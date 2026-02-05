package pubsub

import (
	"context"
	"errors"
	"log/slog"

	"github.com/challenge-github-events/internal/github"
	"github.com/challenge-github-events/internal/store"
)

// CommitStatsFetcher fetches commit stats (e.g. github.Client).
type CommitStatsFetcher interface {
	GetCommitStats(ctx context.Context, owner, repo, ref string) (*github.CommitStats, error)
}

// Consumer processes commit jobs: fetch stats and persist. Depends only on Store interface.
type Consumer struct {
	store   store.Store
	fetcher CommitStatsFetcher
	jobs    <-chan CommitJob
	log     *slog.Logger
}

// NewConsumer returns a consumer that reads jobs from the given channel.
func NewConsumer(s store.Store, f CommitStatsFetcher, jobs <-chan CommitJob) *Consumer {
	return &Consumer{store: s, fetcher: f, jobs: jobs, log: slog.Default()}
}

// Run starts one worker. Call N times for N workers.
func (c *Consumer) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			c.log.Debug("consumer worker stopping")
			return
		case job, ok := <-c.jobs:
			if !ok {
				c.log.Debug("consumer jobs channel closed")
				return
			}
			c.process(ctx, job)
		}
	}
}

func (c *Consumer) process(ctx context.Context, job CommitJob) {
	stats, err := c.fetcher.GetCommitStats(ctx, job.Owner, job.Repo, job.SHA)
	if err != nil {
		if errors.Is(err, github.ErrNotFound) {
			c.log.Debug("commit not found, skipping", "repo", job.Repo, "sha", job.SHA)
			return
		}
		c.log.Warn("get commit stats", "repo", job.Repo, "sha", job.SHA, "err", err)
		return
	}
	row := &store.CommitStatsRow{
		Sha:         stats.SHA,
		Repo:        job.Owner + "/" + job.Repo,
		Author:      stats.Author,
		CommittedAt: stats.CommittedAt,
		Additions:   stats.Additions,
		Deletions:   stats.Deletions,
		Total:       stats.Total,
		Net:         stats.Net,
	}
	inserted, err := c.store.InsertCommitStats(ctx, row)
	if err != nil {
		c.log.Warn("insert commit stats", "sha", job.SHA, "err", err)
		return
	}
	if inserted {
		c.log.Debug("commit stats saved", "repo", row.Repo, "sha", job.SHA, "net", row.Net)
	}
}
