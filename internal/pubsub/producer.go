package pubsub

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/challenge-github-events/internal/github"
	"github.com/challenge-github-events/internal/store"
)

// EventsFetcher fetches GitHub events (e.g. github.Client).
type EventsFetcher interface {
	FetchEvents(ctx context.Context, etag string) (events []github.Event, newEtag string, err error)
}

// Producer polls GitHub events and enqueues commit jobs. Depends only on Store interface.
type Producer struct {
	store        store.Store
	fetcher      EventsFetcher
	jobs         chan<- CommitJob
	pollInterval time.Duration
	log          *slog.Logger
}

// NewProducer returns a producer that sends jobs to the given channel.
// pollInterval is the delay between event fetches (e.g. from POLL_INTERVAL_SEC).
func NewProducer(s store.Store, f EventsFetcher, jobs chan<- CommitJob, pollInterval time.Duration) *Producer {
	return &Producer{store: s, fetcher: f, jobs: jobs, pollInterval: pollInterval, log: slog.Default()}
}

// Run polls until ctx is cancelled. Uses bounded channel for backpressure.
func (p *Producer) Run(ctx context.Context) {
	p.log.Info("producer running", "poll_interval", p.pollInterval)
	var etag string
	for {
		select {
		case <-ctx.Done():
			p.log.Info("producer stopping")
			return
		default:
		}
		events, newEtag, err := p.fetcher.FetchEvents(ctx, etag)
		if err != nil {
			p.log.Warn("fetch events", "err", err)
			continue
		}
		etag = newEtag
		if len(events) > 0 {
			p.log.Info("events fetched", "count", len(events), "etag", newEtag)
		}
		for _, e := range events {
			if e.Type != "PushEvent" {
				continue
			}
			inserted, err := p.store.InsertPushEvent(ctx, p.eventToRow(&e))
			if err != nil {
				p.log.Warn("insert push event", "id", e.ID, "err", err)
				continue
			}
			if !inserted {
				continue
			}
			payload := new(github.PushEventPayload)
			if err := json.Unmarshal(e.RawPayload, payload); err != nil {
				p.log.Warn("parse push payload", "id", e.ID, "err", err)
				continue
			}
			owner, repo := splitRepo(e.Repo)
			shas := make([]string, 0, len(payload.Commits)+1)
			for _, c := range payload.Commits {
				if c.SHA != "" {
					shas = append(shas, c.SHA)
				}
			}
			// Public /events API omits "commits"; use tip (head/after) so we still enqueue one job per push.
			if len(shas) == 0 {
				if tip := payload.Head; tip != "" {
					shas = append(shas, tip)
				} else if tip := payload.After; tip != "" {
					shas = append(shas, tip)
				}
			}
			p.log.Info("push event processed", "event_id", e.ID, "repo", owner+"/"+repo, "commits", len(shas))
			for _, sha := range shas {
				job := CommitJob{EventID: e.ID, Owner: owner, Repo: repo, SHA: sha}
				select {
				case p.jobs <- job:
				case <-ctx.Done():
					p.log.Info("producer stopping")
					return
				}
			}
		}
		select {
		case <-ctx.Done():
			p.log.Info("producer stopping")
			return
		case <-time.After(p.pollInterval):
		}
	}
}

func (p *Producer) eventToRow(e *github.Event) *store.PushEventRow {
	row := &store.PushEventRow{
		ID:         e.ID,
		Type:       e.Type,
		CreatedAt:  e.CreatedAt,
		Repo:       "",
		RawPayload: e.RawPayload,
	}
	if e.Repo != nil {
		row.Repo = e.Repo.FullName
	}
	if e.Actor != nil {
		row.ActorLogin = e.Actor.Login
	}
	return row
}

func splitRepo(r *github.Repo) (owner, repo string) {
	if r == nil {
		return "", ""
	}
	parts := strings.SplitN(r.FullName, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return r.FullName, ""
}
