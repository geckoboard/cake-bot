package main

import (
	"log"
	"regexp"
	"strings"

	"github.com/google/go-github/github"
)

const (
	WIPLabel          string = "wip"
	CakedLabel        string = "caked"
	AwaitingCakeLabel string = "awaiting-cake"
)

var (
	IssueUrlRegex = regexp.MustCompile("repos/([^/]+)/([^/]+)/issues")
	WIPRegex      = regexp.MustCompile("(?i)wip")
	LabelColors   = map[string]string{
		// Blue
		WIPLabel: "207de5",
		// Green
		CakedLabel: "009800",
		// Orange
		AwaitingCakeLabel: "eb6420",
	}
	deprecatedLabels = []string{"Awaiting Cake"}
)

type PullRequest struct {
	issue    github.Issue
	client   *github.Client
	comments *[]github.IssueComment
	owner    string
	repo     string
}

func (p *PullRequest) IsWIP() bool {
	return WIPRegex.MatchString(*p.issue.Title)
}

func (p *PullRequest) IsCaked() bool {
	comments, err := p.FetchComments()

	if err != nil {
		return false
	}

	for _, c := range comments {
		if strings.Contains(*c.Body, ":cake:") {
			return true
		}
	}

	return false
}

func (p *PullRequest) FetchComments() ([]github.IssueComment, error) {
	var allComments []github.IssueComment

	opts := github.IssueListCommentsOptions{}

	for {
		comments, resp, err := p.client.Issues.ListComments(p.owner, p.repo, *p.issue.Number, &opts)

		if err != nil {
			return nil, err
		}

		allComments = append(allComments, comments...)

		if resp.NextPage == 0 {
			break
		}

		opts.ListOptions.Page = resp.NextPage
	}

	return allComments, nil
}

func (p *PullRequest) SetReviewLabel(l string) error {
	newLabels := []string{l}

	for _, l := range p.issue.Labels {
		switch *l.Name {
		case WIPLabel, CakedLabel, AwaitingCakeLabel:
			continue
		default:
			newLabels = append(newLabels, *l.Name)
		}
	}

	log.Printf("Setting labels to %#v\n", newLabels)

	_, _, err := p.client.Issues.ReplaceLabelsForIssue(p.owner, p.repo, *p.issue.Number, newLabels)

	return err
}

func PullRequestFromIssue(i github.Issue, c *github.Client) PullRequest {
	components := IssueUrlRegex.FindStringSubmatch(*i.URL)
	org := components[1]
	repo := components[2]

	return PullRequest{
		issue:  i,
		client: c,
		owner:  org,
		repo:   repo,
	}
}

func pullRequestIssues(connection *github.Client, org string) ([]PullRequest, error) {
	var allIssues []PullRequest

	opts := github.IssueListOptions{
		Filter:    "all",
		Sort:      "updated",
		Direction: "descending",
	}

	for {
		issues, resp, err := connection.Issues.ListByOrg(org, &opts)

		if err != nil {
			return nil, err
		}

		for _, i := range issues {
			if i.PullRequestLinks == nil {
				break
			}

			allIssues = append(allIssues, PullRequestFromIssue(i, connection))
		}

		if resp.NextPage == 0 {
			break
		}

		opts.ListOptions.Page = resp.NextPage
	}

	return allIssues, nil
}

func ensureOrgReposHaveLabels(org string, client *github.Client) error {
	opts := github.RepositoryListByOrgOptions{}

	for {
		repos, resp, err := client.Repositories.ListByOrg(org, &opts)

		if err != nil {
			return err
		}

		for _, r := range repos {
			err := setupReviewFlagsInRepo(r, client)

			if err != nil {
				return err
			}
		}

		if resp.NextPage == 0 {
			break
		}

		opts.ListOptions.Page = resp.NextPage
	}

	return nil

}

func setupReviewFlagsInRepo(repo github.Repository, client *github.Client) error {
	opts := github.ListOptions{}
	currentLabels, _, err := client.Issues.ListLabels(*repo.Owner.Login, *repo.Name, &opts)

	if err != nil {
		return err
	}

	for _, label := range deprecatedLabels {
		for _, actualLabel := range currentLabels {
			if strings.ToLower(*actualLabel.Name) == strings.ToLower(label) {
				_, err = client.Issues.DeleteLabel(*repo.Owner.Login, *repo.Name, *actualLabel.Name)

				if err != nil {
					return err
				}
			}
		}
	}

	for label, color := range LabelColors {
		found := false
		needsUpdating := false

		for _, actualLabel := range currentLabels {
			if *actualLabel.Name == label {
				found = true

				if *actualLabel.Color != color {
					needsUpdating = true
				}

				break
			}

			if strings.ToLower(*actualLabel.Name) == strings.ToLower(label) {
				found = true
				needsUpdating = true
				break
			}
		}

		if !found {
			_, _, err = client.Issues.CreateLabel(*repo.Owner.Login, *repo.Name, &github.Label{Name: &label, Color: &color})
		} else if needsUpdating {
			_, _, err = client.Issues.EditLabel(*repo.Owner.Login, *repo.Name, label, &github.Label{Name: &label, Color: &color})
		}

		if err != nil {
			return err
		}
	}

	return nil
}
