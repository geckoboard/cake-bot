package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	bugsnag "github.com/bugsnag/bugsnag-go"
	"github.com/dchest/uniuri"
	"github.com/geckoboard/cake-bot/ctx"
	github "github.com/google/go-github/github"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	log15 "gopkg.in/inconshreveable/log15.v2"
)

var (
	GithubApiKey string
	logger       log15.Logger
	gh           *github.Client
	notifier     *Notifier
)

type Config struct {
	Port             int
	GithubOrg        string
	SlackWebhook     string
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

func runBulkSync(conf Config) {
	var wg sync.WaitGroup

	c := ctx.WithLogger(context.Background(), logger.New("bulk.session", uniuri.NewLen(4)))

	ctx.Logger(c).Info("starting bulk sync")

	wg.Add(2)
	go func() {
		defer wg.Done()
		return

		err := ensureOrgReposHaveLabels(c, conf.GithubOrg, gh)

		if err != nil {
			ctx.Logger(c).Error("encountered error while ensuring all repos have lables", "err", err)
		}
	}()

	go func() {
		defer wg.Done()
		issues, err := ReviewRequestsInOrg(c, gh, conf.GithubOrg)

		if err != nil {
			ctx.Logger(c).Error("could not load issues from github org", "err", err)
			return
		}

		for _, rr := range issues {
			c2 := ctx.WithLogger(c, ctx.Logger(c).New("repo.path", rr.RepositoryPath(), "issue.number", rr.Number(), "issue.url", rr.URL()))

			wg.Add(1)

			go func(rr ReviewRequest, c2 context.Context) {
				updateIssueReviewLabels(c2, gh, rr)
				wg.Done()
			}(rr, c2)
		}
	}()

	wg.Wait()

	ctx.Logger(c).Info("finished bulk sync")
}

func periodicallyRunSync(c Config) {
	ticker := time.NewTicker(time.Second * time.Duration(c.BulkSyncInterval))

	for _ = range ticker.C {
		runBulkSync(c)
	}
}

func main() {
	logger = log15.New()
	logger.SetHandler(log15.MultiHandler(
		log15.StreamHandler(os.Stdout, log15.LogfmtFormat()),
	))

	var c Config

	flag.IntVar(&c.Port, "port", 0, "port to run http server on, if not set server does not run")
	flag.IntVar(&c.BulkSyncInterval, "bulk-sync-interval", 30, "when running as a http server run a bulk sync every X seconds")
	flag.StringVar(&c.GithubOrg, "github-org", "geckoboard", "the github org to manage issues for")
	flag.StringVar(&c.SlackWebhook, "slack-hook", "", "Slack webhook")
	flag.Parse()

	notifier = NewNotifier(c.SlackWebhook)

	token := os.Getenv("GITHUB_ACCESS_TOKEN")

	if token == "" {
		logger.Error("GITHUB_ACCESS_TOKEN not specified")
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
			Handler: bugsnag.Handler(NewServer()),
		}

		httpServer.ListenAndServe()
	} else {
		runBulkSync(c)
	}
}

func init() {
	bugsnag.Configure(bugsnag.Configuration{
		APIKey: "4bff6f651b0aa990253ce5520f4e2a51",
	})
}
