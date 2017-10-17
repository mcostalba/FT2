package main

import (
	"github.com/alexedwards/scs"
	"html/template"
	"log"
	"net/http"
)

type Page struct {
	Username string
	Data     DBResults
	Fmt      FmtFunc // Trick to call formatting functions from inside templates
}

var (
	// Initialize a new encrypted-cookie based session manager
	SessionManager = scs.NewCookieManager("u46IpCV9y5Vlur8YvODJEhgOY8m9JVE4")

	// Define and parse at startup our templates, one for each handler
	runsTemplate = template.Must(template.ParseFiles("templ/runs.html", "templ/base.html"))
)

func handleRuns(w http.ResponseWriter, r *http.Request) {

	var page Page

	session := SessionManager.Load(r)
	username, _ := session.GetString("username")
	page.Username = username

	db := DB()
	defer db.Close()

	err := db.Runs(50, &page.Data)
	if err != nil {
		log.Printf("RunQuery : ERROR : %s\n", err)
	}

	runsTemplate.ExecuteTemplate(w, "layout", &page)
}

func main() {

	// Connect to MongoDB
	DialDB()
	defer CloseDB()

	mux := http.NewServeMux()

	// Serve css directory as static files
	fs := http.FileServer(http.Dir("css"))
	mux.Handle("/css/", http.StripPrefix("/css/", fs))

	// Handles used for GitHub OAuth login/logout
	mux.HandleFunc("/login/", HandleGitHubLogin)
	mux.HandleFunc("/logout/", HandleGitHubLogout)
	mux.HandleFunc("/github_oauth_cb", HandleGitHubCallback) // Note missing trailing slash!

	mux.HandleFunc("/", handleRuns)
	http.ListenAndServe(":8080", SessionManager.Use(mux))
}
