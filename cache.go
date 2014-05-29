package main

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// CacheDir is a directory holding temporary/cached files.
type CacheDir struct {
	Path     string
	ByteSize int64
	elems    map[*CacheEntry]bool
	velems   []*CacheEntry
	log      *log.Logger
	mtx      sync.RWMutex
}

// OpenCacheDir opens or creates a cache directory, and
// loads already cached content, if available.
func OpenCacheDir(name string) (*CacheDir, error) {
	p, err := GetCacheDir(name)
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
	cachedname, siz, err := d.cachecontent(r)
	if err != nil {
		return nil, err
	}

	d.mtx.Lock()
	d.ByteSize += siz
	e := &CacheEntry{d, user, filename, cachedname, siz}
	d.elems[e] = true
	d.log.Println("Added", e.Fn, "for", e.Un, "as", filepath.Base(e.Cn))
	d.save()
	d.mtx.Unlock()

	notifier.notify(user)
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

// Userfiles returns a new lists of cached files for the user.
func (d *CacheDir) Userfiles(user string) (r []string) {
	user = strings.ToLower(user)
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	for e := range d.elems {
		if strings.ToLower(e.Un) == user {
			r = append(r, e.Fn)
		}
	}
	return
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
	err = os.Remove(e.Cn)

	d.mtx.Lock()
	d.ByteSize -= e.Siz
	delete(d.elems, e)
	d.save()
	d.mtx.Unlock()

	d.log.Println("Removed", e.Fn, "for", e.Un, "as", filepath.Base(e.Cn))
	if err != nil {
		d.log.Println("Remove failed:", err)
	}

	notifier.notify(e.Un)
	return
}

func (d *CacheDir) load() error {
	f, err := os.Open(d.datafilename())
	switch {
	case err == nil:
		defer f.Close()
		err = json.NewDecoder(f).Decode(&d.velems)
	case os.IsNotExist(err):
		err = nil
	}
	if err != nil {
		d.log.Println("Load error:", err)
		return err
	}
	d.ByteSize = 0
	d.elems = make(map[*CacheEntry]bool)
	if len(d.velems) == 0 {
		return err
	}
	for _, e := range d.velems {
		e.dir = d
		if _, xerr := os.Stat(e.Cn); xerr == nil {
			// add only existing files
			d.ByteSize += e.Siz
			d.elems[e] = true
		}
	}
	d.log.Println("Loaded cache of", d.ByteSize, "bytes in", len(d.elems), "files")
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
	if cap(d.velems) < len(d.elems) {
		d.velems = make([]*CacheEntry, 0, 2*len(d.elems))
	} else {
		d.velems = d.velems[:0]
	}
	for e := range d.elems {
		d.velems = append(d.velems, e)
	}
	f, err := SafeFileWriter(d.datafilename())
	if err == nil {
		defer func() {
			err = f.Close()
			if err != nil {
				d.log.Println("Can't close:", err)
			}
		}()
		err = json.NewEncoder(f).Encode(d.velems)
	}
	if err != nil {
		d.log.Println("Can't save:", err)
	}
}

func (d *CacheDir) datafilename() string {
	return d.Path + "/cachefiles.json"
}

type CachedFile interface {
	User() string
	Filename() string
	Open() (io.ReadCloser, error)
	Discard() error
	Size() int64
}

type CacheEntry struct {
	dir *CacheDir
	Un  string
	Fn  string
	Cn  string
	Siz int64
}

func (e *CacheEntry) User() string                 { return e.Un }
func (e *CacheEntry) Filename() string             { return e.Fn }
func (e *CacheEntry) Open() (io.ReadCloser, error) { return e.dir.open(e) }
func (e *CacheEntry) Discard() error               { return e.dir.remove(e) }
func (e *CacheEntry) Size() int64                  { return e.Siz }
