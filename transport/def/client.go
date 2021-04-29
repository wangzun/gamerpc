package def

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type Client struct {
	mu         sync.Mutex // guard the following variables
	ctx        context.Context
	cancel     context.CancelFunc
	ctxDone    <-chan struct{} // Cache the ctx.Done() chan.
	conn       net.Conn
	remoteAddr net.Addr
	localAddr  net.Addr
	readerDone chan struct{} // sync point to enable testing.
	writerDone chan struct{} // sync point to enable testing.
	controlBuf *controlBuffer
	onPrefaceReceipt func()
	onClose    func(error)
	activeStreams map[uint32]*Stream
	bufferPool *bufferPool
	connectionID uint64
	maxStreamId  uint32
	wbuf        []byte
}

func NewClient(ctx context.Context,addr string,onPrefaceReceipt func(),onClose func(error)) (cli *Client,err error) {
	ctx, cancel := context.WithCancel(ctx)
	defer func() {
		if err != nil {
			cancel()
		}
	}()
	connectCtx ,connectCancel := context.WithTimeout(context.Background(),10*time.Second)
	conn,err := (&net.Dialer{}).DialContext(connectCtx, "tcp", addr)
	if err != nil {
		connectCancel()
		_ = conn.Close()
		return
	}
	t := &Client{
		ctx:                   ctx,
		ctxDone:               ctx.Done(), // Cache Done chan.
		cancel:                cancel,
		conn:                  conn,
		remoteAddr:            conn.RemoteAddr(),
		localAddr:             conn.LocalAddr(),
		readerDone:            make(chan struct{}),
		writerDone:            make(chan struct{}),
		activeStreams:         make(map[uint32]*Stream),
		onPrefaceReceipt:      onPrefaceReceipt,
		bufferPool:            newBufferPool(),
		onClose:               onClose,
		wbuf:                  make([]byte,0),
	}
	t.controlBuf = newControlBuffer(t.ctxDone)
	go t.readPool()
	go t.writePool()
	return t,nil
}

func (t *Client) getStream(id uint32) *Stream {
	t.mu.Lock()
	s := t.activeStreams[id]
	t.mu.Unlock()
	return s
}

func (t *Client) setStream(stream *Stream) {
	t.mu.Lock()
	t.activeStreams[stream.id] = stream
	t.mu.Unlock()
}

func (t *Client) Close(err error) {
	for _,stream := range t.activeStreams {
		stream.recv.put(recvMsg{err:err})
	}
}

func (t *Client) readPool() {
	var err error
	defer func() {
		if err != nil {
			_ = t.conn.Close()
			t.Close(err)
			if t.onClose != nil {
				t.onClose(err)
			}
		}
	}()
	bin := make([]byte,len(HandShakerServer))
	_,err = t.conn.Read(bin)
	if err != nil {
		return
	}
	if string(bin) != HandShakerServer {
		err = errors.New("hand shaker fail")
		return
	}
	t.onPrefaceReceipt()
	headBin := make([]byte,FramePreFix)
	for {
		_,err := io.ReadFull(t.conn,headBin)
		if err != nil {
			return
		}
		frame ,err := getframeHeadFromBin(headBin)
		if err != nil {
			return
		}
		if frame.typ == Cast || frame.typ == CallReq {
			err = errors.New("frame type err")
			return
		}
		streamId := frame.streamId
		stream := t.getStream(streamId)
		if stream == nil {
			str := fmt.Sprintf("no stream in client %d",streamId)
			err = errors.New(str)
			return
		}
		frameData := make([]byte,frame.framelen)
		_,err = io.ReadFull(t.conn,frameData)
		if err != nil {
			return
		}
		if frame.flag.Has(FlagHead){

		}
		if frame.flag.Has(FlagSetting)  {

		}
		if frame.flag.Has(FlagData) {
			buffer := t.bufferPool.get()
			buffer.Reset()
			buffer.Write(frameData)
			stream.recv.put(recvMsg{buffer: buffer})
		}
		if frame.flag.Has(FlagEnd) {
			delete(t.activeStreams,streamId)
		}
	}
}

func (t *Client) writePool() {
	var err error
	defer func() {
		if err != nil {
			_ = t.conn.Close()
			t.Close(err)
			if t.onClose != nil {
				t.onClose(err)
			}
		}
	}()
	_ ,err = t.conn.Write([]byte(HandShakerClient))
	if err != nil {
		return
	}
	for {
		it, err := t.controlBuf.get(true)
		if err != nil {
			return
		}
		streamOut := it.(*StreamOut)
		frameHead := &frameHead{streamId:streamOut.id}
		freamHeadBin := make([]byte,8)
		t.wbuf = append(t.wbuf[:0],freamHeadBin...)
		var length uint16
		if len(streamOut.hdr) != 0 {
			frameHead.flag = frameHead.flag |FlagHead
			length = length + uint16(2 +len(streamOut.hdr))
			bin := make([]byte,2)
			binary.BigEndian.PutUint16(bin,uint16(len(streamOut.hdr)))
			t.wbuf = append(t.wbuf,bin...)
			t.wbuf = append(t.wbuf,streamOut.hdr...)
			streamOut.hdr = streamOut.hdr[:0]
		}
		if len(streamOut.data) != 0 {
			frameHead.flag = frameHead.flag |FlagData
			if len(streamOut.data) > MaxFrameLen {
				length = length + uint16(2+MaxFrameLen)
				bin := make([]byte,2)
				binary.BigEndian.PutUint16(bin,uint16(MaxFrameLen))
				t.wbuf = append(t.wbuf,bin...)
				t.wbuf = append(t.wbuf,streamOut.data[0:MaxFrameLen]...)
				streamOut.data = streamOut.data[MaxFrameLen:]
				err  = t.controlBuf.put(streamOut)
				if err != nil {
					return
				}
			}else {
				frameHead.flag = frameHead.flag |FlagEnd
				length = length + uint16(2+len(streamOut.data))
				bin := make([]byte,2)
				binary.BigEndian.PutUint16(bin,uint16(len(streamOut.data)))
				t.wbuf = append(t.wbuf,bin...)
				t.wbuf = append(t.wbuf,streamOut.data...)
				streamOut.data = streamOut.data[:0]
			}
		}else{
			err = errors.New("empty data")
			return
		}
		frameHead.framelen = length
		frameHead.typ = CallReq
		getframeHeadBin(frameHead,t.wbuf[0:8])
		_,err = t.conn.Write(t.wbuf)
		if err != nil {
			return
		}
	}
}

type Options struct {
	typ uint8
}

type StreamOut struct {
	id   uint32
	hdr  []byte
	data []byte
	typ  uint8
}

func (t *Client) newStreamId() uint32 {
	return atomic.AddUint32(&t.maxStreamId,1)
}

func (t *Client) Write(ctx context.Context,hdr []byte, data []byte, opts *Options) (*Stream,error) {
	if len(hdr) == 0  || len(data) == 0 {
		return nil,errors.New("data nil")
	}
	buf:= newRecvBuffer()
	s := &Stream{
		recv: buf,
		reader: &recvBufferReader{
			ctx:     ctx,
			ctxDone: ctx.Done(),
			recv:    buf,
			freeBuffer: t.bufferPool.put,
		},
	}
	id := t.newStreamId()
	s.id = id
	t.setStream(s)
	out := &StreamOut{
		id:id,
		hdr:hdr,
		data:data,
		typ:CallReq,
	}
	if err := t.controlBuf.put(out);err != nil {
		return nil,err
	}
	return s,nil
}


