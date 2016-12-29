package main

import (
	"strings"

	"github.com/google/go-github/github"
)

type ReviewState string

var (
	APPROVED          ReviewState = "APPROVED"
	CHANGES_REQUESTED ReviewState = "CHANGE_REQUESTED"
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
	ID    int         `json:"id"`
	User  github.User `json:"user"`
	State ReviewState `json:"state"`
	Links links       `json:"_links"`
}

func (p PullRequestReview) IsApproved() bool {
	// In the GH PR reviews beta webhooks use lowercase constants,
	// but the API uses uppercase constants
	return ReviewState(strings.ToUpper(string(p.State))) == APPROVED
}

func (p PullRequestReview) URL() string {
	return p.Links.URLByRel("html")
}
