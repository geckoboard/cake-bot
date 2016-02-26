package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/bugsnag/bugsnag-go"
	"github.com/geckoboard/cake-bot/ctx"
	"github.com/geckoboard/cake-bot/log"
	"github.com/google/go-github/github"
	"github.com/julienschmidt/httprouter"
	"golang.org/x/net/context"
)

func NewServer() http.Handler {
	r := httprouter.New()
	r.GET("/", root)
	r.GET("/ping", ping)
	r.POST("/github", githubWebhook)
	return r
}

func root(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	w.Header().Add("Location", "https://github.com/geckoboard/cake-bot")
	w.WriteHeader(302)
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

func (w *webhookPayload) enhanceLogger(l log.LeveledLogger) log.LeveledLogger {
	l = l.With("endpoint", "webhook", "action", w.Action)

	if w.Repository != nil {
		l = l.With("repo.name", *w.Repository.Name)
	}

	if w.Issue != nil {
		l = l.With(
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
		ctx.Logger(c).Info("at", "webhook.event_includes_pull_request_details")
		return nil
	}

	if w.isPROpenedEvent() {
		ctx.Logger(c).Info("at", "webhook.loading_pull_request_details")

		issue, _, err := gh.Issues.Get(*w.Repository.Owner.Login, *w.Repository.Name, *w.PullRequest.Number)
		w.Issue = issue

		return err
	}

	return errors.New("webhook does not reference a pull request")
}

func githubWebhook(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	c := ctx.WithLogger(
		context.Background(),
		logger.With(
			"endpoint", "webhook",
			"request_id", r.Header.Get("X-Request-ID"),
		),
	)

	event := r.Header.Get("X-GitHub-Event")

	switch event {
	case "pull_request", "issue_comment":
		// handle request
	default:
		ctx.Logger(c).Info("at", "webhook.ignore_event", "github_event", event)
		w.WriteHeader(http.StatusOK)
		return
	}

	var payload webhookPayload

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		bugsnag.Notify(err)
		ctx.Logger(c).Error("at", "webhook.unmarshal_error", "err", err)
		w.WriteHeader(501)
		return
	}

	c = ctx.WithLogger(c, payload.enhanceLogger(ctx.Logger(c)))

	if !payload.referencesPullRequest() {
		ctx.Logger(c).Info("at", "webhook.not_pull_request", "github_event", event)
		w.WriteHeader(http.StatusOK)
		return
	}

	if err := payload.ensurePullRequestLoaded(c); err != nil {
		ctx.Logger(c).Error("at", "webhook.could_not_load_pull_request", "err", err)
		bugsnag.Notify(err)
		w.WriteHeader(501)
		return
	}

	pr := ReviewRequestFromIssue(c, *payload.Repository, *payload.Issue, gh)

	if err := updateIssueReviewLabels(c, gh, pr); err != nil {
		ctx.Logger(c).Error("at", "webhook.could_not_update_pull_request", "err", err)
		bugsnag.Notify(err)
		w.WriteHeader(501)
		return
	}

	ctx.Logger(c).Info("at", "webhook.pull_request_updated")
	w.WriteHeader(http.StatusOK)
}
