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

func tmplBase(n string) *template.Template {
	return template.New(n).Funcs(template.FuncMap{
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
	})
}

var tmplUpload = template.Must(tmplBase("upload").Parse(`<!DOCTYPE html>
<html>
	<head>
		<title>Web-FTP uploader</title>
		<link href="//netdna.bootstrapcdn.com/font-awesome/4.1.0/css/font-awesome.min.css" rel="stylesheet" type="text/css">
		<link href="http://fonts.googleapis.com/css?family=Open+Sans:400italic,400,700" rel="stylesheet" type="text/css">
		{{if .Name}}
		<script src="./ext/js/dropzone.min.js"></script>
		<link rel="stylesheet" href="./ext/css/dropzone.css"/>
		{{end}}
		<style>
body {
	font-family: 'Open Sans', sans-serif;
}
ul.filelist li {
	display: inline;
}
h1 {
	text-align: center;
}
form.name {
	margin: 0 auto;
	width: 250px;
}
div.info {
	border: 1px solid #aaa;
	border-radius: 1em;
	background-color: #eee;
	color: #666;
	margin: 30px auto;
	padding: 1em;
	width: 80%;
	font-style: italic;
	font-size: 80%;
}
		</style>
	</head>
	<body>
		<h1>Web-FTP feltöltőcucc</h1>
		{{if .Name}}
		<p>Üdvözlet, <b>{{.Name}}</b>!</p>
		<p>Add meg a feltöltendő fájlokat, vagy klikk <a href="?login=1">ide</a>, ha másik nevet adnál meg.</p>
		{{if .Userfiles}}
		<p><b>{{.Name}}</b> néven korábban feltöltött fájlok:</p>
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
				<input name="do" type="hidden" value="upload">
			</form>
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
