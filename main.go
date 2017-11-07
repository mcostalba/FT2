package main

import (
	"github.com/alexedwards/scs"
	"golang.org/x/net/websocket"
	"html/template"
	"log"
	"net/http"
	"net/url"
)

type Page struct {
	Username string
	Params   url.Values
	Data     DBResults
	Fmt      FmtFunc // Trick to call formatting functions from inside templates
}

var (
	// Initialize a new encrypted-cookie based session manager
	SessionManager = scs.NewCookieManager("u46IpCV9y5Vlur8YvODJEhgOY8m9JVE4")

	// Define and parse at startup our templates, one for each handler
	runsTemplate    = template.Must(template.ParseFiles("templ/runs.html", "templ/base.html"))
	getRunsTemplate = template.Must(template.ParseFiles("templ/get_runs.html", "templ/machines.html"))
)

func handleRunsWS(ws *websocket.Conn) {

	defer ws.Close()
	c, ok := NewConnection()
	if !ok {
		return
	}
	defer c.Close()

	for msg := range c.ch {
		err := websocket.Message.Send(ws, msg)
		if err != nil {
			log.Println("Can't send, closing websocket")
			break
		}
		// To detect browser disconnection, we try to read a 'pong' signal, sent
		// by the client after receiving our data.
		var pong string
		err = websocket.Message.Receive(ws, &pong)
		if err != nil {
			log.Println("Can't read, closing websocket")
			break
		}
	}
}

func handleGetRuns(w http.ResponseWriter, r *http.Request) {

	db := DB()
	defer db.Close()

	var page Page
	page.Params = r.URL.Query()
	page.Params.Set("limit", "50")

	if GetCachedPage(w, &page) {
		return
	}
	err := db.Runs(page.Params, &page.Data)
	if err != nil {
		log.Printf("RunQuery ERROR: %s\n", err)
	}
	getRunsTemplate.ExecuteTemplate(w, "layout", &page)
}

func handleRuns(w http.ResponseWriter, r *http.Request) {

	var page Page
	session := SessionManager.Load(r)
	username, _ := session.GetString("username")
	page.Username = username
	runsTemplate.ExecuteTemplate(w, "layout", &page)
}

func main() {

	// Connect to MongoDB
	DialDB()
	defer CloseDB()

	// Setup and start caching and websocket service
	StartBroadcasting(getRunsTemplate)
	defer StopBroadcasting()

	mux := http.NewServeMux()

	// Serve static files
	fs := http.FileServer(http.Dir("static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// Handles used for GitHub OAuth login/logout
	mux.HandleFunc("/login/", HandleGitHubLogin)
	mux.HandleFunc("/logout/", HandleGitHubLogout)
	mux.HandleFunc("/github_oauth_cb", HandleGitHubCallback) // Note missing trailing slash!

	// Handle for websockets
	mux.Handle("/runs_ws/", websocket.Handler(handleRunsWS))

	mux.HandleFunc("/", handleRuns)
	mux.HandleFunc("/get_runs/", handleGetRuns)
	http.ListenAndServe(":8080", SessionManager.Use(mux))
}
