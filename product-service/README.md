# KazakhExpress Product Service

Project theme: a national Kazakh analogue of AliExpress.

This repository contains a foundation for the **product service** in Go. The service handles product creation, listing, fetching a product by ID, and updating stock.

## Layout

- `go.mod` — Go module for the product service.
- `cmd/product-service/main.go` — service entrypoint.
- `internal/product/model.go` — product and request models.
- `internal/product/repository.go` — repository interface and in-memory storage.
- `internal/product/service.go` — business logic for creation and stock updates.
- `internal/http/handler.go` — HTTP API.

## Run

```powershell
go run ./cmd/product-service
```

By default the service listens on port `8082`. Override with:

```powershell
$env:PRODUCT_SERVICE_PORT="8083"
go run ./cmd/product-service
```

## API

### Health

```http
GET /health
```

### Create product (CreateProduct)

```http
POST /products
```

Example body:

```json
{
  "name": "Kazakh handmade shapan",
  "description": "Wool, traditional pattern",
  "price_kzt": 25000,
  "stock": 10
}
```

`name` is required; `description` may be an empty string. `price_kzt` and `stock` must not be negative.

### List products (ListProducts)

```http
GET /products
```

### Get product by ID (GetProduct)

```http
GET /products/{id}
```

### Update stock (UpdateStock)

Sets the absolute stock quantity.

```http
PATCH /products/{id}/stock
```

Example body:

```json
{
  "stock": 7
}
```

`stock` must not be negative.

### Errors

Error responses are JSON, for example:

```json
{
  "error": "product not found"
}
```

## Possible next steps

- PostgreSQL and migrations instead of in-memory storage.
- Categories, SKU, images, publish/draft workflow.
- Pagination and filters for `GET /products`.
- Integration with the order service for stock checks at checkout.
- Authorization (e.g. JWT) for catalog changes.
- Tests for the service layer and HTTP handlers.
