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
	"github.com/geckoboard/cake-bot/log"
	github "github.com/google/go-github/github"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
)

var (
	GithubApiKey string
	logger       log.LeveledLogger
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

func bulksyncLabels(conf Config) {
	c := ctx.WithLogger(context.Background(), logger.With("bulk_sync_labels.session", uniuri.NewLen(4)))

	ctx.Logger(c).Info("at", "bulk_sync_labels.start")

	err := ensureOrgReposHaveLabels(c, conf.GithubOrg, gh)

	if err != nil {
		ctx.Logger(c).Error("at", "sync_labels.error", "err", err)
	}

	ctx.Logger(c).Info("at", "bulk_sync_labels.end")
}

func bulkSyncCakes(conf Config) {
	var wg sync.WaitGroup

	c := ctx.WithLogger(context.Background(), logger.With("bulk_sync_cakes.session", uniuri.NewLen(4)))

	ctx.Logger(c).Info("at", "bulk_sync_cakes.start")

	stream := NewReviewRequestStream(gh, conf.GithubOrg).Stream(c)

	for {
		next, channelOpen := <-stream

		if !channelOpen {
			return
		}

		wg.Add(1)

		go func(sessionCtx context.Context, githubConnection *github.Client, review ReviewRequest) {
			defer wg.Done()

			reviewCtx := ctx.WithLogger(
				sessionCtx,
				ctx.Logger(sessionCtx).With(
					"issue.repo", review.RepositoryPath(),
					"issue.number", review.Number(),
				),
			)

			updateIssueReviewLabels(reviewCtx, githubConnection, review)

		}(c, gh, next)
	}

	wg.Wait()

	ctx.Logger(c).Info("at", "bulk_sync_cakes.end")
}

func periodicallyRunSync(c Config) {
	ticker := time.NewTicker(time.Second * time.Duration(c.BulkSyncInterval))

	for _ = range ticker.C {
		bulkSyncCakes(c)
	}
}

func main() {
	logger = log.New()

	var c Config

	flag.IntVar(&c.Port, "port", 0, "port to run http server on, if not set server does not run")
	flag.IntVar(&c.BulkSyncInterval, "bulk-sync-interval", 30, "when running as a http server run a bulk sync every X seconds")
	flag.StringVar(&c.GithubOrg, "github-org", "geckoboard", "the github org to manage issues for")
	flag.StringVar(&c.SlackWebhook, "slack-hook", "", "Slack webhook")
	flag.Parse()

	token := os.Getenv("GITHUB_ACCESS_TOKEN")

	if token == "" {
		logger.Error("msg", "GITHUB_ACCESS_TOKEN not specified")
		os.Exit(1)
	}

	slackToken := os.Getenv("SLACK_TOKEN")

	if slackToken == "" {
		logger.Error("msg", "SLACK_TOKEN not specified")
		os.Exit(1)
	}

	notifier = NewNotifier(c.SlackWebhook, token)
	if err := notifier.BuildSlackUserMap(); err != nil {
		logger.Error("msg", fmt.Sprintf("building Slack user map raised an error: %s", err.Error()))
		os.Exit(1)
	}

	ts := &tokenSource{
		&oauth2.Token{AccessToken: token},
	}

	tc := oauth2.NewClient(oauth2.NoContext, ts)

	gh = github.NewClient(tc)

	bulksyncLabels(c)

	if c.Port > 0 {
		go periodicallyRunSync(c)

		httpServer := http.Server{
			Addr:    fmt.Sprintf(":%d", c.Port),
			Handler: bugsnag.Handler(NewServer()),
		}

		httpServer.ListenAndServe()
	} else {
		bulkSyncCakes(c)
	}
}

func init() {
	bugsnag.Configure(bugsnag.Configuration{
		APIKey: os.Getenv("BUGSNAG_API_KEY"),
	})
}
