# Timeweb Deploy Checklist (MAX bot + mini app)

> Operational deploy policy: `docs/max-prod-deploy-rules.md`

## 1) Что уже подготовлено в коде

- Полнофункциональный mini app: создание заявок, мои заявки, новости, карта, профиль, согласие ПДн.
- Бот передает в mini app параметры `chat_id`, `max_user_id`, `max_login`.
- Добавлен production compose: `docker-compose.timeweb.yaml`.
- Добавлен production Dockerfile: `max-miniapp/Dockerfile.prod`.
- Добавлен `VITE_BASE_PATH` для выката на под-путь занятого домена.

## 2) Что нужно предоставить для реального деплоя в Timeweb

1. SSH-доступ на сервер Timeweb:
   - public IPv4: `77.232.131.248`
   - IPv6: `2a03:6f01:1:2::b5ac`
   - private IPv4: `192.168.0.7`
   - SSH: `ssh root@77.232.131.248`
   - port: `22`
   - user: `root`
   - closed ports: `389, 2525, 25, 587, 53413, 3389, 465`
   - root-пароль хранить только локально в `.env.timeweb.local` как `TIMEWEB_ROOT_PASSWORD=...`
2. Доступ к reverse proxy (Caddy/Nginx/Traefik) на этом сервере.
3. Точный публичный URL mini app:
   - либо отдельный домен: `https://max-miniapp....`
   - либо под-путь: `https://<ваш-домен>/<tail>`
4. Точный URL webhook бота (если переключаемся на webhook).
5. Значения env для production:
   - `MAX_BOT_TOKEN`
   - `MAX_WEBHOOK_SECRET`
   - `MAX_MODE` (`polling` или `webhook`)
   - `MAX_MINIAPP_URL`
   - `ACTIVE_TVER_API_URL`
   - `VITE_API_BASE_URL`
   - `VITE_ACTIVE_TVER_BEARER_TOKEN` (если нужен)
   - `VITE_CHANNEL_HANDLE`
   - `VITE_CHANNEL_NAME`
   - `VITE_BASE_PATH` (например `/tver-miniapp/`)

## 3) Команды деплоя (на сервере)

С этой машины можно отправить проект и запустить деплой одной командой:

```powershell
Copy-Item .env.timeweb.local.example .env.timeweb.local
# Впишите root-пароль в .env.timeweb.local: TIMEWEB_ROOT_PASSWORD=...
.\infra\push-timeweb.ps1
```

Если запускать деплой уже внутри сервера:

```bash
cd /opt/TverActive
cp /opt/secure/max-bot.env ./max-bot-service/.env
cp /opt/secure/max-miniapp.env ./.env.timeweb

./infra/deploy-timeweb.sh
```

В текущем production compose `core-api` работает на Postgres-контейнере, а `max-bot-service` получает внутренний адрес `CORE_API_BASE_URL=http://core-api:8088`. Для проверки после выката можно использовать:

```bash
./infra/smoke-deploy.sh
```

Для переключения MAX на webhook:

```bash
MAX_WEBHOOK_URL=https://tver4ever.ru/integrations/max/webhook ./max-bot-service/scripts/register_webhook.sh
```

## 4) Проверки после деплоя

1. `GET /healthz` у bot-service.
2. Открытие mini app по целевому URL.
3. Команда `/start` в MAX -> ссылка с `chat_id` и `max_login`.
4. Создание тестовой заявки в mini app.
5. Получение ответа бота в чате MAX.

## 5) Если хвостик занят другим приложением

- Публикуем mini app на отдельном под-пути (`VITE_BASE_PATH` + route в reverse proxy).
- Webhook бота ведем на отдельный путь `/integrations/max/webhook`.
- Проверяем, что route webhook не конфликтует с фронтендом.
