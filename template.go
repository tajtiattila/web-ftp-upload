package main

import (
	"fmt"
	"html/template"
)

const (
	MultByte = 1 << (10 * iota)
	MultKiB
	MultMiB
	MultGiB
)

var tmplFuncs = template.FuncMap{
	"filesize": func(n int64) string {
		switch {
		case n < MultKiB:
			return fmt.Sprint(n, " byte")
		case n < MultMiB:
			return fmt.Sprint(n/MultKiB, " KiB")
		case n < MultGiB:
			return fmt.Sprint(n/MultMiB, " MiB")
		}
		return fmt.Sprint(n/MultGiB, " GiB")
	},
}

var tmplUpload = template.Must(template.New("upload").Funcs(tmplFuncs).Parse(`<!DOCTYPE html>
<html>
	<head>
		<title>Web-FTP uploader</title>
		<link href="//netdna.bootstrapcdn.com/font-awesome/4.1.0/css/font-awesome.min.css" rel="stylesheet" type="text/css">
		<link href="http://fonts.googleapis.com/css?family=Open+Sans:400italic,400,700" rel="stylesheet" type="text/css">
		{{if .Name}}
		<script src="./ext/js/dropzone.min.js"></script>
		<link rel="stylesheet" href="./ext/css/dropzone.css"/>
		{{end}}
		<link rel="stylesheet" href="./ext/css/upload.css"/>
	</head>
	<body>
		<h1>Web-FTP feltöltőcucc</h1>
		{{if .Name}}
		<p>Üdvözlet, <b>{{.Name}}</b>!</p>
		<p>Add meg a feltöltendő fájlokat, vagy <a href="?login=1">klikk ide</a>, ha másik nevet adnál meg.</p>
		<div id="dropzone">
			<form id="upload" action="." class="dropzone">
				<input name="do" type="hidden" value="upload">
			</form>
		</div>
		<div id="info">
			{{template "info" .}}
		</div>
		{{else}}
			<form method="post" class="name" action=".">
				<i class="fa fa-user"></i> <input name="name" type="text" placeholder="név"/>
				<input name="do" type="hidden" value="login"/>
				<input type="submit" value="küld"/>
			</form>
			<div class="info"><i class="fa fa-info-circle"></i> Ez az oldal sütiket használ. Az oldal további használatával Ön hozzájárul ezek használatához.</div>
		{{end}}
	</body>
</html>
`))

var tmplInfo = template.Must(tmplUpload.New("info").Funcs(tmplFuncs).Parse(`<div>
{{if .QueueSize}}
<p>Jelenleg összesen {{filesize .QueueSize}} FTP-re töltése van folyamatban.</p>
{{if gt .QueueLoad 80}}
	<p><i class="fa fa-exclamation-triangle"></i> Ajjaj, ha most töltesz fel, nem biztos, hogy sikerül.</p>
{{else if gt .QueueLoad 50}}
	<p><i class="fa fa-info-circle"></i> Elég sok fájl töltődik már fölfele. Inkább később tölts fel.</p>
{{end}}
{{end}}
{{if or .Cachedfiles .Donefiles}}
<p><b>{{.Name}}</b> névvel korábban feltöltött fájlok:</p>
<ul class="filelist">
	{{range .Cachedfiles}}
	<li class="working">{{.}}</li>
	{{end}}
	{{range .Donefiles}}
	<li class="done">{{.}}</li>
	{{end}}
</ul>
{{else}}
<p><i class="fa fa-info-circle"></i> <b>{{.Name}}</b> névvel még nem töltöttek fel fájlokat,
vagy azok már fel lettek dolgozva.</p>
{{end}}
</div>`))
