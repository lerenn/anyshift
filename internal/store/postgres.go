package store

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Postgres implements Store using PostgreSQL. Only this package and main use *sql.DB / *pgxpool.Pool.
type Postgres struct {
	pool *pgxpool.Pool
}

// NewPostgres returns a Store backed by the given pool. Caller must call Close when done.
func NewPostgres(pool *pgxpool.Pool) *Postgres {
	return &Postgres{pool: pool}
}

// InsertPushEvent inserts a push event. Returns (true, nil) if inserted, (false, nil) if duplicate id.
func (p *Postgres) InsertPushEvent(ctx context.Context, event *PushEventRow) (bool, error) {
	cmd, err := p.pool.Exec(ctx, `
		INSERT INTO gh_push_events (id, type, created_at, actor_login, repo, raw_payload)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO NOTHING
	`, event.ID, event.Type, event.CreatedAt, event.ActorLogin, event.Repo, event.RawPayload)
	if err != nil {
		return false, err
	}
	return cmd.RowsAffected() > 0, nil
}

// InsertCommitStats inserts commit stats. Returns (true, nil) if inserted, (false, nil) if duplicate sha.
func (p *Postgres) InsertCommitStats(ctx context.Context, stats *CommitStatsRow) (bool, error) {
	cmd, err := p.pool.Exec(ctx, `
		INSERT INTO commit_stats (sha, repo, author, committed_at, additions, deletions, total, net)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (sha) DO NOTHING
	`, stats.Sha, stats.Repo, stats.Author, stats.CommittedAt, stats.Additions, stats.Deletions, stats.Total, stats.Net)
	if err != nil {
		return false, err
	}
	return cmd.RowsAffected() > 0, nil
}

// GlobalNetLines returns the sum of net from commit_stats.
func (p *Postgres) GlobalNetLines(ctx context.Context) (int64, error) {
	var v int64
	err := p.pool.QueryRow(ctx, `SELECT COALESCE(SUM(net), 0) FROM commit_stats`).Scan(&v)
	return v, err
}

// EventsSeenCount returns the count of rows in gh_push_events.
func (p *Postgres) EventsSeenCount(ctx context.Context) (int64, error) {
	var n int64
	err := p.pool.QueryRow(ctx, `SELECT COUNT(*) FROM gh_push_events`).Scan(&n)
	return n, err
}

// Ping checks the database connection.
func (p *Postgres) Ping(ctx context.Context) error {
	return p.pool.Ping(ctx)
}
