package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/geckoboard/goutils/router"
	github "github.com/google/go-github/github"
	"github.com/julienschmidt/httprouter"
	"golang.org/x/oauth2"
)

const (
	WIPLabel          string = "WIP"
	CakedLabel        string = "Caked"
	AwaitingCakeLabel string = "Awaiting Cake"
)

var GithubApiKey string
var IssueUrlRegex = regexp.MustCompile("repos/([^/]+)/([^/]+)/issues")
var WIPRegex = regexp.MustCompile("(?i)wip")

type Config struct {
	Port int
}

func newWebhookServer() http.Handler {
	r := router.New()
	r.POST("/webhooks/github", ghHandler)
	return r
}

func ghHandler(w http.ResponseWriter, r *http.Request, params httprouter.Params) {

}

// tokenSource is an oauth2.TokenSource which returns a static access token
type tokenSource struct {
	token *oauth2.Token
}

// Token implements the oauth2.TokenSource interface
func (t *tokenSource) Token() (*oauth2.Token, error) {
	return t.token, nil
}

type PullRequest struct {
	i     github.Issue
	c     *github.Client
	Owner string
	Repo  string
}

func (p *PullRequest) IsWIP() bool {
	return WIPRegex.MatchString(*p.i.Title)
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
		comments, resp, err := p.c.Issues.ListComments(p.Owner, p.Repo, *p.i.Number, &opts)

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

	for _, l := range p.i.Labels {
		switch *l.Name {
		case WIPLabel, CakedLabel, AwaitingCakeLabel:
			continue
		default:
			newLabels = append(newLabels, *l.Name)
		}
	}

	log.Printf("Setting labels to %#v\n", newLabels)

	_, _, err := p.c.Issues.ReplaceLabelsForIssue(p.Owner, p.Repo, *p.i.Number, newLabels)

	return err
}

func PullRequestFromIssue(i github.Issue, c *github.Client) PullRequest {
	components := IssueUrlRegex.FindStringSubmatch(*i.URL)
	org := components[1]
	repo := components[2]

	return PullRequest{
		i:     i,
		c:     c,
		Owner: org,
		Repo:  repo,
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

func main() {
	var c Config

	flag.IntVar(&c.Port, "port", 0, "port to run http server on")
	flag.Parse()

	token := os.Getenv("GITHUB_ACCESS_TOKEN")

	if token == "" {
		log.Fatal("GITHUB_ACCESS_TOKEN not specified")
	}

	ts := &tokenSource{
		&oauth2.Token{AccessToken: token},
	}

	tc := oauth2.NewClient(oauth2.NoContext, ts)

	client := github.NewClient(tc)

	issues, err := pullRequestIssues(client, "geckoboard")

	if err != nil {
		log.Fatal(err)
	}

	for _, p := range issues {
		status := AwaitingCakeLabel
		isCaked := p.IsCaked()
		isWIP := p.IsWIP()

		log.Printf("%s, caked: %#v", *p.i.Title, isCaked)

		if isWIP {
			status = WIPLabel
		} else if isCaked {
			status = CakedLabel
		}

		err = p.SetReviewLabel(status)

		if err != nil {
			log.Fatal("error changing the review label for issue", err)
		}
	}
}
