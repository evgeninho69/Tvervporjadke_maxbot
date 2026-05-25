# MAX Prod Deploy Rules (Timeweb)

## 1. Цель

Этот регламент нужен для стабильного тестового prod-тура mini app + bot в MAX.
Применять для сервера `tver4ever.ru` и compose-файла `docker-compose.timeweb.yaml`.

## 2. Обязательные условия перед деплоем

1. Доступ к серверу по SSH (`root`).
2. Заполнены:
   - `.env.timeweb`
   - `core-api/.env`
   - `max-bot-service/.env`
3. Проверены критичные переменные:
   - `MAX_BOT_TOKEN`
   - `MAX_WEBHOOK_SECRET`
   - `MAX_WEBHOOK_URL`
   - `MAX_MINIAPP_URL`
   - `CORE_API_INTERNAL_TOKEN`
   - `DATABASE_URL`
4. Для Postgres на локальном контейнере:
   - `POSTGRES_SSL=false` (после фикса парсинга допустимо также пустое значение).

## 3. Стандартный сценарий деплоя

1. Локально из проекта:
   - `.\infra\push-timeweb.ps1`
2. Если деплой запускается на сервере вручную:
   - `cd /opt/TverActive`
   - `./infra/deploy-timeweb.sh`
3. После деплоя обязательно:
   - `docker compose -f docker-compose.timeweb.yaml ps`
   - `./infra/smoke-deploy.sh` (если доступен `CORE_API_INTERNAL_TOKEN`)

## 4. Критерии успешного деплоя

1. Сервисы `core-api`, `max-bot-service`, `max-miniapp`, `caddy`, `postgres` в состоянии `Up`.
2. `https://tver4ever.ru/max-miniapp/` отвечает `200`.
3. `https://tver4ever.ru/healthz` отвечает `200`.
4. `https://tver4ever.ru/v2/citizen/categories` отвечает `200`.
5. В `max-bot-service` нет ошибок регистрации webhook.

## 5. Проверка интеграции с MAX

1. Принудительно зарегистрировать webhook:
   - `MAX_WEBHOOK_URL=https://tver4ever.ru/integrations/max/webhook ./max-bot-service/scripts/register_webhook.sh`
   - В подписке должны быть `update_types`: `message_created`, `message_callback`, `bot_started`
2. В MAX отправить `/start` боту.
3. Убедиться, что ссылка на mini app открывается с параметрами `chat_id`, `max_user_id`, `max_login`.
4. Создать тестовое обращение из mini app и проверить доставку уведомления в чат.
5. Нажать кнопки меню бота (`Новости`, `Создать заявку`, `Мои заявки`) и убедиться, что бот отвечает/открывает mini app.

## 6. Антизависание сборки

1. В деплой-скрипте уже включен:
   - `BUILDX_NO_DEFAULT_ATTESTATIONS=1`
2. Если `docker compose build` завис:
   - прервать задачу,
   - выполнить `docker compose -f docker-compose.timeweb.yaml up -d --no-build`,
   - отдельно пересобрать только нужный сервис.

## 7. Rollback

1. Не выполнять destructive-команды (`docker system prune -a`, `rm -rf /opt/TverActive`) в аварийном режиме.
2. Минимальный rollback:
   - `docker compose -f docker-compose.timeweb.yaml up -d --no-build`
   - `docker compose -f docker-compose.timeweb.yaml restart caddy max-bot-service core-api`
3. Если проблема после конкретной сборки:
   - вернуть предыдущий рабочий tar-архив проекта,
   - запустить `./infra/deploy-timeweb.sh`.

## 8. Пост-деплой контроль

1. Зафиксировать дату/время релиза и `BUILD_ID`.
2. Сохранить вывод:
   - `docker compose -f docker-compose.timeweb.yaml ps`
   - `docker logs --tail 100 core-api`
   - `docker logs --tail 100 max-bot-service`
3. Подтвердить, что mini app открывается в MAX и в обычном браузере.
