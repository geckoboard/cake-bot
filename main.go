package main

import (
	"encoding/json"
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
	gh           *github.Client
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

func ping(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	fmt.Println(w, "ok")
}
func githubWebhook(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var payload struct {
		Issue struct {
			Number int
		}
		Repository struct {
			Name  string
			Owner struct {
				Login string
			}
		}
	}

	l := log.New("endpoint", "webhook")

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		l.Error("could not unmarshal json", "err", err)
		w.WriteHeader(501)
		return
	}

	if payload.Issue.Number != 0 {
		number := payload.Issue.Number
		owner := payload.Repository.Owner.Login
		repo := payload.Repository.Name

		l = l.New("repo.name", repo, "repo.owner", owner, "issue.number", number)
		l.Info("fetching issue")

		issue, _, err := gh.Issues.Get(owner, repo, number)

		if err != nil {
			l.Error("error fetching issue", "err", err)
		}

		pr := PullRequestFromIssue(*issue, gh)

		l = l.New("issue.url", pr.URL())

		err = updateIssueReviewLabels(gh, l, pr)

		if err != nil {
			w.WriteHeader(501)
			return
		}
	}
}

func runBulkSync(c Config) {
	var wg sync.WaitGroup

	wg.Add(2)
	go func() {
		defer wg.Done()

		err := ensureOrgReposHaveLabels(c.GithubOrg, gh)

		if err != nil {
			log.Error("encountered error while ensuring all repos have lables", "err", err)
		}
	}()

	go func() {
		defer wg.Done()
		issues, err := pullRequestIssues(gh, c.GithubOrg)

		if err != nil {
			log.Error("could not load issues from github org", "err", err)
			return
		}

		for _, pr := range issues {
			l := log.New("issue.number", pr.Number(), "issue.url", pr.URL())

			go updateIssueReviewLabels(gh, l, pr)
		}
	}()

	wg.Wait()
}

func main() {
	log = log15.New()
	log.SetHandler(log15.MultiHandler(
		log15.StreamHandler(os.Stdout, log15.LogfmtFormat()),
	))

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

	gh = github.NewClient(tc)

	runBulkSync(c)

	if c.Port > 0 {
		httpServer := http.Server{
			Addr:    fmt.Sprintf(":%d", c.Port),
			Handler: NewServer(),
		}

		httpServer.ListenAndServe()
	}
}
