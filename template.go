package main

import (
	"fmt"
	"html/template"
	"os"
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

type tmpl struct {
	Title string
	Home  *template.Template
	Info  *template.Template
}

const defaultlang = Language("en")

var (
	langtmpl    = make(map[Language]*tmpl)
	defaulttmpl *tmpl
	languages   []Language
)

/*
func templates(lang string) *tmpl {
	if t, ok := langtmpl[lang]; ok {
		return t
	}
	t := langtmpl[defaultlang]
	if t == nil {
		panic("missing "+defaultlang+" template")
	}
	return t
}
*/

func readtemplates(dir string, titles map[Language]string) (err error) {
	var templates *template.Template
	templates, err = template.New("base").Funcs(tmplFuncs).ParseGlob(dir + "/*.tmpl")
	if err != nil {
		return
	}
	var d *os.File
	d, err = os.Open(dir)
	if err != nil {
		return
	}
	var v []os.FileInfo
	v, err = d.Readdir(0)
	if err != nil {
		return
	}
	for _, fi := range v {
		if fi.IsDir() {
			var t *template.Template
			t, err = templates.Clone()
			if err != nil {
				return
			}
			subdir := dir + "/" + fi.Name()
			_, err = t.ParseGlob(subdir + "/*.tmpl")
			if err != nil {
				return
			}
			th := t.Lookup("home")
			ti := t.Lookup("info")
			if th == nil {
				return fmt.Errorf(`Template "home" is missing in %s`, subdir)
			}
			if ti == nil {
				return fmt.Errorf(`Template "info" is missing in %s`, subdir)
			}
			title, ok := "", false
			if title, ok = titles[Language(fi.Name())]; !ok {
				if title, ok = titles[defaultlang]; !ok {
					title = "Uploader"
				}
			}
			langtmpl[Language(fi.Name())] = &tmpl{title, th, ti}
		}
	}
	defaulttmpl = langtmpl[defaultlang]
	if defaulttmpl == nil {
		fmt.Errorf("missing " + string(defaultlang) + " template")
	}
	languages = make([]Language, 0, len(langtmpl))
	languages = append(languages, defaultlang)
	for k := range langtmpl {
		if k != defaultlang {
			languages = append(languages, k)
		}
	}
	return
}
