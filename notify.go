package main

import (
	"sync"
)

var nofifier = infopagenotifier{m: make(map[string]*userhandler)}

type infopagenotifier struct {
	mtx sync.RWMutex
	m   map[string]*userhandler
}

func (n *infopagenotifier) listen(user string) (chfiles <-chan *InfoPage, chquit chan<- bool) {
	n.mtx.RLock()
	h, ok := n.m[user]
	n.mtx.RUnlock()
	if !ok {
		n.mtx.Lock()
		if h, ok = n.m[user]; !ok {
			h = handlelisteners(user)
			n.m[user] = h
		}
		n.mtx.Unlock()
	}
	chf, chq := make(chan *InfoPage), make(chan bool, 1)
	h.chreg <- &listener{chf, chq}
	return chf, chq
}

func (n *infopagenotifier) notify(user string) {
	n.mtx.RLock()
	h, ok := n.m[user]
	n.mtx.RUnlock()
	if ok {
		h.chupd <- true
	}
}

var notifier = infopagenotifier{m: make(map[string]*userhandler)}

type userhandler struct {
	chreg chan<- *listener
	chupd chan<- bool
}

func handlelisteners(user string) *userhandler {
	chreg := make(chan *listener)
	chupd := make(chan bool)
	go func() {
		m := make(map[*listener]bool)
		for {
			select {
			case <-chupd:
				if len(m) == 0 {
					break
				}
				p := NewInfoPage(user)
				for l := range m {
					select {
					case l.chfiles <- p:
					case <-l.chquit:
						close(l.chfiles)
						delete(m, l)
					}
				}
			case l := <-chreg:
				m[l] = true
			}
		}
	}()
	return &userhandler{chreg, chupd}
}

type listener struct {
	chfiles chan *InfoPage
	chquit  chan bool
}
