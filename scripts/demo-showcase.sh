#!/usr/bin/env bash
set -euo pipefail

API_BASE_URL="${API_BASE_URL:-http://localhost:8080}"
GRAFANA_URL="${GRAFANA_URL:-http://localhost:3000}"
INTERACTIVE="${INTERACTIVE:-false}"
SUFFIX="$(date +%s)"

step() {
  echo
  echo "=== $1 ==="
  [ "${2:-}" != "" ] && echo "$2"
  if [ "$INTERACTIVE" = "true" ]; then
    read -r -p "Press Enter for next step"
  fi
}

wait_for_url() {
  local url="$1"
  local name="$2"
  for _ in $(seq 1 60); do
    if curl -fsS "$url" >/dev/null; then
      echo "$name ready"
      return 0
    fi
    sleep 2
  done
  echo "$name is not ready: $url" >&2
  return 1
}

step "0. Start stack" "Run: docker compose up -d --build && docker compose --profile seed run --rm seed-data"
wait_for_url "$API_BASE_URL/health" gateway
wait_for_url "$API_BASE_URL/payment/health" payment
wait_for_url "$API_BASE_URL/products/health" product
wait_for_url "$API_BASE_URL/reviews/health" review
echo "Grafana: $GRAFANA_URL"

step "1. Register user and trigger welcome email" "Watch service logs in Grafana/Loki or docker compose logs smtp-service."
auth="$(jq -n --arg email "demo-$SUFFIX@maqsatto.dev" '{email:$email,password:"Password123!",first_name:"Demo",last_name:"Buyer",phone:"+77010000000",address:"Almaty"}' | curl -fsS -X POST "$API_BASE_URL/auth/register" -H 'Content-Type: application/json' -d @-)"
customer_id="$(printf '%s' "$auth" | jq -r '.user.id')"
customer_email="$(printf '%s' "$auth" | jq -r '.user.email')"
echo "Registered $customer_email as $customer_id"

step "2. Browse catalog" "Open consumer frontend and Grafana Backend Overview."
products="$(curl -fsS "$API_BASE_URL/products")"
product_id="$(printf '%s' "$products" | jq -r '.[0].id')"
product_name="$(printf '%s' "$products" | jq -r '.[0].name')"
product_price="$(printf '%s' "$products" | jq -r '.[0].price_kzt')"
echo "Using product $product_name / $product_id"

step "3. Create paid orders" "This creates HTTP, gRPC, Postgres, NATS and email activity."
payment_ids=()
last_order_id=""
for i in $(seq 1 8); do
  order_body="$(jq -n --arg cid "$customer_id" --arg pid "$product_id" --arg name "$product_name" --argjson price "$product_price" '{customer_id:$cid,items:[{product_id:$pid,name:$name,quantity:1,price_kzt:$price}]}')"
  order="$(curl -fsS -X POST "$API_BASE_URL/orders" -H 'Content-Type: application/json' -d "$order_body")"
  order_id="$(printf '%s' "$order" | jq -r '.id')"
  total="$(printf '%s' "$order" | jq -r '.total_kzt')"
  method="card"; [ $((i % 2)) -eq 0 ] && method="kaspi"
  payment_body="$(jq -n --arg oid "$order_id" --arg cid "$customer_id" --arg email "$customer_email" --arg method "$method" --arg key "demo-$SUFFIX-$i" --argjson amount "$total" '{order_id:$oid,customer_id:$cid,customer_email:$email,amount_kzt:$amount,method:$method,idempotency_key:$key}')"
  payment="$(curl -fsS -X POST "$API_BASE_URL/payment" -H 'Content-Type: application/json' -d "$payment_body")"
  payment_id="$(printf '%s' "$payment" | jq -r '.id')"
  payment_ids+=("$payment_id")
  last_order_id="$order_id"
  echo "Order $order_id -> payment $payment_id"
done

step "4. Prove idempotency" "Same idempotency key returns the same payment."
duplicate_body="$(jq -n --arg oid "$last_order_id" --arg cid "$customer_id" --arg email "$customer_email" --arg key "demo-$SUFFIX-8" --argjson amount "$product_price" '{order_id:$oid,customer_id:$cid,customer_email:$email,amount_kzt:$amount,method:"kaspi",idempotency_key:$key}')"
curl -fsS -X POST "$API_BASE_URL/payment" -H 'Content-Type: application/json' -d "$duplicate_body" | jq '{id,status}'

step "5. Refund and review" "Watch Payment Flow and Catalog Reviews dashboards."
curl -fsS -X POST "$API_BASE_URL/payment/${payment_ids[0]}/refund" -H 'Content-Type: application/json' -d '{"reason":"teacher demo"}' | jq '{id,status}'
for i in $(seq 1 5); do
  jq -n --arg cid "$customer_id" --arg comment "Demo review $i" '{customer_id:$cid,rating:5,comment:$comment}' | curl -fsS -X POST "$API_BASE_URL/products/$product_id/reviews" -H 'Content-Type: application/json' -d @- >/dev/null
done
curl -fsS "$API_BASE_URL/products/$product_id/rating" | jq .

step "6. Generate API load" "This makes request rate and latency visible in Grafana."
for _ in $(seq 1 45); do
  curl -fsS "$API_BASE_URL/health" >/dev/null
  curl -fsS "$API_BASE_URL/products" >/dev/null
done

echo
echo "Demo flow complete. Open Grafana dashboards and filter by service/payment/order/product/review."
