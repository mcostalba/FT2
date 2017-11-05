package main

import (
	"strconv"
	"sync"
	"time"
)

type Connection struct {
	ch   chan string
	id   int
	open bool
}

var connections struct {
	sync.Mutex
	clients []*Connection
	id      int
	stop    bool
}

func dispatch() bool {

	connections.Lock()
	defer connections.Unlock()

	for i := len(connections.clients) - 1; i >= 0; i-- {

		c := connections.clients[i]

		if !c.open || connections.stop {
			close(c.ch) // Force the ws handler to exit
			l := len(connections.clients)
			connections.clients[i] = connections.clients[l-1]
			connections.clients = connections.clients[:l-1]
		} else {
			c.ch <- strconv.Itoa(c.id)
		}
	}
	return !connections.stop
}

func StartBroadcasting() {

	connections.Lock()
	defer connections.Unlock()

	connections.stop = false
	connections.clients = make([]*Connection, 0, 1000)

	go func() {
		ticker := time.NewTicker(time.Second * 5)
		for range ticker.C {
			if !dispatch() {
				break
			}
		}
	}()
}

func StopBroadcasting() {

	connections.Lock()
	defer connections.Unlock()
	connections.stop = true
}

func NewConnection() (*Connection, bool) {

	connections.Lock()
	defer connections.Unlock()

	if connections.stop {
		return nil, false
	}
	c := new(Connection)
	c.ch = make(chan string, 10)
	c.open = true
	c.id = connections.id
	connections.id++
	connections.clients = append(connections.clients, c)
	return c, true
}

func (c *Connection) Close() {

	connections.Lock()
	defer connections.Unlock()
	c.open = false
}
