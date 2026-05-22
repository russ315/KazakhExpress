# KazakhExpress Product Service

Catalog and inventory service: products CRUD, stock updates, reserve/release, Redis product cache, RabbitMQ stock events.

Reviews and ratings moved to **review-service** (exposed via API Gateway).

## Run

```powershell
go run ./cmd/product-service
```

Port `8082` (`PRODUCT_SERVICE_PORT`).

## API (direct)

| Method | Route |
|--------|-------|
| POST/GET | `/products` |
| GET/PUT/PATCH/DELETE | `/products/{id}` |
| PATCH | `/products/{id}/stock` |
| POST | `/products/{id}/stock/reserve` |
| POST | `/products/{id}/stock/release` |

## Tests

```powershell
go test ./...
```
