# Review Service

Reviews, ratings, eligibility, Redis cache, and RabbitMQ integration for KazakhExpress.

## Run

```powershell
$env:DATABASE_URL="postgres://postgres:postgres@localhost:5432/kazakhexpress_reviews?sslmode=disable"
$env:REDIS_ADDR="localhost:6379"
$env:RABBITMQ_URL="amqp://guest:guest@localhost:5672/"
go run ./cmd/review-service
```

gRPC default port: `9095` (`REVIEW_GRPC_PORT`).

## gRPC API

- `CreateReview`
- `GetReview`
- `ListProductReviews`
- `UpdateReview`
- `DeleteReview`
- `GetProductRating`

## HTTP via API Gateway

| Method | Route |
|--------|-------|
| POST | `/products/{productId}/reviews` |
| GET | `/products/{productId}/reviews` |
| GET | `/products/{productId}/rating` |
| GET | `/reviews/{id}` |
| PUT | `/reviews/{id}` |
| DELETE | `/reviews/{id}` |

## PostgreSQL

- `reviews` — review records (one per user per product)
- `product_ratings` — denormalized averages
- `review_votes` — optional helpful votes
- `review_eligibility` — granted when `order.completed` is consumed

## Redis

- `product:{id}:rating`
- `product:{id}:reviews:page:{n}`

## RabbitMQ

**Publish** (`review.events`):

- `review.created`
- `review.updated`
- `review.deleted`
- `product.rating.updated`

**Consume** (`kazakhexpress.events`):

- `order.completed` → grants review eligibility for order line items

## Tests

```powershell
go test ./...
```
