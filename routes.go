package main

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/bugsnag/bugsnag-go"
	"github.com/geckoboard/cake-bot/ctx"
	"github.com/google/go-github/github"
	"github.com/julienschmidt/httprouter"
)

type NotifyPullRequestReviewStatus interface {
	Approved(context.Context, github.PullRequest, PullRequestReview) error
	ChangesRequested(context.Context, github.PullRequest, PullRequestReview) error
}

func NewServer(notifier NotifyPullRequestReviewStatus) http.Handler {
	r := httprouter.New()
	s := server{notifier}
	r.GET("/", s.root)
	r.POST("/github", s.githubWebhook)
	return r
}

type server struct {
	notifier NotifyPullRequestReviewStatus
}

func (s server) root(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	w.Header().Add("Location", "https://github.com/geckoboard/cake-bot")
	w.WriteHeader(302)
}

func (s server) githubWebhook(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	c := ctx.WithLogger(
		context.Background(),
		logger.With(
			"endpoint", "webhook",
			"request_id", r.Header.Get("X-Request-ID"),
		),
	)

	event := r.Header.Get("X-GitHub-Event")

	switch event {
	case "pull_request_review":
		// handle request
	default:
		ctx.Logger(c).Info("at", "ignore_event", "github_event", event)
		w.WriteHeader(http.StatusOK)
		return
	}

	var payload webhookPayload

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		bugsnag.Notify(err)
		ctx.Logger(c).Error("at", "unmarshal_error", "err", err)
		w.WriteHeader(501)
		return
	}

	if err := payload.checkPayload(); err != nil {
		ctx.Logger(c).Error("at", "payload_error", "err", err)
		w.WriteHeader(501)
		return
	}

	c = ctx.WithLogger(c, payload.enhanceLogger(ctx.Logger(c)))

	if payload.Review.IsApproved() {
		s.notifier.Approved(c, *payload.PullRequest, *payload.Review)
	} else {
		s.notifier.ChangesRequested(c, *payload.PullRequest, *payload.Review)
	}

	ctx.Logger(c).Info("at", "pull_request_updated")
	w.WriteHeader(http.StatusOK)
}
