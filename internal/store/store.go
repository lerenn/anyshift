package store

//go:generate go run go.uber.org/mock/mockgen -destination store_mock.gen.go -package store . Store

import (
	"context"
	"encoding/json"
	"time"
)

// Store is the persistence interface. Producer, consumer, and server depend only on this interface.
// Only main and this package use *sql.DB.
type Store interface {
	InsertPushEvent(ctx context.Context, event *PushEventRow) (inserted bool, err error)
	InsertCommitStats(ctx context.Context, stats *CommitStatsRow) (inserted bool, err error)
	GlobalNetLines(ctx context.Context) (int64, error)
	EventsSeenCount(ctx context.Context) (int64, error)
	Ping(ctx context.Context) error
}

// PushEventRow is the row shape for gh_push_events.
type PushEventRow struct {
	ID         string
	Type       string
	CreatedAt  time.Time
	ActorLogin string
	Repo       string
	RawPayload json.RawMessage
}

// CommitStatsRow is the row shape for commit_stats.
type CommitStatsRow struct {
	Sha         string
	Repo        string
	Author      string
	CommittedAt time.Time
	Additions   int64
	Deletions   int64
	Total       int64
	Net         int64
}
