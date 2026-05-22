#!/usr/bin/env bash
set -euo pipefail

API_BASE_URL="${API_BASE_URL:-http://localhost:8080}"
GRAFANA_URL="${GRAFANA_URL:-http://localhost:3000}"
NATS_URL="${NATS_URL:-http://localhost:8222}"
SUFFIX="$(date +%s)"

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

wait_for_url "$API_BASE_URL/health" "gateway"
wait_for_url "$API_BASE_URL/payment/health" "payment"
wait_for_url "$API_BASE_URL/products/health" "product"
wait_for_url "$API_BASE_URL/reviews/health" "review"

products="$(curl -fsS "$API_BASE_URL/products")"
product_count="$(printf '%s' "$products" | jq 'length')"
if [ "$product_count" -lt 1 ]; then
  echo "expected seed products" >&2
  exit 1
fi

product_id="$(printf '%s' "$products" | jq -r '.[0].id')"
product_name="$(printf '%s' "$products" | jq -r '.[0].name')"
product_price="$(printf '%s' "$products" | jq -r '.[0].price_kzt')"
image_url="$(printf '%s' "$products" | jq -r '.[0].image_url')"
if [ -n "$image_url" ] && [ "$image_url" != "null" ]; then
  if [ -n "${IMAGE_HOST_ALIAS:-}" ]; then
    image_url="${image_url/localhost/$IMAGE_HOST_ALIAS}"
  fi
  curl -fsSI "$image_url" >/dev/null
fi

order_body="$(jq -n \
  --arg customer_id "usr-smoke-$SUFFIX" \
  --arg product_id "$product_id" \
  --arg name "$product_name" \
  --argjson price "$product_price" \
  '{customer_id:$customer_id, items:[{product_id:$product_id, name:$name, quantity:1, price_kzt:$price}]}')"
order="$(curl -fsS -X POST "$API_BASE_URL/orders" -H 'Content-Type: application/json' -d "$order_body")"
order_id="$(printf '%s' "$order" | jq -r '.id')"
total_kzt="$(printf '%s' "$order" | jq -r '.total_kzt')"

payment_body="$(jq -n \
  --arg order_id "$order_id" \
  --arg customer_id "usr-smoke-$SUFFIX" \
  --arg customer_email "teacher-demo@example.com" \
  --arg method "card" \
  --arg idempotency_key "smoke-$SUFFIX" \
  --argjson amount_kzt "$total_kzt" \
  '{order_id:$order_id, customer_id:$customer_id, customer_email:$customer_email, amount_kzt:$amount_kzt, method:$method, idempotency_key:$idempotency_key}')"
payment_a="$(curl -fsS -X POST "$API_BASE_URL/payment" -H 'Content-Type: application/json' -d "$payment_body")"
payment_b="$(curl -fsS -X POST "$API_BASE_URL/payment" -H 'Content-Type: application/json' -d "$payment_body")"
payment_id="$(printf '%s' "$payment_a" | jq -r '.id')"
payment_id_b="$(printf '%s' "$payment_b" | jq -r '.id')"
payment_status="$(printf '%s' "$payment_a" | jq -r '.status')"
if [ "$payment_id" != "$payment_id_b" ] || [ "$payment_status" != "succeeded" ]; then
  echo "payment idempotency failed" >&2
  exit 1
fi

final_status=""
for _ in $(seq 1 20); do
  final_status="$(curl -fsS "$API_BASE_URL/orders/$order_id" | jq -r '.status')"
  [ "$final_status" = "paid" ] && break
  sleep 1
done
if [ "$final_status" != "paid" ]; then
  echo "order was not marked paid, got $final_status" >&2
  exit 1
fi

review_body="$(jq -n --arg customer_id "usr-smoke-$SUFFIX" '{customer_id:$customer_id, rating:5, comment:"Smoke test review"}')"
curl -fsS -X POST "$API_BASE_URL/products/$product_id/reviews" -H 'Content-Type: application/json' -d "$review_body" >/dev/null
rating_count="$(curl -fsS "$API_BASE_URL/products/$product_id/rating" | jq -r '.review_count')"
if [ "$rating_count" -lt 1 ]; then
  echo "rating was not updated" >&2
  exit 1
fi

refund="$(curl -fsS -X POST "$API_BASE_URL/payment/$payment_id/refund" -H 'Content-Type: application/json' -d '{"reason":"smoke test refund"}')"
refund_status="$(printf '%s' "$refund" | jq -r '.status')"
if [ "$refund_status" != "refunded" ]; then
  echo "refund failed, got $refund_status" >&2
  exit 1
fi

curl -fsS "$NATS_URL/healthz" >/dev/null
if curl -fsS "$GRAFANA_URL/api/health" >/dev/null; then
  dashboards="$(curl -fsS -u admin:admin "$GRAFANA_URL/api/search?query=KazakhExpress" | jq 'length')"
  if [ "$dashboards" -lt 4 ]; then
    echo "expected provisioned dashboards, got $dashboards" >&2
    exit 1
  fi
fi

jq -n \
  --arg order_id "$order_id" \
  --arg payment_id "$payment_id" \
  --arg product_id "$product_id" \
  --arg final_order_status "$final_status" \
  --arg refund_status "$refund_status" \
  --argjson products "$product_count" \
  '{ok:true, products:$products, product_id:$product_id, order_id:$order_id, payment_id:$payment_id, final_order_status:$final_order_status, refund_status:$refund_status}'
