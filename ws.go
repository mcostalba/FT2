package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
)

type Connection struct {
	ch   chan string
	open bool
}

var cache struct {
	sync.Mutex
	clients   []*Connection
	stop      bool
	template  *template.Template
	page      []byte
	pageDiff  string
	signature string
}

func computeDiff(page []byte, sign string) (string, error) {

	type Message struct {
		SignOld string
		SignNew string
		Body    string
	}
	m := Message{cache.signature, sign, "ping"}
	b, err := json.Marshal(&m)
	return string(b), err
}

func dispatch() bool {

	cache.Lock()
	defer cache.Unlock()

	for i := len(cache.clients) - 1; i >= 0; i-- {
		c := cache.clients[i]
		if !c.open || cache.stop {
			close(c.ch) // Force the ws handler to exit
			l := len(cache.clients)
			cache.clients[i] = cache.clients[l-1]
			cache.clients = cache.clients[:l-1]
		} else {
			c.ch <- cache.pageDiff
		}
	}
	return !cache.stop
}

func StartBroadcasting(template *template.Template) {

	cache.stop = false
	cache.clients = make([]*Connection, 0, 1000)
	cache.template = template
	updateCachedPage()

	go func() {
		ticker := time.NewTicker(time.Second * 5)
		for range ticker.C {
			updateCachedPage()
			if !dispatch() {
				break
			}
		}
	}()
	log.Println("Broadcasting started")
}

func StopBroadcasting() {

	cache.Lock()
	defer cache.Unlock()
	cache.stop = true
}

func NewConnection() (*Connection, bool) {

	cache.Lock()
	defer cache.Unlock()

	if cache.stop {
		return nil, false
	}
	c := new(Connection)
	c.ch = make(chan string, 10)
	c.open = true
	cache.clients = append(cache.clients, c)
	return c, true
}

func (c *Connection) Close() {

	cache.Lock()
	defer cache.Unlock()
	c.open = false
}

func GetCachedPage(w http.ResponseWriter, p *Page) bool {

	page, err := strconv.Atoi(p.Params.Get("page"))
	username := p.Params.Get("username")
	if err != nil || page != 0 || username != "" {
		return false
	}
	cache.Lock()
	defer cache.Unlock()

	w.Write(cache.page)
	return true
}

func updateCachedPage() error {

	db := DB()
	defer db.Close()

	page := Page{}
	page.Params = url.Values{}
	page.Params.Set("page", "0")
	page.Params.Set("limit", "50")

	b := make([]byte, 16)
	rand.Read(b)
	sign := hex.EncodeToString(b)
	page.Params.Set("signature", sign)

	err := db.Runs(page.Params, &page.Data)
	if err != nil {
		log.Printf("updateCachedTemplate: %s\n", err)
		return err
	}

	buf := bytes.NewBuffer(make([]byte, 0, 256*1024))
	cache.template.ExecuteTemplate(buf, "layout", &page)

	diff, err := computeDiff(buf.Bytes(), sign)
	if err != nil {
		log.Printf("updateCachedTemplate: %s\n", err)
		return err
	}

	cache.Lock()
	defer cache.Unlock()

	cache.page = buf.Bytes()
	cache.pageDiff = diff
	cache.signature = sign
	return nil
}
