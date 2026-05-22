param(
  [string]$ApiBaseUrl = $(if ($env:API_BASE_URL) { $env:API_BASE_URL } else { "http://localhost:8080" }),
  [string]$GrafanaUrl = $(if ($env:GRAFANA_URL) { $env:GRAFANA_URL } else { "http://localhost:3000" }),
  [switch]$Interactive
)

$ErrorActionPreference = "Stop"

function Step($Title, $Body) {
  Write-Host "`n=== $Title ===" -ForegroundColor Cyan
  if ($Body) { Write-Host $Body }
  if ($Interactive) { Read-Host "Press Enter for next step" | Out-Null }
}

function Wait-Url($Url, $Name) {
  for ($i = 0; $i -lt 60; $i++) {
    try {
      Invoke-RestMethod -Uri $Url | Out-Null
      Write-Host "$Name ready"
      return
    } catch {
      Start-Sleep -Seconds 2
    }
  }
  throw "$Name is not ready: $Url"
}

$suffix = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()

Step "0. Start stack" "Run: docker compose up -d --build && docker compose --profile seed run --rm seed-data"
Wait-Url "$ApiBaseUrl/health" "gateway"
Wait-Url "$ApiBaseUrl/payment/health" "payment"
Wait-Url "$ApiBaseUrl/products/health" "product"
Wait-Url "$ApiBaseUrl/reviews/health" "review"
Write-Host "Grafana: $GrafanaUrl"

Step "1. Register user and trigger welcome email" "Watch service logs in Grafana/Loki or docker compose logs smtp-service."
$registerBody = @{
  email = "demo-$suffix@maqsatto.dev"
  password = "Password123!"
  first_name = "Demo"
  last_name = "Buyer"
  phone = "+77010000000"
  address = "Almaty"
} | ConvertTo-Json
$auth = Invoke-RestMethod -Method Post -Uri "$ApiBaseUrl/auth/register" -ContentType application/json -Body $registerBody
$customerId = $auth.user.id
$customerEmail = $auth.user.email
Write-Host "Registered $customerEmail as $customerId"

Step "2. Browse catalog" "Open consumer frontend and Grafana Backend Overview."
$products = Invoke-RestMethod "$ApiBaseUrl/products"
if ($products.Count -lt 1) { throw "seed products missing" }
$product = $products[0]
Write-Host "Using product $($product.name) / $($product.id)"

Step "3. Create paid orders" "This creates HTTP, gRPC, Postgres, NATS and email activity."
$createdPayments = @()
for ($i = 1; $i -le 8; $i++) {
  $orderBody = @{
    customer_id = $customerId
    items = @(@{
      product_id = $product.id
      name = $product.name
      quantity = 1
      price_kzt = [int64]$product.price_kzt
    })
  } | ConvertTo-Json -Depth 6
  $order = Invoke-RestMethod -Method Post -Uri "$ApiBaseUrl/orders" -ContentType application/json -Body $orderBody
  $paymentBody = @{
    order_id = $order.id
    customer_id = $customerId
    customer_email = $customerEmail
    amount_kzt = [int64]$order.total_kzt
    method = if ($i % 2 -eq 0) { "kaspi" } else { "card" }
    idempotency_key = "demo-$suffix-$i"
  } | ConvertTo-Json
  $payment = Invoke-RestMethod -Method Post -Uri "$ApiBaseUrl/payment" -ContentType application/json -Body $paymentBody
  $createdPayments += $payment
  Write-Host "Order $($order.id) -> payment $($payment.id) $($payment.status)"
}

Step "4. Prove idempotency" "Same idempotency key returns the same payment."
$last = $createdPayments[-1]
$duplicateBody = @{
  order_id = $last.order_id
  customer_id = $customerId
  customer_email = $customerEmail
  amount_kzt = [int64]$last.amount_kzt
  method = $last.method
  idempotency_key = "demo-$suffix-8"
} | ConvertTo-Json
$duplicate = Invoke-RestMethod -Method Post -Uri "$ApiBaseUrl/payment" -ContentType application/json -Body $duplicateBody
Write-Host "Duplicate payment id: $($duplicate.id)"

Step "5. Refund and review" "Watch Payment Flow and Catalog Reviews dashboards."
$refund = Invoke-RestMethod -Method Post -Uri "$ApiBaseUrl/payment/$($createdPayments[0].id)/refund" -ContentType application/json -Body '{"reason":"teacher demo"}'
Write-Host "Refund status: $($refund.status)"
for ($i = 1; $i -le 5; $i++) {
  $reviewBody = @{ customer_id = $customerId; rating = 5; comment = "Demo review $i" } | ConvertTo-Json
  Invoke-RestMethod -Method Post -Uri "$ApiBaseUrl/products/$($product.id)/reviews" -ContentType application/json -Body $reviewBody | Out-Null
}
$rating = Invoke-RestMethod "$ApiBaseUrl/products/$($product.id)/rating"
Write-Host "Rating count: $($rating.review_count), average: $($rating.average_rating)"

Step "6. Generate API load" "This makes request rate and latency visible in Grafana."
1..45 | ForEach-Object {
  Invoke-RestMethod "$ApiBaseUrl/health" | Out-Null
  Invoke-RestMethod "$ApiBaseUrl/products" | Out-Null
}

Write-Host "`nDemo flow complete. Open Grafana dashboards and filter by service/payment/order/product/review." -ForegroundColor Green
