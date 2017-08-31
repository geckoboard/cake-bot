package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/geckoboard/cake-bot/ctx"
	"github.com/geckoboard/cake-bot/github"
	"github.com/geckoboard/cake-bot/slack"
)

type Notifier interface {
	Approved(context.Context, *github.Repository, *github.PullRequest, *github.Review) error
	ChangesRequested(context.Context, *github.Repository, *github.PullRequest, *github.Review) error
	ReviewRequested(context.Context, *github.PullRequestWebhook) error
}

const MAX_TITLE_LENGTH = 35

var slackHook = &http.Client{}

type SlackNotifier struct {
	Webhook string
}

type CakeEvent struct {
	Channel   string `json:"channel"`
	Username  string `json:"username"`
	Text      string `json:"text"`
	IconEmoji string `json:"icon_emoji"`
}

func NewSlackNotifier(url string) *SlackNotifier {
	return &SlackNotifier{
		Webhook: url,
	}
}

func (n *SlackNotifier) Approved(c context.Context, repo *github.Repository, pr *github.PullRequest, review *github.Review) error {
	evt := CakeEvent{
		Channel:  "#devs",
		Username: "cake-bot",
		Text: fmt.Sprintf(
			"%s you have received a :cake: for %s",
			buildLinkToUser(pr.User),
			prLink(review.HTMLURL(), repo, pr),
		),
		IconEmoji: ":sheep:",
	}

	return n.sendMessage(c, evt)
}

func (n *SlackNotifier) ChangesRequested(c context.Context, repo *github.Repository, pr *github.PullRequest, review *github.Review) error {
	evt := CakeEvent{
		Channel:  "#devs",
		Username: "cake-bot",
		Text: fmt.Sprintf(
			"%s you have received some feedback on %s",
			buildLinkToUser(pr.User),
			prLink(review.HTMLURL(), repo, pr),
		),
		IconEmoji: ":sheep:",
	}

	return n.sendMessage(c, evt)
}

func (n *SlackNotifier) ReviewRequested(c context.Context, webhook *github.PullRequestWebhook) error {
	url := webhook.PullRequest.HTMLURL
	evt := CakeEvent{
		Channel:  "#devs",
		Username: "cake-bot",
		Text: fmt.Sprintf(
			"%s you have been asked by %s to review %s",
			buildLinkToUser(webhook.RequestedReviewer),
			buildLinkToUser(webhook.Sender),
			prLink(url, webhook.Repository, webhook.PullRequest),
		),
		IconEmoji: ":sheep:",
	}

	return n.sendMessage(c, evt)
}

func (n *SlackNotifier) sendMessage(c context.Context, e CakeEvent) error {
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

func prLink(url string, repo *github.Repository, pr *github.PullRequest) string {
	title := pr.Title
	if len(title) > MAX_TITLE_LENGTH {
		title = fmt.Sprintf("%s...", title[0:MAX_TITLE_LENGTH])
	}
	return fmt.Sprintf("<%s|%s#%d> - %s", url, repo.Name, pr.Number, title)
}

func buildLinkToUser(ghUser *github.User) string {
	users := slack.Users.FindByGithubUsername(ghUser.Login)
	if users != nil {
		return fmt.Sprintf("<@%s>", users[0].ID)
	}
	return ghUser.Login
}
