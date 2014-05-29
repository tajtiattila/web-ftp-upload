package main

import (
	"encoding/gob"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

type page struct {
	Name        string
	Donefiles   []string
	Cachedfiles []string
	QueueSize   int64
	QueueLoad   int
}

type WebServer struct {
	Prefix   string
	Sessions map[string]string
	log      *log.Logger
}

func NewWebServer(prefix string) *WebServer {
	s := &WebServer{
		Prefix:   prefix,
		Sessions: make(map[string]string),
		log:      log.New(os.Stderr, "WWW     ", log.LstdFlags),
	}
	s.load()
	return s
}

func (s *WebServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "GET":
		var sid string
		if ck, err := req.Cookie("sid"); err == nil {
			sid = ck.Value
		}
		if req.FormValue("info") != "" {
			s.showPage(w, tmplInfo, s.Sessions[sid])
			return
		}
		var user string
		if req.FormValue("login") == "" {
			user = s.Sessions[sid]
		} else {
			delete(s.Sessions, sid)
		}
		s.showPage(w, tmplHome, user)
	case "POST":
		d := req.FormValue("do")
		switch d {
		case "login":
			s.handleLogin(w, req)
		case "upload":
			s.handleUpload(w, req)
		default:
			http.Error(w, "Invalid request", http.StatusInternalServerError)
		}
	}
}

func (s *WebServer) handleLogin(w http.ResponseWriter, req *http.Request) {
	user := req.FormValue("name")
	if user == "" {
		http.Error(w, "Missing name", http.StatusInternalServerError)
		return
	}
	sid := gensid()
	http.SetCookie(w, &http.Cookie{Name: "sid", Value: sid})
	s.Sessions[sid] = user
	http.Redirect(w, req, ".", http.StatusMovedPermanently)
	s.save()
}

func (s *WebServer) handleUpload(w http.ResponseWriter, req *http.Request) {
	var user string
	if ck, err := req.Cookie("sid"); err == nil {
		user = s.Sessions[ck.Value]
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

func (s *WebServer) load() {
	f, err := os.Open(s.datafilename())
	switch {
	case err == nil:
		defer f.Close()
		err = gob.NewDecoder(f).Decode(&s.Sessions)
	case os.IsNotExist(err):
		err = nil
	}
	if err != nil {
		s.log.Println("Load error:", err)
	}
}

func (s *WebServer) save() {
	f, err := SafeFileWriter(s.datafilename())
	if err == nil {
		defer func() {
			err = f.Close()
			if err != nil {
				s.log.Println("Can't close:", err)
			}
		}()
		err = gob.NewEncoder(f).Encode(s.Sessions)
	}
	if err != nil {
		s.log.Println("Can't save:", err)
	}
}

func (s *WebServer) datafilename() string {
	p, _ := GetCacheDir("")
	return p + "/session.dat"
}

func (s *WebServer) showPage(w http.ResponseWriter, tmpl *template.Template, user string) {
	p := &page{}
	if user != "" {
		p.Name = strings.Title(user)
		p.Donefiles = uploader.Userfiles(user)
		sort.Strings(p.Donefiles)
		p.Cachedfiles = cachedir.Userfiles(user)
		sort.Strings(p.Cachedfiles)
	}
	p.QueueSize = cachedir.ByteSize
	p.QueueLoad = int(p.QueueSize * 100 / (8 * 1024 * 1024 * 1024))
	err := tmplHome.Execute(w, p)
	if err != nil {
		s.log.Println(err)
	}
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
