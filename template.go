package main

import (
	"html/template"
)

var tmplUpload = template.Must(template.New("foo").Parse(`<!DOCTYPE html>
<html>
	<head>
		<title>Web-FTP uploader</title>
		<script src="./ext/js/dropzone.min.js"></script>
		<link rel="stylesheet" href="./ext/css/dropzone.css"/>
	</head>
	<body>
		<h1>Web-FTP feltöltőcucc</h1>
		<p>Ez az oldal sütiket és egyéb nyalánkságokat használ a feltöltés lebonyolítására. Ha nem tetszik, fel is út, le is út!</p>
		<div id="dropzone">
			<form id="upload" action="." class="dropzone"></form>
			<div id="fallback">
				<input name="file" type="file" multiple />
			</div>
		</div>
		<script>
			document.getElementById("fallback").style.display = "hidden";
		</script>
	</body>
</html>
`))
