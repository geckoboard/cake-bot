package main

import (
	"fmt"
	"net/http"
	"os"

	bugsnag "github.com/bugsnag/bugsnag-go"
	"github.com/geckoboard/cake-bot/log"
	"github.com/geckoboard/cake-bot/slack"
	"github.com/joho/godotenv"
)

var (
	logger log.LeveledLogger = log.New()
)

func main() {
	if err := godotenv.Load(); err != nil {
		logger.Error("msg", "Error loading .env file")
	}

	var (
		httpPort   = mustGetenv("PORT")
		slackToken = mustGetenv("SLACK_TOKEN")
		slackHook  = mustGetenv("SLACK_HOOK")
	)

	users := NewSlackUserDirectory(slack.New(slackToken))
	if err := users.ScanSlackTeam(); err != nil {
		logger.Error("msg", fmt.Sprintf("Error building Slack user map: %v", err))
		os.Exit(1)
	}

	notifier := NewNotifier(users, slackHook)
	httpServer := http.Server{
		Addr:    ":" + httpPort,
		Handler: bugsnag.Handler(NewServer(notifier)),
	}

	logger.Info("msg", fmt.Sprintf("Listening on port %s", httpPort))
	httpServer.ListenAndServe()
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
