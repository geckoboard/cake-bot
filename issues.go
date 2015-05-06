package main

import (
	"log"
	"regexp"
	"strings"

	"github.com/google/go-github/github"
)

const (
	WIPLabel          string = "WIP"
	CakedLabel        string = "Caked"
	AwaitingCakeLabel string = "Awaiting Cake"
)

var (
	IssueUrlRegex = regexp.MustCompile("repos/([^/]+)/([^/]+)/issues")
	WIPRegex      = regexp.MustCompile("(?i)wip")
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
