# max-bot-service

MAX бот для проекта ActiveTver.

## Реализованный функционал

- `/start`, `/help`, `/miniapp`
- `/auth` - согласие на ПДн и подтверждение MAX-авторизации
- `/create` - открыть mini app для создания заявки
- `/myclaims` - открыть mini app со списком заявок
- `/status [id]` - проверить статус последней/конкретной заявки через `core-api`
- `/support` - контакты поддержки
- `/news` - перейти к новостям канала
- `/adminbind +7...` - привязать чат администратора для личных уведомлений
- `/logout`, `/cancel`
- Бот работает как gateway/notifier для `core-api`
- Фоновый воркер отправляет push-уведомления по domain events (`claim_status_changed`)
- Ошибки доставки возвращаются в `core-api` для retry/backoff и DLQ
- Режимы `webhook` и `polling`
- Передача в mini app параметров: `chat_id`, `max_user_id`

## Админка

- Админ-контур полностью вынесен в `core-api` + `/admin` UI в `max-miniapp`
- В боте оставлена только привязка админ-чата (`/adminbind`) и доставка уведомлений

## Быстрый старт

```bash
npm install
npm run dev
```

## Переменные окружения

- `MAX_MODE=webhook|polling|hybrid`
- `MAX_POLL_INTERVAL_MS=1500`
- `CORE_API_EVENTS_POLL_MS=4000`
- `MAX_API_BASE_URL=https://platform-api.max.ru`
- `CORE_API_BASE_URL=http://core-api:8088`
- `CORE_API_INTERNAL_TOKEN=...`
- `MAX_BOT_TOKEN=...`
- `MAX_WEBHOOK_SECRET=...`
- `MAX_WEBHOOK_URL=...` (optional; auto-registers webhook in `webhook|hybrid` mode)
- `MAX_MINIAPP_URL=...`
- `DEFAULT_CHANNEL_HANDLE=@id695005444203_biz`
- `DEFAULT_CHANNEL_NAME=Тверь 900`
- `SUPER_ADMIN_PHONES=+79105313393,+79108325238`

## Endpoints

- `GET /healthz`
- `POST /integrations/max/webhook`

## Smoke / webhook scripts

- `scripts/max_api_smoke.sh` - checks `/me`, `/subscriptions`, and dry `/updates`.
- `scripts/register_webhook.sh` - registers `MAX_WEBHOOK_URL`.
- `scripts/delete_webhook.sh` - removes `MAX_WEBHOOK_URL` subscription.
