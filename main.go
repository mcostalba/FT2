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
}

var (
	// Initialize a new encrypted-cookie based session manager
	SessionManager = scs.NewCookieManager("u46IpCV9y5Vlur8YvODJEhgOY8m9JVE4")
	templates      = template.Must(template.ParseFiles("templ/tests.html", "templ/sidebar.html", "templ/base.html"))
)

func handleDashboard(w http.ResponseWriter, r *http.Request) {

	var page Page

	session := SessionManager.Load(r)
	username, _ := session.GetString("username")
	page.Username = username

	db := DB()
	defer db.Close()

	err := db.Runs(5, &page.Data)
	if err != nil {
		log.Printf("RunQuery : ERROR : %s\n", err)
	}

	templates.ExecuteTemplate(w, "layout", &page)
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

	mux.HandleFunc("/", handleDashboard)
	http.ListenAndServe(":8080", SessionManager.Use(mux))
}
