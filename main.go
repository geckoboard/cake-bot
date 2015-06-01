package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/dchest/uniuri"
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
	Port             int
	GithubOrg        string
	BulkSyncInterval int
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
	l := log.New("endpoint", "webhook")

	event := r.Header.Get("X-GitHub-Event")

	switch event {
	case "pull_request", "issue_comment":
		// handle request
	default:
		l.Info("not handling webhook", "github_event", event)
		w.WriteHeader(http.StatusOK)
		return
	}

	var payload struct {
		Action      string
		Issue       *github.Issue
		Repository  *github.Repository
		PullRequest *github.PullRequest `json:"pull_request"`
	}
	var err error

	var triggerInspection bool

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		l.Error("could not unmarshal json", "err", err)
		w.WriteHeader(501)
		return
	}

	l = log.New("endpoint", "webhook", "action", payload.Action)

	if payload.Repository != nil {
		l = l.New("repo.path", fmt.Sprintf("%s/%s", *payload.Repository.Name, *payload.Repository.Owner.Login))
	}

	if payload.Issue != nil {
		l = l.New(
			"issue.number", *payload.Issue.Number,
			"issue.url", *payload.Issue.HTMLURL,
		)
	}

	if payload.Issue != nil && *payload.Issue.Number != 0 && payload.Issue.PullRequestLinks != nil {
		triggerInspection = true

		l.Info("found issue with pr links")
	} else if payload.PullRequest != nil && payload.Action != "" {
		triggerInspection = true

		l.Info("found pr opened event, inferring issue")

		payload.Issue, _, err = gh.Issues.Get(*payload.Repository.Owner.Login, *payload.Repository.Name, *payload.PullRequest.Number)

		if err != nil {
			l.Error("encountered error while loading issue", "err", err)
			w.WriteHeader(501)
			return
		}
	} else {
		l.Info("payload does not refer to pull request", "github_event", event)
	}

	if triggerInspection {

		pr := ReviewRequestFromIssue(*payload.Repository, *payload.Issue, gh)

		err = updateIssueReviewLabels(gh, l, pr)

		if err != nil {
			w.WriteHeader(501)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

func runBulkSync(c Config) {
	var wg sync.WaitGroup

	bl := log.New("bulk.session", uniuri.NewLen(4))

	bl.Info("starting bulk sync")

	wg.Add(2)
	go func() {
		defer wg.Done()
		return

		err := ensureOrgReposHaveLabels(c.GithubOrg, gh)

		if err != nil {
			bl.Error("encountered error while ensuring all repos have lables", "err", err)
		}
	}()

	go func() {
		defer wg.Done()
		issues, err := ReviewRequestsInOrg(gh, c.GithubOrg)

		if err != nil {
			bl.Error("could not load issues from github org", "err", err)
			return
		}

		for _, rr := range issues {
			l := bl.New("repo.path", rr.RepositoryPath(), "issue.number", rr.Number(), "issue.url", rr.URL())

			wg.Add(1)

			go func(pr ReviewRequest, l log15.Logger) {
				updateIssueReviewLabels(gh, l, pr)
				wg.Done()
			}(rr, l)
		}
	}()

	wg.Wait()

	bl.Info("finished bulk sync")
}

func periodicallyRunSync(c Config) {
	ticker := time.NewTicker(time.Second * time.Duration(c.BulkSyncInterval))

	for _ = range ticker.C {
		runBulkSync(c)
	}
}

func main() {
	log = log15.New()
	log.SetHandler(log15.MultiHandler(
		log15.StreamHandler(os.Stdout, log15.LogfmtFormat()),
	))

	var c Config

	flag.IntVar(&c.Port, "port", 0, "port to run http server on, if not set server does not run")
	flag.IntVar(&c.BulkSyncInterval, "bulk-sync-interval", 30, "when running as a http server run a bulk sync every X seconds")
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

	if c.Port > 0 {
		go periodicallyRunSync(c)

		httpServer := http.Server{
			Addr:    fmt.Sprintf(":%d", c.Port),
			Handler: NewServer(),
		}

		httpServer.ListenAndServe()
	} else {
		runBulkSync(c)
	}
}
