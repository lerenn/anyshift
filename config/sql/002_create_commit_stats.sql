-- commit_stats: per-commit line stats (idempotent on sha)
CREATE TABLE IF NOT EXISTS commit_stats (
    sha         TEXT PRIMARY KEY,
    repo        TEXT NOT NULL,
    author      TEXT,
    committed_at TIMESTAMPTZ,
    additions   BIGINT NOT NULL DEFAULT 0,
    deletions   BIGINT NOT NULL DEFAULT 0,
    total      BIGINT NOT NULL DEFAULT 0,
    net        BIGINT NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_commit_stats_repo ON commit_stats (repo);
CREATE INDEX IF NOT EXISTS idx_commit_stats_committed_at ON commit_stats (committed_at);
