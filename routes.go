package main

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/bugsnag/bugsnag-go"
	"github.com/geckoboard/cake-bot/ctx"
	"github.com/geckoboard/cake-bot/log"
	"github.com/google/go-github/github"
	"github.com/julienschmidt/httprouter"
)

type NotifyPullRequestReviewStatus interface {
	Approved(context.Context, *github.Repository, *github.PullRequest, *PullRequestReview) error
	ChangesRequested(context.Context, *github.Repository, *github.PullRequest, *PullRequestReview) error
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
	event := r.Header.Get("X-GitHub-Event")

	l := logger.With(
		"endpoint", "webhook",
		"request_id", r.Header.Get("X-Request-ID"),
		"github_delivery_id", r.Header.Get("X-GitHub-Delivery"),
		"github_event", event,
	)

	switch event {
	case "pull_request_review":
		s.handlePullRequestReview(w, r, l)
	default:
		l.Info("at", "ignore_event")
		w.WriteHeader(http.StatusOK)
	}
}

func (s server) handlePullRequestReview(w http.ResponseWriter, r *http.Request, l log.LeveledLogger) {
	var payload webhookPayload

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		bugsnag.Notify(err)
		l.Error("at", "unmarshal_error", "err", err)
		w.WriteHeader(501)
		return
	}

	l = payload.enhanceLogger(l)

	if payload.Action != "submitted" {
		l.Info("at", "ignore_review_action")
		w.WriteHeader(http.StatusOK)
		return
	}

	if err := payload.checkPayload(); err != nil {
		l.Error("at", "payload_error", "err", err)
		w.WriteHeader(501)
		return
	}

	c := ctx.WithLogger(context.Background(), l)

	if payload.Review.IsApproved() {
		s.notifier.Approved(c, payload.Repository, payload.PullRequest, payload.Review)
	} else if payload.Review.User.ID != payload.PullRequest.User.ID {
		s.notifier.ChangesRequested(c, payload.Repository, payload.PullRequest, payload.Review)
	}

	l.Info("at", "pull_request_updated")
	w.WriteHeader(http.StatusOK)
}
