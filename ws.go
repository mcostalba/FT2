package main

import (
	"bytes"
	"encoding/json"
	"html/template"
	"labix.org/v2/mgo/bson"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
)

type (
	Connection struct {
		ch   chan string
		open bool
	}
	diffItem struct {
		Field string
		Value string
	}
	diffEntry struct {
		Id   bson.ObjectId
		Item diffItem
		Mkey string
	}
)

var cache struct {
	sync.Mutex
	clients   []*Connection
	stop      bool
	template  *template.Template
	page      []byte
	pageDiff  string
	pageDB    DBResults
	signature string
}

func deltaMachines(id bson.ObjectId, wNew, wOld []bson.M, diffList *[]diffEntry) {

	for i, _ := range wNew {
		kn := wNew[i]["unique_key"].(string)
		for j, _ := range wOld {
			ko := wOld[j]["unique_key"].(string)
			if kn == ko {
				for key, v := range wNew[i] {
					vn, isString := v.(string)
					if vo, ok := wOld[j][key]; ok && isString {
						if vn != vo.(string) {
							*diffList = append(*diffList, diffEntry{id, diffItem{key, vn}, kn})
						}
					}
				}
				break
			}
		}
	}
}

func delta(id bson.ObjectId, ledNew, eloNew, n, o bson.M, wNew []bson.M, diffList *[]diffEntry) {

	var f FmtFunc
	ledOld := f.Led(o["finished"].(bool), o["workers"], o["games"])
	eloOld := f.Elo(o["finished"].(bool), o["results"].(bson.M), o["args"].(bson.M), o["results_info"])
	wOld := f.Machines(o)["workers"].([]bson.M)

	for key, vn := range ledNew {
		if vo, ok := ledOld[key]; ok {
			if vn.(string) != vo.(string) {
				*diffList = append(*diffList, diffEntry{id, diffItem{key, vn.(string)}, ""})
			}
		}
	}
	for key, vn := range eloNew {
		if vo, ok := eloOld[key]; ok {
			if vn.(string) != vo.(string) {
				*diffList = append(*diffList, diffEntry{id, diffItem{key, vn.(string)}, ""})
			}
		}
	}
	deltaMachines(id, wNew, wOld, diffList)
}

func computeDiff(page []byte, results DBResults, sign string) (string, error) {

	type Message struct {
		SignOld string
		SignNew string
		Diff    []diffEntry
	}
	var f FmtFunc
	diffList := make([]diffEntry, 0, 50)

	for i := range results.M {
		n := results.M[i]
		newId := n["_id"].(bson.ObjectId)
		ledNew := f.Led(n["finished"].(bool), n["workers"], n["games"])
		eloNew := f.Elo(n["finished"].(bool), n["results"].(bson.M), n["args"].(bson.M), n["results_info"])
		wNew := f.Machines(n)["workers"].([]bson.M)
		for j := range cache.pageDB.M {
			o := cache.pageDB.M[j]
			oldId := o["_id"].(bson.ObjectId)
			if oldId == newId {
				delta(newId, ledNew, eloNew, n, o, wNew, &diffList)
			}
		}
	}
	m := Message{cache.signature, sign, diffList}
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

	c := new(Connection)
	c.ch = make(chan string, 10)
	c.open = true

	cache.Lock()
	defer cache.Unlock()

	if cache.stop {
		return nil, false
	}
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

	sign := time.Now().String()
	page.Params.Set("signature", sign)

	err := db.Runs(page.Params, &page.Data)
	if err != nil {
		log.Printf("updateCachedTemplate: %s\n", err)
		return err
	}

	buf := bytes.NewBuffer(make([]byte, 0, 256*1024))
	cache.template.ExecuteTemplate(buf, "layout", &page)

	diff, err := computeDiff(buf.Bytes(), page.Data, sign)
	if err != nil {
		log.Printf("updateCachedTemplate: %s\n", err)
		return err
	}

	cache.Lock()
	defer cache.Unlock()

	cache.page = buf.Bytes()
	cache.pageDiff = diff
	cache.pageDB = page.Data
	cache.signature = sign
	return nil
}
