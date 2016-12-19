package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	bugsnag "github.com/bugsnag/bugsnag-go"
	"github.com/geckoboard/cake-bot/log"
	"github.com/geckoboard/cake-bot/slack"
	github "github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

var (
	GithubApiKey string
	logger       log.LeveledLogger
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

	gh := github.NewClient(
		oauth2.NewClient(
			oauth2.NoContext,
			&tokenSource{
				&oauth2.Token{AccessToken: token},
			},
		),
	)

	sl := slack.New(slackToken)

	users := NewSlackUserDirectory(gh, sl)
	if err := users.ScanSlackTeam(); err != nil {
		logger.Error("msg", fmt.Sprintf("building Slack user map raised an error: %s", err.Error()))
		os.Exit(1)
	}

	notifier := NewNotifier(users, c.SlackWebhook, slackToken)

	httpServer := http.Server{
		Addr:    fmt.Sprintf(":%d", c.Port),
		Handler: bugsnag.Handler(NewServer(notifier)),
	}

	httpServer.ListenAndServe()
}

func init() {
	bugsnag.Configure(bugsnag.Configuration{
		APIKey: os.Getenv("BUGSNAG_API_KEY"),
	})
}
