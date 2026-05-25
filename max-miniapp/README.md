# max-miniapp

Production mini app + admin console для канала MAX "Тверь 900".

## Что реализовано

- UI/UX на основе нового прототипа: `Home`, `History`, `Profile`, overlay `Form`
- Быстрый сценарий создания обращения (< 1 минуты)
- Контракт обращения v2:
  - `category`
  - `description <= 1000`
  - `media[0..3]`
  - `location{lat,lng,address_fallback}`
- Канонические статусы: `moderation`, `in_work`, `resolved`, `rejected`
- Профиль: согласие 152-ФЗ + настройки push
- Админ-консоль `/admin`:
  - аналитический dashboard
  - очередь модерации (`queue`)
  - назначение исполнителя
  - официальный ответ и смена статуса
  - управление ролями и инвайтами

## Быстрый старт

```bash
npm install
npm run dev -- --host 0.0.0.0 --port 5174
```

## Переменные окружения

- `VITE_API_BASE_URL` - URL `core-api` (в проде обычно `https://tver4ever.ru`)
- `VITE_ACTIVE_TVER_BEARER_TOKEN` - опциональный токен
- `VITE_CHANNEL_HANDLE` - ник канала MAX
- `VITE_CHANNEL_NAME` - отображаемое имя канала

## URL-параметры mini app

- `chat_id`
- `max_user_id`
- `max_login`
- `view` (`home|history|form|profile`)
- `access_token` (опционально)
