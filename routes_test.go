package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/geckoboard/cake-bot/log"
	"github.com/google/go-github/github"
)

type notification struct {
	action ReviewState
	repo   github.Repository
	pr     github.PullRequest
	review PullRequestReview
}

type fakeNotifier struct {
	notifications []notification
}

func (f *fakeNotifier) Approved(_ context.Context, repo github.Repository, pr github.PullRequest, review PullRequestReview) error {
	f.notifications = append(f.notifications, notification{APPROVED, repo, pr, review})
	return nil
}

func (f *fakeNotifier) ChangesRequested(_ context.Context, repo github.Repository, pr github.PullRequest, review PullRequestReview) error {
	f.notifications = append(f.notifications, notification{CHANGES_REQUESTED, repo, pr, review})
	return nil
}

func TestHandleReviewRequiresChanges(t *testing.T) {
	outcome := &fakeNotifier{}

	s := httptest.NewServer(NewServer(outcome))
	defer s.Close()
	webhookURL := s.URL + "/github"

	file, err := os.Open("./example-webhooks/pull_request_review_submitted.json")

	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest("POST", webhookURL, file)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("X-Github-Event", "pull_request_review")

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("expected status code to be 200, got %d", resp.StatusCode)
	}

	if len(outcome.notifications) != 1 {
		t.Fatalf("expected 1 notification that PR has been approved, instead got: %#v", outcome.notifications)
	}

	if outcome.notifications[0].action != CHANGES_REQUESTED {
		t.Fatalf("expected notification to be CHANGES_REQUESTED, got: %q", outcome.notifications[0])
	}

	if *outcome.notifications[0].pr.Number != 12 {
		t.Fatalf("expected PR number %d, got: %q", 12, *outcome.notifications[0].pr.Number)
	}

	if outcome.notifications[0].review.ID != 13449121 {
		t.Fatalf("unexpected review passed to notifier: %q", outcome.notifications[0].review)
	}
}

func TestHandleReviewApproved(t *testing.T) {
	outcome := &fakeNotifier{}

	s := httptest.NewServer(NewServer(outcome))
	defer s.Close()
	webhookURL := s.URL + "/github"

	file, err := os.Open("./example-webhooks/pull_request_review_approved.json")

	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest("POST", webhookURL, file)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("X-Github-Event", "pull_request_review")

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("expected status code to be 200, got %d", resp.StatusCode)
	}

	if len(outcome.notifications) != 1 {
		t.Fatalf("expected 1 notification that PR has been approved, instead got: %#v", outcome.notifications)
	}

	if outcome.notifications[0].action != APPROVED {
		t.Fatalf("expected notification to be APPROVED, got: %q", outcome.notifications[0])
	}

	if *outcome.notifications[0].pr.Number != 12 {
		t.Fatalf("expected PR number %d, got: %q", 12, *outcome.notifications[0].pr.Number)
	}

	if outcome.notifications[0].review.ID != 13449164 {
		t.Fatalf("unexpected review passed to notifier: %q", outcome.notifications[0].review)
	}
}

func init() {
	logger = log.New()
}
