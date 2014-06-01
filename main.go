package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
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
	sock := flag.String("sock", "", `file to listen on`)
	prefix := flag.String("pfx", "/web-ftp-upload", `web server path prefix`)
	wdir := flag.String("dir", ".", `directory for template and external files`)
	cfg := flag.String("cfg", "config.json", `config file`)
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

	config, err := readconfig(*cfg)
	if err != nil {
		die(err)
	}

	err = readtemplates(*wdir+"/template", config.Title)
	if err != nil {
		die(err)
	}

	err = inituploader(config.FTPUrl)
	if err != nil {
		die("can't init uploader", err)
	}

	var listener net.Listener
	if *addr != "" {
		listener, err = net.Listen("tcp", *addr)
	} else {
		listener, err = net.Listen("unix", *sock)
	}
	check(err)
	defer listener.Close()

	server := NewWebServer(*prefix, *wdir+"/ext")
	http.Serve(listener, server)
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

type config struct {
	Title  map[Language]string
	FTPUrl string
}

func readconfig(fn string) (*config, error) {
	f, err := os.Open(fn)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	c := new(config)
	err = json.NewDecoder(f).Decode(c)
	if err != nil {
		return nil, err
	}
	return c, err
}
