package main

import (
	"code.google.com/p/go.net/idna"
	"fmt"
	"github.com/jlaffaye/ftp"
	"io"
	"log"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	STATUS_CONNECTING = iota
	STATUS_CONNECTED
	STATUS_ERROR
	STATUS_DISCONNECTING
	STATUS_INACTIVE

	FTP_RECONNECT_DELAY   = 60 * time.Second
	FTP_DISCONNECT_DELAY  = 60 * time.Second
	FTP_UPLOAD_FAIL_DELAY = 60 * time.Second

	USER_DIR_PREFIX = "uplood-"
)

var ErrQuit = fmt.Errorf("Quitting")

type Uploader struct {
	Addr, User, Pass, RemoteDir string

	log    *log.Logger
	conn   *ftp.ServerConn
	status int
	err    error
	queue  *uploadqueue
	chquit chan bool

	fmtx  sync.RWMutex
	files map[string][]string
}

func NewUploader(rawurl string) (*Uploader, error) {
	url, err := url.Parse(rawurl)
	if err != nil {
		return nil, err
	}
	if url.Scheme != "ftp" {
		return nil, fmt.Errorf("URL scheme '%s' not supported", url.Scheme)
	}
	addr := url.Host
	if strings.IndexRune(addr, ':') == -1 {
		addr += ":21"
	}
	var user, pass string
	if url.User != nil {
		user = url.User.Username()
		var ok bool
		if pass, ok = url.User.Password(); !ok {
			pass = "anonymous"
		}
	}
	u := &Uploader{
		Addr:      addr,
		User:      user,
		Pass:      pass,
		RemoteDir: url.Path,

		log:    log.New(os.Stderr, "FTP     ", log.LstdFlags),
		queue:  newuploadqueue(nil),
		chquit: make(chan bool),
		files:  make(map[string][]string),
	}
	err = u.find_files()
	if err != nil {
		return nil, err
	}
	go u.run()
	return u, nil
}

func (u *Uploader) Add(f CachedFile) error {
	_, err := Encodename(f.User())
	if err != nil {
		return err
	}
	u.queue.push(f)
	return nil
}

func (u *Uploader) Close() error {
	close(u.chquit)
	return nil
}

func (u *Uploader) Userfiles(uploader string) []string {
	uploader = strings.ToLower(uploader)
	u.fmtx.RLock()
	defer u.fmtx.RUnlock()
	return u.files[uploader]
}

func (u *Uploader) setStatus(status int, err error) {
}

func (u *Uploader) connect(once bool) error {
	for u.conn == nil {
		u.log.Println("Connecting")
		u.setStatus(STATUS_CONNECTING, nil)
		conn, err := ftp.Connect(u.Addr)
		if err == nil {
			u.log.Println("Logging in", u.User)
			err = conn.Login(u.User, u.Pass)
			if err == nil {
				u.log.Println("Login successful")
				u.conn = conn
				u.setStatus(STATUS_CONNECTED, nil)
				break
			} else {
				u.log.Println("Login failed:", err)
			}
		} else {
			u.log.Println("Connection failed:", err)
		}
		u.setStatus(STATUS_ERROR, err)
		if once {
			return err
		}
		select {
		case <-time.After(FTP_RECONNECT_DELAY):
		case <-u.chquit:
			return ErrQuit
		}
	}
	return nil
}

func (u *Uploader) disconnect(xerr error) {
	if u.conn != nil {
		u.log.Println("Disconnecting")
		u.setStatus(STATUS_DISCONNECTING, nil)
		if err := u.conn.Quit(); err != nil {
			u.log.Println("Disconnect failed:", err)
		}
		u.conn = nil
		if xerr == nil {
			u.setStatus(STATUS_INACTIVE, nil)
		} else {
			u.setStatus(STATUS_ERROR, xerr)
			time.Sleep(FTP_UPLOAD_FAIL_DELAY)
		}
	}
}

func (u *Uploader) upload(encname, filename string, content io.ReadCloser) bool {
	err := u.conn.ChangeDir(u.RemoteDir)
	if err != nil {
		u.log.Println("CD to remote dir", u.RemoteDir, "failed:", err)
		u.disconnect(err)
		return false
	}
	userdir := USER_DIR_PREFIX + encname
	errmk := u.conn.MakeDir(userdir) // don't check, may exist
	err = u.conn.ChangeDir(userdir)
	if err != nil {
		u.log.Println("Creating/changing to directory", userdir, "failed:", errmk, err)
		u.disconnect(err)
		return false
	}
	err = u.conn.Stor(filename, content)
	if err != nil {
		u.log.Println("Upload of", encname+"/"+filename, "failed:", err)
		u.disconnect(err)
		return false
	}
	u.log.Println("File", encname+"/"+filename, "uploaded")
	err = content.Close()
	if err != nil {
		u.log.Println("Close failed:", err)
	}
	return true
}

func (u *Uploader) add_file(user, filename string) {
	u.fmtx.Lock()
	defer u.fmtx.Unlock()
	user = strings.ToLower(user)
	v := u.files[user]
	for _, fn := range v {
		if filename == fn {
			return
		}
	}
	u.files[user] = append(v, filename)
}

func (u *Uploader) find_files() (err error) {
	if err = u.connect(true); err != nil {
		return
	}
	defer func() {
		if err != nil {
			u.disconnect(err)
		}
	}()
	if err = u.conn.ChangeDir(u.RemoteDir); err != nil {
		u.log.Println("CD to remote dir", u.RemoteDir, "failed:", err)
		return err
	}
	var unames []string
	if unames, err = u.conn.NameList("."); err != nil {
		u.log.Println("Can't get list of users")
		return err
	}
	nu, nf := 0, 0
	for _, name := range unames {
		if strings.HasPrefix(name, USER_DIR_PREFIX) {
			user, err := Decodename(name[len(USER_DIR_PREFIX):])
			if user == "" || err != nil {
				u.log.Println("Unexpected file/folder:", name[len(USER_DIR_PREFIX):], err)
				continue
			}
			if user != "" {
				nu++
				var fnames []string
				if fnames, err = u.conn.NameList(name); err != nil {
					u.log.Println("Getting filenames for", user, "failed:", err)
				} else {
					u.files[user] = fnames
					nf += len(fnames)
				}
			}
		}
	}
	u.log.Println("Found", nf, "files for", nu, "users")
	return
}

func (u *Uploader) run() {
	defer func() {
		u.disconnect(nil)
	}()
	for {
		if f := u.queue.peek(); f != nil {
			content, err := f.Open()
			if err != nil {
				log.Println("Can't open content for ", f.User()+"/"+f.Filename(), ", dropping")
				u.queue.pop()
				f.Discard()
				continue
			}
			if u.connect(false) != nil {
				return
			}
			encname, err := Encodename(f.User())
			if err != nil {
				panic(err) // can't happen Encodename is called in Add()
			}
			if u.upload(encname, f.Filename(), content) {
				u.add_file(f.User(), f.Filename())
				u.queue.pop()
				f.Discard()
			}
		} else {
			select {
			case <-u.queue.ch:
				// no op
			case <-time.After(60 * time.Second):
				u.disconnect(nil)
			case <-u.chquit:
				return
			}
		}
	}
}

func Encodename(x string) (string, error) {
	return idna.ToASCII(strings.ToLower(x))
}

func Decodename(s string) (string, error) {
	return idna.ToUnicode(s)
}
