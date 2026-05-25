---
name: max-chatbot-platform
description: Используй, когда нужно разработать, интегрировать, протестировать или поддерживать чат-бота MAX через Bot API (webhook/polling, subscriptions, updates, chats, messages, auth, rate limits, production hardening и диагностика).
---

# MAX Chatbot Platform

Используй этот skill для задач по MAX Bot API: подключение бота, обработка событий, отправка сообщений, эксплуатация в production.

## Когда применять

- Пользователь просит "сделать бота в MAX", "подключить webhook", "настроить Bot API".
- Нужны методы `platform-api.max.ru` (`/subscriptions`, `/updates`, `/chats`, `/messages` и т.д.).
- Нужны требования к production-надежности и безопасности бота.

## Базовые правила

1. Для production основной режим `Webhook`; fallback-режим `Polling` держи как аварийный.
2. Токен передавай только в `Authorization` header.
3. Закладывай лимит до `30 rps` суммарно на вызовы `platform-api.max.ru`.
4. Делай идемпотентную обработку входящих апдейтов (защита от дублей и ретраев).
5. Любые секреты храни в secret manager, не в коде и не в логах.

## Реальные API-детали (проверено в рабочем контуре)

1. Формат входящего `message_created`:
   - текст: `message.body.text`
   - chat id: `message.recipient.chat_id`
2. Отправка сообщения:
   - `POST /messages?chat_id=<id>`
   - body: `{ "text": "..." }`
3. Подписки:
   - список: `GET /subscriptions`
   - создание: `POST /subscriptions`
   - удаление: `DELETE /subscriptions?url=<encoded_url>`
4. Для polling полезно использовать:
   - `types=message_created,bot_started`
   - `timeout` и `limit`
   - `marker` из предыдущего ответа

## Рабочий процесс

1. Проверить входные данные:
   - bot token
   - публичный HTTPS URL для webhook
   - целевое окружение (dev/stage/prod)
2. Поднять endpoint `/integrations/max/webhook`.
3. Зарегистрировать webhook через `/subscriptions`.
4. Реализовать обработчик событий (router + handlers + retries + DLQ).
5. Реализовать исходящие уведомления через outbox (гарантированная доставка).
6. Добавить наблюдаемость: metrics, logs, alerts.
7. Прогнать smoke tests из `scripts/max_api_smoke.sh`.
8. Если webhook недоступен или неверно маршрутизируется:
   - временно отключить webhook подписку
   - переключить бота в polling
   - подтвердить входящие через `GET /updates`

## Диагностический runbook

1. `GET /me` должен вернуть текущего бота.
2. `GET /subscriptions`:
   - если webhook есть, но бот не отвечает, проверить reverse proxy route.
3. `GET /updates`:
   - если `updates: []` при активном чате, проверить конфликт webhook/polling.
4. Если есть входящие в `/updates`, но нет ответа:
   - проверить формат парсинга (`message.body.text`, `message.recipient.chat_id`).
5. Если ответ отправляется с 400:
   - проверить что используется `POST /messages?chat_id=...` вместо старого payload.

## Production checklist (минимум)

- Входящие апдейты: idempotency key + dedupe table.
- Исходящие сообщения: outbox + retry + backoff + DLQ.
- Логи: без токенов/PII, с correlation id.
- Метрики: latency p95/p99, delivery ratio, queue backlog.
- Алерты: spike ошибок, деградация доставки, рост очереди.
