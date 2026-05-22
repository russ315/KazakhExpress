#!/usr/bin/env bash
set -euo pipefail

export DATABASE_URL="${DATABASE_URL:-postgres://postgres:postgres@localhost:5432/kazakhexpress?sslmode=disable}"
export NATS_URL="${NATS_URL:-nats://localhost:4222}"

docker compose up -d postgres nats migrate

for service in user-service order-service product-service payment-service review-service; do
  echo
  echo "== integration: ${service} =="
  (cd "$service" && go test -tags=integration ./...)
done

echo
echo "PostgreSQL and NATS integration tests passed."
