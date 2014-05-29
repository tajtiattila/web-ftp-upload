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

type WebServer struct {
	*http.ServeMux
	Prefix   string
	Sessions map[string]string
	log      *log.Logger
}

func NewWebServer(p, ext string) *WebServer {
	s := &WebServer{
		ServeMux: http.NewServeMux(),
		Prefix:   p,
		Sessions: make(map[string]string),
		log:      log.New(os.Stderr, "WWW     ", log.LstdFlags),
	}
	s.load()
	if len(s.Prefix) != 0 && s.Prefix[len(s.Prefix)-1] != '/' {
		s.HandleFunc(s.Prefix, func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, s.Prefix+"/home", http.StatusMovedPermanently)
		})
		s.Prefix += "/"
	}
	s.HandleFunc(s.Prefix, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, s.Prefix+"home", http.StatusMovedPermanently)
	})
	s.HandleFunc(s.Prefix+"login", s.handleLogin)
	s.HandleFunc(s.Prefix+"home", s.handleHome)
	s.HandleFunc(s.Prefix+"upload", s.handleUpload)
	s.Handle(s.Prefix+"ext/", http.StripPrefix(s.Prefix+"ext/", http.FileServer(http.Dir(ext))))
	return s
}

func (s *WebServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	s.log.Println("accessing:", req.URL.String())
	s.ServeMux.ServeHTTP(w, req)
}

func (s *WebServer) handleHome(w http.ResponseWriter, req *http.Request) {
	user := req.FormValue("name")
	if user != "" {
		sid := gensid()
		http.SetCookie(w, &http.Cookie{Name: "sid", Value: sid})
		s.Sessions[sid] = user
		// clear query from URL (req.Method == "GET") and make reload
		// possible without "confirm form resubmission" (req.Method == "POST")
		http.Redirect(w, req, "home", http.StatusMovedPermanently)
		s.save()
		return
	}
	var sid string
	if ck, err := req.Cookie("sid"); err == nil {
		sid = ck.Value
	}
	s.showPage(w, tmplHome, s.Sessions[sid])
}

func (s *WebServer) handleLogin(w http.ResponseWriter, req *http.Request) {
	var sid string
	if ck, err := req.Cookie("sid"); err == nil {
		sid = ck.Value
	}
	delete(s.Sessions, sid)
	s.showPage(w, tmplHome, "")
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

type page struct {
	Prefix      string
	Name        string
	Donefiles   []string
	Cachedfiles []string
	QueueSize   int64
	QueueLoad   int
}

func (s *WebServer) showPage(w http.ResponseWriter, tmpl *template.Template, user string) {
	p := &page{Prefix: s.Prefix}
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
