# KazakhExpress Payment Service

Тема проекта: национальный казахский аналог AliExpress.

`payment-service` отвечает за оплату заказов, возвраты, публикацию событий оплаты и отправку email-чеков.

## Что покрывает критерии

- Clean architecture: бизнес-логика в `internal/payment`, внешние адаптеры в `internal/http`, `internal/messaging`, `internal/email`.
- gRPC endpoints: 4 метода в `proto/payment.proto`.
- Message Queue: NATS publisher для `payments.created` и `payments.refunded`.
- Database: PostgreSQL repository и миграция `migrations/001_create_payments.sql`.
- Email: SMTP sender для receipt/refund email.
- Tests: unit-тесты для payment service.

Вместе с `user-service`, `product-service` и `order-service` проект может набрать минимум 12 gRPC endpoints, если каждый сервис имеет хотя бы 4 метода.

## gRPC методы payment-service

- `CreatePayment`
- `GetPayment`
- `ListPayments`
- `RefundPayment`

## Переменные окружения

```powershell
$env:PAYMENT_SERVICE_PORT="8083"
$env:DATABASE_URL="postgres://postgres:postgres@localhost:5432/kazakhexpress?sslmode=disable"
$env:NATS_URL="nats://localhost:4222"
$env:SMTP_HOST="smtp.gmail.com"
$env:SMTP_PORT="587"
$env:SMTP_USERNAME="your-email@gmail.com"
$env:SMTP_PASSWORD="your-app-password"
$env:SMTP_FROM="noreply@kazakhexpress.kz"
```

## Запуск

Сначала применить миграцию к PostgreSQL:

```powershell
psql $env:DATABASE_URL -f migrations/001_create_payments.sql
```

Потом запустить NATS и сервис:

```powershell
go run ./cmd/payment-service
```

## HTTP API для демонстрации

### Health

```http
GET /health
```

### Создать оплату

```http
POST /payments
```

```json
{
  "order_id": "ord-1",
  "customer_id": "usr-1",
  "customer_email": "buyer@example.com",
  "amount_kzt": 25000,
  "method": "kaspi"
}
```

### Получить оплаты

```http
GET /payments
GET /payments/{id}
```

### Сделать возврат

```http
POST /payments/{id}/refund
```

```json
{
  "reason": "customer request"
}
```

## Тесты

```powershell
go test ./...
```
