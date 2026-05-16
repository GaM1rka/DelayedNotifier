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

Зарегистрируйте новую почту на https://mail.ru/. 
После регистрации:
→ Настройки
→ Пароль и безопасность
→ Пароли для внешних приложений
Генерируете пароль и добавляете .env файл внутри /sender-serivce:
bash
SMTP_HOST=smtp.mail.ru
SMTP_PORT=465
SMTP_USERNAME=your-email
SMTP_PASSWORD=your-generated-password
SMTP_FROM=your-email
SMTP_USE_TLS=true

KAFKA_BROKERS=kafka:9092
KAFKA_TOPIC=notification.sent
KAFKA_GROUP_ID=notification-service
