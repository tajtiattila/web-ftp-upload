package main

import (
	"fmt"
	"html/template"
)

const (
	MultByte = iota << 10
	MultKiB
	MultMiB
	MultGiB
)

var tmplUpload = template.Must(template.New("foo").Funcs(template.FuncMap{
	"filesize": func(n int64) string {
		switch {
		case n < MultKiB:
			return fmt.Sprint(n, "byte")
		case n < MultMiB:
			return fmt.Sprint(n/MultKiB, "KiB")
		case n < MultGiB:
			return fmt.Sprint(n/MultMiB, "MiB")
		}
		return fmt.Sprint(n/MultGiB, "GiB")
	},
}).Parse(`<!DOCTYPE html>
<html>
	<head>
		<title>Web-FTP uploader</title>
		{{if .Name}}
		<script src="./ext/js/dropzone.min.js"></script>
		<link rel="stylesheet" href="./ext/css/dropzone.css"/>
		{{end}}
		<style>
ul.filelist li {
	display: inline;
}
		</style>
	</head>
	<body>
		<h1>Web-FTP feltöltőcucc</h1>
		{{if .Name}}
		<p>Szia <b>{{.Name}}</b>!</p>
		<p>Add meg a feltöltendő fájlokat, vagy klikk <a href=".">ide</a>, ha másik nevet adnál meg.</p>
		{{if .Userfiles}}
		<p><b>{{.Name}}</b> néven feltöltött fájlok:</p>
		<ul class="filelist">
			{{range .Userfiles}}
			<li>{{.}}</li>
			{{end}}
		</ul>
		{{else}}
		<p><b>{{.Name}}</b> névvel még nem töltöttek fel fájlokat.</p>
		{{end}}
		<p>Jelenleg {{filesize .QueueSize}} FTP-re töltése van folyamatban.</p>
		<div id="dropzone">
			<form id="upload" action="." class="dropzone">
				<input name="name" value="{{.Name}}" type="hidden">
			</form>
		</div>
		{{else}}
			<p>Ez az oldal sütiket és egyéb nyalánkságokat használ a feltöltés lebonyolítására. Ha nem tetszik, fel is út, le is út!</p>
			<form action=".">
				Neved:<input name="name" type="text"/>
				<input type="submit"/>
			</form>
		{{end}}
	</body>
</html>
`))
