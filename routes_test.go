package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/geckoboard/cake-bot/github"
	"github.com/geckoboard/cake-bot/log"
	"github.com/slack-go/slack"
)

type notification struct {
	action   string
	repo     *github.Repository
	pr       *github.PullRequest
	reviewer *github.User
}

type fakeNotifier struct {
	notifications []notification
}

func (f *fakeNotifier) Approved(_ context.Context, repo *github.Repository, pr *github.PullRequest, review *github.Review) error {
	f.notifications = append(f.notifications, notification{"approved", repo, pr, review.User})
	return nil
}

func (f *fakeNotifier) ChangesRequested(_ context.Context, repo *github.Repository, pr *github.PullRequest, review *github.Review) error {
	f.notifications = append(f.notifications, notification{"changes_requested", repo, pr, review.User})
	return nil
}

func (f *fakeNotifier) ReviewRequested(_ context.Context, repo *github.Repository, pr *github.PullRequest, reviewer *github.User) error {
	f.notifications = append(f.notifications, notification{"review_requested", repo, pr, reviewer})
	return nil
}

func (f *fakeNotifier) RespondToSlackAction(_ context.Context, _ *slack.InteractionCallback, _ string) error {
	return nil
}

type fakeWebhookValidator struct{}

func (f *fakeWebhookValidator) ValidateSignature(r *http.Request) error {
	return nil
}

func TestHandlePullRequestEvent(t *testing.T) {
	outcome := &fakeNotifier{}
	validator := &fakeWebhookValidator{}

	s := httptest.NewServer(NewServer(outcome, validator))
	defer s.Close()
	webhookURL := s.URL + "/github"

	file, err := os.Open("./example-webhooks/pull_request_review_requested.json")

	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest("POST", webhookURL, file)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("X-Github-Event", "pull_request")

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

	if outcome.notifications[0].action != "review_requested" {
		t.Fatalf("expected notification to be review_requested, got: %q", outcome.notifications[0].action)
	}

	if outcome.notifications[0].pr.Number != 14 {
		t.Fatalf("expected PR number %d, got: %v", 14, outcome.notifications[0].pr.Number)
	}

	if outcome.notifications[0].reviewer.ID != 20394 {
		t.Fatalf("unexpected review passed to notifier: %v", outcome.notifications[0].reviewer)
	}
}

func TestHandleReviewRequiresChanges(t *testing.T) {
	outcome := &fakeNotifier{}
	validator := &fakeWebhookValidator{}

	s := httptest.NewServer(NewServer(outcome, validator))
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

	if outcome.notifications[0].action != "changes_requested" {
		t.Fatalf("expected notification to be changes_requested, got: %q", outcome.notifications[0].action)
	}

	if outcome.notifications[0].pr.Number != 12 {
		t.Fatalf("expected PR number %d, got: %q", 12, outcome.notifications[0].pr.Number)
	}

	if outcome.notifications[0].reviewer.ID != 12532520 {
		t.Fatalf("unexpected review passed to notifier: %v", outcome.notifications[0].reviewer)
	}
}

func TestHandleReviewApproved(t *testing.T) {
	outcome := &fakeNotifier{}
	validator := &fakeWebhookValidator{}

	s := httptest.NewServer(NewServer(outcome, validator))
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

	if outcome.notifications[0].action != "approved" {
		t.Fatalf("expected notification to be approved, got: %q", outcome.notifications[0].action)
	}

	if outcome.notifications[0].pr.Number != 12 {
		t.Fatalf("expected PR number %d, got: %d", 12, outcome.notifications[0].pr.Number)
	}

	if outcome.notifications[0].reviewer.ID != 12532520 {
		t.Fatalf("unexpected review passed to notifier: %v", outcome.notifications[0].reviewer)
	}
}

func init() {
	logger = log.New()
}
