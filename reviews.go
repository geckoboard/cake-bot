package main

import (
	"strings"

	"github.com/google/go-github/github"
)

type links map[string]struct {
	HREF string
}

func (lm links) URLByRel(rel string) string {
	l, exists := lm[rel]

	if !exists {
		return ""
	}

	return l.HREF
}

type PullRequestReview struct {
	ID   int         `json:"id"`
	User github.User `json:"user"`

	// State can be either "approved" or "change_requested".
	State string `json:"state"`

	Links links `json:"_links"`
}

func (p PullRequestReview) IsApproved() bool {
	// In the GH PR reviews beta, webhooks use lowercase constants,
	// but the API uses uppercase constants.
	return strings.ToLower(p.State) == "approved"
}

func (p PullRequestReview) URL() string {
	return p.Links.URLByRel("html")
}
