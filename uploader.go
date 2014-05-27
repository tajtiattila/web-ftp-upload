package main

import (
	"code.google.com/p/go.net/idna"
	"fmt"
	"github.com/jlaffaye/ftp"
	"log"
	"net/url"
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

type Upload struct {
	user     string
	encname  string
	filename string
	size     int64
	content  CachedContent
}

type Uploader struct {
	Addr, User, Pass, RemoteDir string

	conn   *ftp.ServerConn
	status int
	err    error
	queue  *uploadqueue
	chquit chan bool

	fmtx  sync.RWMutex
	files map[string][]string

	smtx      sync.RWMutex
	queuesize int64
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

func (u *Uploader) Add(uploader, filename string, content CachedContent) error {
	size, err := content.Size()
	if err != nil {
		return err
	}
	uploader = strings.ToLower(uploader)
	encname, err := Encodename(uploader)
	if err != nil {
		return err
	}
	u.add_file(uploader, filename)
	u.queue.push(&Upload{uploader, encname, filename, size, content})
	return nil
}

func (u *Uploader) Close() error {
	close(u.chquit)
	return nil
}

func (u *Uploader) Files(uploader string) []string {
	uploader = strings.ToLower(uploader)
	u.fmtx.RLock()
	defer u.fmtx.RUnlock()
	return u.files[uploader]
}

func (u *Uploader) QueueSize() int64 {
	u.smtx.RLock()
	defer u.smtx.RUnlock()
	return u.queuesize
}

func (u *Uploader) setStatus(status int, err error) {
}

func (u *Uploader) connect(once bool) error {
	for u.conn == nil {
		log.Println("Connecting")
		u.setStatus(STATUS_CONNECTING, nil)
		conn, err := ftp.Connect(u.Addr)
		if err == nil {
			log.Println("Logging in ", u.User)
			err = conn.Login(u.User, u.Pass)
			if err == nil {
				log.Println("Login successful")
				u.conn = conn
				u.setStatus(STATUS_CONNECTED, nil)
				break
			} else {
				log.Println("Login failed: ", err)
			}
		} else {
			log.Println("Connection failed: ", err)
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
		log.Println("Disconnecting")
		u.setStatus(STATUS_DISCONNECTING, nil)
		if err := u.conn.Quit(); err != nil {
			log.Println("Disconnect failed:", err)
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

func (u *Uploader) upload(encname, filename string, content CachedContent) bool {
	err := u.conn.ChangeDir(u.RemoteDir)
	if err != nil {
		log.Println("CD to remote dir ", u.RemoteDir, " failed: ", err)
		u.disconnect(err)
		return false
	}
	userdir := USER_DIR_PREFIX + encname
	errmk := u.conn.MakeDir(userdir) // don't check, may exist
	err = u.conn.ChangeDir(userdir)
	if err != nil {
		log.Println("Creating/changing to directory ", userdir, " failed: ", errmk, err)
		u.disconnect(err)
		return false
	}
	err = u.conn.Stor(filename, content)
	if err != nil {
		log.Println("Upload of ", encname+"/"+filename, " failed: ", err)
		u.disconnect(err)
		return false
	}
	log.Println("File ", encname+"/"+filename, " uploaded")
	err = content.Close()
	if err != nil {
		log.Println("Cleanup failed: ", err)
	}
	return true
}

func (u *Uploader) add_file(user, filename string) {
	u.fmtx.Lock()
	defer u.fmtx.Unlock()
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
		log.Println("CD to remote dir ", u.RemoteDir, " failed: ", err)
		return err
	}
	var unames []string
	if unames, err = u.conn.NameList("."); err != nil {
		log.Println("Can't get list of users")
		return err
	}
	nu, nf := 0, 0
	for _, name := range unames {
		if strings.HasPrefix(name, USER_DIR_PREFIX) {
			user, err := Decodename(name[len(USER_DIR_PREFIX):])
			if user == "" || err != nil {
				log.Println("Unexpected file/folder: ", name[len(USER_DIR_PREFIX):], err)
				continue
			}
			if user != "" {
				nu++
				var fnames []string
				if fnames, err = u.conn.NameList(name); err != nil {
					log.Println("Getting filenames for ", user, " failed: ", err)
				} else {
					u.files[user] = fnames
					nf += len(fnames)
				}
			}
		}
	}
	log.Println("Found", nf, "files for", nu, "users")
	return
}

func (u *Uploader) run() {
	defer func() {
		u.disconnect(nil)
	}()
	for {
		if e := u.queue.peek(); e != nil {
			if _, err := e.content.Seek(0, 0); err != nil {
				log.Println("Can't seek content for ", e.user+"/"+e.filename, ", dropping")
				u.queue.pop()
				continue
			}
			if u.connect(false) != nil {
				return
			}
			if u.upload(e.encname, e.filename, e.content) {
				u.queue.pop()
				u.diffsize(-e.size)
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

func (u *Uploader) diffsize(d int64) {
	u.smtx.Lock()
	defer u.smtx.Unlock()
	u.queuesize += d
}

func Encodename(x string) (string, error) {
	return idna.ToASCII(x)
}

func Decodename(s string) (string, error) {
	return idna.ToUnicode(s)
}
