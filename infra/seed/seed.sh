#!/bin/sh
set -eu

API_BASE_URL="${API_BASE_URL:-http://localhost:8080}"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

wait_for_gateway() {
  for _ in $(seq 1 60); do
    if curl -fsS "$API_BASE_URL/health" >/dev/null &&
      curl -fsS "$API_BASE_URL/products/health" >/dev/null &&
      curl -fsS "$API_BASE_URL/reviews/health" >/dev/null; then
      return 0
    fi
    sleep 2
  done
  echo "gateway is not ready" >&2
  return 1
}

json_escape() {
  jq -Rn --arg v "$1" '$v'
}

create_product() {
  name="$1"
  description="$2"
  price="$3"
  stock="$4"
  image_seed="$5"

  existing="$(curl -fsS "$API_BASE_URL/products?q=$(printf '%s' "$name" | sed 's/ /%20/g')" | jq -r --arg name "$name" '.[] | select(.name == $name) | .id' | head -n 1)"
  if [ -n "$existing" ]; then
    echo "$existing"
    return 0
  fi

  body="$(jq -n \
    --arg name "$name" \
    --arg description "$description" \
    --argjson price "$price" \
    --argjson stock "$stock" \
    '{name:$name, description:$description, price_kzt:$price, stock:$stock}')"

  product_id="$(curl -fsS -X POST "$API_BASE_URL/products" \
    -H 'Content-Type: application/json' \
    -d "$body" | jq -r '.id')"

  image_file="$TMP_DIR/$product_id.jpg"
  if ! curl -fsSL "https://picsum.photos/seed/$image_seed/900/700" -o "$image_file"; then
    # Tiny fallback image keeps seed deterministic when the public image endpoint is unavailable.
    printf '%s' 'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/p9sAAAAASUVORK5CYII=' | base64 -d > "$image_file"
  fi
  curl -fsS -X POST "$API_BASE_URL/products/$product_id/images" \
    -F "image=@$image_file;type=image/jpeg" >/dev/null

  echo "$product_id"
}

create_review() {
  product_id="$1"
  customer_id="$2"
  rating="$3"
  comment="$4"
  existing="$(curl -fsS "$API_BASE_URL/products/$product_id/reviews" | jq -r --arg customer "$customer_id" '.[] | select(.customer_id == $customer) | .id' | head -n 1)"
  if [ -n "$existing" ]; then
    return 0
  fi
  body="$(jq -n \
    --arg customer_id "$customer_id" \
    --arg comment "$comment" \
    --argjson rating "$rating" \
    '{customer_id:$customer_id, rating:$rating, comment:$comment}')"
  curl -fsS -X POST "$API_BASE_URL/products/$product_id/reviews" \
    -H 'Content-Type: application/json' \
    -d "$body" >/dev/null
}

wait_for_gateway

phone_id="$(create_product "Kaspi Smart X1" "Fast Android phone with AMOLED screen, NFC, and 256GB storage." 189990 42 "kazakhexpress-phone")"
headphones_id="$(create_product "Nomad ANC Headphones" "Wireless headphones with active noise cancelling and 40 hour battery life." 59990 80 "kazakhexpress-headphones")"
watch_id="$(create_product "Steppe Watch Pro" "Smart watch with heart rate tracking, GPS, and water resistance." 89990 35 "kazakhexpress-watch")"
bag_id="$(create_product "Alatau Travel Backpack" "Durable 28L backpack with laptop pocket and weather resistant fabric." 29990 120 "kazakhexpress-backpack")"

create_review "$phone_id" "usr-seed-aida" 5 "Fast delivery and the screen is bright."
create_review "$phone_id" "usr-seed-arman" 4 "Good phone for the price."
create_review "$headphones_id" "usr-seed-dana" 5 "Noise cancelling works well in the bus."
create_review "$watch_id" "usr-seed-nur" 4 "Battery is solid, setup was simple."
create_review "$bag_id" "usr-seed-saule" 5 "Comfortable backpack with enough space."

curl -fsS "$API_BASE_URL/products" | jq '{products: length}'
echo "seed data ready"
