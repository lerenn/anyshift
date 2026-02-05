-- gh_push_events: raw GitHub PushEvent records (idempotent on event id)
CREATE TABLE IF NOT EXISTS gh_push_events (
    id          TEXT PRIMARY KEY,
    type        TEXT NOT NULL,
    created_at  TIMESTAMPTZ,
    actor_login TEXT,
    repo        TEXT NOT NULL,
    raw_payload JSONB
);

CREATE INDEX IF NOT EXISTS idx_gh_push_events_created_at ON gh_push_events (created_at);
