package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
)

// WebhookValidator interface validates the incoming webhook request.
type WebhookValidator interface {
	ValidateSignature(r *http.Request) error
}

// GitHubWebhookValidator implements the validation of GitHub webhook requests
// using a secret key.
type GitHubWebhookValidator struct {
	secret string
}

// NewGitHubWebhookValidator instantiates a new GitHubWebhookValidator given the
// secret key.
func NewGitHubWebhookValidator(secret string) *GitHubWebhookValidator {
	return &GitHubWebhookValidator{secret}
}

// GitHubWebhookValidator implements the WebhookValidator interface.
// The implementation uses HMAC encryption-decryption using the "secret" key.
//
// Refer: https://developer.github.com/webhooks/securing/#securing-your-webhooks
func (g *GitHubWebhookValidator) ValidateSignature(r *http.Request) error {
	signature := r.Header.Get("X-Hub-Signature")
	if signature == "" {
		return errors.New("No signature header provided")
	}

	// The value of the header is of the format: sha1=<actualhash>
	gotHash := strings.SplitN(signature, "=", 2)
	if gotHash[0] != "sha1" {
		return errors.New("Invalid signature header provided")
	}
	defer r.Body.Close()

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}

	hash := hmac.New(sha1.New, []byte(g.secret))
	if _, err := hash.Write(b); err != nil {
		return err
	}

	expectedHash := hex.EncodeToString(hash.Sum(nil))
	if expectedHash != gotHash[1] {
		return errors.New("Hashes do not match")
	}
	return nil
}
