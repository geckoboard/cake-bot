package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/geckoboard/cake-bot/ctx"
	"github.com/geckoboard/goutils/router"
	"github.com/google/go-github/github"
	"github.com/julienschmidt/httprouter"
	"golang.org/x/net/context"
	"gopkg.in/inconshreveable/log15.v2"
)

func NewServer() http.Handler {
	r := router.New()
	r.GET("/ping", ping)
	r.POST("/github", githubWebhook)
	return r
}

func ping(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	fmt.Println(w, "ok")
}

type webhookPayload struct {
	Action      string
	Issue       *github.Issue
	Repository  *github.Repository
	PullRequest *github.PullRequest `json:"pull_request"`
}

func (w *webhookPayload) enhanceLogger(l log15.Logger) log15.Logger {
	l = l.New("endpoint", "webhook", "action", w.Action)

	if w.Repository != nil {
		l = l.New("repo.path", fmt.Sprintf("%s/%s", *w.Repository.Name, *w.Repository.Owner.Login))
	}

	if w.Issue != nil {
		l = l.New(
			"issue.number", *w.Issue.Number,
			"issue.url", *w.Issue.HTMLURL,
		)
	}

	return l
}

func (w *webhookPayload) referencesPullRequest() bool {
	return w.isPullRequestEvent() || w.isPROpenedEvent()
}

func (w *webhookPayload) isPullRequestEvent() bool {
	return w.Issue != nil && *w.Issue.Number != 0 && w.Issue.PullRequestLinks != nil
}

func (w *webhookPayload) isPROpenedEvent() bool {
	return w.PullRequest != nil && w.Action != ""

}

func (w *webhookPayload) ensurePullRequestLoaded(c context.Context) error {
	if w.referencesPullRequest() {
		ctx.Logger(c).Info("found issue with pr links")
		return nil
	}

	if w.isPROpenedEvent() {
		ctx.Logger(c).Info("found pr opened event, inferring issue")

		issue, _, err := gh.Issues.Get(*w.Repository.Owner.Login, *w.Repository.Name, *w.PullRequest.Number)

		w.Issue = issue

		return err
	}

	return errors.New("webhook does not reference a pull request")
}

func githubWebhook(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	c := ctx.WithLogger(context.Background(), logger.New("endpoint", "webhook"))

	event := r.Header.Get("X-GitHub-Event")

	switch event {
	case "pull_request", "issue_comment":
		// handle request
	default:
		ctx.Logger(c).Info("not handling webhook", "github_event", event)
		w.WriteHeader(http.StatusOK)
		return
	}

	var payload webhookPayload
	var err error

	var triggerInspection bool

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		ctx.Logger(c).Error("could not unmarshal json", "err", err)
		w.WriteHeader(501)
		return
	}

	c = ctx.WithLogger(c, payload.enhanceLogger(ctx.Logger(c)))

	if !payload.referencesPullRequest() {
		ctx.Logger(c).Info("payload does not refer to pull request", "github_event", event)
		w.WriteHeader(http.StatusOK)
		return
	}

	err = payload.ensurePullRequestLoaded(c)

	if err != nil {
		ctx.Logger(c).Error("encountered error while loading issue", "err", err)
		w.WriteHeader(501)
		return
	}

	pr := ReviewRequestFromIssue(c, *payload.Repository, *payload.Issue, gh)

	err = updateIssueReviewLabels(c, gh, pr)

	if err != nil {
		w.WriteHeader(501)
		return
	}

	w.WriteHeader(http.StatusOK)
}
