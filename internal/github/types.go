package github

import (
	"encoding/json"
	"time"
)

// Event is a minimal shape for GitHub API events (we only use PushEvent).
type Event struct {
	ID         string          `json:"id"`
	Type       string          `json:"type"`
	CreatedAt  time.Time       `json:"created_at"`
	Actor      *Actor          `json:"actor"`
	Repo       *Repo           `json:"repo"`
	RawPayload json.RawMessage `json:"payload"`
}

// Actor holds actor login.
type Actor struct {
	Login string `json:"login"`
}

// Repo holds full name (owner/repo).
type Repo struct {
	FullName string `json:"full_name"`
}

// PushEventPayload is the payload for type PushEvent.
// Head is set by the public /events API when Commits is omitted.
type PushEventPayload struct {
	Before  string       `json:"before"`
	After   string       `json:"after"`
	Head    string       `json:"head"`
	Commits []PushCommit `json:"commits"`
}

// PushCommit has sha for each commit in a push.
type PushCommit struct {
	SHA string `json:"sha"`
}

// CommitStats is the result of GET /repos/{owner}/{repo}/commits/{ref} stats.
type CommitStats struct {
	SHA         string
	Additions   int64
	Deletions   int64
	Total       int64
	Net         int64
	Author      string
	CommittedAt time.Time
}

// CommitAPIResponse is the relevant part of the commit API JSON.
type CommitAPIResponse struct {
	SHA    string       `json:"sha"`
	Commit CommitDetail `json:"commit"`
	Stats  *struct {
		Additions *int `json:"additions"`
		Deletions *int `json:"deletions"`
		Total     *int `json:"total"`
	} `json:"stats"`
}

// CommitDetail has author and date.
type CommitDetail struct {
	Author struct {
		Name  string    `json:"name"`
		Email string    `json:"email"`
		Date  time.Time `json:"date"`
	} `json:"author"`
}
