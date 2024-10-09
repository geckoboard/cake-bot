package slack

import (
	"log"
	"strings"
	"sync"

	"github.com/slack-go/slack"
)

var Users = &users{
	userMap: make(map[string][]*slack.User),
}

type users struct {
	userMap map[string][]*slack.User
	mu      sync.RWMutex
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

	var wg sync.WaitGroup
	wg.Add(len(users))

	for _, u := range users {
		go func(u *slack.User) {
			defer wg.Done()

			profile, err := api.GetUserProfile(u.ID)
			if err != nil {
				log.Println(err)
				return
			}

			if name := findGitHubUsernameFromCustomFieldID(githubFieldID, profile); name != "" {
				c.mu.Lock()
				if c.userMap[name] == nil {
					c.userMap[name] = []*slack.User{}
				}
				c.userMap[name] = append(c.userMap[name], u)
				c.mu.Unlock()
			}
		}(&u)
	}

	wg.Wait()

	return nil
}

func (c *users) FindByGitHubUsername(name string) []*slack.User {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if u, ok := c.userMap[strings.ToLower(name)]; ok {
		return u
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
			return strings.TrimSpace(strings.ToLower(field.Value))
		}
	}

	return ""
}
