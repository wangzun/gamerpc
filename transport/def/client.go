package def

import (
	"context"
	"errors"
	"net"
	"sync"
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
	}
	t.controlBuf = newControlBuffer(t.ctxDone)
	go t.read()
	go t.write()
	return t,nil
}

func (t *Client) read() {
	bin := make([]byte,len(HandShakerServer))
	_,err := t.conn.Read(bin)
	if err != nil {
		t.onClose(err)
		return
	}
	if string(bin) != HandShakerServer {
		t.onClose(errors.New("hand shaker fail"))
		return
	}
	t.onPrefaceReceipt()
}

func (t *Client) write() {
	_,err := t.conn.Write([]byte(HandShakerClient))
	if err != nil {
		t.onClose(err)
		return
	}
}

