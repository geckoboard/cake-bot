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

	if err := n.notifyChannel(c, devsChannel, text); err != nil {
		return err
	}

	return n.tryNotifyUser(c, pr.User, text)
}

func (n *SlackNotifier) ChangesRequested(c context.Context, repo *github.Repository, pr *github.PullRequest, review *github.Review) error {
	text := fmt.Sprintf(
		"%s you have received some feedback on %s",
		buildLinkToUser(pr.User),
		prLink(review.HTMLURL(), repo, pr),
	)

	if err := n.notifyChannel(c, devsChannel, text); err != nil {
		return err
	}

	return n.tryNotifyUser(c, pr.User, text)
}

func (n *SlackNotifier) ReviewRequested(c context.Context, webhook *github.PullRequestWebhook) error {
	url := webhook.PullRequest.HTMLURL
	text := fmt.Sprintf(
		"%s you have been asked by %s to review %s",
		buildLinkToUser(webhook.RequestedReviewer),
		buildLinkToUser(webhook.Sender),
		prLink(url, webhook.Repository, webhook.PullRequest),
	)

	if err := n.notifyChannel(c, devsChannel, text); err != nil {
		return err
	}

	return n.tryNotifyUser(c, webhook.RequestedReviewer, text)
}

func (n *SlackNotifier) tryNotifyUser(c context.Context, ghUser *github.User, text string) error {
	if user := findSlackUser(ghUser); user != nil {
		return n.notifyUser(c, user.ID, text)
	}
	return nil
}

func (n *SlackNotifier) notifyUser(c context.Context, userID, text string) error {
	_, _, channel, err := n.client.OpenIMChannel(userID)
	if err != nil {
		return err
	}
	return n.notifyChannel(c, channel, text)
}

func (n *SlackNotifier) notifyChannel(c context.Context, channel, text string) error {
	l := ctx.Logger(c).With("at", "slack.ping-user")

	params := slackapi.NewPostMessageParameters()
	params.AsUser = true
	params.EscapeText = false

	_, _, err := n.client.PostMessage(channel, text, params)
	if err != nil {
		l.Error("msg", "unable to post message", "err", err)
		return err
	}

	l.Info("msg", "ping successful")
	return nil
}

func buildLinkToUser(ghUser *github.User) string {
	if user := findSlackUser(ghUser); user != nil {
		return fmt.Sprintf("<@%s>", user.ID)
	}
	return ghUser.Login
}

func findSlackUser(ghUser *github.User) *slack.User {
	users := slack.Users.FindByGithubUsername(ghUser.Login)
	if len(users) > 0 {
		return &users[0]
	}
	return nil
}

func prLink(url string, repo *github.Repository, pr *github.PullRequest) string {
	title := pr.Title
	if len(title) > maxTitleLength {
		title = fmt.Sprintf("%s...", title[0:maxTitleLength])
	}
	return fmt.Sprintf("<%s|%s#%d> - %s", url, repo.Name, pr.Number, title)
}
