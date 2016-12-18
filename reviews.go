package main

import "strings"

type ReviewState string

var (
	APPROVED          ReviewState = "APPROVED"
	CHANGES_REQUESTED ReviewState = "CHANGE_REQUESTED"
)

type reviewAuthor struct {
	ID    int
	Login string
}

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
	ID     int          `json:"id"`
	Author reviewAuthor `json:"user"`
	State  ReviewState  `json:"state"`
	Links  links        `json:"_links"`
}

func (p PullRequestReview) IsApproved() bool {
	// In the GH PR reviews beta webhooks use lowercase constants,
	// but the API uses uppercase constants
	return ReviewState(strings.ToUpper(string(p.State))) == APPROVED
}
