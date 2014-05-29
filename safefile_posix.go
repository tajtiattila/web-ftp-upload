// +build !windows

package main

import (
	"io"
	"os"
)

type bfile struct {
	n string
	f *os.File
}

const tmp_suffix = ".tmp"

func SafeFileWriter(filename string) (io.WriteCloser, error) {
	f, err := os.Create(filename + tmp_suffix)
	if err != nil {
		return nil, err
	}
	return &bfile{filename, f}, nil
}

func (f *bfile) Write(p []byte) (int, error) {
	return f.f.Write(p)
}

func (f *bfile) Close() (err error) {
	err = f.f.Close()
	if err != nil {
		return
	}
	// we have atomic rename
	return os.Rename(f.n+tmp_suffix, f.n)
}
