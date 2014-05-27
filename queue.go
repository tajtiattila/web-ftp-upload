package main

import (
	"sync"
)

type uploadqueue struct {
	mtx  sync.RWMutex
	buf  []*Upload            // circular buffer
	size int                  // number of elements in buffer
	head int                  // read position
	tail int                  // write position
	ch   chan bool            // element added to empty queue
	cb   func(<-chan *Upload) // update callback
}

func newuploadqueue(q []*Upload) *uploadqueue {
	n := 16
	for n < len(q) {
		n *= 2
	}
	buf := make([]*Upload, n)
	copy(buf, q)
	return &uploadqueue{
		buf:  buf,
		size: len(q),
		head: 0,
		tail: len(q),
		ch:   make(chan bool, 1),
	}
}

func (q *uploadqueue) push(u *Upload) {
	q.mtx.Lock()
	defer q.mtx.Unlock()
	if q.head == q.tail && q.size > 0 {
		nsiz := len(q.buf) * 2
		nbuf := make([]*Upload, nsiz)
		copy(nbuf, q.buf[q.head:])
		copy(nbuf[len(q.buf)-q.head:], q.buf[:q.head])
		q.head = 0
		q.tail = len(q.buf)
		q.buf = nbuf
	}
	q.buf[q.tail] = u
	q.tail = (q.tail + 1) % len(q.buf)
	q.size++
	if q.ch != nil && q.size == 1 {
		q.ch <- true
	}
	q.docallback()
}

func (q *uploadqueue) peek() *Upload {
	q.mtx.RLock()
	defer q.mtx.RUnlock()
	if q.size == 0 {
		return nil
	}
	return q.buf[q.head]
}

func (q *uploadqueue) pop() {
	q.mtx.Lock()
	defer q.mtx.Unlock()
	if q.size == 0 {
		return // panic?
	}
	q.size--
	if q.size == 0 {
		q.head, q.tail = 0, 0
		if q.ch != nil {
			select {
			case <-q.ch:
			default:
			}
		}
	} else {
		q.head = (q.head + 1) % len(q.buf)
	}
	q.docallback()
}

func (q *uploadqueue) docallback() {
	if q.cb == nil {
		return
	}
	ch := make(chan *Upload)
	go func() {
		h := q.head
		for i := 0; i < q.size; i++ {
			ch <- q.buf[h]
			h = (h + 1) % len(q.buf)
		}
		close(ch)
	}()
	q.cb(ch)
}
