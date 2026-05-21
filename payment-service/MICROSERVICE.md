# Payment Microservice

## Purpose

`payment-service` закрывает оплату заказа: создание платежа, идемпотентность повторного checkout request, mock provider charge, возвраты, отмены, payment events, read API и вызовы общего SMTP service.

## Runtime Interfaces

```txt
HTTP Gin     : internal only; docker-compose does not publish it
gRPC server  : :9093 kazakhexpress.payment.v1.PaymentService, internal Docker network only
gRPC client  : smtp-service:9094 kazakhexpress.smtp.v1.SMTPService, internal Docker network only
PostgreSQL  : payments, payment_events, refunds
Redis       : payment:idempotency:{key}
RabbitMQ    : kazakhexpress.events topic exchange
SMTP service : receipt, refund, failure emails
Grafana     : PostgreSQL 17 datasource for payment tables
```

External clients must call payment through API Gateway:

```txt
http://localhost:8080/payment
```

## Package Layout

```txt
cmd/payment-service       thin entrypoint, imports internal/paymentapp once
internal/paymentapp       dependency wiring and HTTP/gRPC server startup
internal/payment          domain model, use cases, repository interface
internal/http             Gin HTTP adapter
internal/grpcapi          gRPC server adapter
internal/cache            Redis idempotency adapter
internal/messaging        RabbitMQ and NATS publishers
internal/email            SMTP service gRPC client adapter
internal/provider         mock payment provider
migrations                PostgreSQL schema migrations
proto                     moved to github.com/maqsatto/kazakhexpress-proto
```

## Main Flow

```txt
1. Gateway receives POST /payment through Gin.
2. Gateway internal/paymentservice calls payment-service gRPC CreatePayment.
3. payment-service checks Redis idempotency key.
4. payment-service inserts pending payment into PostgreSQL.
5. payment-service publishes payment.created.
6. mock provider returns succeeded or failed.
7. payment-service updates status and appends payment_events row.
8. payment-service publishes payment.succeeded or payment.failed.
9. payment-service calls smtp-service to send receipt or failure email.
```

## Local Infra

```powershell
docker compose -f infra/payment-compose.yml up -d
```

The compose file uses PostgreSQL 17, Redis, RabbitMQ with management UI, smtp-service, api-gateway, payment-service, and Grafana with a PostgreSQL datasource.

## Proto Source

The gRPC source of truth lives in a separate repository:

```txt
github.com/maqsatto/kazakhexpress-proto
```

Local development uses a Go module `replace` pointing to the sibling `../kazakhexpress-proto` checkout.

## Verification

```powershell
go test ./...
go vet ./...
```
