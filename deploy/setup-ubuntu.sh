#!/usr/bin/env bash
set -euo pipefail

DOMAIN="${DOMAIN:-kazexp.maqsatto.dev}"
APP_DIR="${APP_DIR:-/opt/KazakhExpress}"
EMAIL="${LETSENCRYPT_EMAIL:-admin@maqsatto.dev}"
PUBLIC_IP="${PUBLIC_IP:-}"

if [[ "$(id -u)" -ne 0 ]]; then
  echo "run as root"
  exit 1
fi

apt-get update
apt-get install -y ca-certificates curl git nginx certbot python3-certbot-nginx

if ! command -v docker >/dev/null 2>&1; then
  install -m 0755 -d /etc/apt/keyrings
  curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
  chmod a+r /etc/apt/keyrings/docker.asc
  . /etc/os-release
  echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu ${VERSION_CODENAME} stable" > /etc/apt/sources.list.d/docker.list
  apt-get update
  apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
fi

mkdir -p "$APP_DIR"
cd "$APP_DIR"

if [[ ! -d .git ]]; then
  git clone https://github.com/russ315/KazakhExpress.git .
fi

git fetch origin
git checkout feat/maqsat-commits
git pull --ff-only origin feat/maqsat-commits

if [[ ! -d ../kazakhexpress-proto ]]; then
  git clone https://github.com/maqsatto/kazakhexpress-proto.git ../kazakhexpress-proto
fi
git -C ../kazakhexpress-proto fetch origin
git -C ../kazakhexpress-proto checkout feat/maqsat-commits
git -C ../kazakhexpress-proto pull --ff-only origin feat/maqsat-commits

if [[ ! -f .env ]]; then
  cp .env.example .env
fi

grep -q '^VITE_API_BASE_URL=' .env || echo 'VITE_API_BASE_URL=/api' >> .env
grep -q '^GRAFANA_ROOT_URL=' .env || echo "GRAFANA_ROOT_URL=https://${DOMAIN}/metrics" >> .env
grep -q '^GRAFANA_SERVE_FROM_SUB_PATH=' .env || echo 'GRAFANA_SERVE_FROM_SUB_PATH=true' >> .env

COMPOSE_PARALLEL_LIMIT=1 docker compose up -d --build
docker compose --profile seed run --rm seed-data || true

install -d /etc/nginx/sites-available /etc/nginx/sites-enabled /var/www/certbot
cat >"/etc/nginx/sites-available/${DOMAIN}.conf" <<NGINX
server {
  listen 80;
  server_name ${DOMAIN};

  location /.well-known/acme-challenge/ {
    root /var/www/certbot;
  }

  location /api/ {
    proxy_pass http://127.0.0.1:8080/;
    proxy_http_version 1.1;
    proxy_set_header Host \$host;
    proxy_set_header X-Real-IP \$remote_addr;
    proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto \$scheme;
  }

  location = /metrics {
    return 301 /metrics/;
  }

  location /metrics/ {
    proxy_pass http://127.0.0.1:3000/metrics/;
    proxy_http_version 1.1;
    proxy_set_header Host \$host;
    proxy_set_header X-Real-IP \$remote_addr;
    proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto \$scheme;
  }

  location / {
    proxy_pass http://127.0.0.1:5173;
    proxy_http_version 1.1;
    proxy_set_header Host \$host;
    proxy_set_header X-Real-IP \$remote_addr;
    proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto \$scheme;
  }
}
NGINX
ln -sf "/etc/nginx/sites-available/${DOMAIN}.conf" "/etc/nginx/sites-enabled/${DOMAIN}.conf"
nginx -t
systemctl reload nginx

resolved_ip="$(getent ahostsv4 "$DOMAIN" | awk 'NR==1 {print $1}')"
if [[ -n "$PUBLIC_IP" && "$resolved_ip" != "$PUBLIC_IP" ]]; then
  echo "DNS for ${DOMAIN} resolves to '${resolved_ip:-empty}', expected '${PUBLIC_IP}'. Skipping HTTPS for now."
  docker compose ps
  exit 0
fi

if certbot certificates -d "$DOMAIN" >/dev/null 2>&1; then
  cp deploy/nginx/kazexp.maqsatto.dev.conf "/etc/nginx/sites-available/${DOMAIN}.conf"
  nginx -t
  systemctl reload nginx
else
  certbot certonly --webroot -w /var/www/certbot -d "$DOMAIN" --non-interactive --agree-tos -m "$EMAIL"
  cp deploy/nginx/kazexp.maqsatto.dev.conf "/etc/nginx/sites-available/${DOMAIN}.conf"
  nginx -t
  systemctl reload nginx
fi

docker compose ps
