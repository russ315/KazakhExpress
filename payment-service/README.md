# Payment Service

Payment microservice for checkout, payment status, refunds, idempotency, NATS events, PostgreSQL persistence, Redis cache, and email receipts through the shared SMTP service.

## gRPC

```txt
kazakhexpress.payment.v1.PaymentService
```

Methods:

```txt
HealthCheck
CreatePayment
GetPayment
GetPaymentByOrderID
ListPayments
RefundPayment
ConfirmPayment
CancelPayment
```

## Runtime Dependencies

```txt
PostgreSQL 17
Redis
NATS
smtp-service
```

## Events

Published subjects:

```txt
payment.created
payment.succeeded
payment.failed
payment.refunded
payment.cancelled
```

## Environment

```powershell
$env:PAYMENT_SERVICE_PORT="8083"
$env:PAYMENT_GRPC_PORT="9093"
$env:DATABASE_URL="postgres://postgres:postgres@localhost:5432/kazakhexpress?sslmode=disable"
$env:REDIS_ADDR="localhost:6379"
$env:NATS_URL="nats://localhost:4222"
$env:SMTP_GRPC_ADDR="localhost:9094"
```

In Docker, payment HTTP is internal. Public calls go through API Gateway:

```txt
GET  /payment/health
POST /payment
GET  /payment
GET  /payment/:id
GET  /payment/order/:orderId
POST /payment/:id/refund
POST /payment/:id/confirm
POST /payment/:id/cancel
```

## Idempotency

`CreatePayment` stores `payment:idempotency:{key}` in Redis. Repeating the same checkout request with the same key returns the existing payment instead of creating a duplicate.

## Tests

```powershell
go test ./...
go vet ./...
```
