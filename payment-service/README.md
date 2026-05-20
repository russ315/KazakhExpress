# KazakhExpress Payment Service

Тема проекта: национальный казахский аналог AliExpress.

`payment-service` отвечает за оплату заказов, возвраты, идемпотентность оплат, публикацию событий оплаты и отправку email-чеков.

## Что покрывает критерии

- Clean architecture: бизнес-логика в `internal/payment`, app wiring в `internal/paymentapp`, внешние адаптеры в `internal/http`, `internal/grpcapi`, `internal/messaging`, `internal/cache`, `internal/email`.
- HTTP: Gin routes для локальной демонстрации и health checks.
- gRPC: payment-service слушает отдельный порт, gateway ходит к нему через gRPC.
- gRPC contract: 7 методов из отдельного repo `github.com/maqsatto/kazakhexpress-proto`.
- Message Queue: RabbitMQ topic exchange `kazakhexpress.events`; NATS можно включить через `MESSAGE_BROKER=nats`.
- Database: PostgreSQL repository и миграция для `payments`, `payment_events`, `refunds`.
- Redis: ключи идемпотентности `payment:idempotency:{key}`.
- Email: gRPC client к общему `smtp-service` для receipt/refund/failure email.
- Tests: unit-тесты payment use case layer.
- Observability: `GET /metrics` plus Grafana with PostgreSQL 17 datasource.

Вместе с `user-service`, `product-service` и `order-service` проект может набрать минимум 12 gRPC endpoints, если каждый сервис имеет хотя бы 4 метода.

## gRPC методы payment-service

- `CreatePayment`
- `GetPayment`
- `ListPayments`
- `GetPaymentByOrderID`
- `RefundPayment`
- `ConfirmPayment`
- `CancelPayment`

## Переменные окружения

```powershell
$env:PAYMENT_SERVICE_PORT="8083"
$env:PAYMENT_GRPC_PORT="9093"
$env:SMTP_GRPC_ADDR="localhost:9094"
$env:DATABASE_URL="postgres://postgres:postgres@localhost:5432/kazakhexpress?sslmode=disable"
$env:REDIS_ADDR="localhost:6379"
$env:MESSAGE_BROKER="rabbitmq"
$env:RABBITMQ_URL="amqp://guest:guest@localhost:5672/"
$env:RABBITMQ_EXCHANGE="kazakhexpress.events"
$env:NATS_URL="nats://localhost:4222"
```

## Запуск

Сначала применить миграцию к PostgreSQL:

```powershell
psql $env:DATABASE_URL -f migrations/001_create_payments.sql
psql $env:DATABASE_URL -f migrations/002_extend_payments.sql
```

Потом запустить Redis, RabbitMQ, `smtp-service` и сервис:

```powershell
go run ./cmd/payment-service
```

## HTTP API для демонстрации

### Health

```http
GET /health
GET /metrics
```

### Создать оплату

```http
POST /payment
```

```json
{
  "order_id": "ord-1",
  "customer_id": "usr-1",
  "customer_email": "buyer@example.com",
  "amount_kzt": 25000,
  "method": "kaspi",
  "idempotency_key": "checkout-ord-1-attempt-1"
}
```

### Получить оплаты

```http
GET /payment
GET /payment/{id}
GET /payment/order/{orderId}
GET /payment?customer_id=usr-1
```

### Сделать возврат

```http
POST /payment/{id}/refund
```

```json
{
  "reason": "customer request"
}
```

### Отменить оплату

```http
POST /payment/{id}/cancel
```

```json
{
  "reason": "order cancelled"
}
```

### Mock webhook

```http
POST /payment/webhook/mock
```

```json
{
  "payment_id": "pay-...",
  "status": "succeeded",
  "provider_transaction_id": "mock-txn-1"
}
```

## События

RabbitMQ exchange:

```txt
kazakhexpress.events
```

## Gateway integration

Gateway должен ходить в payment-service по gRPC. Payment-service должен ходить в общий SMTP service по gRPC:

```powershell
$env:PAYMENT_GRPC_ADDR="localhost:9093"
$env:SMTP_GRPC_ADDR="localhost:9094"
```

HTTP routes gateway держит у себя в `api-gateway/internal/paymentservice`, а payment-service остается отдельным микросервисом.

Routing keys:

```txt
payment.created
payment.succeeded
payment.failed
payment.refunded
payment.cancelled
```

## Тесты

```powershell
go test ./...
```
