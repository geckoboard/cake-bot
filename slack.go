package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/geckoboard/cake-bot/ctx"
	"github.com/google/go-github/github"
)

var slackHook = &http.Client{}

type Notifier struct {
	directory SlackUserDirectory
	Webhook   string
	Token     string
}

type CakeEvent struct {
	Channel   string `json:"channel"`
	Username  string `json:"username"`
	Text      string `json:"text"`
	IconEmoji string `json:"icon_emoji"`
	Parse     string `json:"parse"`
}

func NewNotifier(d SlackUserDirectory, url, token string) *Notifier {
	return &Notifier{
		directory: d,
		Webhook:   url,
		Token:     token,
	}
}

func (n Notifier) Approved(c context.Context, pr github.PullRequest, review PullRequestReview) error {
	user := n.directory.FindUserByGithubUser(pr.User)

	e := CakeEvent{
		Channel:   "#devs",
		Username:  "cake-bot",
		Text:      "@" + user + " you have received a :cake: for " + review.URL(),
		IconEmoji: ":sheep:",
		Parse:     "full",
	}

	return n.sendMessage(c, e)
}

func (n Notifier) ChangesRequested(c context.Context, pr github.PullRequest, review PullRequestReview) error {
	user := n.directory.FindUserByGithubUser(pr.User)

	e := CakeEvent{
		Channel:   "#devs",
		Username:  "cake-bot",
		Text:      "@" + user + " you have received some feedback on this PR " + review.URL(),
		IconEmoji: ":sheep:",
		Parse:     "full",
	}

	return n.sendMessage(c, e)
}

func (n Notifier) sendMessage(c context.Context, e CakeEvent) error {
	l := ctx.Logger(c).With("at", "slack.ping-user")

	payload, err := json.Marshal(e)
	if err != nil {
		l.Error("msg", "unable to encode cake event", "err", err)
		return err
	}

	req, err := http.NewRequest("POST", n.Webhook, bytes.NewBuffer(payload))
	if err != nil {
		l.Error("msg", "unable to create request", "err", err)
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := slackHook.Do(req)
	if err != nil {
		l.Error("msg", "unable to create request", "err", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		l.Error("msg", "unexpected response status", "resp.Status", resp.Status)
		return err
	}

	l.Info("msg", "ping successful")
	return nil
}
