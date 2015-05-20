package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/geckoboard/goutils/router"
	github "github.com/google/go-github/github"
	"github.com/julienschmidt/httprouter"
	"golang.org/x/oauth2"
	log15 "gopkg.in/inconshreveable/log15.v2"
)

var (
	GithubApiKey string
	log          log15.Logger
)

type Config struct {
	Port      int
	GithubOrg string
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

func NewServer() http.Handler {
	r := router.New()
	r.GET("/ping", ping)
	r.POST("/github", githubWebhook)
	return r
}

func ping(_ http.ResponseWriter, _ *http.Request, _ httprouter.Params) {

}
func githubWebhook(_ http.ResponseWriter, _ *http.Request, _ httprouter.Params) {

}

func runBulkSync(client *github.Client, c Config) {
	var wg sync.WaitGroup

	wg.Add(2)
	go func() {
		defer wg.Done()

		err := ensureOrgReposHaveLabels(c.GithubOrg, client)

		if err != nil {
			log.Error("encountered error while ensuring all repos have lables", "err", err)
		}
	}()

	go func() {
		defer wg.Done()
		issues, err := pullRequestIssues(client, c.GithubOrg)

		if err != nil {
			log.Error("could not load issues from github org", "err", err)
			return
		}

		for _, pr := range issues {
			err := updateIssueReviewLabels(client, &pr)

			if err != nil {
				log.Error("cannot change review label for issue", "err", err, "issue.number", pr.Number(), "issue.html_url", pr.URL())
			}
		}
	}()

	wg.Wait()
}

func main() {
	log = log15.New()

	var c Config

	flag.IntVar(&c.Port, "port", 0, "port to run http server on, if not set server does not run")
	flag.StringVar(&c.GithubOrg, "github-org", "geckoboard", "the github org to manage issues for")
	flag.Parse()

	token := os.Getenv("GITHUB_ACCESS_TOKEN")

	if token == "" {
		log.Error("GITHUB_ACCESS_TOKEN not specified")
		os.Exit(1)
	}

	ts := &tokenSource{
		&oauth2.Token{AccessToken: token},
	}

	tc := oauth2.NewClient(oauth2.NoContext, ts)

	client := github.NewClient(tc)

	runBulkSync(client, c)

	if c.Port > 0 {
		httpServer := http.Server{
			Addr:    fmt.Sprintf(":%d", c.Port),
			Handler: NewServer(),
		}

		httpServer.ListenAndServe()
	}
}
