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
	Path    string `json:"-"`
	MaxSize int64  `json:"-"`
	size    int64  `json:"-"`
	Entries []*CacheEntry
	scratch []*CacheEntry
	log     *log.Logger
	mtx     sync.RWMutex
}

// OpenCacheDir opens or creates a cache directory, and
// loads already cached content, if available.
func OpenCacheDir(name string, maxsiz int64) (*CacheDir, error) {
	p, err := GetCacheDir(name)
	if err != nil {
		return nil, err
	}
	d := &CacheDir{
		Path:    p,
		MaxSize: maxsiz,
		log:     log.New(os.Stderr, "CACHE   ", log.LstdFlags),
	}
	d.log.Println("initializing in", d.Path, "with limit", d.MaxSize)
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
	// d.size is already increased in cachecontent/LimitWriter
	e := &CacheEntry{d, user, filename, cachedname, siz}
	d.Entries = append(d.Entries, e)
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
		for _, e := range d.Entries {
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
	for _, e := range d.Entries {
		if strings.ToLower(e.Un) == user {
			r = append(r, e.Fn)
		}
	}
	return
}

func (d *CacheDir) Size() int64 {
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	return d.size
}

func (d *CacheDir) cachecontent(r io.Reader) (cachedname string, siz int64, err error) {
	var f *os.File
	if f, err = ioutil.TempFile(d.Path, "cache-"); err != nil {
		return
	}
	cachedname = f.Name()
	w := NewLimitWriter(f, d)
	defer func() {
		f.Close()
		w.Finish()
		if err != nil {
			if os.Remove(cachedname) == nil {
				w.Free()
			}
		}
	}()
	siz, err = io.Copy(w, r)
	return
}

func (d *CacheDir) open(e *CacheEntry) (io.ReadCloser, error) {
	return os.Open(e.Cn)
}

func (d *CacheDir) remove(old *CacheEntry) (err error) {
	err = os.Remove(old.Cn)

	d.mtx.Lock()
	d.size -= old.Siz
	d.filterentries(func(e *CacheEntry) bool {
		return e != old
	})
	d.save()
	d.mtx.Unlock()

	d.log.Println("Removed", old.Fn, "for", old.Un, "as", filepath.Base(old.Cn))
	if err != nil {
		d.log.Println("Remove failed:", err)
	}

	notifier.notify(old.Un)
	return
}

func (d *CacheDir) clearoldfiles() error {
	dir, err := os.Open(d.Path)
	if err != nil {
		return err
	}
	nold, nclear := 0, 0
	names, err := dir.Readdirnames(0)
	if err != nil {
		return err
	}
	for _, fn := range names {
		if len(fn) < 6 || fn[:6] != "cache-" {
			continue
		}
		old := true
		for _, e := range d.Entries {
			if fn == filepath.Base(e.Cn) {
				old = false
				break
			}
		}
		if old {
			nold++
			errx := os.Remove(filepath.Join(d.Path, fn))
			if err == nil {
				err = errx
			} else {
				nclear++
			}
		}
	}
	if nold != 0 {
		if nclear == nold {
			d.log.Println(nold, "incomplete/invalid files were cleared")
		} else {
			d.log.Println(nold, "incomplete/invalid files found, of which", nclear, "could be cleared")
		}
	}
	return err
}

func (d *CacheDir) load() error {
	f, err := os.Open(d.datafilename())
	switch {
	case err == nil:
		defer f.Close()
		err = json.NewDecoder(f).Decode(&d)
	case os.IsNotExist(err):
		err = nil
	}
	if err != nil {
		d.log.Println("Load error:", err)
		return err
	}
	d.filterentries(func(e *CacheEntry) bool {
		// keep only existing files
		_, xerr := os.Stat(e.Cn)
		return xerr == nil
	})
	d.size = 0
	for _, e := range d.Entries {
		e.dir = d
		d.size += e.Siz
	}
	d.log.Println("Loaded cache of", d.size, "bytes in", len(d.Entries), "files")
	if errc := d.clearoldfiles(); errc != nil {
		d.log.Println("error clearing old files:", errc)
	}
	return err
}

func (d *CacheDir) save() {
	if len(d.Entries) == 0 {
		err := os.Remove(d.datafilename())
		if err != nil {
			d.log.Println("Can't cleanup:", err)
		}
		return
	}
	f, err := SafeFileWriter(d.datafilename())
	if err == nil {
		defer func() {
			err = f.Close()
			if err != nil {
				d.log.Println("Can't close:", err)
			}
		}()
		err = json.NewEncoder(f).Encode(d)
	}
	if err != nil {
		d.log.Println("Can't save:", err)
	}
}

func (d *CacheDir) filterentries(f func(e *CacheEntry) bool) {
	d.scratch = d.scratch[:0]
	for _, e := range d.Entries {
		if f(e) {
			d.scratch = append(d.scratch, e)
		}
	}
	d.Entries, d.scratch = d.scratch, d.Entries
}

func (d *CacheDir) datafilename() string {
	return d.Path + "/cachefiles.json"
}

func (d *CacheDir) AllocBytes(n int) bool {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	if d.size+int64(n) <= d.MaxSize {
		d.size += int64(n)
		return true
	}
	d.log.Println("buffer full: couldn't allocate", n, "bytes")
	return false
}

func (d *CacheDir) FreeBytes(n int) {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	d.size -= int64(n)
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
