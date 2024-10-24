package main

import (
	"context"
	"fmt"
	"os"

	"github.com/geckoboard/cake-bot/ctx"
	"github.com/geckoboard/cake-bot/github"
	"github.com/geckoboard/cake-bot/slack"
	slackapi "github.com/slack-go/slack"
)

type Notifier interface {
	ReviewRequested(context.Context, *github.Repository, *github.PullRequest, *github.User) error
	Approved(context.Context, *github.Repository, *github.PullRequest, *github.Review) error
	ChangesRequested(context.Context, *github.Repository, *github.PullRequest, *github.Review) error
}

const (
	maxTitleLength = 80
)

var (
	notificationChannel string
)

type SlackNotifier struct {
	client *slackapi.Client
}

func NewSlackNotifier(client *slackapi.Client) *SlackNotifier {
	targetChannel, ok := os.LookupEnv("SLACK_NOTIFICATION_CHANNEL")
	if !ok {
		notificationChannel = "#devs"
	}
	notificationChannel = targetChannel
	return &SlackNotifier{client}
}

func (n *SlackNotifier) Approved(c context.Context, repo *github.Repository, pr *github.PullRequest, review *github.Review) error {
	text := fmt.Sprintf(
		"%s you have received a :cake: for %s",
		buildLinkToUser(pr.User),
		prLink(review.HTMLURL(), repo, pr),
	)

	messageBlocks := []slackapi.Block{buildTextMessageBlock(text)}
	return n.notifyChannel(c, notificationChannel, messageBlocks)
}

func (n *SlackNotifier) ChangesRequested(c context.Context, repo *github.Repository, pr *github.PullRequest, review *github.Review) error {
	text := fmt.Sprintf(
		"%s you have received some feedback on %s",
		buildLinkToUser(pr.User),
		prLink(review.HTMLURL(), repo, pr),
	)

	messageBlocks := []slackapi.Block{buildTextMessageBlock(text)}
	return n.notifyChannel(c, notificationChannel, messageBlocks)
}

func (n *SlackNotifier) ReviewRequested(c context.Context, repo *github.Repository, pr *github.PullRequest, reviewer *github.User) error {
	text := fmt.Sprintf(
		"%s you have been asked by %s to review %s",
		buildLinkToUser(reviewer), buildUserName(pr.User),
		prLink(pr.HTMLURL, repo, pr),
	)

	messageBlocks := []slackapi.Block{buildTextMessageBlock(text)}
	err := n.notifyChannel(c, notificationChannel, messageBlocks)
	if err != nil {
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

func (n *SlackNotifier) notifyUser(c context.Context, userID, text string) error {
	channel, _, _, err := n.client.OpenConversation(&slackapi.OpenConversationParameters{
		Users: []string{userID},
	})
	if err != nil {
		return err
	}

	messageBlocks := []slackapi.Block{buildTextMessageBlock(text)}
	return n.notifyChannel(c, channel.ID, messageBlocks)
}

// notifyChannel sends a message to a channel constructed from blocks
// Callers should pass the channel ID, not the channel name (e.g. "C1234567890")
func (n *SlackNotifier) notifyChannel(c context.Context, channel string, blocks []slackapi.Block) error {
	params := slackapi.NewPostMessageParameters()
	params.AsUser = true
	params.EscapeText = false

	_, _, err := n.client.PostMessageContext(
		c,
		channel,
		slackapi.MsgOptionBlocks(blocks...),
		slackapi.MsgOptionPostMessageParameters(params),
	)
	return err
}

func (n *SlackNotifier) findSlackUserPresence(user *slackapi.User) string {
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

func findSlackUser(ghUser *github.User) *slackapi.User {
	return slack.Users.FindByGitHubUsername(ghUser.Login)
}

func prLink(url string, repo *github.Repository, pr *github.PullRequest) string {
	title := pr.Title
	if len(title) > maxTitleLength {
		title = fmt.Sprintf("%s...", title[0:maxTitleLength])
	}
	return fmt.Sprintf("<%s|%s#%d> - %s", url, repo.Name, pr.Number, title)
}

// buildTextMessageBlock returns a single slackapi.Block text
// The block supports slack markdown formatting.
func buildTextMessageBlock(text string) slackapi.Block {
	textBlock := slackapi.NewSectionBlock(
		slackapi.NewTextBlockObject(slackapi.MarkdownType, text, false, false),
		nil,
		nil,
	)
	return textBlock
}
