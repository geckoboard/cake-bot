package github

import (
	"errors"

	"github.com/geckoboard/cake-bot/log"
)

const (
	PullRequestEvent       = "pull_request"
	PullRequestReviewEvent = "pull_request_review"
)

type PullRequestReviewWebhook struct {
	// Action can be "submitted", "edited", or "dismissed".
	Action      string       `json:"action"`
	Review      *Review      `json:"review"`
	PullRequest *PullRequest `json:"pull_request"`
	Repository  *Repository  `json:"repository"`
}

func (w *PullRequestReviewWebhook) EnhanceLogger(l log.LeveledLogger) log.LeveledLogger {
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

func (w *PullRequestReviewWebhook) Validate() error {
	if w.Review == nil {
		return errors.New(`"review" field is missing from webhook payload`)
	}

	if w.PullRequest == nil {
		return errors.New(`"pull_request" field is missing from webhook payload`)
	}

	return nil
}
