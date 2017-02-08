package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/geckoboard/cake-bot/ctx"
	"github.com/google/go-github/github"
)

const MAX_TITLE_LENGTH = 30

var slackHook = &http.Client{}

type Notifier struct {
	directory SlackUserDirectory
	Webhook   string
}

type CakeEvent struct {
	Channel   string `json:"channel"`
	Username  string `json:"username"`
	Text      string `json:"text"`
	IconEmoji string `json:"icon_emoji"`
}

func NewNotifier(d SlackUserDirectory, url string) *Notifier {
	return &Notifier{
		directory: d,
		Webhook:   url,
	}
}

func prLink(repo github.Repository, pr github.PullRequest, review PullRequestReview) string {
	title := *pr.Title
	if len(title) > MAX_TITLE_LENGTH {
		title = fmt.Sprintf("%s...", title[0:MAX_TITLE_LENGTH])
	}

	return fmt.Sprintf("<%s|%s#%d> - %s", review.URL(), *repo.Name, *pr.Number, title)
}

func (n Notifier) Approved(c context.Context, repo github.Repository, pr github.PullRequest, review PullRequestReview) error {
	e := CakeEvent{
		Channel:  "#devs",
		Username: "cake-bot",
		Text: fmt.Sprintf(
			"%s you have received a :cake: for %s",
			n.directory.BuildLinkToUser(pr.User),
			prLink(repo, pr, review),
		),
		IconEmoji: ":sheep:",
	}

	return n.sendMessage(c, e)
}

func (n Notifier) ChangesRequested(c context.Context, repo github.Repository, pr github.PullRequest, review PullRequestReview) error {
	e := CakeEvent{
		Channel:  "#devs",
		Username: "cake-bot",
		Text: fmt.Sprintf(
			"%s you have received some feedback on this PR: %s",
			n.directory.BuildLinkToUser(pr.User),
			prLink(repo, pr, review),
		),
		IconEmoji: ":sheep:",
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
