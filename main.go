package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"html/template"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/mail"
	"net/smtp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
)

type Person struct {
	ID    int
	Name  string
	Email string
	out   *Person
	addr  *mail.Address
}
type ByID []Person

func (p ByID) Len() int           { return len(p) }
func (p ByID) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p ByID) Less(i, j int) bool { return p[i].ID < p[j].ID }

type SantaServer struct {
	people []Person
	err    error
	tRoot  *template.Template
	tMail  *template.Template
	root   string
	from   string
	auth   smtp.Auth
}

func (ss *SantaServer) rootHandler(rw http.ResponseWriter, req *http.Request) {
	ss.tRoot.Execute(rw, ss.root)
}
func (ss *SantaServer) postHandler(rw http.ResponseWriter, req *http.Request) {
	dec := json.NewDecoder(req.Body)
	if ss.err = dec.Decode(&ss.people); ss.err != nil {
		rw.Header().Set("Santa-Mail-Status", "Invalid Request")
		return
	}
	if ss.validateAddresses(); ss.err != nil {
		rw.Header().Set("Santa-Mail-Status", "Invalid Email Address")
		return
	}
	ss.secretsanta()
	if ss.sendAll(); ss.err != nil {
		rw.Header().Set("Santa-Mail-Status", "Email Error")
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
	rw.Header().Set("Santa-Mail-Status", "Success")
}

func (ss *SantaServer) validateAddresses() {
	for i := range ss.people {
		stAddr := strings.Join([]string{ss.people[i].Name, " <", ss.people[i].Email, ">"}, "")
		if ss.people[i].addr, ss.err = mail.ParseAddress(stAddr); ss.err != nil {
			return
		}
	}
}

func (ss *SantaServer) secretsanta() {
	rand.Seed(time.Now().UnixNano())
	for i := len(ss.people) - 1; i >= 0; i-- {
		j := rand.Intn(i + 1)
		ss.people[i], ss.people[j] = ss.people[j], ss.people[i]
	}
	for i := range ss.people {
		ss.people[i].out = &ss.people[(i+1)%len(ss.people)]
	}
	sort.Sort(ByID(ss.people))
}

func (ss *SantaServer) sendEmail(wg *sync.WaitGroup, p Person) error {
	defer wg.Done()
	const address = "smtp.gmail.com:587"
	conn, err := net.Dial("tcp4", address)
	if err != nil {
		return err
	}
	host, _, _ := net.SplitHostPort(address)
	c, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer c.Close()
	if err = c.Hello(host); err != nil {
		return err
	}
	if ok, _ := c.Extension("STARTTLS"); ok {
		config := &tls.Config{ServerName: host}
		if err = c.StartTLS(config); err != nil {
			return err
		}
	}
	if ss.auth != nil {
		if ok, _ := c.Extension("AUTH"); ok {
			if err = c.Auth(ss.auth); err != nil {
				return err
			}
		}
	}
	if err = c.Mail(ss.from); err != nil {
		return err
	}
	if err := c.Rcpt(p.addr.Address); err != nil {
		return err
	}
	// Send body
	w, err := c.Data()
	if err != nil {
		return err
	}
	data := struct {
		From, To, Name string
	}{ss.from, p.addr.Address, p.out.Name}
	if err = ss.tMail.Execute(w, data); err != nil {
		return err
	}
	if err = w.Close(); err != nil {
		return err
	}
	return c.Quit()
}

func (ss *SantaServer) sendAll() {
	var wg sync.WaitGroup
	for i := range ss.people {
		wg.Add(1)
		go ss.sendEmail(&wg, ss.people[i])
	}
	wg.Wait()
}

func main() {
	t := flag.String("t", "./config.toml", "path to toml configuration file")
	flag.Parse()
	var config struct {
		Port     string
		Root     string
		From     string
		Password string
		TmpRoot  string
		TmpMail  string
	}
	if _, err := toml.DecodeFile(*t, &config); err != nil {
		log.Fatalln(err)
	}
	ss := &SantaServer{
		tRoot: template.Must(template.ParseFiles(config.TmpRoot)),
		tMail: template.Must(template.ParseFiles(config.TmpMail)),
		root:  config.Root,
		from:  config.From,
		auth:  smtp.PlainAuth("", config.From, config.Password, "smtp.gmail.com"),
	}
	http.HandleFunc("/", ss.rootHandler)
	http.HandleFunc("/post/", ss.postHandler)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	log.Fatalln(http.ListenAndServe(config.Port, nil))
}
