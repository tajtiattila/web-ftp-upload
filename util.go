package main

import (
	"os"
	"path/filepath"
	"strings"
)

// GetCacheDir creates a cache directory name
// according to XDG Base Directory Specification.
// If name is empty, the base name from os.Args[0]
// without extensions will be used.
// An error is returned when the directory can't
// be accessed or created.
func GetCacheDir(name string) (string, error) {
	if name == "" {
		name = filepath.Base(os.Args[0])
		if n := strings.IndexRune(name, '.'); n != -1 {
			name = name[:n]
		}
	}
	p := os.Getenv("XDG_CACHE_HOME")
	if p == "" {
		h := os.Getenv("HOME")
		if h == "" {
			// if HOME is not set, try the
			// Windows-specific %HOMEPATH%
			h = os.Getenv("HOMEPATH")
		}
		p = filepath.Join(h, ".cache")
	}
	p = filepath.Join(p, name)
	err := os.MkdirAll(p, 0700)
	return p, err
}
