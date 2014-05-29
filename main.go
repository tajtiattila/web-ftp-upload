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
	sock := flag.String("sock", "", `socket file to listen on`)
	prefix := flag.String("pfx", "/web-ftp-upload/", `web server path prefix`)
	ext := flag.String("ext", "ext", `directory for external files`)
	tmpl := flag.String("tmpl", "template", `directory for templates`)
	usefcgi := flag.Bool("fcgi", false, `fastcgi mode`)
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

	var l net.Listener
	if *addr != "" {
		l, err = net.Listen("tcp", *addr)
	} else {
		l, err = net.Listen("unix", *sock)
	}
	check(err)
	defer l.Close()

	ws := NewWebServer(*prefix)
	m := http.NewServeMux()
	m.Handle(*prefix, ws)
	m.Handle(*prefix+"ext/", http.StripPrefix(*prefix+"ext/", http.FileServer(http.Dir(*ext))))
	if *usefcgi {
		err = fcgi.Serve(l, m)
	} else {
		err = http.Serve(l, m)
	}
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
