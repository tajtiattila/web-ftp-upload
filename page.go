package main

import (
	"sort"
	"strings"
)

type InfoPage struct {
	Name        string
	Donefiles   []string
	Cachedfiles []string
	QueueSize   int64
	QueueLoad   int
}

func NewInfoPage(user string) *InfoPage {
	if user == "" {
		return nil
	}
	p := &InfoPage{}
	p.Name = strings.Title(user)
	p.Donefiles = uploader.Userfiles(user)
	sort.Strings(p.Donefiles)
	p.Cachedfiles = cachedir.Userfiles(user)
	sort.Strings(p.Donefiles)
	p.QueueSize = cachedir.Size()
	p.QueueLoad = int(p.QueueSize * 100 / cachedir.MaxSize)
	return p
}
