package def

import (
	"context"
	"net"
	"reflect"
	"sync"
	"testing"
)



func TestNewClient(t *testing.T) {
	type args struct {
		ctx              context.Context
		addr             string
		onPrefaceReceipt func()
		onClose          func(error)
	}
	tests := []struct {
		name    string
		args    args
		wantCli *Client
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCli, err := NewClient(tt.args.ctx, tt.args.addr, tt.args.onPrefaceReceipt, tt.args.onClose)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotCli, tt.wantCli) {
				t.Errorf("NewClient() = %v, want %v", gotCli, tt.wantCli)
			}
		})
	}
}

func TestClient_getStream(t *testing.T) {
	type fields struct {
		mu               sync.Mutex
		ctx              context.Context
		cancel           context.CancelFunc
		ctxDone          <-chan struct{}
		conn             net.Conn
		remoteAddr       net.Addr
		localAddr        net.Addr
		readerDone       chan struct{}
		writerDone       chan struct{}
		controlBuf       *controlBuffer
		onPrefaceReceipt func()
		onClose          func(error)
		activeStreams    map[uint32]*Stream
		bufferPool       *bufferPool
		connectionID     uint64
		maxStreamId      uint32
		wbuf             []byte
	}
	type args struct {
		id uint32
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *Stream
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &Client{
				mu:               tt.fields.mu,
				ctx:              tt.fields.ctx,
				cancel:           tt.fields.cancel,
				ctxDone:          tt.fields.ctxDone,
				conn:             tt.fields.conn,
				remoteAddr:       tt.fields.remoteAddr,
				localAddr:        tt.fields.localAddr,
				readerDone:       tt.fields.readerDone,
				writerDone:       tt.fields.writerDone,
				controlBuf:       tt.fields.controlBuf,
				onPrefaceReceipt: tt.fields.onPrefaceReceipt,
				onClose:          tt.fields.onClose,
				activeStreams:    tt.fields.activeStreams,
				bufferPool:       tt.fields.bufferPool,
				connectionID:     tt.fields.connectionID,
				maxStreamId:      tt.fields.maxStreamId,
				wbuf:             tt.fields.wbuf,
			}
			if got := tr.getStream(tt.args.id); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Client.getStream() = %v, want %v", got, tt.want)
			}
		})
	}
}
