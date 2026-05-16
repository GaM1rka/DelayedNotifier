# DelayedNotifier

Сервис отложенных уведомлений: HTTP API, очередь RabbitMQ, фоновая отправка в заданное время с повторными попытками.

## Структура проекта

```text
DelayedNotifier/
├── backend/
│   ├── api-service/       # HTTP API, Postgres, Redis, RabbitMQ publisher
│   ├── worker-service/    # Consumer очереди, планирование и retry
│   └── sender-service/    # Отправка email (SMTP)
├── frontend/              # Простой UI (HTML + JS)
├── openapi/               # OpenAPI спецификация
└── docker-compose.yml
```

## Запуск через Docker Compose

```bash
docker compose up --build
```

| Сервис | URL |
|--------|-----|
| UI | http://localhost:3000 |
| API | http://localhost:8080 |
| Swagger UI | http://localhost:8082 |
| RabbitMQ Management | http://localhost:15672 (guest/guest) |
| Mailpit | http://localhost:8025 |

## HTTP API

- `POST /notify` — создать уведомление
- `GET /notify/{id}` — статус уведомления
- `DELETE /notify/{id}` — отмена
- `GET /notify` — список для UI

## Локальная разработка

Конфигурация задаётся в `.env` файлах сервисов (`backend/*/.env`) или через переменные окружения. Для `sender-service` уже перенесён `.env` из notification-service донора; пример лежит в `backend/sender-service/.env.example`.

Если хотите отправлять через Mailpit вместо внешнего SMTP, задайте в `backend/sender-service/.env`:

```bash
HTTP_ADDR=:8004
SMTP_HOST=mailpit
SMTP_PORT=1025
SMTP_USERNAME=test
SMTP_PASSWORD=test
SMTP_FROM=test@example.com
SMTP_USE_TLS=false
```
