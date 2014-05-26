package main

import (
	"net/http"
)

func NewServeMux() *http.ServeMux {
	sm := http.NewServeMux()
	sm.HandleFunc("/web-ftp-upload", handleUpload)
	return sm
}

var up = NewUploader()

func handleUpload(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "GET":
		tmplUpload.Execute(w, nil)
	case "POST":
		f, fh, err := req.FormFile("file")
		if err != nil {
			http.Error(w, "Missing file: "+err.Error(), http.StatusInternalServerError)
			return
		}
		err = up.Add(fh.Filename, f)
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
