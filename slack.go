package main

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/google/go-github/github"

	"net/http"
	"strings"
)

var slackApi *http.Client

type Notifier struct {
	Webhook string
}

var userMap = map[string]string{
	"t-o-m-":    "tomhirst",
	"tomrandle": "tomr",
}

func GuessSlackUsername(user *github.User) string {
	specialUser := userMap[*user.Login]
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
}

func NewNotifier(url string) *Notifier {
	return &Notifier{
		Webhook: url,
	}
}

func (n *Notifier) PingUser(r ReviewRequest) {
	user := GuessSlackUsername(r.issue.User)
	if user == "" {
		log.Info("User not found for", r.issue.User.Name)
		return
	}

	log.Info("Ping user", user, "for a cake")

	e := CakeEvent{
		Channel:   "#dev",
		Username:  "cake-bot",
		Text:      "@" + user + " you have received a :cake: for " + r.URL(),
		IconEmoji: ":sheep:",
	}

	payload, err := json.Marshal(e)
	if err != nil {
		fmt.Println(err)
		return
	}

	req, err := http.NewRequest("POST", n.Webhook, bytes.NewBuffer(payload))
	if err != nil {
		fmt.Println(err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := slackApi.Do(req)

	defer resp.Body.Close()
}

func init() {
	slackApi = &http.Client{}
}
