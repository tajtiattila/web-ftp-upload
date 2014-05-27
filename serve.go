package main

import (
	"net/http"
	"strings"
)

func NewServeMux() *http.ServeMux {
	sm := http.NewServeMux()
	sm.HandleFunc("/web-ftp-upload", handleUpload)
	return sm
}

type page struct {
	Name      string
	Userfiles []string
	QueueSize int64
}

func handleUpload(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "GET":
		v := req.URL.Query()
		nv := v["name"]
		p := &page{}
		if len(nv) != 0 {
			n := nv[0]
			p.Name = strings.Title(n)
			p.Userfiles = up.Files(n)
		}
		p.QueueSize = up.QueueSize()
		tmplUpload.Execute(w, p)
	case "POST":
		u := req.FormValue("name")
		if u == "" {
			http.Error(w, "Missing name", http.StatusInternalServerError)
			return
		}
		f, fh, err := req.FormFile("file")
		if err != nil {
			http.Error(w, "Missing file: "+err.Error(), http.StatusInternalServerError)
			return
		}
		err = upload(u, fh.Filename, f)
		if err != nil {
			http.Error(w, "Upload error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
	/*
		f, err := ioutil.TempFile("", "web-ftp-upload")
		if err != nil {
			http.Error(w, "Error creating temporary file: " + err.Error(), 500)
			return
		}
		defer os.Remove(f.Name())
		defer f.Close()
		n, err := io.Copy(f, req)
		if err != nil {
			http.Error(w, "Error uploading file: " + err.Error(), 500)
			return
		}
		f.Seek(0, 0)
	*/
}
