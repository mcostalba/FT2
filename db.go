package main

import (
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"log"
	"net/url"
	"os"
	"strconv"
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

func parseFilter(params url.Values) bson.M {

	match := bson.M{"deleted": bson.M{"$ne": 1}}

	username := params.Get("username")

	if username != "" {
		match["args.username"] = username
	}
	return match
}

// Global visibility: our API
func (d *DBSession) Runs(params url.Values, results *DBResults) error {

	limit, _ := strconv.Atoi(params.Get("limit"))
	ofs, err := strconv.Atoi(params.Get("page"))
	ofs *= limit
	if err != nil {
		return err
	}

	// Setup conditions to limit data and to sort results
	noSpsaNoTasks := bson.M{"args.spsa.param_history": 0, "args.spsa.params": 0, "tasks": bson.M{"$slice": 0}}
	stateAndTime := []string{"finished", "-last_updated", "-start_time"}

	match := parseFilter(params)

	c := d.s.DB(dbname).C("runs")
	err = c.Find(match).Select(noSpsaNoTasks).Sort(stateAndTime...).Skip(ofs).Limit(limit).All(&results.M)

	// In case all runs are finished, we can just return. Otherwise load
	// active tasks, used by machines view. Note that 'finished' is sorted.
	if err != nil || len(results.M) == 0 || results.M[0]["finished"].(bool) {
		return err
	}

	// We could ideally get all the runs and active tasks in one go, but
	// Fishtest MongoDB version is an old and quite limited 2.4
	getActiveTasks := []bson.M{
		bson.M{"$match": bson.M{"finished": false}},
		bson.M{"$match": bson.M{"deleted": bson.M{"$ne": 1}}},

		bson.M{"$project": bson.M{"_id": 1, "tasks": 1}},

		bson.M{"$unwind": "$tasks"},
		bson.M{"$match": bson.M{"tasks.active": true}},
		bson.M{"$group": bson.M{"_id": "$_id",
			"tasks":   bson.M{"$push": "$tasks"},
			"workers": bson.M{"$sum": 1}}},
	}
	var activeTasks []bson.M
	err = c.Pipe(getActiveTasks).All(&activeTasks)
	if err != nil {
		return err
	}

	// Re-add the active tasks into active runs
	for i := range activeTasks {
		id := activeTasks[i]["_id"].(bson.ObjectId)
		for j := range results.M {
			if results.M[j]["_id"] == id {
				results.M[j]["tasks"] = activeTasks[i]["tasks"]
				results.M[j]["workers"] = activeTasks[i]["workers"]
				break
			}
		}
	}
	params.Set("machines", "load")
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
	oneDay, _ := time.ParseDuration("24h") // FIXME remove when in production
	index := mgo.Index{Background: true, ExpireAfter: oneDay}
	c := session.DB(dbname).C("runs")
	index.Key = []string{"deleted"}
	c.EnsureIndex(index)
	index.Key = []string{"finished", "-last_updated", "-start_time"}
	c.EnsureIndex(index)
	index.Key = []string{"args.username"}
	c.EnsureIndex(index)

	log.Println("DB connected!")
}

func CloseDB() {
	masterDBSession.s.Close()
}
