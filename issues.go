package main

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/bugsnag/bugsnag-go"
	"github.com/geckoboard/cake-bot/ctx"
	"github.com/google/go-github/github"
	"golang.org/x/net/context"
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

func loadComments(_ context.Context, client *github.Client, review *ReviewRequest) error {
	var allComments []github.IssueComment

	opts := github.IssueListCommentsOptions{}

	for {
		comments, resp, err := client.Issues.ListComments(*review.repo.Owner.Login, *review.repo.Name, *review.issue.Number, &opts)

		if err != nil {
			return err
		}

		allComments = append(allComments, comments...)

		if resp.NextPage == 0 {
			break
		}

		opts.ListOptions.Page = resp.NextPage
	}

	review.comments = allComments

	return nil
}

func updateIssueReviewLabels(c context.Context, client *github.Client, review ReviewRequest) error {
	oldLabels := []string{}
	newLabels := []string{review.CalculateAppropriateStatus(c)}

	foundReviewLabel, incorrectReviewLabel := false, false

	for _, l := range review.issue.Labels {
		oldLabels = append(oldLabels, *l.Name)

		switch *l.Name {
		case WIPLabel, CakedLabel, AwaitingCakeLabel:
			foundReviewLabel = true

			if *l.Name != newLabels[0] {
				incorrectReviewLabel = true
			}

			continue
		default:
			newLabels = append(newLabels, *l.Name)
		}
	}

	var labelsNeedUpdating bool

	l := ctx.Logger(c).With("old_labels", fmt.Sprintf("%v", oldLabels), "new_labels", fmt.Sprintf("%v", newLabels))

	switch {
	case !foundReviewLabel:
		labelsNeedUpdating = true
		l.Info("at", "update_review.assigning_initial_label")
	case incorrectReviewLabel:
		labelsNeedUpdating = true
		l.Info("at", "update_review.correcting_label", "rr.path", review.RepositoryPath())
		for _, comment := range review.comments {
			ctx.Logger(c).Error("msg", "found unexpected comment", "comment.pointer", fmt.Sprintf("%#v", comment.ID), "comment.url", *comment.HTMLURL)
		}
	default:
		l.Debug("at", "update_review.current_label_accurate")
	}

	if labelsNeedUpdating {
		_, resp, err := client.Issues.ReplaceLabelsForIssue(*review.repo.Owner.Login, *review.repo.Name, review.Number(), newLabels)

		if resp != nil && resp.StatusCode == 404 {
			ctx.Logger(c).Error("at", "update_review.no_permission")
			return nil
		}

		if err != nil {
			bugsnag.Notify(err)
			ctx.Logger(c).Error("at", "update_review.failed", "err", err)
			return err
		}

		if review.IsCaked(c) {
			notifier.PingUser(c, review)
		}
	}

	return nil
}

type Issue struct {
	github.Issue

	Repository github.Repository `json:"repository,omitempty"`
}

type ReviewRequest struct {
	issue    github.Issue
	repo     github.Repository
	comments []github.IssueComment
}

func (p *ReviewRequest) IsWIP(_ context.Context) bool {
	return WIPRegex.MatchString(*p.issue.Title)
}

func (p *ReviewRequest) IsCaked(c context.Context) bool {
	for _, comment := range p.comments {
		if strings.Contains(*comment.Body, ":cake:") || strings.Contains(*comment.Body, "🍰") {
			return true
		}
	}

	return false
}

func (p *ReviewRequest) CalculateAppropriateStatus(c context.Context) string {
	switch {
	case p.IsWIP(c):
		return WIPLabel
	case p.IsCaked(c):
		return CakedLabel
	default:
		return AwaitingCakeLabel
	}
}

func (p *ReviewRequest) ExtractTrelloCardUrls() []string {
	urls := TrelloUrlRegex.FindAllString(*p.issue.Body, -1)

	for _, c := range p.comments {
		urls = append(urls, TrelloUrlRegex.FindAllString(*c.Body, -1)...)
	}

	return urls
}

func (p *ReviewRequest) RepositoryPath() string {
	return fmt.Sprintf("%s/%s", *p.repo.Owner.Login, *p.repo.Name)
}

func (p *ReviewRequest) Number() int {
	return *p.issue.Number
}

func (p *ReviewRequest) URL() string {
	return *p.issue.HTMLURL
}

func ReviewRequestFromIssue(c context.Context, r github.Repository, i github.Issue, cl *github.Client) ReviewRequest {
	review := ReviewRequest{
		issue: i,
		repo:  r,
	}

	loadComments(c, cl, &review)

	return review
}

func ghGet(gh *github.Client, url string, v interface{}) (*github.Response, error) {
	r, err := gh.NewRequest("GET", url, nil)

	if err != nil {
		return nil, err
	}

	return gh.Do(r, v)
}

func ghNextPageURL(r *github.Response) string {
	if r == nil {
		return ""
	}

	if links, ok := r.Response.Header["Link"]; ok && len(links) > 0 {
		for _, link := range strings.Split(links[0], ",") {
			segments := strings.Split(strings.TrimSpace(link), ";")

			// link must at least have href and rel
			if len(segments) < 2 {
				continue
			}

			// ensure href is properly formatted
			if !strings.HasPrefix(segments[0], "<") || !strings.HasSuffix(segments[0], ">") {
				continue
			}

			url := segments[0][1 : len(segments[0])-1]

			for _, segment := range segments[1:] {
				if strings.Contains(segment, `rel="next"`) {
					return url
				}
			}
		}
	}

	return ""
}

type ReviewRequestStream struct {
	connection *github.Client
	org        string
}

func NewReviewRequestStream(connection *github.Client, org string) *ReviewRequestStream {
	return &ReviewRequestStream{
		connection: connection,
		org:        org,
	}
}

func (s *ReviewRequestStream) Stream(c context.Context) chan ReviewRequest {
	out := make(chan ReviewRequest)

	go s.findReviews(c, out)

	return out
}

func (s *ReviewRequestStream) findReviews(c context.Context, out chan ReviewRequest) {
	var numIssues, numPRs int
	defer close(out)

	url := fmt.Sprintf("https://api.github.com/orgs/%s/issues?filter=all&sort=updated&direction=ascending", s.org)

	for {
		ctx.Logger(c).Info("at", "fetch_org_reviews.load", "url", url)

		var pageIssues []Issue
		resp, err := ghGet(s.connection, url, &pageIssues)

		if err != nil {
			ctx.Logger(c).Error("at", "fetch_org_reviews.unknown_error", "err", err)
			return
		}

		for _, i := range pageIssues {
			numIssues += 1

			if i.Issue.PullRequestLinks == nil {
				ctx.Logger(c).Debug("at", "fetch_org_reviews.ignore_issue", "issue.number", *i.Issue.Number, "url", *i.Issue.HTMLURL)
				continue
			}

			ctx.Logger(c).Debug("at", "fetch_org_reviews.found_review", "issue.number", *i.Issue.Number, "url", *i.Issue.HTMLURL)

			numPRs += 1
			out <- ReviewRequestFromIssue(c, i.Repository, i.Issue, s.connection)
		}

		url = ghNextPageURL(resp)

		if url == "" {
			ctx.Logger(c).Info("msg", "loaded all review requests", "issues.len", numIssues, "reviews.len", numPRs)
			return
		}
	}
}

func ensureOrgReposHaveLabels(c context.Context, org string, client *github.Client) error {
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

				repoCtx := ctx.WithLogger(c, ctx.Logger(c).With("repo.name", *r.Name))

				err := setupReviewFlagsInRepo(repoCtx, r, client)

				if err != nil {
					ctx.Logger(c).Error("msg", "error syncing repo review labels", "err", err, "repo", r.Name)
				}
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

func setupReviewFlagsInRepo(c context.Context, repo github.Repository, client *github.Client) error {
	c = ctx.WithLogger(c, ctx.Logger(c).With("repo.name", repo.Name))
	opts := github.ListOptions{}
	currentLabels, _, err := client.Issues.ListLabels(*repo.Owner.Login, *repo.Name, &opts)

	if err != nil {
		ctx.Logger(c).Error("at", "setup_repo_labels.unknown_error", "err", err)
		bugsnag.Notify(err)
		return err
	}

	for _, label := range deprecatedLabels {
		for _, actualLabel := range currentLabels {
			if strings.ToLower(*actualLabel.Name) == strings.ToLower(label) {
				ctx.Logger(c).Info("at", "setup_repo_labels.duplicate_label", "label", *actualLabel.Name)

				_, err = client.Issues.DeleteLabel(*repo.Owner.Login, *repo.Name, *actualLabel.Name)

				if err != nil {
					bugsnag.Notify(err)
					return err
				}
			}
		}
	}

	for label, color := range LabelColors {
		var response *github.Response
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
			ctx.Logger(c).Info("msg", "creating label", "label.name", label, "label.color", color)

			_, response, err = client.Issues.CreateLabel(*repo.Owner.Login, *repo.Name, &github.Label{Name: &label, Color: &color})
		} else if needsUpdating {
			ctx.Logger(c).Info("msg", "updating label", "label.name", label, "label.color", color)

			_, response, err = client.Issues.EditLabel(*repo.Owner.Login, *repo.Name, label, &github.Label{Name: &label, Color: &color})
		}

		// The bot does not have write access to this repo
		if response != nil && response.StatusCode == 404 {
			ctx.Logger(c).Info("at", "setup_repo_labels.not_allowed")
			return nil
		}

		if err != nil {
			return err
		}
	}

	return nil
}
