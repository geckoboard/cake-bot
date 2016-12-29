package main

import (
	"strings"
	"sync"

	"github.com/geckoboard/cake-bot/slack"
	"github.com/google/go-github/github"
)

func NewSlackUserDirectory(gh *github.Client, sl *slack.Client) SlackUserDirectory {
	return SlackUserDirectory{
		gh,
		sl,
		map[string]string{},
	}
}

type SlackUserDirectory struct {
	githubClient *github.Client
	slackClient  *slack.Client

	directory map[string]string
}

func (s *SlackUserDirectory) FindUserByGithubUser(user *github.User) string {
	specialUser := s.directory[strings.ToLower(*user.Login)]
	if specialUser != "" {
		return specialUser
	}

	// Not all API responses/webhook payloads embed the user's name,
	// so we need to look the user up separately
	info, _, err := s.githubClient.Users.Get(*user.Login)
	if err != nil {
		return ""
	}

	if info.Name != nil {
		name := strings.SplitN(*info.Name, " ", 2)
		if len(name) > 0 {
			return strings.ToLower(name[0])
		}
	}

	return *user.Login
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
	GHID := findGithubFieldID(team)

	var wg sync.WaitGroup
	var l sync.RWMutex

	d := map[string]string{}

	wg.Add(len(users))
	for _, u := range users {
		go func(u slack.User) {
			defer wg.Done()
			profile, err := s.slackClient.GetUserProfile(u.ID)
			if err != nil {
				return
			}

			if name := findGithubUsername(GHID, profile); name != "" {
				l.Lock()
				d[name] = u.Name
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
