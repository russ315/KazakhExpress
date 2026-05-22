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
frontend         consumer marketplace UI on :5173
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
Frontend     http://localhost:5173
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

The frontend is consumer-first: catalog, registration, cart, order, mock payment through the gateway, receipt email, and reviews. Backend/admin checks live under:

```txt
http://localhost:5173/ops
```

Gateway rate limiting is backed by Redis. Defaults:

```txt
RATE_LIMIT_REQUESTS=120
RATE_LIMIT_WINDOW_SECONDS=60
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

Email uses the shared `smtp-service`. If no credentials are configured it logs dry-run messages and does not crash. Resend is preferred for the hosted domain; classic SMTP still works as fallback.

```powershell
$env:RESEND_API_KEY="your-resend-key"
$env:RESEND_FROM="KazakhExpress <noreply@send.maqsatto.dev>"
docker compose up --build
```

Classic SMTP fallback:

```powershell
$env:SMTP_USERNAME="your@gmail.com"
$env:SMTP_PASSWORD="your-app-password"
$env:SMTP_FROM="noreply@kazakhexpress.kz"
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

Push-Location frontend
npm ci
npm run lint
npm run build
Pop-Location
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

End-to-end smoke test through the API Gateway:

```powershell
docker compose up --build -d
docker compose --profile seed run --rm seed-data
powershell -ExecutionPolicy Bypass -File scripts/smoke.ps1
```

PostgreSQL and NATS integration tests:

```powershell
powershell -ExecutionPolicy Bypass -File scripts/integration.ps1
```

Full unit + mock + integration test report (creates docs/test-execution-report.md):

```powershell
powershell -ExecutionPolicy Bypass -File scripts/run-all-tests.ps1
```

Teacher demo flow with step-by-step API load for Grafana:

```powershell
powershell -ExecutionPolicy Bypass -File scripts/demo-showcase.ps1 -Interactive
```

Sequential load generator (phase-by-phase spikes for Grafana):

```powershell
powershell -ExecutionPolicy Bypass -File scripts/demo-load-generator.ps1
```

Useful dashboards while the demo script runs:

```txt
KazakhExpress Ultimate Performance Dashboard
KazakhExpress Backend Overview
KazakhExpress Payment Flow
KazakhExpress Catalog And Reviews
KazakhExpress Messaging And Infra
```

Demo runbook:

```txt
docs/demo-workflow.md
docs/backend-status.md
```

## Observability

Grafana is provisioned with Prometheus, Loki, Tempo, and ready dashboards:

```txt
KazakhExpress Ultimate Performance Dashboard
KazakhExpress Backend Overview
KazakhExpress Payment Flow
KazakhExpress Catalog And Reviews
KazakhExpress Messaging And Infra
```

GitHub Actions workflows:

```txt
service-ci -> unit tests, vet, Docker build, compose validation, seed, smoke
backend-integration -> Postgres + NATS integration tests
test-report -> unit + integration report artifact
proto-generation -> buf lint/generate + proto tests
```

Current services expose `/metrics` where HTTP is enabled, and all service logs are structured enough for local debugging. Tracing pipeline is ready through the OTel Collector and Tempo.
