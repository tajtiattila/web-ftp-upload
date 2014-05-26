package main

import (
	"io"
	"io/ioutil"
	"os"
)

type Uploader struct {
	CacheLimit int64
	Prefix     string
	ch         chan *upload
}

type upload struct {
	filename string
	content  io.ReadCloser
}

func NewUploader() *Uploader {
	return &Uploader{CacheLimit: 8 * 1024 * 1024 * 1024, Prefix: "upload-cache", ch: make(chan *upload)}
}

func (u *Uploader) Add(fn string, r io.Reader) (err error) {
	var f *os.File
	if f, err = ioutil.TempFile("", u.Prefix); err != nil {
		return
	}
	defer func() {
		if err != nil {
			f.Close()
			os.Remove(f.Name())
		}
	}()
	_, err = io.Copy(f, r)
	if err != nil {
		return
	}
	_, err = f.Seek(0, 0)
	if err != nil {
		return
	}
	//u.ch<- upload{fn, f}
	return nil
}
