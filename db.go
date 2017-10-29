package main

import (
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"log"
	"os"
	"time"
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

	var finishedRuns []bson.M
	var err error

	// Setup conditions to load only needed data.
	// FIXME We should use a Pipe() here, but Fishtest MongoDB version is 2.4.4
	// and aggregations are still too basic, we need at least 3.2
	notDeleted := bson.M{"deleted": bson.M{"$ne": 1}}
	finished := bson.M{"$and": []bson.M{notDeleted, bson.M{"finished": true}}}
	active := bson.M{"$and": []bson.M{notDeleted, bson.M{"finished": false}}}
	noSpsa := bson.M{"args.spsa.param_history": 0, "args.spsa.params": 0}
	noSpsaNoTasks := bson.M{"args.spsa.param_history": 0, "args.spsa.params": 0, "tasks": bson.M{"$slice": 0}}
	onTime := []string{"-last_updated", "-start_time"}

	c := d.s.DB(dbname).C("runs")

	// We need task information for active runs but not for finished runs, so run a full query for page 0,
	// for all other pages just count number of active runs, to consistently adjust ofs and limit.
	if ofs == 0 {
		err = c.Find(active).Select(noSpsa).Sort(onTime...).All(&results.M)
		limit -= len(results.M)
	} else {
		n, _ := c.Find(active).Select(bson.M{"_id": 1}).Count()
		ofs -= n
	}
	if err != nil {
		return err
	}
	err = c.Find(finished).Select(noSpsaNoTasks).Sort(onTime...).Skip(ofs).Limit(limit).All(&finishedRuns)
	results.M = append(results.M, finishedRuns...)
	return err
}

func (d *DBSession) Users(limit int, results *DBResults) error {
	c := d.s.DB(dbname).C("users")
	return c.Find(nil).Limit(limit).All(&results.M)
}

func (d *DBSession) Close() {
	d.s.Close()
}

// Allow concurrent requests, so that each handler (in its goroutine) uses a
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
	// MongoDB is assumed to be on the same machine, if not user should use
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
	// Ensure an index on 'runs' collection to speed up matching and sorting
	oneDay, _ := time.ParseDuration("24h")
	index1 := mgo.Index{
		Key:         []string{"deleted", "finished"},
		Unique:      false,
		DropDups:    false,
		Background:  true,
		Sparse:      false,
		ExpireAfter: oneDay, // FIXME remove when in production
	}
	index2 := mgo.Index{
		Key:         []string{"-last_updated", "-start_time"},
		Unique:      false,
		DropDups:    false,
		Background:  true,
		Sparse:      false,
		ExpireAfter: oneDay, // FIXME remove when in production
	}
	c := session.DB(dbname).C("runs")
	err = c.EnsureIndex(index1)
	if err == nil {
		err = c.EnsureIndex(index2)
	}
	if err != nil {
		log.Fatal("Error while creating an index on 'runs': %s", err)
	}
	log.Println("DB connected!")
}

func CloseDB() {
	masterDBSession.s.Close()
}
