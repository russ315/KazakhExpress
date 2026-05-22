$ErrorActionPreference = "Stop"

$ApiBaseUrl = if ($env:API_BASE_URL) { $env:API_BASE_URL } else { "http://localhost:8080" }
$GrafanaUrl = if ($env:GRAFANA_URL) { $env:GRAFANA_URL } else { "http://localhost:3000" }
$NatsUrl = if ($env:NATS_URL) { $env:NATS_URL } else { "http://localhost:8222" }
$Suffix = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()

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

Wait-Url "$ApiBaseUrl/health" "gateway"
Wait-Url "$ApiBaseUrl/payment/health" "payment"
Wait-Url "$ApiBaseUrl/products/health" "product"
Wait-Url "$ApiBaseUrl/reviews/health" "review"

$products = Invoke-RestMethod "$ApiBaseUrl/products"
if ($products.Count -lt 1) { throw "expected seed products" }
$product = $products[0]
if ($product.image_url) {
  Invoke-WebRequest -Uri $product.image_url -Method Head -UseBasicParsing | Out-Null
}

$orderBody = @{
  customer_id = "usr-smoke-$Suffix"
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
  customer_id = "usr-smoke-$Suffix"
  customer_email = "teacher-demo@example.com"
  amount_kzt = [int64]$order.total_kzt
  method = "card"
  idempotency_key = "smoke-$Suffix"
} | ConvertTo-Json
$paymentA = Invoke-RestMethod -Method Post -Uri "$ApiBaseUrl/payment" -ContentType application/json -Body $paymentBody
$paymentB = Invoke-RestMethod -Method Post -Uri "$ApiBaseUrl/payment" -ContentType application/json -Body $paymentBody
if ($paymentA.id -ne $paymentB.id -or $paymentA.status -ne "succeeded") { throw "payment idempotency failed" }

$finalStatus = ""
for ($i = 0; $i -lt 20; $i++) {
  $finalStatus = (Invoke-RestMethod "$ApiBaseUrl/orders/$($order.id)").status
  if ($finalStatus -eq "paid") { break }
  Start-Sleep -Seconds 1
}
if ($finalStatus -ne "paid") { throw "order was not marked paid, got $finalStatus" }

$reviewBody = @{
  customer_id = "usr-smoke-$Suffix"
  rating = 5
  comment = "Smoke test review"
} | ConvertTo-Json
Invoke-RestMethod -Method Post -Uri "$ApiBaseUrl/products/$($product.id)/reviews" -ContentType application/json -Body $reviewBody | Out-Null
$rating = Invoke-RestMethod "$ApiBaseUrl/products/$($product.id)/rating"
if ($rating.review_count -lt 1) { throw "rating was not updated" }

$refund = Invoke-RestMethod -Method Post -Uri "$ApiBaseUrl/payment/$($paymentA.id)/refund" -ContentType application/json -Body '{"reason":"smoke test refund"}'
if ($refund.status -ne "refunded") { throw "refund failed, got $($refund.status)" }

Invoke-RestMethod "$NatsUrl/healthz" | Out-Null
try {
  Invoke-RestMethod "$GrafanaUrl/api/health" | Out-Null
  $auth = [Convert]::ToBase64String([Text.Encoding]::ASCII.GetBytes("admin:admin"))
  $dashboards = Invoke-RestMethod -Headers @{Authorization = "Basic $auth"} "$GrafanaUrl/api/search?query=KazakhExpress"
  if ($dashboards.Count -lt 4) { throw "expected provisioned dashboards, got $($dashboards.Count)" }
} catch {
  throw
}

[pscustomobject]@{
  ok = $true
  products = $products.Count
  product_id = $product.id
  order_id = $order.id
  payment_id = $paymentA.id
  final_order_status = $finalStatus
  refund_status = $refund.status
} | ConvertTo-Json -Compress
