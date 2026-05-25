# core-api

Выделенный API слой для TverActive (citizen/admin/internal bot endpoints).

## Что внутри

- `v2` API для mini app (citizen)
- `admin/v2` API для админки
- `internal/v2` API для bot gateway/notifier
- SQLite или Postgres storage с доменными событиями
- retry/backoff/DLQ для outbox уведомлений

## Команды

```bash
npm install
npm run dev
npm run migrate:sqlite-to-postgres
```

Миграция читает `data/tveractive.db` по умолчанию. Для другого файла укажите `SQLITE_DB_PATH`, для целевого Postgres нужен `DATABASE_URL`.

## Эндпоинты

- `GET /healthz`
- `GET /v2/citizen/bootstrap`
- `POST /v2/citizen/claims`
- `GET /v2/citizen/claims`
- `GET /v2/citizen/claims/:id`
- `POST /admin/v2/auth/login`
- `GET /admin/v2/dashboard`
- `GET /admin/v2/queue`
- `POST /admin/v2/claims/:id/status`
- `GET /internal/v2/events/pending`
- `POST /internal/v2/events/:id/ack`
- `POST /internal/v2/events/:id/fail`
- `POST /internal/v2/bot/incoming-updates/claim`
- `POST /internal/v2/bot/incoming-updates/complete`

