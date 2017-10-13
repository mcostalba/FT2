package main

import (
	"html/template"
	"net/http"
)

var templates = template.Must(template.ParseFiles("templ/dashboard.html"))

func handleDashboard(w http.ResponseWriter, r *http.Request) {

	templates.ExecuteTemplate(w, "dashboard.html", nil)
}

func main() {

	// Serve css directory as static files
	fs := http.FileServer(http.Dir("css"))
	http.Handle("/css/", http.StripPrefix("/css/", fs))

    // Handles used for GitHub OAuth login
	http.HandleFunc("/login/", HandleGitHubLogin)
	http.HandleFunc("/github_oauth_cb", HandleGitHubCallback) // Note missing trailing slash!

	http.HandleFunc("/", handleDashboard)
	http.ListenAndServe(":8080", nil)
}
