# 2025.11 Challenge

# Coding Challenge — Producer/Consumer with Go Channels + PostgreSQL (Docker)

## Requirements

- **Language**: Go (only)
- **Database**: PostgreSQL (Docker)
- **GitHub Personal Access Token (PAT)**: recommended but optional  
  → [Rate limits documentation](https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api?apiVersion=2022-11-28#primary-rate-limit-for-authenticated-users)

## Docs

- [GitHub API Root](https://api.github.com/)
- [GitHub Global Events](https://api.github.com/events)
- [GitHub Event Types](https://docs.github.com/en/rest/using-the-rest-api/github-event-types)
- [Compare Two Commits](https://docs.github.com/en/rest/commits/commits?apiVersion=2022-11-28#compare-two-commits)
- [Commits API](https://docs.github.com/en/rest/commits/commits)
- [Conditional Requests / ETag](https://docs.github.com/en/rest/using-the-rest-api/best-practices-for-using-the-rest-api#conditional-requests)

---

## Description

Build a small Go service that:

- Polls **GitHub Global Events** (`https://api.github.com/events`) and selects **PushEvent** items.
- Enumerates commit SHAs for each PushEvent.
- Processes each commit (consumer workers) to fetch **line stats** from the **Commit API** (`/repos/{owner}/{repo}/commits/{sha}`).
- Persists **raw events** and **per-commit stats** into **PostgreSQL**.
- Maintains a single **global net lines** metric = additions − deletions.
- Exposes an HTTP **/stats** endpoint showing:
  - current global net lines value,
  - total number of events processed since startup,
  - (optional) delta over a rolling window.

---

## Hard Requirements

### Concurrency
- Implement a **producer/consumer pattern** using **Go channels** (no external queue).
- Use a **bounded channel** (backpressure required).

### Polling
- Poll the **GitHub Global Events** endpoint.
- Extract `{owner, repo, before, head}` from each **PushEvent**.
- List commit SHAs from each PushEvent.
- For each commit SHA, fetch:
  - `stats.additions`, `stats.deletions`, commit timestamp, and author (if available).
- Avoid reprocessing already-seen events (ensure idempotency).

### Rate Limits & Errors
- Support optional **PAT** via `GH_TOKEN` (Authorization header).
- Respect `X-RateLimit-*` headers; back off on `403` until reset.
- Retry transient `5xx` responses with backoff.
- Skip and log `404` (private or missing commits).
- Ensure the poller does **not overwhelm** the consumer (pause or skip).

### Database (PostgreSQL)
- Provide Docker (or Compose) instructions to start PostgreSQL locally.
- Ensure **idempotency**:
  - `gh_push_events.id` (GitHub event ID) is unique.
  - `commit_stats.sha` is unique.
- **Minimum tables:**
  - `gh_push_events`: stores event id, type, created_at, actor_login, repo, and raw payload.
  - `commit_stats`: stores sha, repo, author, committed_at, additions, deletions, total, net.
- Maintain one global counter (net lines) derived from `commit_stats`.

### HTTP Endpoints
- `GET /health` → returns simple OK JSON.
- `GET /stats` → returns JSON including:
  - `global_net_lines_current`
  - `events_seen_since_start`
  - (optional) `global_net_lines_delta_window`

---

## Deliverables

- **README** explaining:
  - How to start PostgreSQL in Docker.
  - How to run the service.
  - Example request to `/stats`.
- A **runnable service** satisfying all requirements above.
