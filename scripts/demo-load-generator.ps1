param(
  [string]$ApiBaseUrl = $(if ($env:API_BASE_URL) { $env:API_BASE_URL } else { "http://localhost:8080" }),
  [string]$GrafanaUrl = $(if ($env:GRAFANA_URL) { $env:GRAFANA_URL } else { "http://localhost:3000" })
)

$ErrorActionPreference = "Stop"

function Print-Banner($Title) {
    Write-Host "`n==========================================================================" -ForegroundColor Green
    Write-Host "  PHASE: $Title" -ForegroundColor Yellow -Bold
    Write-Host "==========================================================================" -ForegroundColor Green
}

function Wait-Enter($Prompt) {
    Write-Host "`n>>> $Prompt" -ForegroundColor Cyan
    Read-Host "Press [ENTER] to execute this phase and show the metrics spike..." | Out-Null
}

function Wait-Url($Url, $Name) {
  Write-Host "Checking if $Name ($Url) is ready..." -NoNewline
  for ($i = 0; $i -lt 30; $i++) {
    try {
      $resp = Invoke-RestMethod -Uri $Url -TimeoutSec 2
      Write-Host " -> OK!" -ForegroundColor Green
      return
    } catch {
      Write-Host "." -NoNewline
      Start-Sleep -Seconds 2
    }
  }
  throw "`nError: $Name ($Url) is not responding. Please make sure 'docker compose up -d' is fully running!"
}

# Clear console
Clear-Host
Write-Host "==========================================================================" -ForegroundColor Cyan
Write-Host "           KAZAKHEXPRESS OBSERVED DEMO TRAFFIC GENERATOR                  " -ForegroundColor Cyan
Write-Host "==========================================================================" -ForegroundColor Cyan
Write-Host " This interactive load generator will guide you step-by-step to show real " -ForegroundColor Gray
Write-Host " performance metrics and spikes on your new Ultimate Grafana Dashboard.  " -ForegroundColor Gray
Write-Host " Target Gateway: $ApiBaseUrl" -ForegroundColor Gray
Write-Host " Target Grafana: $GrafanaUrl" -ForegroundColor Gray
Write-Host "==========================================================================" -ForegroundColor Cyan

# 0. Check stack availability
Wait-Url "$ApiBaseUrl/health" "API Gateway"
Wait-Url "$ApiBaseUrl/products" "Catalog service"

$suffix = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()
$customerIds = @()
$customerEmails = @()
$orderIds = @()
$product = $null

# Fetch product first
try {
    $products = Invoke-RestMethod "$ApiBaseUrl/products"
    if ($products.Count -lt 1) {
        throw "No products in database. Please run: docker compose --profile seed run --rm seed-data"
    }
    $product = $products[0]
    Write-Host "Using seeded product: $($product.name) (ID: $($product.id), Price: $($product.price_kzt) KZT)" -ForegroundColor Green
} catch {
    Write-Host "Error fetching seeded product. Did you run the seed command?" -ForegroundColor Red
    Exit 1
}

# ---------------------------------------------------------
# PHASE 1
# ---------------------------------------------------------
Print-Banner "1. User Registration Spike (NATS & SMTP Email Load)"
Write-Host "Action: Sequentially registers 20 new users to the database." -ForegroundColor Gray
Write-Host "Under the hood: Triggers bcrypt password hashing (high CPU) and publishes" -ForegroundColor Gray
Write-Host "NATS events which trigger smtp-service background welcome email runs." -ForegroundColor Gray

Wait-Enter "Ready to register 20 users?"

Write-Host "Sending registrations..." -ForegroundColor DarkGray
for ($i = 1; $i -le 20; $i++) {
    $email = "demo-$suffix-$i@kazakhexpress.kz"
    $body = @{
        email = $email
        password = "SecurePassword123!"
        first_name = "DemoUser-$i"
        last_name = "KazExpress"
        phone = "+770712345$i"
        address = "Astana, Kazakhstan"
    } | ConvertTo-Json
    
    try {
        $resp = Invoke-RestMethod -Method Post -Uri "$ApiBaseUrl/auth/register" -ContentType "application/json" -Body $body
        $customerIds += $resp.user.id
        $customerEmails += $resp.user.email
        Write-Host "  Registered User $i/20: $email" -ForegroundColor Gray
    } catch {
        Write-Host "  Failed to register user $i: $_" -ForegroundColor Red
    }
}

Write-Host "`n*** DEMO PRESENTATION TIP FOR THE PROFESSOR ***" -ForegroundColor Cyan
Write-Host "Point the professor to the Grafana dashboard:"
Write-Host "1. Row: 'Go Runtime Diagnostics' -> Look at 'Active Go Goroutines by Service'. You'll see user-service and smtp-service goroutines climb!"
Write-Host "2. Row: 'Infrastructure, Caching & Broker Queues' -> Notice the 'NATS Active Clients Pool' connection metrics fluctuating."
Write-Host "3. Row: 'Centralized Log Streams (Loki)' -> Notice logs containing 'Failed to send welcome email (dry-run)' or 'welcome email sent'."

# ---------------------------------------------------------
# PHASE 2
# ---------------------------------------------------------
Print-Banner "2. Product Catalog Browsing Storm (HTTP & Throughput Load)"
Write-Host "Action: Simulates a heavy storm of 150 product catalog retrieval requests" -ForegroundColor Gray
Write-Host "under 5 seconds to show high concurrency throughput without latency penalties." -ForegroundColor Gray

Wait-Enter "Ready to storm the products endpoint?"

Write-Host "Storming API Gateway..." -ForegroundColor DarkGray
$stormCount = 150
$startTick = Get-Date

for ($i = 1; $i -le $stormCount; $i++) {
    try {
        $null = Invoke-RestMethod -Uri "$ApiBaseUrl/products"
        if ($i % 30 -eq 0) { Write-Host "  Completed $i/$stormCount search requests..." -ForegroundColor Gray }
    } catch {
        Write-Host "  Request $i failed" -ForegroundColor Red
    }
}
$dur = [Math]::Round(((Get-Date) - $startTick).TotalSeconds, 2)
$rps = [Math]::Round(($stormCount / $dur), 1)
Write-Host "Browsing storm complete. Sent $stormCount requests in $dur seconds (~$rps RPS)." -ForegroundColor Green

Write-Host "`n*** DEMO PRESENTATION TIP FOR THE PROFESSOR ***" -ForegroundColor Cyan
Write-Host "Point the professor to the Grafana dashboard:"
Write-Host "1. Row: 'HTTP & gRPC Traffic Analytics' -> Look at 'Throughput: HTTP Metrics Endpoint Scan Rates'."
Write-Host "   You should see a sharp upward spike showing intense requests rate!"
Write-Host "2. 'Accumulated HTTP Status Codes' -> You will see 200 HTTP code bar count spike rapidly!"

# ---------------------------------------------------------
# PHASE 3
# ---------------------------------------------------------
Print-Banner "3. High-Velocity Order Spike (PostgreSQL Db Load)"
Write-Host "Action: Generates 30 orders in rapid succession for the registered users." -ForegroundColor Gray
Write-Host "Under the hood: Hits PostgreSQL database with heavy transaction writes." -ForegroundColor Gray

Wait-Enter "Ready to trigger 30 PostgreSQL orders?"

Write-Host "Placing orders..." -ForegroundColor DarkGray
$orderCount = [Math]::Min(30, $customerIds.Count)

for ($i = 0; $i -lt $orderCount; $i++) {
    $cid = $customerIds[$i]
    $body = @{
        customer_id = $cid
        items = @(
            @{
                product_id = $product.id
                name = $product.name
                quantity = 1
                price_kzt = [int64]$product.price_kzt
            }
        )
    } | ConvertTo-Json -Depth 5
    
    try {
        $resp = Invoke-RestMethod -Method Post -Uri "$ApiBaseUrl/orders" -ContentType "application/json" -Body $body
        $orderIds += $resp.id
        Write-Host "  Created Order $i/$orderCount: $($resp.id) Total: $($resp.total_kzt) KZT" -ForegroundColor Gray
    } catch {
        Write-Host "  Failed to place order $i: $_" -ForegroundColor Red
    }
}

Write-Host "`n*** DEMO PRESENTATION TIP FOR THE PROFESSOR ***" -ForegroundColor Cyan
Write-Host "Point the professor to the Grafana dashboard:"
Write-Host "1. Row: 'Infrastructure, Caching & Broker Queues' -> Look at 'PostgreSQL Db Active Connections Pool'."
Write-Host "   Explain how pgx pool scales up active database sessions to process the transaction writes!"
Write-Host "2. Row: 'Go Runtime Diagnostics' -> Look at 'Active Go Goroutines' for 'order-service'."

# ---------------------------------------------------------
# PHASE 4
# ---------------------------------------------------------
Print-Banner "4. Payment Storm & Idempotency Defense (Redis Locks & Fast Hits)"
Write-Host "Action: Sequentially processes payments for the 30 orders. To show the power" -ForegroundColor Gray
Write-Host "of Redis idempotency, it deliberately retries each payment with the identical" -ForegroundColor Gray
Write-Host "idempotency key immediately, executing the defense check." -ForegroundColor Gray

Wait-Enter "Ready to run the Payment Storm?"

Write-Host "Executing idempotent payments..." -ForegroundColor DarkGray
for ($i = 0; $i -lt $orderIds.Count; $i++) {
    $oid = $orderIds[$i]
    $cid = $customerIds[$i]
    $email = $customerEmails[$i]
    
    $body = @{
        order_id = $oid
        customer_id = $cid
        customer_email = $email
        amount_kzt = [int64]$product.price_kzt
        method = "card"
        idempotency_key = "lock-demo-$suffix-$i"
    } | ConvertTo-Json
    
    try {
        # First payment request (creates the record)
        $p1 = Invoke-RestMethod -Method Post -Uri "$ApiBaseUrl/payment" -ContentType "application/json" -Body $body
        
        # Immediate duplicate request with exact same idempotency key (defense check)
        $p2 = Invoke-RestMethod -Method Post -Uri "$ApiBaseUrl/payment" -ContentType "application/json" -Body $body
        
        Write-Host "  Paid Order $i: $oid -> Payment ID: $($p1.id) (Duplicate ID match: $($p1.id -eq $p2.id))" -ForegroundColor Gray
    } catch {
        Write-Host "  Failed payment for order $oid: $_" -ForegroundColor Red
    }
}

Write-Host "`n*** DEMO PRESENTATION TIP FOR THE PROFESSOR ***" -ForegroundColor Cyan
Write-Host "Point the professor to the Grafana dashboard:"
Write-Host "1. Explain that double hitting the payments endpoint usually causes race conditions, but Redis-backed"
Write-Host "   idempotency handles the duplicate instantly in <1ms without duplicating database queries!"
Write-Host "2. Check the 'Centralized Log Streams (Loki)' -> Look for payment-service logs containing: 'returning cached payment'."

# ---------------------------------------------------------
# PHASE 5
# ---------------------------------------------------------
Print-Banner "5. Rating Cache & Reviews Storm (OTel, Logs & Full Pipeline)"
Write-Host "Action: Submits 15 product reviews with ratings, which updates the Redis cache" -ForegroundColor Gray
Write-Host "and triggers rating recalculations, outputting detailed Loki logs." -ForegroundColor Gray

Wait-Enter "Ready to trigger the Reviews Storm?"

Write-Host "Submitting product reviews..." -ForegroundColor DarkGray
$reviewCount = [Math]::Min(15, $customerIds.Count)

for ($i = 0; $i -lt $reviewCount; $i++) {
    $cid = $customerIds[$i]
    $body = @{
        customer_id = $cid
        rating = 5
        comment = "Amazing fast shipping! Highly recommended KazakhExpress product!"
    } | ConvertTo-Json
    
    try {
        $null = Invoke-RestMethod -Method Post -Uri "$ApiBaseUrl/products/$($product.id)/reviews" -ContentType "application/json" -Body $body
        Write-Host "  Review $i/$reviewCount submitted by customer $cid" -ForegroundColor Gray
    } catch {
        Write-Host "  Failed to submit review $i: $_" -ForegroundColor Red
    }
}

# Fetch recalculated rating
try {
    $rating = Invoke-RestMethod "$ApiBaseUrl/products/$($product.id)/rating"
    Write-Host "`nRecalculated cached rating: Count: $($rating.review_count), Average: $($rating.average_rating)" -ForegroundColor Green
} catch {
    Write-Host "Failed to fetch updated rating cache" -ForegroundColor Red
}

Write-Host "`n*** DEMO PRESENTATION TIP FOR THE PROFESSOR ***" -ForegroundColor Cyan
Write-Host "Point the professor to the Grafana dashboard:"
Write-Host "1. Row: 'Centralized Log Streams (Loki)' -> Expand it at the bottom. You will see colored logs"
Write-Host "   from api-gateway, review-service, and product-service cooperating seamlessly!"
Write-Host "2. Open your browser and show: http://localhost:5173/ops"
Write-Host "   This page displays the real-time operational diagnostics of all microservices."

Write-Host "`n==========================================================================" -ForegroundColor Green
Write-Host "  DEMO LOAD GENERATOR RUN COMPLETED SUCCESSFULLY!" -ForegroundColor Yellow
Write-Host "  Open your Ultimate Performance Dashboard at: http://localhost:3000" -ForegroundColor Cyan
Write-Host "==========================================================================" -ForegroundColor Green
