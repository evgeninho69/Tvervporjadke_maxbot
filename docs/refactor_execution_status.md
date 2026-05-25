# Refactor Execution Status

## Implemented in current update

1. Mini app claim flow upgrade:
- added quick templates for common incidents (road pit, lighting, waste)
- added interactive mini-map with pin placement in claim form
- added pre-submit preview step before claim creation

2. Citizen claim tracking improvements:
- added visual claim lifecycle tracker in claim card
- added SLA helper text (time to deadline / overdue / closed in N hours)
- added direct map link for claim coordinates

3. Admin dashboard and queue polish:
- added trend line chart for timeline (`created` vs `resolved`)
- added queue filter by status in moderation view
- added quick actions in queue table (`В работу`, `Решено`)

## Implemented in iteration 1

1. Introduced new `core-api` service:
- citizen API: `/v2/citizen/*`
- admin API: `/admin/v2/*`
- bot internal API: `/internal/v2/*`
- canonical statuses: `moderation`, `in_work`, `resolved`, `rejected`
- domain events table and delivery flow for `claim_status_changed`

2. Switched bot to gateway/notifier role:
- removed embedded mini/admin API from `max-bot-service`
- updated bot commands to v2 UX (`/start`, `/status [id]`, `/support`, deep links)
- added notification worker that reads pending domain events from `core-api` and sends MAX messages

3. Reworked mini app + admin UI to new prototype style:
- citizen IA: `Home`, `History`, `Profile`, overlay `Form`
- claim form constraints: `description <= 1000`, media up to `3`, GPS + manual coordinates + address fallback
- admin IA: dashboard + moderation queue + assignment + official response + role/invite management

4. Updated infrastructure routing:
- `docker-compose.*` now includes `core-api`
- `infra/Caddyfile` routes `/v2/*`, `/admin/v2/*`, `/internal/v2/*` to `core-api`

## Implemented in follow-up sprint

1. Finished the Postgres transition path:
- added `core-api/scripts/migrate-sqlite-to-postgres.ts`
- kept runtime storage switchable via `CORE_API_STORAGE=sqlite|postgres`
- added local Postgres to `docker-compose.max.yaml`

2. Hardened the notification outbox:
- domain events now have `next_retry_at` and `last_error`
- notifier marks delivery failures through `/internal/v2/events/:id/fail`
- retry backoff is configurable and exhausted events move to `dead` DLQ status
- incoming MAX updates are deduped in persistent `incoming_updates`

3. Extended operations metrics:
- admin dashboard now receives SLA breach/due-today/p95 resolution metrics
- dashboard also exposes outbox delivery counts (`pending`, `delivered`, `failed`, `dead`)

4. Prepared deployment automation:
- production compose runs `core-api` on Postgres and injects `CORE_API_BASE_URL=http://core-api:8088` into the bot
- added webhook register/delete scripts and MAX API smoke script
- bot can auto-register `MAX_WEBHOOK_URL` on startup in `webhook|hybrid` mode
- added `infra/deploy-timeweb.sh` and `infra/smoke-deploy.sh`
- added `BUILD_ID` to service health responses for deployment verification

## Not yet implemented (next increments)

- legacy `at-*` read-only cutover/decommission steps
- physical Timeweb deploy from this workstation is blocked until SSH/host credentials or Docker CLI are available
- production DB password must be set via `POSTGRES_PASSWORD` before first real compose up
- MAX webhook smoke on public HTTPS after real deploy
- alerting dashboards outside the app UI
