# KazakhExpress Backend

Microservice backend with one public Gin API Gateway and internal gRPC services.

## Services

```txt
api-gateway      public HTTP API on :8080
user-service     auth, profile, JWT, welcome email
product-service  products, stock, MinIO image storage
order-service    orders, status flow, payment event consumer
payment-service  idempotent payments, refunds, SMTP receipts
review-service   product reviews and cached rating
smtp-service     shared SMTP sender used by other services
```

Shared protobuf contracts live in:

```txt
github.com/maqsatto/kazakhexpress-proto
```

Local modules use:

```txt
replace github.com/maqsatto/kazakhexpress-proto => ../../kazakhexpress-proto
```

Keep the proto repository checked out next to this repository:

```txt
Projects/
  KazakhExpress/
  kazakhexpress-proto/
```

## Run

```powershell
docker compose up --build
```

Main URLs:

```txt
API Gateway  http://localhost:8080
Grafana      http://localhost:3000  admin/admin
NATS monitor http://localhost:8222
MinIO        http://localhost:9001  minioadmin/minioadmin
PostgreSQL   localhost:5432
Redis        localhost:6379
```

Business services are internal. Use the gateway, for example:

```powershell
curl http://localhost:8080/health
curl http://localhost:8080/payment/health
curl http://localhost:8080/products
```

Seed catalog products with image uploads to MinIO and demo reviews:

```powershell
docker compose --profile seed run --rm seed-data
```

OpenAPI contract:

```txt
api-gateway/openapi.yaml
```

## Payment Smoke Flow

```powershell
$body = @{
  order_id = "ord-demo"
  customer_id = "usr-demo"
  customer_email = "dev@example.com"
  amount_kzt = 15000
  method = "card"
  idempotency_key = "demo-key-1"
} | ConvertTo-Json

Invoke-RestMethod -Method Post -Uri http://localhost:8080/payment -ContentType application/json -Body $body
Invoke-RestMethod -Method Post -Uri http://localhost:8080/payment -ContentType application/json -Body $body
```

The second request returns the same payment through Redis idempotency. If SMTP credentials are empty, the SMTP service logs a dry-run email.

## SMTP

Set these when real email is needed:

```powershell
$env:SMTP_USERNAME="your@gmail.com"
$env:SMTP_PASSWORD="your-app-password"
$env:SMTP_FROM="noreply@kazakhexpress.kz"
docker compose up --build
```

## Tests

Run all modules:

```powershell
foreach ($svc in "api-gateway","user-service","order-service","product-service","payment-service","review-service","smtp-service") {
  Push-Location $svc
  go test ./...
  go vet ./...
  Pop-Location
}
```

Proto repo:

```powershell
Push-Location ..\kazakhexpress-proto
buf lint
buf generate
go test ./...
git diff --exit-code
Pop-Location
```

## Observability

Grafana is provisioned with Prometheus, Loki, Tempo, and ready dashboards:

```txt
KazakhExpress Backend Overview
KazakhExpress Payment Flow
```

Current services expose `/metrics` where HTTP is enabled, and all service logs are structured enough for local debugging. Tracing pipeline is ready through the OTel Collector and Tempo.
