package main

import (
	"errors"

	"github.com/geckoboard/cake-bot/github"
	"github.com/geckoboard/cake-bot/log"
)

type pullRequestReviewWebhook struct {
	Action      string              `json:"action"`
	Repository  *github.Repository  `json:"repository"`
	PullRequest *github.PullRequest `json:"pull_request"`
	Review      *PullRequestReview  `json:"review"`
}

func (w *pullRequestReviewWebhook) EnhanceLogger(l log.LeveledLogger) log.LeveledLogger {
	l = l.With("action", w.Action)

	if w.Repository != nil {
		l = l.With("repo.name", w.Repository.Name)
	}

	if w.PullRequest != nil {
		l = l.With(
			"pr.number", w.PullRequest.Number,
			"pr.url", w.PullRequest.HTMLURL,
		)
	}

	return l
}

func (w *pullRequestReviewWebhook) ValidatePayload() error {
	if w.Review == nil {
		return errors.New(`"review" field is missing from webhook payload`)
	}

	if w.PullRequest == nil {
		return errors.New(`"pull_request" field is missing from webhook payload`)
	}

	return nil
}
