package main

import (
	"io"
	"io/ioutil"
	"os"
)

func cache(r io.Reader) (content CachedContent, err error) {
	var cf *CacheFile
	if cf, err = NewCacheFile("", "upload-cache"); err != nil {
		return
	}
	defer func() {
		if err != nil {
			cf.Close()
		}
	}()
	_, err = io.Copy(cf, r)
	if err != nil {
		return
	}
	return cf, nil
}

func upload(user, filename string, r io.Reader) error {
	content, err := cache(r)
	if err != nil {
		return err
	}
	return up.Add(user, filename, content)
}

var up *Uploader

func inituploader(url string) (err error) {
	up, err = NewUploader(url)
	return
}

type CachedContent interface {
	io.Reader
	io.Writer
	io.Seeker
	io.Closer
	Size() (int64, error)
}

type CacheFile struct {
	f *os.File
}

func NewCacheFile(dir, prefix string) (*CacheFile, error) {
	f, err := ioutil.TempFile(dir, prefix)
	if err != nil {
		return nil, err
	}
	return &CacheFile{f}, nil
}

func (t *CacheFile) Read(p []byte) (int, error)                   { return t.f.Read(p) }
func (t *CacheFile) Write(p []byte) (int, error)                  { return t.f.Write(p) }
func (t *CacheFile) Seek(offset int64, whence int) (int64, error) { return t.f.Seek(offset, whence) }
func (t *CacheFile) Close() error {
	n := t.f.Name()
	err1 := t.f.Close()
	err2 := os.Remove(n)
	if err1 != nil {
		return err1
	}
	return err2
}
func (t *CacheFile) Size() (int64, error) {
	fi, err := t.f.Stat()
	return fi.Size(), err
}

func CacheDir(name string) (string, error) {
	c := os.Getenv("XDG_CACHE_HOME")
	if c == "" {
		h := os.Getenv("HOME")
		if h == "" {
			h = "/"
		}
		c = h + ".cache"
	}
	c += "/" + name
	return c, os.MkdirAll(c, 0700)
}
