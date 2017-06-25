package main

import (
	"fmt"
	fb "github.com/huandu/facebook"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/facebook"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"time"
)

var (
	oauthConf = &oauth2.Config{
		ClientID:     facebookClientId,
		ClientSecret: facebookClientSecret,
		RedirectURL:  facebookRedirectUrl,
		Scopes:       []string{"public_profile"},
		Endpoint:     facebook.Endpoint,
	}
	oauthStateString = facebookOauthStateString
)

func handleMain(w http.ResponseWriter, r *http.Request) {
	const htmlIndex = `<html><body>
Logged in with <a href="/login">facebook</a>
</body></html>
`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(htmlIndex))
}

func handleFacebookLogin(w http.ResponseWriter, r *http.Request) {
	authUrl, err := url.Parse(oauthConf.Endpoint.AuthURL)
	if err != nil {
		log.Fatal("Parse: ", err)
	}
	parameters := url.Values{}
	parameters.Add("client_id", oauthConf.ClientID)
	parameters.Add("scope", strings.Join(oauthConf.Scopes, " "))
	parameters.Add("redirect_uri", oauthConf.RedirectURL)
	parameters.Add("response_type", "code")
	parameters.Add("state", oauthStateString)
	authUrl.RawQuery = parameters.Encode()
	url := authUrl.String()
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func handleFacebookCallback(w http.ResponseWriter, r *http.Request) {
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

	fmt.Println(token.AccessToken)

	var tokens = ReadTokens()
	tokens.Facebook = token.AccessToken
	WriteTokens(tokens)

	os.Exit(0)
}

type FacebookEvent struct {
	Id              string `facebook:",required"`
	StartTime       string
	StartTimeParsed time.Time
	Name            string
}

type FacebookEvents struct {
	Events []FacebookEvent `facebook:"data"`
}

type VenueInformation struct {
	Name string
	Id   string
	Slug string
}

func AuthenticateFacebook() (session *fb.Session, err error) {
	var tokens = ReadTokens()

	log.Printf("Authenticating with FB...")

	app := fb.New(facebookClientId, facebookClientSecret)
	session = app.Session(tokens.Facebook)

	err = session.Validate()
	if tokens.Facebook == "" || err != nil {
		http.HandleFunc("/", handleMain)
		http.HandleFunc("/login", handleFacebookLogin)
		http.HandleFunc("/facebook/callback", handleFacebookCallback)

		fmt.Println("Started running on http://local.page5of4.com:9090")

		http.ListenAndServe(":9090", nil)
	}

	return
}

func GetVenueInformation(session *fb.Session, venueId string) (venueInfo VenueInformation, err error) {
	res, err := session.Get("/"+venueId, nil)
	if err != nil {
		return
	}

	err = res.Decode(&venueInfo)
	if err != nil {
		fmt.Println("Error", err)
	}

	return
}

func GetUpcomingFacebookEvents(session *fb.Session, venueId string) (events []FacebookEvent, err error) {
	res, err := session.Get("/"+venueId+"/events", nil)
	if err != nil {
		return
	}

	var unfiltered []FacebookEvent
	var page FacebookEvents
	paging, _ := res.Paging(session)

	for numberOfPages := 0; numberOfPages < 10; numberOfPages++ {
		paging.Decode(&page)
		done := false
		for _, event := range page.Events {
			event.StartTimeParsed, err = time.Parse("2006-01-02T15:04:05-0700", event.StartTime)

			if event.StartTimeParsed.Before(time.Now()) {
				done = true
			}

			unfiltered = append(unfiltered, event)
		}

		if done {
			break
		}

		paging.Next()
	}

	aWeekFromNow := time.Now().AddDate(0, 0, 7)
	events = unfiltered[:0]
	for _, event := range unfiltered {
		if event.StartTimeParsed.After(time.Now()) && event.StartTimeParsed.Before(aWeekFromNow) {
			events = append(events, event)
		}
	}

	return
}
