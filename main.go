package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	bugsnag "github.com/bugsnag/bugsnag-go"
	"github.com/geckoboard/cake-bot/log"
	"github.com/geckoboard/cake-bot/slack"
	"github.com/joho/godotenv"
	slackapi "github.com/slack-go/slack"
)

var (
	logger log.LeveledLogger = log.New()
)

func main() {
	if err := godotenv.Load(); err != nil {
		logger.Error("msg", "Error loading .env file")
	}

	var (
		httpPort     = mustGetenv("PORT")
		githubSecret = mustGetenv("GITHUB_SECRET")
		slackToken   = mustGetenv("SLACK_TOKEN")
	)

	slackClient := slack.New(slackToken)

	go func() {
		refreshSlackUsers(slackClient)
		for range time.Tick(5 * time.Minute) {
			refreshSlackUsers(slackClient)
		}
	}()

	notifier := NewSlackNotifier(slackapi.New(slackToken))
	webhookValidator := NewGitHubWebhookValidator(githubSecret)
	httpServer := http.Server{
		Addr:    ":" + httpPort,
		Handler: bugsnag.Handler(NewServer(notifier, webhookValidator)),
	}

	logger.Info("msg", fmt.Sprintf("Listening on port %s", httpPort))
	_ = httpServer.ListenAndServe()
}

func refreshSlackUsers(slackClient *slack.Client) {
	if err := slack.Users.Load(slackClient); err != nil {
		logger.Error("msg", "couldn't load Slack users", "err", err)
	}
}

func mustGetenv(key string) string {
	str := os.Getenv(key)
	if str == "" {
		logger.Error("msg", fmt.Sprintf("Missing environment variable: %s", key))
		os.Exit(1)
	}
	return str
}

func init() {
	bugsnag.Configure(bugsnag.Configuration{
		APIKey: os.Getenv("BUGSNAG_API_KEY"),
	})
}
