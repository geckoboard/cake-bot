package slack

import (
	"strings"
	"sync/atomic"

	"github.com/slack-go/slack"
)

var Users = &users{}

type users struct {
	users atomic.Pointer[map[string]slack.User]
}

func (c *users) Load(api *Client) error {
	team, err := api.GetTeamProfile()
	if err != nil {
		return err
	}

	githubFieldID := findCustomFieldID(team)

	users, err := api.GetUsers()
	if err != nil {
		return err
	}

	m := make(map[string]slack.User)

	for _, u := range users {
		profile, err := api.GetUserProfile(u.ID)
		if err != nil {
			return err
		}

		if name := findGitHubUsernameFromCustomFieldID(githubFieldID, profile); name != "" {
			m[strings.ToLower(name)] = u
		}
	}

	c.users.Store(&m)
	return nil
}

func (c *users) FindByGitHubUsername(name string) *slack.User {
	m := c.users.Load()
	if m == nil {
		// We haven't loaded all users yet, bail out.
		return nil
	}

	if u, ok := (*m)[strings.ToLower(name)]; ok {
		return &u
	}

	return nil
}

func findCustomFieldID(team *slack.TeamProfile) string {
	for _, f := range team.Fields {
		if strings.Contains(strings.ToLower(f.Label), "github") {
			return f.ID
		}
	}

	return ""
}

func findGitHubUsernameFromCustomFieldID(fieldId string, profile *slack.UserProfile) string {
	for id, field := range profile.Fields.ToMap() {
		if id == fieldId {
			return strings.TrimSpace(field.Value)
		}
	}

	return ""
}
