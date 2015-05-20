package main

import (
	"regexp"
	"strings"
	"sync"

	"github.com/google/go-github/github"
	log15 "gopkg.in/inconshreveable/log15.v2"
)

const (
	WIPLabel          string = "wip"
	CakedLabel        string = "caked"
	AwaitingCakeLabel string = "awaiting-cake"
)

var (
	IssueUrlRegex  = regexp.MustCompile("repos/([^/]+)/([^/]+)/issues")
	TrelloUrlRegex = regexp.MustCompile("(https?://trello.com/(?:[^\\s()]+))")

	WIPRegex    = regexp.MustCompile("(?i)wip")
	LabelColors = map[string]string{
		// Blue
		WIPLabel: "207de5",
		// Green
		CakedLabel: "009800",
		// Orange
		AwaitingCakeLabel: "eb6420",
	}
	deprecatedLabels = []string{"Awaiting Cake"}
)

func loadComments(client *github.Client, pr *PullRequest) error {
	var allComments []github.IssueComment

	opts := github.IssueListCommentsOptions{}

	for {
		comments, resp, err := client.Issues.ListComments(pr.owner, pr.repo, *pr.issue.Number, &opts)

		if err != nil {
			return err
		}

		allComments = append(allComments, comments...)

		if resp.NextPage == 0 {
			break
		}

		opts.ListOptions.Page = resp.NextPage
	}

	pr.comments = &allComments

	return nil
}

func updateIssueReviewLabels(client *github.Client, log log15.Logger, pr *PullRequest) error {
	newLabels := []string{pr.CalculateAppropriateStatus()}

	for _, l := range pr.issue.Labels {
		switch *l.Name {
		case WIPLabel, CakedLabel, AwaitingCakeLabel:
			continue
		default:
			newLabels = append(newLabels, *l.Name)
		}
	}

	log.Info("updating issue review label", "new_label", newLabels[0])

	_, _, err := client.Issues.ReplaceLabelsForIssue(pr.owner, pr.repo, *pr.issue.Number, newLabels)

	if err != nil {
		log.Error("unable to update issue review label", "err", err)
	}

	return err
}

type PullRequest struct {
	issue    github.Issue
	comments *[]github.IssueComment
	owner    string
	repo     string
}

func (p *PullRequest) IsWIP() bool {
	return WIPRegex.MatchString(*p.issue.Title)
}

func (p *PullRequest) IsCaked() bool {
	for _, c := range *p.comments {
		if strings.Contains(*c.Body, ":cake:") {
			return true
		}
	}

	return false
}

func (p *PullRequest) CalculateAppropriateStatus() string {
	switch {
	case p.IsWIP():
		return WIPLabel
	case p.IsCaked():
		return CakedLabel
	default:
		return AwaitingCakeLabel
	}
}

func (p *PullRequest) ExtractTrelloCardUrls() []string {
	urls := TrelloUrlRegex.FindAllString(*p.issue.Body, -1)

	for _, c := range *p.comments {
		urls = append(urls, TrelloUrlRegex.FindAllString(*c.Body, -1)...)
	}

	return urls
}

func (p *PullRequest) Number() int {
	return *p.issue.Number
}

func (p *PullRequest) URL() string {
	return *p.issue.HTMLURL
}

func PullRequestFromIssue(i *github.Issue, c *github.Client) PullRequest {
	components := IssueUrlRegex.FindStringSubmatch(*i.URL)
	org := components[1]
	repo := components[2]

	pr := PullRequest{
		issue: *i,
		owner: org,
		repo:  repo,
	}

	loadComments(c, &pr)

	return pr
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

			allIssues = append(allIssues, PullRequestFromIssue(&i, connection))
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

	var wg sync.WaitGroup

	for {
		repos, resp, err := client.Repositories.ListByOrg(org, &opts)

		if err != nil {
			return err
		}

		for _, r := range repos {
			wg.Add(1)

			go func(r github.Repository) {
				defer wg.Done()
				log.Info("start syncing labels for repo", "repo.name", *r.Name)
				err := setupReviewFlagsInRepo(r, client)

				if err != nil {
					log.Error("error syncing repo review labels", "err", err, "repo", r.Name)
				}

				log.Info("done syncing labels for repo", "repo.name", *r.Name)
			}(r)
		}

		if resp.NextPage == 0 {
			break
		}

		opts.ListOptions.Page = resp.NextPage
	}

	wg.Wait()
	return nil

}

func setupReviewFlagsInRepo(repo github.Repository, client *github.Client) error {
	l := log.New("repo.name", repo.Name)
	opts := github.ListOptions{}
	currentLabels, _, err := client.Issues.ListLabels(*repo.Owner.Login, *repo.Name, &opts)

	if err != nil {
		l.Error("unable to fetch current labels", "err", err)
		return err
	}

	for _, label := range deprecatedLabels {
		for _, actualLabel := range currentLabels {
			if strings.ToLower(*actualLabel.Name) == strings.ToLower(label) {
				l.Info("deleting deprecated label", "repo.name", *repo.Name, "label", *actualLabel.Name)

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
			l.Info("creating label", "repo.Name", *repo.Name, "label.name", label, "label.color", color)

			_, _, err = client.Issues.CreateLabel(*repo.Owner.Login, *repo.Name, &github.Label{Name: &label, Color: &color})
		} else if needsUpdating {
			l.Info("updating label", "repo.Name", *repo.Name, "label.name", label, "label.color", color)

			_, _, err = client.Issues.EditLabel(*repo.Owner.Login, *repo.Name, label, &github.Label{Name: &label, Color: &color})
		}

		if err != nil {
			return err
		}
	}

	return nil
}
