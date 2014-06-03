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
	fmt.Fprintln(os.Stderr, v...)
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

	maxcachesize, err := parsebytesize(config.MaxCacheSize)
	if err != nil {
		die("config.MaxCacheSize:", err)
	}

	err = inituploader(config.FTPUrl, maxcachesize)
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

func inituploader(url string, siz int64) (err error) {
	cachedir, err = OpenCacheDir("", siz)
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

func parsebytesize(s string) (int64, error) {
	var value int64
	// 0 start
	// 1 number
	// 2 after number
	// 3 had suffix char (state-2)
	state, power, mult := 0, 0, 1000
	for _, ch := range s {
		switch ch {
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			if state == 0 {
				state++
			}
			if state != 1 {
				return 0, invalidsize(s)
			}
			value = value*10 + int64(ch-'0')
		case ' ':
			if state == 1 {
				state++
			}
		case 't', 'g', 'm', 'k', 'T', 'G', 'M', 'K':
			if state == 0 || state > 2 {
				return 0, invalidsize(s)
			}
			switch ch {
			case 't', 'T':
				power = 4
			case 'g', 'G':
				power = 3
			case 'm', 'M':
				power = 2
			case 'k', 'K':
				power = 1
			}
			state = 3
		case 'i':
			if state != 3 {
				return 0, invalidsize(s)
			}
			mult = 1024
			state = 4
		case 'b', 'B':
			if state > 4 {
				return 0, invalidsize(s)
			}
			state = 5
		default:
			return 0, invalidsize(s)
		}
	}
	if state == 0 {
		return 0, invalidsize(s)
	}
	for power > 0 {
		value *= int64(mult)
		power--
	}
	return value, nil
}

func invalidsize(s string) error {
	return fmt.Errorf("invalid size %s", s)
}

type ByteSize int64

func (b *ByteSize) UnmarshalJSON(p []byte) error {
	fmt.Println(string(p))
	*b = 8192 * 1024 * 1024
	return nil
}

type config struct {
	MaxCacheSize string
	Title        map[Language]string
	FTPUrl       string
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
