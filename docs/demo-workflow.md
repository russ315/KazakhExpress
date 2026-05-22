# Backend Demo Workflow

This file is the ready checklist for showing the backend.

## 1. Start Everything

```powershell
docker compose up --build -d
docker compose --profile seed run --rm seed-data
```

Wait until gateway is ready:

```powershell
curl http://localhost:8080/health
curl http://localhost:8080/payment/health
curl http://localhost:8080/products/health
curl http://localhost:8080/reviews/health
```

## 2. Run Automated Smoke Test

Windows:

```powershell
powershell -ExecutionPolicy Bypass -File scripts/smoke.ps1
```

Linux/macOS:

```bash
bash scripts/smoke.sh
```

The smoke test proves:

```txt
gateway health
payment health
product health
review health
seed products exist
product image is reachable from MinIO
order creation works through gateway
payment idempotency returns the same payment
payment.succeeded event updates order to paid
review creation updates rating cache
refund works
NATS monitor is reachable
Grafana dashboards are provisioned
```

## 3. Show Main Flow Manually

List seeded products:

```powershell
curl http://localhost:8080/products
```

Create an order:

```powershell
$orderBody = @{
  customer_id = "usr-demo-teacher"
  items = @(@{
    product_id = "replace-with-product-id"
    name = "Demo product"
    quantity = 1
    price_kzt = 59990
  })
} | ConvertTo-Json -Depth 6

$order = Invoke-RestMethod -Method Post -Uri http://localhost:8080/orders -ContentType application/json -Body $orderBody
$order
```

Create a payment twice with the same idempotency key:

```powershell
$paymentBody = @{
  order_id = $order.id
  customer_id = "usr-demo-teacher"
  customer_email = "teacher-demo@example.com"
  amount_kzt = $order.total_kzt
  method = "card"
  idempotency_key = "teacher-demo-key-1"
} | ConvertTo-Json

$p1 = Invoke-RestMethod -Method Post -Uri http://localhost:8080/payment -ContentType application/json -Body $paymentBody
$p2 = Invoke-RestMethod -Method Post -Uri http://localhost:8080/payment -ContentType application/json -Body $paymentBody
$p1.id
$p2.id
```

The two payment IDs should match.

Check that the order became paid:

```powershell
Invoke-RestMethod http://localhost:8080/orders/$($order.id)
```

Create review and show cached rating:

```powershell
$reviewBody = @{
  customer_id = "usr-demo-teacher"
  rating = 5
  comment = "Works through gateway"
} | ConvertTo-Json

Invoke-RestMethod -Method Post -Uri http://localhost:8080/products/replace-with-product-id/reviews -ContentType application/json -Body $reviewBody
Invoke-RestMethod http://localhost:8080/products/replace-with-product-id/rating
```

## 4. Show Observability

Open:

```txt
Grafana:      http://localhost:3000  admin/admin
NATS monitor: http://localhost:8222
MinIO:        http://localhost:9001  minioadmin/minioadmin
```

Grafana dashboards:

```txt
KazakhExpress Backend Overview
KazakhExpress Payment Flow
KazakhExpress Catalog And Reviews
KazakhExpress Messaging And Infra
```

Recommended story for the demo:

```txt
1. Gateway is the only public backend entrypoint.
2. Gateway maps HTTP routes to internal gRPC services.
3. PostgreSQL stores users, products, orders, payments, and reviews.
4. Redis handles payment idempotency and review rating cache.
5. NATS sends payment/order/review/product events.
6. SMTP service sends real email when credentials exist, otherwise dry-run logs.
7. MinIO stores product images.
8. Grafana shows service health, logs, payment flow, catalog data, and infrastructure metrics.
```
