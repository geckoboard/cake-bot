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
	ReviewRequested(context.Context, *github.Repository, *github.PullRequest, *github.User) error
	Approved(context.Context, *github.Repository, *github.PullRequest, *github.Review) error
	ChangesRequested(context.Context, *github.Repository, *github.PullRequest, *github.Review) error
}

const (
	maxTitleLength = 80
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

func (n *SlackNotifier) ReviewRequested(c context.Context, repo *github.Repository, pr *github.PullRequest, reviewer *github.User) error {
	text := fmt.Sprintf(
		"%s you have been asked by %s to review %s",
		buildLinkToUser(reviewer), buildUserName(pr.User),
		prLink(pr.HTMLURL, repo, pr),
	)

	if err := n.notifyChannel(c, devsChannel, text); err != nil {
		return err
	}

	if err := n.tryNotifyUser(c, reviewer, text); err != nil {
		return err
	}

	presenceText := fmt.Sprintf(
		"%s seems away and might not able to review %s",
		buildLinkToUser(reviewer),
		prLink(pr.HTMLURL, repo, pr),
	)

	return n.tryNotifyPresence(c, reviewer, pr.User, presenceText)
}

func (n *SlackNotifier) tryNotifyPresence(c context.Context, ghReviewer *github.User, ghReviewee *github.User, text string) error {
	reviewer := findSlackUser(ghReviewer)
	if reviewer == nil {
		return nil
	}
	presence := n.findSlackUserPresence(reviewer)

	if presence == "away" {
		if reviewee := findSlackUser(ghReviewee); reviewee != nil {
			return n.notifyUser(c, reviewee.ID, text)
		}
	}

	return nil
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

func (n *SlackNotifier) findSlackUserPresence(user *slack.User) string {
	up, err := n.client.GetUserPresence(user.ID)
	if err != nil {
		return ""
	}

	return up.Presence
}

func buildLinkToUser(ghUser *github.User) string {
	if user := findSlackUser(ghUser); user != nil {
		return fmt.Sprintf("<@%s>", user.ID)
	}
	return ghUser.Login
}

func buildUserName(ghUser *github.User) interface{} {
	if user := findSlackUser(ghUser); user != nil {
		return user.Name
	}
	return ghUser.Login
}

func findSlackUser(ghUser *github.User) *slack.User {
	users := slack.Users.FindByGitHubUsername(ghUser.Login)
	if len(users) > 0 {
		return users[0]
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
