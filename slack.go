package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/bugsnag/bugsnag-go"
	"github.com/geckoboard/cake-bot/ctx"
	"github.com/google/go-github/github"
	"golang.org/x/net/context"
)

var slackApi *http.Client

type Notifier struct {
	Webhook string
}

var userMap = map[string]string{
	"t-o-m-":     "tomhirst",
	"tomrandle":  "tomr",
	"kliriklara": "klara",
}

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

func NewNotifier(url string) *Notifier {
	return &Notifier{
		Webhook: url,
	}
}

func (n *Notifier) PingUser(c context.Context, r ReviewRequest) {
	l := ctx.Logger(c).With("at", "slack.ping-user")

	user := GuessSlackUsername(r.issue.User)
	if user == "" {
		l.Error("msg", "user not found")
		return
	}

	e := CakeEvent{
		Channel:   "#devs",
		Username:  "cake-bot",
		Text:      "@" + user + " you have received a :cake: for " + r.URL(),
		IconEmoji: ":sheep:",
		Parse:     "full",
	}

	l = l.With("slack.user", user, "slack.channel", e.Channel)

	payload, err := json.Marshal(e)
	if err != nil {
		l.Error("msg", "unable to encode cake event", "err", err)
		bugsnag.Notify(err)
		return
	}

	req, err := http.NewRequest("POST", n.Webhook, bytes.NewBuffer(payload))
	if err != nil {
		l.Error("msg", "unable to create request", "err", err)
		bugsnag.Notify(err)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := slackApi.Do(req)
	if err != nil {
		l.Error("msg", "unable to create request", "err", err)
		bugsnag.Notify(err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		l.Error("msg", "unexpected response status", "resp.Status", resp.Status)
		bugsnag.Notify(fmt.Errorf("unexpected response code %d, expected %d", resp.StatusCode, http.StatusOK))
		return
	}

	l.Info("msg", "ping successful")
	return
}

func init() {
	slackApi = &http.Client{}
}
