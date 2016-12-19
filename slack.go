package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"sync"

	"github.com/geckoboard/cake-bot/ctx"
	"github.com/geckoboard/cake-bot/slack"
	"github.com/google/go-github/github"
	"golang.org/x/net/context"
)

var slackHook = &http.Client{}

type Notifier struct {
	Webhook string
	Token   string
}

var userMap = map[string]string{}

func GuessSlackUsername(user *github.User) string {
	specialUser := userMap[strings.ToLower(*user.Login)]
	if specialUser != "" {
		return specialUser
	}

	info, _, err := gh.Users.Get(*user.Login)
	if err != nil {
		return ""
	}

	if info.Name != nil {
		name := strings.SplitN(*info.Name, " ", 2)
		if len(name) > 0 {
			return strings.ToLower(name[0])
		}
	}

	return ""
}

type CakeEvent struct {
	Channel   string `json:"channel"`
	Username  string `json:"username"`
	Text      string `json:"text"`
	IconEmoji string `json:"icon_emoji"`
	Parse     string `json:"parse"`
}

func NewNotifier(url, token string) *Notifier {
	return &Notifier{
		Webhook: url,
		Token:   token,
	}
}

func (n Notifier) Approved(pr github.PullRequest, review PullRequestReview) error {
	user := GuessSlackUsername(pr.User)

	e := CakeEvent{
		Channel:   "#devs",
		Username:  "cake-bot",
		Text:      "@" + user + " you have received a :cake: for " + review.URL(),
		IconEmoji: ":sheep:",
		Parse:     "full",
	}

	return n.sendMessage(context.TODO(), e)
}

func (n Notifier) ChangesRequested(pr github.PullRequest, review PullRequestReview) error {
	user := GuessSlackUsername(pr.User)

	e := CakeEvent{
		Channel:   "#devs",
		Username:  "cake-bot",
		Text:      "@" + user + " you have received some feedback on this PR " + review.URL(),
		IconEmoji: ":sheep:",
		Parse:     "full",
	}

	return n.sendMessage(context.TODO(), e)
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

func (n Notifier) BuildSlackUserMap() error {
	api := slack.New(n.Token)

	users, err := api.GetUsers()
	if err != nil {
		return err
	}

	team, err := api.GetTeamProfile()
	if err != nil {
		return err
	}
	GHID := findGithubFieldID(team)

	var wg sync.WaitGroup
	wg.Add(len(users))
	for _, u := range users {
		go func(u slack.User) {
			defer wg.Done()
			profile, err := api.GetUserProfile(u.ID)
			if err != nil {
				return
			}

			if name := findGithubUsername(GHID, profile); name != "" {
				userMap[name] = u.Name
			}
		}(u)
	}

	wg.Wait()

	return nil
}

func findGithubFieldID(team *slack.TeamProfile) string {
	for _, f := range team.Fields {
		if strings.Contains(strings.ToLower(f.Label), "github") {
			return f.ID
		}
	}

	return ""
}

func findGithubUsername(fieldId string, profile *slack.UserProfile) string {
	for id, field := range profile.Fields {
		if id == fieldId {
			return strings.TrimSpace(strings.ToLower(field.Value))
		}
	}

	return ""
}
