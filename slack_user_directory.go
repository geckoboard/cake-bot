package main

import (
	"fmt"
	"strings"
	"sync"

	"github.com/geckoboard/cake-bot/github"
	"github.com/geckoboard/cake-bot/slack"
)

func NewSlackUserDirectory(slackClient *slack.Client) SlackUserDirectory {
	return SlackUserDirectory{
		slackClient: slackClient,
		directory:   make(map[string]slack.User),
	}
}

type SlackUserDirectory struct {
	slackClient *slack.Client
	directory   map[string]slack.User
}

func (s *SlackUserDirectory) BuildLinkToUser(ghUser *github.User) string {
	u, exists := s.directory[strings.ToLower(ghUser.Login)]
	if exists {
		return fmt.Sprintf("<@%s>", u.ID)
	}
	return ghUser.Login
}

func (s *SlackUserDirectory) ScanSlackTeam() error {
	users, err := s.slackClient.GetUsers()
	if err != nil {
		return err
	}

	team, err := s.slackClient.GetTeamProfile()
	if err != nil {
		return err
	}

	githubFieldID := findGithubFieldID(team)

	var wg sync.WaitGroup
	var l sync.RWMutex

	d := map[string]slack.User{}

	wg.Add(len(users))
	for _, u := range users {
		go func(u slack.User) {
			defer wg.Done()
			profile, err := s.slackClient.GetUserProfile(u.ID)
			if err != nil {
				return
			}

			if name := findGithubUsername(githubFieldID, profile); name != "" {
				l.Lock()
				d[name] = u
				l.Unlock()
			}
		}(u)
	}

	wg.Wait()

	s.directory = d

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
