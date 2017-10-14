package main

import (
	"fmt"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	"net/http"
	"os"
)

var (
	oauthConf = &oauth2.Config{
		ClientID:     os.Getenv("githubkey"),
		ClientSecret: os.Getenv("githubsecret"),
		Scopes:       []string{"user:email"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://github.com/login/oauth/authorize",
			TokenURL: "https://github.com/login/oauth/access_token",
		},
	}
	// Random string for oauth2 API calls to protect against CSRF
	oauthStateString = "thisshouldberandom"
)

// Login through GitHub’s authorization page
func HandleGitHubLogin(w http.ResponseWriter, r *http.Request) {

	url := oauthConf.AuthCodeURL(oauthStateString)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// Logout from current session
func HandleGitHubLogout(w http.ResponseWriter, r *http.Request) {

	session := SessionManager.Load(r)
	session.PutString(w, "username", "")
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

// Called by github after authorization is granted
func HandleGitHubCallback(w http.ResponseWriter, r *http.Request) {

	session := SessionManager.Load(r)
	session.PutString(w, "username", "")

	state := r.FormValue("state")
	if state != oauthStateString {
		fmt.Printf("invalid oauth state, expected '%s', got '%s'\n", oauthStateString, state)
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	code := r.FormValue("code")
	token, err := oauthConf.Exchange(oauth2.NoContext, code)
	if err != nil {
		fmt.Printf("oauthConf.Exchange() failed with '%s'\n", err)
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	oauthClient := oauthConf.Client(oauth2.NoContext, token)
	client := github.NewClient(oauthClient)
	user, _, err := client.Users.Get(oauth2.NoContext, "")
	if err != nil {
		fmt.Printf("client.Users.Get() faled with '%s'\n", err)
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	// Save credentials on the client side using a secure cookie
	session.PutString(w, "username", *user.Login)

	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}
