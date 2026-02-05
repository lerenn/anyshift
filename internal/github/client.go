package github

//go:generate go run go.uber.org/mock/mockgen -destination client_mock.gen.go -package github . EventsFetcher,CommitStatsFetcher

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	eventsURL    = "https://api.github.com/events"
	commitAPIFmt = "https://api.github.com/repos/%s/%s/commits/%s"
)

var (
	ErrNotFound   = errors.New("not found")
	ErrRateLimited = errors.New("rate limited")
)

// EventsFetcher fetches GitHub events (used by producer).
type EventsFetcher interface {
	FetchEvents(ctx context.Context, etag string) (events []Event, newEtag string, err error)
}

// CommitStatsFetcher fetches commit stats for a given repo/ref (used by consumer).
type CommitStatsFetcher interface {
	GetCommitStats(ctx context.Context, owner, repo, ref string) (*CommitStats, error)
}

// Client implements EventsFetcher and CommitStatsFetcher using the GitHub API.
// BaseURL is optional; when set (e.g. in tests) it replaces the default API host.
type Client struct {
	httpClient *http.Client
	token      string
	BaseURL    string // for tests: e.g. httptest.Server.URL
	log        *slog.Logger
}

// NewClient returns a GitHub API client. token is optional (PAT for higher rate limits).
func NewClient(token string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		token:      token,
		log:        slog.Default(),
	}
}

func (c *Client) eventsURL() string {
	if c.BaseURL != "" {
		return strings.TrimSuffix(c.BaseURL, "/") + "/events"
	}
	return eventsURL
}

func (c *Client) commitURL(owner, repo, ref string) string {
	if c.BaseURL != "" {
		return fmt.Sprintf("%s/repos/%s/%s/commits/%s", strings.TrimSuffix(c.BaseURL, "/"), owner, repo, ref)
	}
	return fmt.Sprintf(commitAPIFmt, owner, repo, ref)
}

// FetchEvents fetches global events. If etag is non-empty, sends If-None-Match; on 304 returns nil, newEtag, nil.
func (c *Client) FetchEvents(ctx context.Context, etag string) ([]Event, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.eventsURL(), nil)
	if err != nil {
		return nil, "", err
	}
	c.setAuth(req)
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	newEtag := resp.Header.Get("ETag")
	newEtag = strings.Trim(newEtag, `"`)

	switch resp.StatusCode {
	case http.StatusNotModified:
		return nil, newEtag, nil
	case http.StatusForbidden:
		if reset := resp.Header.Get("X-RateLimit-Reset"); reset != "" {
			if ts, _ := strconv.ParseInt(reset, 10, 64); ts > 0 {
				until := time.Until(time.Unix(ts, 0))
				if until > 0 && until < 5*time.Minute {
					c.log.Info("rate limited, backing off", "until", time.Unix(ts, 0))
					time.Sleep(until)
					return c.FetchEvents(ctx, etag) // retry once after backoff
				}
			}
		}
		return nil, newEtag, ErrRateLimited
	case http.StatusOK:
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, newEtag, err
		}
		var events []Event
		if err := json.Unmarshal(body, &events); err != nil {
			return nil, newEtag, err
		}
		return events, newEtag, nil
	default:
		if resp.StatusCode >= 500 {
			for attempt := 0; attempt < 3; attempt++ {
				backoff := time.Duration(1<<uint(attempt)) * time.Second
				time.Sleep(backoff)
				out, et, err := c.FetchEvents(ctx, etag)
				if err == nil {
					return out, et, nil
				}
				if attempt == 2 {
					return nil, newEtag, err
				}
			}
		}
		return nil, newEtag, fmt.Errorf("events API: %s", resp.Status)
	}
}

// GetCommitStats fetches commit stats for the given repo/ref. Returns ErrNotFound on 404.
func (c *Client) GetCommitStats(ctx context.Context, owner, repo, ref string) (*CommitStats, error) {
	url := c.commitURL(owner, repo, ref)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	c.setAuth(req)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNotFound:
		return nil, ErrNotFound
	case http.StatusForbidden:
		if reset := resp.Header.Get("X-RateLimit-Reset"); reset != "" {
			if ts, _ := strconv.ParseInt(reset, 10, 64); ts > 0 {
				until := time.Until(time.Unix(ts, 0))
				if until > 0 && until < 5*time.Minute {
					time.Sleep(until)
					return c.GetCommitStats(ctx, owner, repo, ref)
				}
			}
		}
		return nil, ErrRateLimited
	case http.StatusOK:
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		var api CommitAPIResponse
		if err := json.Unmarshal(body, &api); err != nil {
			return nil, err
		}
		additions := int64(0)
		deletions := int64(0)
		total := int64(0)
		if api.Stats != nil {
			if api.Stats.Additions != nil {
				additions = int64(*api.Stats.Additions)
			}
			if api.Stats.Deletions != nil {
				deletions = int64(*api.Stats.Deletions)
			}
			if api.Stats.Total != nil {
				total = int64(*api.Stats.Total)
			}
		}
		net := additions - deletions
		return &CommitStats{
			SHA:         api.SHA,
			Additions:   additions,
			Deletions:   deletions,
			Total:       total,
			Net:         net,
			Author:      api.Commit.Author.Name,
			CommittedAt: api.Commit.Author.Date,
		}, nil
	default:
		if resp.StatusCode >= 500 {
			for attempt := 0; attempt < 3; attempt++ {
				backoff := time.Duration(1<<uint(attempt)) * time.Second
				time.Sleep(backoff)
				out, err := c.GetCommitStats(ctx, owner, repo, ref)
				if err == nil {
					return out, nil
				}
				if attempt == 2 {
					return nil, err
				}
			}
		}
		return nil, fmt.Errorf("commit API: %s", resp.Status)
	}
}

func (c *Client) setAuth(req *http.Request) {
	if c.token != "" {
		req.Header.Set("Authorization", "token "+c.token)
	}
}
