$ErrorActionPreference = "Stop"

$env:DATABASE_URL = if ($env:DATABASE_URL) { $env:DATABASE_URL } else { "postgres://postgres:postgres@localhost:5432/kazakhexpress?sslmode=disable" }
$env:NATS_URL = if ($env:NATS_URL) { $env:NATS_URL } else { "nats://localhost:4222" }

docker compose up -d postgres nats migrate

$services = @(
  "user-service",
  "order-service",
  "product-service",
  "payment-service",
  "review-service"
)

foreach ($service in $services) {
  Write-Host "`n== integration: $service ==" -ForegroundColor Cyan
  Push-Location $service
  go test -tags=integration ./...
  Pop-Location
}

Write-Host "`nPostgreSQL and NATS integration tests passed." -ForegroundColor Green
