# MAX Bot Working Runbook

## Статус рабочего контура

1. Бот подтвержден в MAX API (`GET /me`).
2. Входящие события приходят через `GET /updates`.
3. Исходящие сообщения отправляются через `POST /messages?chat_id=<id>`.

## Проверенный формат payload

1. Входящий текст:
   - `message.body.text`
2. Идентификатор чата:
   - `message.recipient.chat_id`
3. Тип полезного события:
   - `update_type = message_created`

## Webhook vs Polling

1. Основной production режим: `Webhook`.
2. Аварийный режим: `Polling`.
3. Если webhook маршрут сломан (например, путь отдает frontend):
   - временно удалить подписку webhook
   - переключить сервис на polling
   - восстановить webhook route и вернуть webhook режим.

## Удаление конфликтующей webhook подписки

```bash
curl -X DELETE \
  -H "Authorization: $MAX_BOT_TOKEN" \
  "https://platform-api.max.ru/subscriptions?url=https%3A%2F%2F<domain>%2Fintegrations%2Fmax%2Fwebhook"
```

## Минимальная диагностика

1. Проверка бота:
```bash
curl -H "Authorization: $MAX_BOT_TOKEN" https://platform-api.max.ru/me
```
2. Проверка подписок:
```bash
curl -H "Authorization: $MAX_BOT_TOKEN" https://platform-api.max.ru/subscriptions
```
3. Проверка входящих:
```bash
curl -H "Authorization: $MAX_BOT_TOKEN" \
  "https://platform-api.max.ru/updates?types=message_created,bot_started&timeout=25&limit=100"
```

## Ошибки, которые уже встречались

1. `400 Unknown recipient` при отправке:
   - причина: неверный формат отправки.
   - решение: использовать query `chat_id`.
2. `updates: []` при активном диалоге:
   - причина: конфликт webhook/polling.
   - решение: убрать webhook подписку и проверить polling.
3. Webhook URL отдает HTML/405:
   - причина: route идет в mini app, а не в bot backend.
   - решение: отдельное reverse proxy правило для `/integrations/max/webhook`.

## Безопасность

1. При утечке токена в переписке/логах:
   - немедленно перевыпустить токен в MAX.
   - обновить `.env` и перезапустить сервис.
