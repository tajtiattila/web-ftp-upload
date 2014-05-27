package main

import (
	"math/rand"
	"net/http"
	"strings"
	"time"
)

type page struct {
	Name      string
	Userfiles []string
	QueueSize int64
}

var sessions = make(map[string]string)

func handlehttp(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "GET":
		var sid string
		if ck, err := req.Cookie("sid"); err == nil {
			sid = ck.Value
		}
		var user string
		if req.FormValue("login") == "" {
			user = sessions[sid]
		} else {
			delete(sessions, sid)
		}
		showPage(w, user)
	case "POST":
		d := req.FormValue("do")
		switch d {
		case "login":
			handleLogin(w, req)
		case "upload":
			handleUpload(w, req)
		default:
			http.Error(w, "Invalid request", http.StatusInternalServerError)
		}
	}
}

func handleLogin(w http.ResponseWriter, req *http.Request) {
	user := req.FormValue("name")
	if user == "" {
		http.Error(w, "Missing name", http.StatusInternalServerError)
		return
	}
	sid := gensid()
	http.SetCookie(w, &http.Cookie{Name: "sid", Value: sid})
	sessions[sid] = user
	http.Redirect(w, req, ".", http.StatusMovedPermanently)
	//showPage(w, user)
}

func handleUpload(w http.ResponseWriter, req *http.Request) {
	var user string
	if ck, err := req.Cookie("sid"); err == nil {
		user = sessions[ck.Value]
	}
	if user == "" {
		http.Error(w, "Session id missing or invalid", http.StatusInternalServerError)
		return
	}
	f, fh, err := req.FormFile("file")
	if err != nil {
		http.Error(w, "Missing file: "+err.Error(), http.StatusInternalServerError)
		return
	}
	cached, err := cachedir.Add(user, fh.Filename, f)
	if err == nil {
		err = uploader.Add(cached)
		if err != nil {
			cached.Discard()
		}
	}
	if err != nil {
		http.Error(w, "Upload error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func showPage(w http.ResponseWriter, user string) {
	p := &page{}
	if user != "" {
		p.Name = strings.Title(user)
		p.Userfiles = uploader.Files(user)
	}
	p.QueueSize = cachedir.ByteSize
	tmplUpload.Execute(w, p)
}

func gensid() string {
	p := rand.Perm(26 * 26)
	buf := make([]byte, 14)
	for i := range buf {
		buf[i] = 'a' + byte(p[i]%26)
	}
	return string(buf)
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
