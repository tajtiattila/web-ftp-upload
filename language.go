package main

import (
	"net/http"
	"sort"
	"strconv"
)

func Selectlang(req *http.Request, formvalname string, avail []string) string {
	if len(avail) == 0 {
		panic("no language available")
	}
	if req != nil {
		if l := req.FormValue(formvalname); l != "" {
			if has(avail, l) {
				return l
			}
		}
		for _, al := range parseAcceptLang(req.Header["Accept-Language"]) {
			if l, ok := matchlang(avail, al); ok {
				return l
			}
		}
	}
	return avail[0]
}

func has(list []string, s string) bool {
	for _, e := range list {
		if e == s {
			return true
		}
	}
	return false
}

func matchlang(list []string, s string) (string, bool) {
	for _, l := range list {
		if l == s {
			return l, true
		}
		if len(s) > len(l) && s[:len(l)] == l && s[len(l)] == '-' {
			return l, true
		}
	}
	return "", false
}

func parseAcceptLang(headers []string) []string {
	var (
		lv []string
		qv []float64
	)
	for _, h := range headers {
		p := 0
		for i, ch := range h {
			if ch == ',' {
				if l, q := parseelem(h[p:i]); q >= 0 {
					lv, qv = append(lv, l), append(qv, q)
				}
				p = i + 1
			}
		}
		if p < len(h) {
			if l, q := parseelem(h[p:]); q >= 0 {
				lv, qv = append(lv, l), append(qv, q)
			}
		}
	}
	sort.Stable(&byql{lv, qv})
	return lv
}

func parseelem(elem string) (string, float64) {
	for len(elem) > 0 && elem[0] == ' ' {
		elem = elem[1:]
	}
	lang, vars := elem, ""
	for i, ch := range elem {
		if ch == ';' {
			lang, vars = elem[:i], elem[i:]
			break
		}
	}
	if lang == "" {
		return "", -1
	}
	p, qs, qe := 0, 0, len(vars)
	for i, ch := range vars {
		switch ch {
		case ';':
			if qs != 0 {
				qe = i
				break
			}
			p = i + 1
		case '=':
			if i == p+1 && vars[p] == 'q' {
				qs = i + 1
			}
		}
	}
	if qs != 0 && qs < qe {
		if q, err := strconv.ParseFloat(vars[qs:qe], 64); err != nil {
			return lang, q
		}
	}
	return lang, 1
}

type byql struct {
	l []string
	q []float64
}

func (a *byql) Len() int           { return len(a.l) }
func (a *byql) Less(i, j int) bool { return a.q[i] > a.q[j] }
func (a *byql) Swap(i, j int) {
	a.l[i], a.l[j] = a.l[j], a.l[i]
	a.q[i], a.q[j] = a.q[j], a.q[i]
}
