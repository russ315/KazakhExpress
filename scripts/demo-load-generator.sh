#!/bin/bash

# KazakhExpress Observed Demo Traffic Generator for Linux/macOS
set -e

API_BASE_URL="${API_BASE_URL:-http://localhost:8080}"
GRAFANA_URL="${GRAFANA_URL:-http://localhost:3000}"

print_banner() {
    echo -e "\n\033[0;32m==========================================================================\033[0m"
    echo -e "  \033[1;33mPHASE: $1\033[0m"
    echo -e "\033[0;32m==========================================================================\033[0m"
}

wait_enter() {
    echo -e "\n\033[0;36m>>> $1\033[0m"
    read -p "Press [ENTER] to execute this phase and show the metrics spike..."
}

wait_url() {
    echo -n "Checking if $2 ($1) is ready..."
    for i in {1..30}; do
        if curl -s --max-time 2 "$1" > /dev/null; then
            echo -e " -> \033[0;32mOK!\033[0m"
            return 0
        fi
        echo -n "."
        sleep 2
    done
    echo -e "\n\033[0;31mError: $2 ($1) is not responding. Make sure docker compose is fully up!\033[0m"
    exit 1
}

# Clear console
clear
echo -e "\033[0;36m==========================================================================\033[0m"
echo -e "\033[0;36m           KAZAKHEXPRESS OBSERVED DEMO TRAFFIC GENERATOR                  \033[0m"
echo -e "\033[0;36m==========================================================================\033[0m"
echo -e " This interactive load generator will guide you step-by-step to show real "
echo -e " performance metrics and spikes on your new Ultimate Grafana Dashboard.  "
echo -e " Target Gateway: $API_BASE_URL"
echo -e " Target Grafana: $GRAFANA_URL"
echo -e "\033[0;36m==========================================================================\033[0m"

# 0. Check stack availability
wait_url "$API_BASE_URL/health" "API Gateway"
wait_url "$API_BASE_URL/products" "Catalog service"

suffix=$(date +%s)
customer_ids=()
customer_emails=()
order_ids=()

# Fetch product first
products_json=$(curl -s "$API_BASE_URL/products")
product_id=$(echo "$products_json" | grep -oE '"id":"[^"]+"' | head -n 1 | cut -d'"' -f4)
product_name=$(echo "$products_json" | grep -oE '"name":"[^"]+"' | head -n 1 | cut -d'"' -f4)
product_price=$(echo "$products_json" | grep -oE '"price_kzt":[0-9]+' | head -n 1 | cut -d':' -f2)

if [ -z "$product_id" ]; then
    echo -e "\033[0;31mNo products found. Please seed the catalog first!\033[0m"
    exit 1
fi
echo -e "Using seeded product: \033[0;32m$product_name\033[0m (ID: $product_id, Price: $product_price KZT)"

# ---------------------------------------------------------
# PHASE 1
# ---------------------------------------------------------
print_banner "1. User Registration Spike (NATS & SMTP Email Load)"
echo -e "Action: Sequentially registers 20 new users to the database."
echo -e "Under the hood: Triggers bcrypt password hashing (high CPU) and publishes"
echo -e "NATS events which trigger smtp-service welcome email runs."

wait_enter "Ready to register 20 users?"

echo -e "Sending registrations..."
for i in {1..20}; do
    email="demo-$suffix-$i@kazakhexpress.kz"
    body="{\"email\":\"$email\",\"password\":\"SecurePassword123!\",\"first_name\":\"DemoUser-$i\",\"last_name\":\"KazExpress\",\"phone\":\"+770712345$i\",\"address\":\"Astana, Kazakhstan\"}"
    
    resp=$(curl -s -X POST -H "Content-Type: application/json" -d "$body" "$API_BASE_URL/auth/register" || echo "")
    cid=$(echo "$resp" | grep -oE '"id":"[^"]+"' | head -n 1 | cut -d'"' -f4 || "")
    cemail=$(echo "$resp" | grep -oE '"email":"[^"]+"' | head -n 1 | cut -d'"' -f4 || "")
    
    if [ -n "$cid" ]; then
        customer_ids+=("$cid")
        customer_emails+=("$cemail")
        echo -e "  Registered User $i/20: $email"
    else
        echo -e "  \033[0;31mFailed to register user $i\033[0m"
    fi
done

echo -e "\n\033[0;36m*** DEMO PRESENTATION TIP FOR THE PROFESSOR ***\033[0m"
echo -e "Point the professor to the Grafana dashboard:"
echo -e "1. Row: 'Go Runtime Diagnostics' -> Look at 'Active Go Goroutines by Service'. You'll see user-service and smtp-service goroutines climb!"
echo -e "2. Row: 'Infrastructure, Caching & Broker Queues' -> Notice the 'NATS Active Clients Pool' connection metrics fluctuating."
echo -e "3. Row: 'Centralized Log Streams (Loki)' -> Notice logs containing 'welcome email sent'."

# ---------------------------------------------------------
# PHASE 2
# ---------------------------------------------------------
print_banner "2. Product Catalog Browsing Storm (HTTP & Throughput Load)"
echo -e "Action: Simulates a heavy storm of 150 product catalog retrieval requests"
echo -e "under 5 seconds to show high concurrency throughput without latency penalties."

wait_enter "Ready to storm the products endpoint?"

echo -e "Storming API Gateway..."
storm_count=150
start_time=$(date +%s)

for i in {1..150}; do
    curl -s "$API_BASE_URL/products" > /dev/null || true
    if [ $((i % 30)) -eq 0 ]; then
        echo -e "  Completed $i/$storm_count search requests..."
    fi
done

end_time=$(date +%s)
duration=$((end_time - start_time))
if [ "$duration" -eq 0 ]; then duration=1; fi
rps=$((storm_count / duration))
echo -e "Browsing storm complete. Sent $storm_count requests in $duration seconds (~$rps RPS)."

echo -e "\n\033[0;36m*** DEMO PRESENTATION TIP FOR THE PROFESSOR ***\033[0m"
echo -e "Point the professor to the Grafana dashboard:"
echo -e "1. Row: 'HTTP & gRPC Traffic Analytics' -> Look at 'Throughput: HTTP Metrics Endpoint Scan Rates'."
echo -e "   You should see a sharp upward spike showing intense requests rate!"
echo -e "2. 'Accumulated HTTP Status Codes' -> You will see 200 HTTP code bar count spike rapidly!"

# ---------------------------------------------------------
# PHASE 3
# ---------------------------------------------------------
print_banner "3. High-Velocity Order Spike (PostgreSQL Db Load)"
echo -e "Action: Generates 30 orders in rapid succession for the registered users."
echo -e "Under the hood: Hits PostgreSQL database with heavy transaction writes."

wait_enter "Ready to trigger 30 PostgreSQL orders?"

echo -e "Placing orders..."
order_count=${#customer_ids[@]}
if [ "$order_count" -gt 30 ]; then order_count=30; fi

for ((i=0; i<order_count; i++)); do
    cid="${customer_ids[$i]}"
    body="{\"customer_id\":\"$cid\",\"items\":[{\"product_id\":\"$product_id\",\"name\":\"$product_name\",\"quantity\":1,\"price_kzt\":$product_price}]}"
    
    resp=$(curl -s -X POST -H "Content-Type: application/json" -d "$body" "$API_BASE_URL/orders" || echo "")
    oid=$(echo "$resp" | grep -oE '"id":"[^"]+"' | head -n 1 | cut -d'"' -f4 || "")
    
    if [ -n "$oid" ]; then
        order_ids+=("$oid")
        echo -e "  Created Order $i/$order_count: $oid"
    else
        echo -e "  \033[0;31mFailed to place order $i\033[0m"
    fi
done

echo -e "\n\033[0;36m*** DEMO PRESENTATION TIP FOR THE PROFESSOR ***\033[0m"
echo -e "Point the professor to the Grafana dashboard:"
echo -e "1. Row: 'Infrastructure, Caching & Broker Queues' -> Look at 'PostgreSQL Db Active Connections Pool'."
echo -e "   Explain how pgx pool scales up active database sessions to process the transaction writes!"
echo -e "2. Row: 'Go Runtime Diagnostics' -> Look at 'Active Go Goroutines' for 'order-service'."

# ---------------------------------------------------------
# PHASE 4
# ---------------------------------------------------------
print_banner "4. Payment Storm & Idempotency Defense (Redis Locks & Fast Hits)"
echo -e "Action: Sequentially processes payments for the 30 orders. To show the power"
echo -e "of Redis idempotency, it deliberately retries each payment with the identical"
echo -e "idempotency key immediately, executing the defense check."

wait_enter "Ready to run the Payment Storm?"

echo -e "Executing idempotent payments..."
for ((i=0; i<${#order_ids[@]}; i++)); do
    oid="${order_ids[$i]}"
    cid="${customer_ids[$i]}"
    email="${customer_emails[$i]}"
    key="lock-demo-$suffix-$i"
    
    body="{\"order_id\":\"$oid\",\"customer_id\":\"$cid\",\"customer_email\":\"$email\",\"amount_kzt\":$product_price,\"method\":\"card\",\"idempotency_key\":\"$key\"}"
    
    p1=$(curl -s -X POST -H "Content-Type: application/json" -d "$body" "$API_BASE_URL/payment" || echo "")
    p2=$(curl -s -X POST -H "Content-Type: application/json" -d "$body" "$API_BASE_URL/payment" || echo "")
    
    pid=$(echo "$p1" | grep -oE '"id":"[^"]+"' | head -n 1 | cut -d'"' -f4 || "")
    pid2=$(echo "$p2" | grep -oE '"id":"[^"]+"' | head -n 1 | cut -d'"' -f4 || "")
    
    if [ -n "$pid" ]; then
        match="true"
        if [ "$pid" != "$pid2" ]; then match="false"; fi
        echo -e "  Paid Order $i: $oid -> Payment ID: $pid (Duplicate lock match: $match)"
    else
        echo -e "  \033[0;31mFailed payment for order $oid\033[0m"
    fi
done

echo -e "\n\033[0;36m*** DEMO PRESENTATION TIP FOR THE PROFESSOR ***\033[0m"
echo -e "Point the professor to the Grafana dashboard:"
echo -e "1. Explain that double hitting the payments endpoint usually causes race conditions, but Redis-backed"
echo -e "   idempotency handles the duplicate instantly in <1ms without duplicating database queries!"
echo -e "2. Check the 'Centralized Log Streams (Loki)' -> Look for payment-service logs containing: 'returning cached payment'."

# ---------------------------------------------------------
# PHASE 5
# ---------------------------------------------------------
print_banner "5. Rating Cache & Reviews Storm (OTel, Logs & Full Pipeline)"
echo -e "Action: Submits 15 product reviews with ratings, which updates the Redis cache"
echo -e "and triggers rating recalculations, outputting detailed Loki logs."

wait_enter "Ready to trigger the Reviews Storm?"

echo -e "Submitting product reviews..."
review_count=${#customer_ids[@]}
if [ "$review_count" -gt 15 ]; then review_count=15; fi

for ((i=0; i<review_count; i++)); do
    cid="${customer_ids[$i]}"
    body="{\"customer_id\":\"$cid\",\"rating\":5,\"comment\":\"Amazing fast shipping! Highly recommended KazakhExpress product!\"}"
    
    curl -s -X POST -H "Content-Type: application/json" -d "$body" "$API_BASE_URL/products/$product_id/reviews" > /dev/null || true
    echo -e "  Review $i/$review_count submitted by customer $cid"
done

# Fetch recalculated rating
rating_json=$(curl -s "$API_BASE_URL/products/$product_id/rating" || echo "")
cnt=$(echo "$rating_json" | grep -oE '"review_count":[0-9]+' | cut -d':' -f2 || "0")
avg=$(echo "$rating_json" | grep -oE '"average_rating":[0-9.]+' | cut -d':' -f2 || "0.0")
echo -e "\nRecalculated cached rating: Count: $cnt, Average: $avg"

echo -e "\n\033[0;36m*** DEMO PRESENTATION TIP FOR THE PROFESSOR ***\033[0m"
echo -e "Point the professor to the Grafana dashboard:"
echo -e "1. Row: 'Centralized Log Streams (Loki)' -> Expand it at the bottom. You will see colored logs"
echo -e "   from api-gateway, review-service, and product-service cooperating seamlessly!"
echo -e "2. Open your browser and show: http://localhost:5173/ops"
echo -e "   This page displays the real-time operational diagnostics of all microservices."

echo -e "\n\033[0;32m==========================================================================\033[0m"
echo -e "  \033[1;33mDEMO LOAD GENERATOR RUN COMPLETED SUCCESSFULLY!\033[0m"
echo -e "  Open your Ultimate Performance Dashboard at: \033[0;36mhttp://localhost:3000\033[0m"
echo -e "\033[0;32m==========================================================================\033[0m"
