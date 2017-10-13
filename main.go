package main

import (
	"github.com/alexedwards/scs"
	"html/template"
	"net/http"
)

var templates = template.Must(template.ParseFiles("templ/tests.html", "templ/sidebar.html", "templ/base.html"))

// Initialize a new encrypted-cookie based session manager
var SessionManager = scs.NewCookieManager("u46IpCV9y5Vlur8YvODJEhgOY8m9JVE4")

func handleDashboard(w http.ResponseWriter, r *http.Request) {

	session := SessionManager.Load(r)
	username, _ := session.GetString("username")

	data := struct{ Username string }{username}
	templates.ExecuteTemplate(w, "layout", &data)
}

func main() {

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
