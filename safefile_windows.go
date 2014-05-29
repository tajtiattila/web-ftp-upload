// +build windows

package main

import (
	"io"
	"os"
)

type bfile struct {
	n string
	f *os.File
}

const (
	tmp_suffix    = ".tmp"
	backup_suffix = "~"
)

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
	_, err = os.Stat(f.n)
	success := false
	switch {
	case err == nil:
		_, err = os.Stat(f.n + backup_suffix)
		switch {
		case err == nil:
			err = os.Remove(f.n + backup_suffix)
			if err != nil {
				return
			}
		case os.IsNotExist(err):
			// no op
		default:
			return
		}
		err = os.Rename(f.n, f.n+backup_suffix)
		if err != nil {
			return
		}
		defer func() {
			if success {
				// remove backup
				err = os.Remove(f.n + backup_suffix)
			} else {
				// try to restore backup
				err = os.Rename(f.n+backup_suffix, f.n)
			}
		}()
	case os.IsNotExist(err):
		// nothing to back up
	default:
		return
	}
	err = os.Rename(f.n+tmp_suffix, f.n)
	success = err == nil
	return
}
