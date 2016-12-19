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
	"github.com/joho/godotenv"
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

	err := godotenv.Load()
	if err != nil {
		logger.Error("Error loading .env file")
		os.Exit(1)
	}

	var c Config

	flag.IntVar(&c.Port, "port", 8090, "port to run http server on, if not set server does not run")
	flag.Parse()

	required := map[string]string{
		"GITHUB_ACCESS_TOKEN": "",
		"SLACK_TOKEN":         "",
		"SLACK_HOOK":          "",
	}

	for k, _ := range required {
		val := os.Getenv(k)
		if val == "" {
			logger.Error("msg", val+" not specified")
			os.Exit(1)
		}
		required[k] = val
	}

	gh := github.NewClient(
		oauth2.NewClient(
			oauth2.NoContext,
			&tokenSource{
				&oauth2.Token{AccessToken: required["GITHUB_ACCESS_TOKEN"]},
			},
		),
	)

	sl := slack.New(required["SLACK_TOKEN"])

	users := NewSlackUserDirectory(gh, sl)
	if err := users.ScanSlackTeam(); err != nil {
		logger.Error("msg", fmt.Sprintf("building Slack user map raised an error: %s", err.Error()))
		os.Exit(1)
	}

	notifier := NewNotifier(users, required["SLACK_HOOK"])

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
