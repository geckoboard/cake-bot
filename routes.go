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

func NewServer(notifier Notifier, validator WebhookValidator) http.Handler {
	s := &Server{
		Notifier:         notifier,
		WebhookValidator: validator,
	}

	r := httprouter.New()
	r.GET("/", s.root)
	r.POST("/github", s.githubWebhook)
	return r
}

type Server struct {
	Notifier         Notifier
	WebhookValidator WebhookValidator
}

func (s *Server) validateSignature(r *http.Request) error {
	return s.WebhookValidator.ValidateSignature(r)
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

	if err := s.validateSignature(r); err != nil {
		l.Error("at", "invalid_signature", "err", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	switch event {
	case github.PullRequestEvent:
		s.handlePullRequestEvent(w, r, l)
	case github.PullRequestReviewEvent:
		s.handlePullRequestReviewEvent(w, r, l)
	default:
		l.Info("at", "ignore_event")
		w.WriteHeader(http.StatusOK)
	}
}

func (s *Server) handlePullRequestEvent(w http.ResponseWriter, r *http.Request, l log.LeveledLogger) {
	var webhook github.PullRequestWebhook

	if err := json.NewDecoder(r.Body).Decode(&webhook); err != nil {
		bugsnag.Notify(err)
		l.Error("at", "unmarshal_error", "err", err)
		w.WriteHeader(http.StatusNotImplemented)
		return
	}

	l = webhook.EnhanceLogger(l)

	switch webhook.Action {
	case "review_requested":
		c := ctx.WithLogger(context.Background(), l)
		s.Notifier.ReviewRequested(c, webhook.Repository, webhook.PullRequest, webhook.RequestedReviewer)
		w.WriteHeader(http.StatusOK)
	default:
		l.Info("at", "ignore_pull_request_action")
		w.WriteHeader(http.StatusOK)
	}
}

func (s *Server) handlePullRequestReviewEvent(w http.ResponseWriter, r *http.Request, l log.LeveledLogger) {
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
		s.Notifier.Approved(c, webhook.Repository, webhook.PullRequest, webhook.Review)
	} else if webhook.Review.User.ID != webhook.PullRequest.User.ID {
		s.Notifier.ChangesRequested(c, webhook.Repository, webhook.PullRequest, webhook.Review)
	}

	l.Info("at", "pull_request_updated")
	w.WriteHeader(http.StatusOK)
}
