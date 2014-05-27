package main

import (
	"encoding/gob"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync"
)

// CacheDir is a directory holding temporary/cached files.
type CacheDir struct {
	Path     string
	ByteSize int64
	elems    map[*CacheEntry]bool
	log      *log.Logger
	mtx      sync.Mutex
}

// OpenCacheDir opens or creates a cache directory, and
// loads already cached content, if available.
func OpenCacheDir(name string) (*CacheDir, error) {
	if name == "" {
		name = "upload-cache"
	}
	p := os.Getenv("XDG_CACHE_HOME")
	if p == "" {
		h := os.Getenv("HOME")
		if h == "" {
			h = "/"
		}
		p = h + ".cache"
	}
	p += "/" + name
	err := os.MkdirAll(p, 0700)
	if err != nil {
		return nil, err
	}
	d := &CacheDir{
		Path:  p,
		elems: make(map[*CacheEntry]bool),
		log:   log.New(os.Stderr, "CACHE   ", log.LstdFlags),
	}
	d.log.Println("initializing in", d.Path)
	if err = d.load(); err != nil {
		return nil, err
	}
	return d, nil
}

// Add a new cache entry for the user and filename using the provided io.Reader.
func (d *CacheDir) Add(user, filename string, r io.Reader) (f CachedFile, err error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	cachedname, siz, err := d.cachecontent(r)
	if err != nil {
		return nil, err
	}
	d.ByteSize += siz
	e := &CacheEntry{d, user, filename, cachedname, siz}
	d.elems[e] = true
	d.log.Println("Added", e.Fn, "for", e.Un, "as", filepath.Base(e.Cn))
	d.save()
	return e, nil
}

// Files returns a channel that lists cached files. Useful for
// handling cached, but not yet processed files after a crash or reload.
func (d *CacheDir) Files() <-chan CachedFile {
	ch := make(chan CachedFile)
	go func() {
		for e := range d.elems {
			ch <- e
		}
		close(ch)
	}()
	return ch
}

func (d *CacheDir) cachecontent(r io.Reader) (cachedname string, siz int64, err error) {
	var f *os.File
	if f, err = ioutil.TempFile(d.Path, "cache-"); err != nil {
		return
	}
	cachedname = f.Name()
	defer func() {
		f.Close()
		if err != nil {
			os.Remove(cachedname)
		}
	}()
	siz, err = io.Copy(f, r)
	return
}

func (d *CacheDir) open(e *CacheEntry) (io.ReadCloser, error) {
	return os.Open(e.Cn)
}

func (d *CacheDir) remove(e *CacheEntry) (err error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	d.ByteSize -= e.Siz
	err = os.Remove(e.Cn)
	delete(d.elems, e)
	d.log.Println("Removed", e.Fn, "for", e.Un, "as", filepath.Base(e.Cn))
	if err != nil {
		d.log.Println("Remove failed:", err)
	}
	d.save()
	return
}

func (d *CacheDir) load() error {
	f, err := os.Open(d.datafilename())
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		d.log.Println("Load error:", err)
		return err
	}
	defer f.Close()
	e0 := make(map[*CacheEntry]bool)
	err = gob.NewDecoder(f).Decode(&e0)
	if err == nil {
		d.ByteSize = 0
		for e := range e0 {
			e.Dir = d
			if _, err = os.Stat(e.Cn); err == nil {
				// add only existing files
				d.ByteSize += e.Siz
				d.elems[e] = true
			}
		}
		d.log.Println("Loaded cache of", d.ByteSize, "bytes in", len(d.elems), "files")
	}
	return err
}

func (d *CacheDir) save() {
	if len(d.elems) == 0 {
		err := os.Remove(d.datafilename())
		if err != nil {
			d.log.Println("Can't cleanup:", err)
		}
		return
	}
	f, err := os.Create(d.datafilename())
	if err == nil {
		defer f.Close()
		err = gob.NewEncoder(f).Encode(d.elems)
	}
	if err != nil {
		d.log.Println("Can't save:", err)
	}
}

func (d *CacheDir) datafilename() string {
	return d.Path + "/files.dat"
}

type CachedFile interface {
	User() string
	Filename() string
	Open() (io.ReadCloser, error)
	Discard() error
	Size() int64
}

type CacheEntry struct {
	Dir *CacheDir
	Un  string
	Fn  string
	Cn  string
	Siz int64
}

func (e *CacheEntry) User() string                 { return e.Un }
func (e *CacheEntry) Filename() string             { return e.Fn }
func (e *CacheEntry) Open() (io.ReadCloser, error) { return e.Dir.open(e) }
func (e *CacheEntry) Discard() error               { return e.Dir.remove(e) }
func (e *CacheEntry) Size() int64                  { return e.Siz }
