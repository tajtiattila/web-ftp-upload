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
	err := inituploader(FTP_URL)
	if err != nil {
		die("can't init uploader", err)
	}

	addr := flag.String("addr", "", `address to listen on, eg. ":8080"`)
	sock := flag.String("sock", "", `socket file to listen on`)
	ext := flag.String("ext", "ext", `directory for external files`)
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

	var l net.Listener
	if *addr != "" {
		l, err = net.Listen("tcp", *addr)
	} else {
		l, err = net.Listen("unix", *sock)
	}
	check(err)
	defer l.Close()

	m := http.NewServeMux()
	m.HandleFunc("/web-ftp-upload/", handleUpload)
	m.Handle("/web-ftp-upload/ext/", http.StripPrefix("/web-ftp-upload/ext/", http.FileServer(http.Dir(*ext))))
	if *usefcgi {
		err = fcgi.Serve(l, m)
	} else {
		err = http.Serve(l, m)
	}
	check(err)
}
