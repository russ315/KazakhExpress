# KazakhExpress API Gateway

Gateway отдает HTTP через Gin, а до микросервисов ходит через generated gRPC clients из `github.com/maqsatto/kazakhexpress-proto`. Каждый сервис подключается отдельным package внутри `internal/<service>`, поэтому команда не будет постоянно конфликтовать в одном большом gateway handler.

## Structure

```txt
cmd/api-gateway/main.go          thin entrypoint
internal/gateway                 shared Gin router, health, CORS
internal/gatewayapp              app wiring
internal/paymentservice          /payment HTTP routes + payment gRPC client
```

Чтобы добавить новый сервис, создается новый package, например `internal/orderservice`, и в `internal/gatewayapp/app.go` добавляются только client init + `orderservice.RegisterRoutes(router, client)`.

## Routes

```txt
GET  /health
POST /payment
GET  /payment
GET  /payment/:id
GET  /payment/order/:orderID
POST /payment/:id/refund
POST /payment/:id/confirm
POST /payment/:id/cancel
```

## Environment

```powershell
$env:API_GATEWAY_PORT="8080"
$env:PAYMENT_GRPC_ADDR="localhost:9093"
```

## Run

```powershell
go run ./cmd/api-gateway
```

## Test

```powershell
go test ./...
go vet ./...
```
