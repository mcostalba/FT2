package main

import (
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"log"
	"os"
)

type (
	DBSession struct {
		s *mgo.Session
	}
	DBResults struct {
		M []bson.M
	}
)

// File scope visibility
var (
	masterDBSession DBSession
	dbname          = os.Getenv("dbname")
)

// Global visibility: our API
func (d *DBSession) Runs(ofs, limit int, results *DBResults) error {

	notDeleted := bson.M{"deleted": bson.M{"$ne": 1}}
	stateAndTime := []string{"finished", "-last_updated", "-start_time"}

	c := d.s.DB(dbname).C("runs")
	return c.Find(notDeleted).Sort(stateAndTime...).Skip(ofs).Limit(limit).All(&results.M)
}

func (d *DBSession) Users(limit int, results *DBResults) error {
	c := d.s.DB(dbname).C("users")
	return c.Find(nil).Limit(limit).All(&results.M)
}

func (d *DBSession) Close() {
	d.s.Close()
}

// Allow concurrent requests, so that each handler (in its coroutine) uses a
// different and concurrent connection.
// Usage:
//    db := DB()
//    defer db.Close()
//
//    r := db.Runs(...)
//
func DB() *DBSession {
	return &DBSession{masterDBSession.s.Copy()}
}

func DialDB() {
	// MongoDB server is assumed to be on the same machine, if not user should use
	// ssh with port forwarding to access the remote host.
	session, err := mgo.Dial("localhost:27017")
	if err != nil {
		log.Fatal("Cannot dial mongo ", err)
	}

	session.SetMode(mgo.Monotonic, true)
	masterDBSession.s = session

	var names []string
	names, _ = session.DB(dbname).CollectionNames()
	cnt := 0
	for _, n := range names {
		if n == "runs" || n == "users" {
			cnt++
		}
	}

	if cnt != 2 {
		log.Fatal("Cannot find expected collections")
	}

	log.Println("DB connected!")
}

func CloseDB() {
	masterDBSession.s.Close()
}
