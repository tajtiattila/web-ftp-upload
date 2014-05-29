package main

import (
	"fmt"
	"html/template"
	"strings"
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
	"tolower": func(s string) string {
		return strings.ToLower(s)
	},
}

var tmplHome, tmplInfo *template.Template

func readtemplates(dir string) (err error) {
	var templates *template.Template
	templates, err = template.New("base").Funcs(tmplFuncs).ParseGlob(dir + "/*.html")
	if err != nil {
		return
	}
	tmplHome = templates.Lookup("home")
	tmplInfo = templates.Lookup("info")
	if tmplHome == nil || tmplInfo == nil {
		panic("missing template")
	}
	return
}
