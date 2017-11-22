package main

import (
	"context"
	"fmt"

	"github.com/geckoboard/cake-bot/ctx"
	"github.com/geckoboard/cake-bot/github"
	"github.com/geckoboard/cake-bot/slack"
	slackapi "github.com/nlopes/slack"
)

type Notifier interface {
	Approved(context.Context, *github.Repository, *github.PullRequest, *github.Review) error
	ChangesRequested(context.Context, *github.Repository, *github.PullRequest, *github.Review) error
	ReviewRequested(context.Context, *github.PullRequestWebhook) error
}

const (
	maxTitleLength = 35
	devsChannel    = "#devs"
)

type SlackNotifier struct {
	client *slackapi.Client
}

func NewSlackNotifier(client *slackapi.Client) *SlackNotifier {
	return &SlackNotifier{client}
}

func (n *SlackNotifier) Approved(c context.Context, repo *github.Repository, pr *github.PullRequest, review *github.Review) error {
	text := fmt.Sprintf(
		"%s you have received a :cake: for %s",
		buildLinkToUser(pr.User),
		prLink(review.HTMLURL(), repo, pr),
	)
	return n.sendMessage(c, devsChannel, text)
}

func (n *SlackNotifier) ChangesRequested(c context.Context, repo *github.Repository, pr *github.PullRequest, review *github.Review) error {
	text := fmt.Sprintf(
		"%s you have received some feedback on %s",
		buildLinkToUser(pr.User),
		prLink(review.HTMLURL(), repo, pr),
	)
	return n.sendMessage(c, devsChannel, text)
}

func (n *SlackNotifier) ReviewRequested(c context.Context, webhook *github.PullRequestWebhook) error {
	url := webhook.PullRequest.HTMLURL
	text := fmt.Sprintf(
		"%s you have been asked by %s to review %s",
		buildLinkToUser(webhook.RequestedReviewer),
		buildLinkToUser(webhook.Sender),
		prLink(url, webhook.Repository, webhook.PullRequest),
	)
	return n.sendMessage(c, devsChannel, text)
}

func (n *SlackNotifier) sendMessage(c context.Context, channel, text string) error {
	l := ctx.Logger(c).With("at", "slack.ping-user")

	_, _, err := n.client.PostMessage(channel, text, slackapi.PostMessageParameters{})
	if err != nil {
		l.Error("msg", "unable to post message", "err", err)
		return err
	}

	l.Info("msg", "ping successful")
	return nil
}

func prLink(url string, repo *github.Repository, pr *github.PullRequest) string {
	title := pr.Title
	if len(title) > maxTitleLength {
		title = fmt.Sprintf("%s...", title[0:maxTitleLength])
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
