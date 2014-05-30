package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/http/fcgi"
	"os"
)

func die(v ...interface{}) {
	fmt.Fprint(os.Stderr, v...)
	os.Exit(1)
}

func check(err error) {
	if err != nil {
		die(err)
	}
}

func main() {
	addr := flag.String("addr", "", `address to listen on, eg. ":8080"`)
	sock := flag.String("fcgi", "", `fastcgi file to listen on`)
	prefix := flag.String("pfx", "/web-ftp-upload", `web server path prefix`)
	ext := flag.String("ext", "ext", `directory for external files`)
	tmpl := flag.String("tmpl", "template", `directory for templates`)
	flag.Parse()

	if flag.NArg() != 0 {
		die("no positional arguments allowed")
	}

	if *addr != "" && *sock != "" {
		die("-sock and -addr are mutually exclusive")
	}

	if *addr == "" && *sock == "" {
		die("either -sock or -addr necessary")
	}

	err := readtemplates(*tmpl)
	if err != nil {
		die(err)
	}

	err = inituploader(FTP_URL)
	if err != nil {
		die("can't init uploader", err)
	}

	var (
		listener net.Listener
		servefn  func(net.Listener, http.Handler) error
	)
	if *addr != "" {
		listener, err = net.Listen("tcp", *addr)
		servefn = http.Serve
	} else {
		listener, err = net.Listen("unix", *sock)
		servefn = fcgi.Serve
	}
	check(err)
	defer listener.Close()

	server := NewWebServer(*prefix, *ext)
	servefn(listener, server)
	check(err)
}

var (
	uploader *Uploader
	cachedir *CacheDir
)

func inituploader(url string) (err error) {
	cachedir, err = OpenCacheDir("")
	if err != nil {
		return
	}
	uploader, err = NewUploader(url)
	if err != nil {
		return
	}
	for cached := range cachedir.Files() {
		err = uploader.Add(cached)
		if err != nil {
			cached.Discard()
		}
	}
	return
}
