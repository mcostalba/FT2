package main

import (
	"github.com/alexedwards/scs"
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

func handleGetRuns(w http.ResponseWriter, r *http.Request) {

	db := DB()
	defer db.Close()

	var page Page
	page.Params = r.URL.Query()
	page.Params.Set("limit", "50")

	err := db.Runs(page.Params, &page.Data)
	if err != nil {
		log.Printf("RunQuery : ERROR : %s\n", err)
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

	mux := http.NewServeMux()

	// Serve static files
	fs := http.FileServer(http.Dir("static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// Handles used for GitHub OAuth login/logout
	mux.HandleFunc("/login/", HandleGitHubLogin)
	mux.HandleFunc("/logout/", HandleGitHubLogout)
	mux.HandleFunc("/github_oauth_cb", HandleGitHubCallback) // Note missing trailing slash!

	mux.HandleFunc("/", handleRuns)
	mux.HandleFunc("/get_runs/", handleGetRuns)
	http.ListenAndServe(":8080", SessionManager.Use(mux))
}
