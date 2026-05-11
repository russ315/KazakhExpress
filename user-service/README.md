# KazakhExpress User Service

Тема проекта: национальный казахский аналог AliExpress.

Этот репозиторий содержит `user service` на Go. Сервис отвечает за регистрацию пользователей, аутентификацию, управление профилями и отправку email уведомлений.

## Что добавлено

- `go.mod` - модуль Go для user service с необходимыми зависимостями.
- `cmd/user-service/main.go` - точка запуска сервиса.
- `internal/user/model.go` - модели пользователя и структуры для API запросов.
- `internal/user/repository.go` - репозиторий с PostgreSQL для хранения данных пользователей.
- `internal/user/service.go` - бизнес-логика аутентификации и управления профилями.
- `internal/http/handler.go` - HTTP API для работы с пользователями.
- `internal/email/service.go` - сервис для отправки email уведомлений.

## Запуск

```powershell
go run ./cmd/user-service
```

По умолчанию сервис запускается на порту `8081`. Можно изменить порт через переменную окружения:

```powershell
$env:USER_SERVICE_PORT="8082"
go run ./cmd/user-service
```

## Переменные окружения

- `USER_SERVICE_PORT` - порт сервиса (по умолчанию: 8081)
- `DATABASE_URL` - строка подключения к PostgreSQL (по умолчанию: postgres://user:password@localhost/kazakhexpress_users?sslmode=disable)
- `JWT_SECRET` - секретный ключ для JWT токенов (по умолчанию: your-secret-key-change-in-production)
- `SMTP_HOST` - SMTP сервер для отправки email (по умолчанию: smtp.gmail.com)
- `SMTP_PORT` - SMTP порт (по умолчанию: 587)
- `SMTP_USERNAME` - логин для SMTP
- `SMTP_PASSWORD` - пароль для SMTP
- `FROM_EMAIL` - email отправителя (по умолчанию: noreply@kazakhexpress.kz)

## API

### Проверка сервиса

```http
GET /health
```

### Регистрация пользователя

```http
POST /auth/register
```

Пример тела запроса:

```json
{
  "email": "user@example.com",
  "password": "password123",
  "first_name": "Айгерім",
  "last_name": "Нұрмаханова",
  "phone": "+7 775 123 45 67",
  "address": "г. Алматы, ул. Абая 123"
}
```

### Вход в систему

```http
POST /auth/login
```

Пример тела запроса:

```json
{
  "email": "user@example.com",
  "password": "password123"
}
```

### Получение профиля

```http
GET /users/profile
Authorization: Bearer <token>
```

### Обновление профиля

```http
PUT /users/profile
Authorization: Bearer <token>
```

Пример тела запроса:

```json
{
  "first_name": "Айгерім",
  "last_name": "Нұрмаханова",
  "phone": "+7 775 123 45 67",
  "address": "г. Алматы, ул. Абая 456"
}
```

## Структура ответов

### Успешная аутентификация

```json
{
  "user": {
    "id": "uuid-user-id",
    "email": "user@example.com",
    "first_name": "Айгерім",
    "last_name": "Нұрмаханова",
    "phone": "+7 775 123 45 67",
    "address": "г. Алматы, ул. Абая 123",
    "created_at": "2024-01-01T12:00:00Z",
    "updated_at": "2024-01-01T12:00:00Z"
  },
  "token": "jwt-token-string"
}
```

### Профиль пользователя

```json
{
  "id": "uuid-user-id",
  "email": "user@example.com",
  "first_name": "Айгерім",
  "last_name": "Нұрмаханова",
  "phone": "+7 775 123 45 67",
  "address": "г. Алматы, ул. Абая 123",
  "created_at": "2024-01-01T12:00:00Z",
  "updated_at": "2024-01-01T12:00:00Z"
}
```

## Email уведомления

Сервис поддерживает отправку email уведомлений на двух языках (казахский и русский):

1. **Welcome Email** - отправляется при успешной регистрации пользователя
2. **Receipt Email** - отправляется при оформлении заказа (интеграция с payment service)

## Безопасность

- Пароли хешируются с использованием bcrypt
- JWT токены для аутентификации с сроком действия 7 дней
- Валидация входных данных
- Защита от SQL-инъекций через параметризованные запросы

## Что можно добавить дальше

- Валидация входных данных с использованием библиотеки validator
- Rate limiting для API endpoints
- Логирование запросов и ошибок
- Интеграция с Redis для сессий
- Поддержка OAuth2 (Google, Facebook)
- Тесты для service layer и HTTP handlers
- Docker контейнеризация
- Kubernetes deployment конфигурация
- Интеграция с NATS для асинхронной отправки email
- Поддержка множества языков в интерфейсе
