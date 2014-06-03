package main

import (
	"fmt"
	"io"
)

type WriteLimiter interface {
	AllocBytes(n int) bool
	FreeBytes(n int)
}

type LimitWriter struct {
	w     io.Writer
	l     WriteLimiter
	avail int
	alloc int
}

func NewLimitWriter(w io.Writer, l WriteLimiter) *LimitWriter {
	return &LimitWriter{w: w, l: l}
}

func (w *LimitWriter) Write(p []byte) (n int, err error) {
	if w.avail < 0 {
		return 0, fmt.Errorf("write after buffer full/finish")
	}
	if w.avail < len(p) {
		needed := 1024 * 1024
		if needed < len(p) {
			needed = len(p)
		}
		if w.l.AllocBytes(needed) {
			w.alloc += needed
			w.avail += needed
		} else {
			w.avail = -1
			return 0, fmt.Errorf("buffer full")
		}
	}
	n, err = w.w.Write(p)
	w.avail -= n
	return
}

func (w *LimitWriter) Finish() {
	if w.avail > 0 {
		// free unused bytes
		w.l.FreeBytes(w.avail)
		w.alloc -= w.avail
	}
	w.avail = -1
}

func (w *LimitWriter) Free() {
	w.l.FreeBytes(w.alloc)
	w.alloc = 0
}
