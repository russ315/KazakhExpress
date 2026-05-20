# KazakhExpress Payment Stack

This branch contains the payment-focused microservice stack:

```txt
api-gateway      Gin HTTP gateway, /payment routes, gRPC client to payment-service
payment-service  payment domain, PostgreSQL, Redis idempotency, RabbitMQ events
smtp-service     shared SMTP microservice called through gRPC
infra            Grafana provisioning
```

The protobuf source of truth lives in a separate repository:

```txt
github.com/maqsatto/kazakhexpress-proto
```

For local development, the Go modules use a `replace` directive to a sibling checkout:

```txt
../kazakhexpress-proto
```

## Run Everything

From this repository:

```powershell
docker compose up --build
```

Services:

```txt
API Gateway    http://localhost:8080
Payment API    http://localhost:8080/payment
RabbitMQ UI    http://localhost:15672
Grafana        http://localhost:3000
PostgreSQL 17  localhost:5432
Redis          localhost:6379
```

`payment-service` and `smtp-service` are internal Docker-network services. Do not call them directly from the host; use API Gateway.

## Verify

```powershell
go test ./...
go vet ./...
```

Run those commands inside each Go module: `api-gateway`, `payment-service`, and `smtp-service`.

## Smoke Test

```powershell
Invoke-RestMethod http://localhost:8080/health
Invoke-RestMethod http://localhost:8080/payment/health
```
