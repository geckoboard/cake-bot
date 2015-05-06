package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/geckoboard/goutils/router"
	github "github.com/google/go-github/github"
	"github.com/julienschmidt/httprouter"
	"golang.org/x/oauth2"
)

var GithubApiKey string

type Config struct {
	Port int
}

func newWebhookServer() http.Handler {
	r := router.New()
	r.POST("/webhooks/github", ghHandler)
	return r
}

func ghHandler(w http.ResponseWriter, r *http.Request, params httprouter.Params) {

}

// tokenSource is an oauth2.TokenSource which returns a static access token
type tokenSource struct {
	token *oauth2.Token
}

// Token implements the oauth2.TokenSource interface
func (t *tokenSource) Token() (*oauth2.Token, error) {
	return t.token, nil
}

func main() {
	var c Config

	flag.IntVar(&c.Port, "port", 0, "port to run http server on")
	flag.Parse()

	token := os.Getenv("GITHUB_ACCESS_TOKEN")

	if token == "" {
		log.Fatal("GITHUB_ACCESS_TOKEN not specified")
	}

	ts := &tokenSource{
		&oauth2.Token{AccessToken: token},
	}

	tc := oauth2.NewClient(oauth2.NoContext, ts)

	client := github.NewClient(tc)

	issues, err := pullRequestIssues(client, "geckoboard")

	if err != nil {
		log.Fatal(err)
	}

	for _, p := range issues {
		status := AwaitingCakeLabel
		isCaked := p.IsCaked()
		isWIP := p.IsWIP()

		log.Printf("%s, caked: %#v", *p.issue.Title, isCaked)

		if isWIP {
			status = WIPLabel
		} else if isCaked {
			status = CakedLabel
		}

		err = p.SetReviewLabel(status)

		if err != nil {
			log.Fatal("error changing the review label for issue", err)
		}
	}
}
