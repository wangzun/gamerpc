package def

import (
	"bytes"
	"context"
	"sync"
	"sync/atomic"
)

const HandShakerClient string = "*gamer_rpc_client*"
const HandShakerServer string = "*gamer_rpc_server*"

var ErrConnClosing error

type bufferPool struct {
	pool sync.Pool
}

func newBufferPool() *bufferPool {
	return &bufferPool{
		pool: sync.Pool{
			New: func() interface{} {
				return new(bytes.Buffer)
			},
		},
	}
}

func (p *bufferPool) get() *bytes.Buffer {
	return p.pool.Get().(*bytes.Buffer)
}

func (p *bufferPool) put(b *bytes.Buffer) {
	p.pool.Put(b)
}


type recvMsg struct {
	buffer *bytes.Buffer
	err error
}

type recvBuffer struct {
	c       chan recvMsg
	mu      sync.Mutex
	backlog []recvMsg
	err     error
}

func newRecvBuffer() *recvBuffer {
	b := &recvBuffer{
		c: make(chan recvMsg, 1),
	}
	return b
}

func (b *recvBuffer) put(r recvMsg) {
	b.mu.Lock()
	if b.err != nil {
		b.mu.Unlock()
		return
	}
	b.err = r.err
	if len(b.backlog) == 0 {
		select {
		case b.c <- r:
			b.mu.Unlock()
			return
		default:
		}
	}
	b.backlog = append(b.backlog, r)
	b.mu.Unlock()
}

func (b *recvBuffer) load() {
	b.mu.Lock()
	if len(b.backlog) > 0 {
		select {
		case b.c <- b.backlog[0]:
			b.backlog[0] = recvMsg{}
			b.backlog = b.backlog[1:]
		default:
		}
	}
	b.mu.Unlock()
}


func (b *recvBuffer) get() <-chan recvMsg {
	return b.c
}


type recvBufferReader struct {
	ctx         context.Context
	ctxDone     <-chan struct{}
	recv        *recvBuffer
	last        *bytes.Buffer
	err         error
	freeBuffer  func(*bytes.Buffer)
}


func (r *recvBufferReader) Read(p []byte) (n int, err error) {
	if r.err != nil {
		return 0, r.err
	}
	if r.last != nil {
		copied, _ := r.last.Read(p)
		if r.last.Len() == 0 {
			r.freeBuffer(r.last)
			r.last = nil
		}
		return copied, nil
	}
	n, r.err = r.read(p)
	return n, r.err
}

func (r *recvBufferReader) read(p []byte) (n int, err error) {
	select {
	case <-r.ctxDone:
		return 0, r.ctx.Err()
	case m := <-r.recv.get():
		return r.readAdditional(m, p)
	}
}

func (r *recvBufferReader) readAdditional(m recvMsg, p []byte) (n int, err error) {
	r.recv.load()
	if m.err != nil {
		return 0, m.err
	}
	copied, _ := m.buffer.Read(p)
	if m.buffer.Len() == 0 {
		r.freeBuffer(m.buffer)
		r.last = nil
	} else {
		r.last = m.buffer
	}
	return copied, nil
}



type itemNode struct {
	it   interface{}
	next *itemNode
}

type itemList struct {
	head *itemNode
	tail *itemNode
}

func (il *itemList) enqueue(i interface{}) {
	n := &itemNode{it: i}
	if il.tail == nil {
		il.head, il.tail = n, n
		return
	}
	il.tail.next = n
	il.tail = n
}

func (il *itemList) peek() interface{} {
	return il.head.it
}

func (il *itemList) dequeue() interface{} {
	if il.head == nil {
		return nil
	}
	i := il.head.it
	il.head = il.head.next
	if il.head == nil {
		il.tail = nil
	}
	return i
}

func (il *itemList) dequeueAll() *itemNode {
	h := il.head
	il.head, il.tail = nil, nil
	return h
}

func (il *itemList) isEmpty() bool {
	return il.head == nil
}

type controlBuffer struct {
	ch              chan struct{}
	done            <-chan struct{}
	mu              sync.Mutex
	consumerWaiting bool
	list            *itemList
	err             error
	trfChan         atomic.Value // *chan struct{}
}

func newControlBuffer(done <-chan struct{}) *controlBuffer {
	return &controlBuffer{
		ch:   make(chan struct{}, 1),
		list: &itemList{},
		done: done,
	}
}

func (c *controlBuffer) throttle() {
	ch, _ := c.trfChan.Load().(*chan struct{})
	if ch != nil {
		select {
		case <-*ch:
		case <-c.done:
		}
	}
}

func (c *controlBuffer) put(it interface{}) error {
	_, err := c.executeAndPut(nil, it)
	return err
}

func (c *controlBuffer) executeAndPut(f func(it interface{}) bool, it interface{}) (bool, error) {
	var wakeUp bool
	c.mu.Lock()
	if c.err != nil {
		c.mu.Unlock()
		return false, c.err
	}
	if f != nil {
		if !f(it) { // f wasn't successful
			c.mu.Unlock()
			return false, nil
		}
	}
	if c.consumerWaiting {
		wakeUp = true
		c.consumerWaiting = false
	}
	c.list.enqueue(it)
	c.mu.Unlock()
	if wakeUp {
		select {
		case c.ch <- struct{}{}:
		default:
		}
	}
	return true, nil
}

func (c *controlBuffer) execute(f func(it interface{}) bool, it interface{}) (bool, error) {
	c.mu.Lock()
	if c.err != nil {
		c.mu.Unlock()
		return false, c.err
	}
	if !f(it) { // f wasn't successful
		c.mu.Unlock()
		return false, nil
	}
	c.mu.Unlock()
	return true, nil
}

func (c *controlBuffer) get(block bool) (interface{}, error) {
	for {
		c.mu.Lock()
		if c.err != nil {
			c.mu.Unlock()
			return nil, c.err
		}
		if !c.list.isEmpty() {
			h := c.list.dequeue()
			c.mu.Unlock()
			return h, nil
		}
		if !block {
			c.mu.Unlock()
			return nil, nil
		}
		c.consumerWaiting = true
		c.mu.Unlock()
		select {
		case <-c.ch:
		case <-c.done:
			c.finish()
			return nil, ErrConnClosing
		}
	}
}

func (c *controlBuffer) finish() {
	c.mu.Lock()
	if c.err != nil {
		c.mu.Unlock()
		return
	}
	c.err = ErrConnClosing
	for head := c.list.dequeueAll(); head != nil; head = head.next {
		//todo
	}
	c.mu.Unlock()
}




