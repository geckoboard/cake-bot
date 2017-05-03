package main

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/bugsnag/bugsnag-go"
	"github.com/geckoboard/cake-bot/ctx"
	"github.com/geckoboard/cake-bot/github"
	"github.com/geckoboard/cake-bot/log"
	"github.com/julienschmidt/httprouter"
)

type NotifyPullRequestReviewStatus interface {
	Approved(context.Context, *github.Repository, *github.PullRequest, *github.Review) error
	ChangesRequested(context.Context, *github.Repository, *github.PullRequest, *github.Review) error
}

func NewServer(notifier NotifyPullRequestReviewStatus) http.Handler {
	s := &Server{
		notifier: notifier,
	}

	r := httprouter.New()
	r.GET("/", s.root)
	r.POST("/github", s.githubWebhook)
	return r
}

type Server struct {
	notifier NotifyPullRequestReviewStatus
}

func (s *Server) root(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	w.Header().Add("Location", "https://github.com/geckoboard/cake-bot")
	w.WriteHeader(http.StatusFound)
}

func (s *Server) githubWebhook(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	event := r.Header.Get("X-GitHub-Event")

	l := logger.With(
		"endpoint", "webhook",
		"request_id", r.Header.Get("X-Request-ID"),
		"github_delivery_id", r.Header.Get("X-GitHub-Delivery"),
		"github_event", event,
	)

	switch event {
	case github.PullRequestReviewEvent:
		s.handlePullRequestReview(w, r, l)
	default:
		l.Info("at", "ignore_event")
		w.WriteHeader(http.StatusOK)
	}
}

func (s *Server) handlePullRequestReview(w http.ResponseWriter, r *http.Request, l log.LeveledLogger) {
	var webhook github.PullRequestReviewWebhook

	if err := json.NewDecoder(r.Body).Decode(&webhook); err != nil {
		bugsnag.Notify(err)
		l.Error("at", "unmarshal_error", "err", err)
		w.WriteHeader(http.StatusNotImplemented)
		return
	}

	l = webhook.EnhanceLogger(l)

	if webhook.Action != "submitted" {
		l.Info("at", "ignore_review_action")
		w.WriteHeader(http.StatusOK)
		return
	}

	if err := webhook.Validate(); err != nil {
		l.Error("at", "payload_error", "err", err)
		w.WriteHeader(http.StatusNotImplemented)
		return
	}

	c := ctx.WithLogger(context.Background(), l)

	if webhook.Review.IsApproved() {
		s.notifier.Approved(c, webhook.Repository, webhook.PullRequest, webhook.Review)
	} else if webhook.Review.User.ID != webhook.PullRequest.User.ID {
		s.notifier.ChangesRequested(c, webhook.Repository, webhook.PullRequest, webhook.Review)
	}

	l.Info("at", "pull_request_updated")
	w.WriteHeader(http.StatusOK)
}
