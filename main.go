package main

import (
	"html/template"
	"net/http"
)

var templates = template.Must(template.ParseFiles("templ/dashboard.html"))

func handler(w http.ResponseWriter, r *http.Request) {

	templates.ExecuteTemplate(w, "dashboard.html", nil)
}

func main() {

	// Serve css directory as static files
	fs := http.FileServer(http.Dir("css"))
	http.Handle("/css/", http.StripPrefix("/css/", fs))

	http.HandleFunc("/", handler)
	http.ListenAndServe(":8080", nil)
}
