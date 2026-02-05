# SQL migrations

Migrations in this directory are run by PostgreSQL on **first startup** when using Docker Compose.

- **How it works**: `docker-compose.yml` mounts this directory as `/docker-entrypoint-initdb.d` in the Postgres container. The official Postgres image runs all `.sql` (and `.sh`) files in that directory in alphabetical order when the data volume is initialized.
- **Order**: Files are executed in name order; use numeric prefixes (e.g. `001_...`, `002_...`) to control migration order.
- **Re-running**: Scripts in `docker-entrypoint-initdb.d` run only when the database is created for the first time. To re-run migrations from scratch, remove the Postgres volume and start again: `docker compose down -v && docker compose up -d`.
