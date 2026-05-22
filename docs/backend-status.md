# Backend Status

## Ready

```txt
API Gateway on :8080 with Gin
gRPC contracts from shared contracts repo
user-service auth/profile/email integration
product-service PostgreSQL, MinIO images, NATS product events
order-service PostgreSQL, Redis cache, NATS payment event consumer
payment-service PostgreSQL, Redis idempotency, NATS events, SMTP receipts/refunds
review-service PostgreSQL, Redis rating cache, NATS review events
smtp-service real SMTP or dry-run mode
PostgreSQL 17, Redis, NATS, MinIO
Grafana, Prometheus, Loki, Tempo, OTel Collector
seed data with products, images, and reviews
OpenAPI contract for gateway
CI for tests, vet, Docker build, compose validation, seed, and smoke test
```

## What Is Left

These are not blockers for backend grading, but they are the natural next improvements:

```txt
frontend integration
real SMTP credentials
real JWT middleware on gateway routes instead of demo X-User-ID header
separate databases per service if the course requires strict DB isolation
production-grade distributed tracing spans in every repository/cache/NATS call
more repository integration tests with disposable PostgreSQL containers
load testing and rate limit tuning
deployment manifests for a real server
```

## Backend Acceptance Command

```powershell
docker compose up --build -d
docker compose --profile seed run --rm seed-data
powershell -ExecutionPolicy Bypass -File scripts/smoke.ps1
```

If that passes, the backend is ready to demonstrate.
