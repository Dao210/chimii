#!/usr/bin/env bash
# Native Linux deployment for Chimii (no Docker).
#
# The Go binaries are cross-compiled locally. The Web workspace is uploaded as
# source and built on the target Linux host so native Next.js dependencies match
# the production OS/architecture. Desktop and mobile are never packaged.
#
# Usage:
#   ./scripts/deploy.sh deploy   # provision, build, migrate, activate, verify
#   ./scripts/deploy.sh verify   # read-only service and HTTP checks
#   TLS_EMAIL=ops@example.com ./scripts/deploy.sh tls
#
# Common overrides:
#   SSH_HOST=root@server SSH_KEY=/path/to/key DOMAIN=example.com ./scripts/deploy.sh

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ACTION="${1:-deploy}"
SSH_HOST="${SSH_HOST:-root@47.90.152.4}"
SSH_KEY="${SSH_KEY:-}"
DOMAIN="${DOMAIN:-chimii.com}"
SERVER_IP="${SERVER_IP:-47.90.152.4}"
REMOTE_ROOT="${REMOTE_ROOT:-/opt/chimii}"
BACKEND_PORT="${BACKEND_PORT:-18080}"
WEB_PORT="${WEB_PORT:-13000}"
PNPM_VERSION="${PNPM_VERSION:-10.28.2}"
KEEP_BUILD_INPUT="${KEEP_BUILD_INPUT:-false}"
ALLOW_SIGNUP="${ALLOW_SIGNUP:-true}"

SSH_ARGS=(
  -o BatchMode=yes
  -o ConnectTimeout=15
  -o ServerAliveInterval=30
  -o ServerAliveCountMax=6
  -o StrictHostKeyChecking=accept-new
)
if [[ -n "$SSH_KEY" ]]; then
  SSH_ARGS+=(-i "$SSH_KEY")
fi

log() { printf '\033[36m[chimii] %s\033[0m\n' "$*"; }
ok() { printf '\033[32m[chimii] ✓ %s\033[0m\n' "$*"; }
warn() { printf '\033[33m[chimii] ! %s\033[0m\n' "$*"; }
die() { printf '\033[31m[chimii] ✗ %s\033[0m\n' "$*" >&2; exit 1; }

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "missing local command: $1"
}

ssh_run() {
  ssh "${SSH_ARGS[@]}" "$SSH_HOST" "$@"
}

scp_push() {
  scp "${SSH_ARGS[@]}" "$@" "$SSH_HOST:$REMOTE_INCOMING/"
}

validate_config() {
  [[ "$DOMAIN" =~ ^[A-Za-z0-9.-]+$ ]] || die "invalid DOMAIN: $DOMAIN"
  [[ "$SERVER_IP" =~ ^[0-9A-Fa-f:.]+$ ]] || die "invalid SERVER_IP: $SERVER_IP"
  [[ "$BACKEND_PORT" =~ ^[0-9]+$ ]] || die "invalid BACKEND_PORT"
  [[ "$WEB_PORT" =~ ^[0-9]+$ ]] || die "invalid WEB_PORT"
  [[ "$REMOTE_ROOT" =~ ^/[A-Za-z0-9._/-]+$ ]] || die "REMOTE_ROOT must be a safe absolute path"
  [[ "$PNPM_VERSION" =~ ^[0-9]+([.][0-9]+){2}$ ]] || die "invalid PNPM_VERSION"
  [[ "$ALLOW_SIGNUP" == true || "$ALLOW_SIGNUP" == false ]] || die "ALLOW_SIGNUP must be true or false"
  [[ "$SSH_HOST" != "sshmd" ]] || die "shell aliases do not expand in scripts; set SSH_HOST to the real SSH destination"
}

preflight() {
  log "preflight $SSH_HOST"
  ssh_run 'set -eu
    test "$(id -u)" -eq 0 || { echo "remote deployment requires root" >&2; exit 1; }
    . /etc/os-release
    case "$ID" in debian|ubuntu) ;; *) echo "unsupported OS: $ID" >&2; exit 1 ;; esac
    test "$(uname -m)" = x86_64 || { echo "target must be x86_64" >&2; exit 1; }
    printf "os=%s %s arch=%s mem_kb=%s disk_free_kb=%s\n" "$ID" "$VERSION_ID" "$(uname -m)" "$(awk '\''/MemTotal/{print $2}'\'' /proc/meminfo)" "$(df -Pk / | awk '\''NR==2{print $4}'\'')"'
  ok "preflight passed"
}

provision_remote() {
  log "provision PostgreSQL 17, pgvector, Redis, Nginx, service user, and low-memory swap"
  ssh_run "DOMAIN='$DOMAIN' REMOTE_ROOT='$REMOTE_ROOT' bash -s" <<'REMOTE'
set -euo pipefail
export DEBIAN_FRONTEND=noninteractive

if ! command -v nginx >/dev/null 2>&1; then
  apt-get update -qq
  apt-get install -y -qq nginx ca-certificates curl gnupg openssl
fi

if ! command -v redis-server >/dev/null 2>&1; then
  apt-get update -qq
  apt-get install -y -qq redis-server
fi

if ! command -v psql >/dev/null 2>&1 || ! psql --version | grep -q ' 17\.' || ! dpkg-query -W postgresql-17-pgvector >/dev/null 2>&1; then
  apt-get update -qq
  apt-get install -y -qq ca-certificates curl gnupg postgresql-common
  install -d -m 0755 /usr/share/postgresql-common/pgdg
  curl -fsSL https://www.postgresql.org/media/keys/ACCC4CF8.asc -o /usr/share/postgresql-common/pgdg/apt.postgresql.org.asc
  . /etc/os-release
  printf 'deb [signed-by=/usr/share/postgresql-common/pgdg/apt.postgresql.org.asc] https://apt.postgresql.org/pub/repos/apt %s-pgdg main\n' "$VERSION_CODENAME" > /etc/apt/sources.list.d/pgdg.list
  apt-get update -qq
  apt-get install -y -qq postgresql-17 postgresql-17-pgvector postgresql-contrib
fi

node_major=0
if command -v node >/dev/null 2>&1; then node_major="$(node -p 'process.versions.node.split(".")[0]')"; fi
if (( node_major < 22 )); then
  echo "Node.js 22+ is required on the Linux host (found: ${node_major:-none})" >&2
  exit 1
fi

sed -i '/# >>> chimii >>>/,/# <<< chimii <<</d' /etc/redis/redis.conf
cat >> /etc/redis/redis.conf <<'CONF'
# >>> chimii >>>
bind 127.0.0.1 -::1
protected-mode yes
port 6379
maxmemory 128mb
maxmemory-policy allkeys-lru
# <<< chimii <<<
CONF

systemctl enable --now postgresql redis-server nginx >/dev/null
systemctl restart redis-server
redis-cli -h 127.0.0.1 ping | grep -Fx PONG >/dev/null
if ! id chimii >/dev/null 2>&1; then
  useradd --system --home-dir /var/lib/chimii --create-home --shell /usr/sbin/nologin chimii
fi
install -d -m 0755 "$REMOTE_ROOT" "$REMOTE_ROOT/incoming" "$REMOTE_ROOT/build" "$REMOTE_ROOT/releases/backend" "$REMOTE_ROOT/releases/web"
install -d -o chimii -g chimii -m 0755 /var/lib/chimii/uploads
install -d -m 0700 /etc/chimii
install -d -m 0755 /var/www/chimii-acme/.well-known/acme-challenge

mem_kb="$(awk '/MemTotal/{print $2}' /proc/meminfo)"
swap_kb="$(awk '/SwapTotal/{print $2}' /proc/meminfo)"
if (( mem_kb < 3145728 && swap_kb < 3145728 )); then
  swap_path=/swapfile-chimii
  if [[ ! -e "$swap_path" ]]; then
    fallocate -l 2G "$swap_path"
    chmod 600 "$swap_path"
    mkswap "$swap_path" >/dev/null
  fi
  swapon --show=NAME --noheadings | grep -Fxq "$swap_path" || swapon "$swap_path"
  grep -Fq "$swap_path none swap sw 0 0" /etc/fstab || printf '%s\n' "$swap_path none swap sw 0 0" >> /etc/fstab
fi

umask 077
if [[ ! -s /etc/chimii/postgres-password ]]; then openssl rand -hex 24 > /etc/chimii/postgres-password; fi
pg_password="$(cat /etc/chimii/postgres-password)"
if ! runuser -u postgres -- psql -Atqc "SELECT 1 FROM pg_roles WHERE rolname='chimii'" | grep -qx 1; then
  runuser -u postgres -- psql -v ON_ERROR_STOP=1 -c "CREATE ROLE chimii LOGIN PASSWORD '$pg_password'"
fi
runuser -u postgres -- psql -v ON_ERROR_STOP=1 -c "ALTER ROLE chimii PASSWORD '$pg_password'" >/dev/null
if ! runuser -u postgres -- psql -Atqc "SELECT 1 FROM pg_database WHERE datname='chimii'" | grep -qx 1; then
  runuser -u postgres -- createdb -O chimii chimii
fi
runuser -u postgres -- psql -v ON_ERROR_STOP=1 -d chimii -c 'CREATE EXTENSION IF NOT EXISTS vector' >/dev/null
runuser -u postgres -- psql -v ON_ERROR_STOP=1 -d chimii -c 'CREATE EXTENSION IF NOT EXISTS pg_trgm' >/dev/null
runuser -u postgres -- psql -v ON_ERROR_STOP=1 -d chimii -c 'CREATE EXTENSION IF NOT EXISTS btree_gin' >/dev/null

ss -lnt | grep -Eq '127\.0\.0\.1:5432|\[::1\]:5432' || { echo "PostgreSQL is not listening on loopback" >&2; exit 1; }
ss -lnt | grep -Eq '127\.0\.0\.1:6379' || { echo "Redis is not listening on loopback" >&2; exit 1; }
REMOTE
  ok "remote prerequisites ready"
}

build_local_inputs() {
  require_cmd go
  require_cmd node
  require_cmd tar
  require_cmd ssh
  require_cmd scp

  VERSION="${VERSION:-$(node -p "require('$ROOT_DIR/package.json').version")}"
  COMMIT="${COMMIT:-$(git -C "$ROOT_DIR" rev-parse --short HEAD 2>/dev/null || printf unknown)}"
  RELEASE_ID="${RELEASE_ID:-$(date -u +%Y%m%d-%H%M%S)-$VERSION}"
  [[ "$VERSION" =~ ^[0-9A-Za-z._+-]+$ ]] || die "invalid VERSION: $VERSION"
  [[ "$RELEASE_ID" =~ ^[0-9A-Za-z._+-]+$ ]] || die "invalid RELEASE_ID: $RELEASE_ID"
  REMOTE_INCOMING="$REMOTE_ROOT/incoming/$RELEASE_ID"
  BUILD_TMP="$(mktemp -d "${TMPDIR:-/tmp}/chimii-deploy.XXXXXX")"
  cleanup_build_tmp() {
    [[ -n "${BUILD_TMP:-}" && -d "$BUILD_TMP" ]] && rm -rf -- "$BUILD_TMP"
  }
  trap cleanup_build_tmp EXIT

  log "cross-compile Go backend for linux/amd64 ($VERSION, $COMMIT)"
  install -d "$BUILD_TMP/backend/migrations"
  (
    cd "$ROOT_DIR/server"
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w -X main.version=$VERSION -X main.commit=$COMMIT" -o "$BUILD_TMP/backend/server" ./cmd/server
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags '-s -w' -o "$BUILD_TMP/backend/migrate" ./cmd/migrate
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags '-s -w' -o "$BUILD_TMP/backend/backfill_task_usage_hourly" ./cmd/backfill_task_usage_hourly
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags '-s -w' -o "$BUILD_TMP/backend/backfill_codex_usage_cache" ./cmd/backfill_codex_usage_cache
  )
  cp "$ROOT_DIR"/server/migrations/*.sql "$BUILD_TMP/backend/migrations/"
  COPYFILE_DISABLE=1 tar --no-xattrs -czf "$BUILD_TMP/backend.tar.gz" -C "$BUILD_TMP" backend

  log "package Web source only (desktop/mobile excluded)"
  COPYFILE_DISABLE=1 tar --no-xattrs -cf "$BUILD_TMP/web-source.tar" \
    --exclude='.DS_Store' --exclude='node_modules' --exclude='.next' \
    --exclude='.turbo' --exclude='*.tsbuildinfo' --exclude='test-results' \
    -C "$ROOT_DIR" \
    package.json pnpm-lock.yaml pnpm-workspace.yaml turbo.json .npmrc \
    apps/web packages/core packages/ui packages/views packages/tsconfig packages/eslint-config

  if [[ -f "$ROOT_DIR/.env" ]]; then
    cp "$ROOT_DIR/.env" "$BUILD_TMP/project.env"
  else
    cp "$ROOT_DIR/.env.example" "$BUILD_TMP/project.env"
  fi
  ok "local deployment inputs built: $RELEASE_ID"
}

upload_inputs() {
  log "upload versioned deployment inputs"
  ssh_run "install -d -m 0755 '$REMOTE_INCOMING'"
  scp_push "$BUILD_TMP/backend.tar.gz" "$BUILD_TMP/web-source.tar" "$BUILD_TMP/project.env"
  ssh_run "sha256sum '$REMOTE_INCOMING/backend.tar.gz' '$REMOTE_INCOMING/web-source.tar' '$REMOTE_INCOMING/project.env'"
  ok "deployment inputs uploaded"
}

build_web_remote() {
  log "install Web dependencies and build standalone on Linux"
  ssh_run "RELEASE_ID='$RELEASE_ID' REMOTE_ROOT='$REMOTE_ROOT' PNPM_VERSION='$PNPM_VERSION' VERSION='$VERSION' bash -s" <<'REMOTE'
set -euo pipefail
incoming="$REMOTE_ROOT/incoming/$RELEASE_ID"
build_dir="$REMOTE_ROOT/build/$RELEASE_ID"
[[ ! -e "$build_dir" ]] || { echo "build directory already exists: $build_dir" >&2; exit 1; }
install -d -m 0755 "$build_dir"
tar -xf "$incoming/web-source.tar" -C "$build_dir"
cd "$build_dir"
export COREPACK_ENABLE_DOWNLOAD_PROMPT=0
corepack pnpm --version | grep -Fx "$PNPM_VERSION"
corepack pnpm install --frozen-lockfile --filter @chimii/web...

unit="chimii-web-build-${RELEASE_ID//./-}"
systemctl reset-failed "$unit.service" 2>/dev/null || true
systemd-run --unit="$unit" \
  --property=Type=exec \
  --property="WorkingDirectory=$build_dir" \
  --property=Nice=5 \
  --property=IOSchedulingClass=best-effort \
  --property=IOSchedulingPriority=6 \
  /usr/bin/env \
    STANDALONE=true \
    NEXT_PUBLIC_APP_VERSION="$VERSION" \
    NEXT_TELEMETRY_DISABLED=1 \
    CHIMII_LOW_MEMORY_BUILD=true \
    NODE_OPTIONS=--max-old-space-size=1536 \
    COREPACK_ENABLE_DOWNLOAD_PROMPT=0 \
    /usr/bin/corepack pnpm --filter @chimii/web build

deadline=$((SECONDS + 5400))
while systemctl is-active --quiet "$unit.service"; do
  (( SECONDS < deadline )) || { systemctl stop "$unit.service"; echo "Web build timed out" >&2; exit 1; }
  sleep 10
done
result="$(systemctl show "$unit.service" -p Result --value)"
status="$(systemctl show "$unit.service" -p ExecMainStatus --value)"
journalctl -u "$unit.service" -n 120 --no-pager
[[ "$result" == success && "$status" == 0 ]] || exit 1
test -f "$build_dir/apps/web/.next/standalone/apps/web/server.js"
REMOTE
  ok "Linux Web build completed"
}

install_and_activate() {
  log "install release, production .env, systemd, Nginx, and migrations"
  ssh_run "RELEASE_ID='$RELEASE_ID' REMOTE_ROOT='$REMOTE_ROOT' DOMAIN='$DOMAIN' BACKEND_PORT='$BACKEND_PORT' WEB_PORT='$WEB_PORT' ALLOW_SIGNUP='$ALLOW_SIGNUP' KEEP_BUILD_INPUT='$KEEP_BUILD_INPUT' bash -s" <<'REMOTE'
set -euo pipefail
incoming="$REMOTE_ROOT/incoming/$RELEASE_ID"
build_dir="$REMOTE_ROOT/build/$RELEASE_ID"
backend_release="$REMOTE_ROOT/releases/backend/$RELEASE_ID"
web_release="$REMOTE_ROOT/releases/web/$RELEASE_ID"
[[ ! -e "$backend_release" && ! -e "$web_release" ]] || { echo "release already exists: $RELEASE_ID" >&2; exit 1; }

install -d -m 0755 "$backend_release" "$web_release"
tar -xzf "$incoming/backend.tar.gz" -C "$backend_release" --strip-components=1
cp -a "$build_dir/apps/web/.next/standalone/." "$web_release/"
install -d -m 0755 "$web_release/apps/web/.next"
cp -a "$build_dir/apps/web/.next/static" "$web_release/apps/web/.next/static"
cp -a "$build_dir/apps/web/public" "$web_release/apps/web/public"
chmod 0755 "$backend_release/server" "$backend_release/migrate" "$backend_release/backfill_task_usage_hourly" "$backend_release/backfill_codex_usage_cache"
chown -R chimii:chimii "$backend_release" "$web_release"
test -x "$backend_release/server"
test -f "$web_release/apps/web/server.js"

umask 077
if [[ ! -s /etc/chimii/jwt-secret ]]; then openssl rand -hex 32 > /etc/chimii/jwt-secret; fi
if [[ ! -s /etc/chimii/vcs-secret ]]; then openssl rand -base64 32 | tr -d '\n' > /etc/chimii/vcs-secret; printf '\n' >> /etc/chimii/vcs-secret; fi
pg_password="$(cat /etc/chimii/postgres-password)"
jwt_secret="$(cat /etc/chimii/jwt-secret)"
vcs_secret="$(cat /etc/chimii/vcs-secret)"
cp "$incoming/project.env" /etc/chimii/backend.env

set_env() {
  local key="$1" value="$2" file="$3" tmp
  tmp="$(mktemp /etc/chimii/env.XXXXXX)"
  grep -v "^${key}=" "$file" > "$tmp" || true
  printf '%s=%s\n' "$key" "$value" >> "$tmp"
  mv "$tmp" "$file"
}
set_env APP_ENV production /etc/chimii/backend.env
set_env CHIMII_DEV_VERIFICATION_CODE '' /etc/chimii/backend.env
set_env POSTGRES_DB chimii /etc/chimii/backend.env
set_env POSTGRES_USER chimii /etc/chimii/backend.env
set_env POSTGRES_PASSWORD "$pg_password" /etc/chimii/backend.env
set_env DATABASE_URL "postgres://chimii:${pg_password}@127.0.0.1:5432/chimii?sslmode=disable" /etc/chimii/backend.env
set_env DATABASE_MAX_CONNS 10 /etc/chimii/backend.env
set_env DATABASE_MIN_CONNS 2 /etc/chimii/backend.env
set_env REDIS_URL redis://127.0.0.1:6379/0 /etc/chimii/backend.env
set_env PORT "$BACKEND_PORT" /etc/chimii/backend.env
set_env CHIMII_BIND_HOST 127.0.0.1 /etc/chimii/backend.env
set_env BACKEND_PORT "$BACKEND_PORT" /etc/chimii/backend.env
set_env FRONTEND_PORT "$WEB_PORT" /etc/chimii/backend.env
set_env FRONTEND_ORIGIN "https://$DOMAIN" /etc/chimii/backend.env
set_env CORS_ALLOWED_ORIGINS "https://$DOMAIN" /etc/chimii/backend.env
set_env CHIMII_APP_URL "https://$DOMAIN" /etc/chimii/backend.env
set_env CHIMII_PUBLIC_URL "https://$DOMAIN" /etc/chimii/backend.env
set_env CHIMII_SERVER_URL "wss://$DOMAIN/ws" /etc/chimii/backend.env
set_env CHIMII_TRUSTED_PROXIES 127.0.0.1/32 /etc/chimii/backend.env
set_env RATE_LIMIT_TRUSTED_PROXIES 127.0.0.1/32 /etc/chimii/backend.env
set_env ALLOW_SIGNUP "$ALLOW_SIGNUP" /etc/chimii/backend.env
set_env COOKIE_DOMAIN '' /etc/chimii/backend.env
set_env JWT_SECRET "$jwt_secret" /etc/chimii/backend.env
set_env LOCAL_UPLOAD_DIR /var/lib/chimii/uploads /etc/chimii/backend.env
set_env LOCAL_UPLOAD_BASE_URL "https://$DOMAIN" /etc/chimii/backend.env
set_env GOOGLE_REDIRECT_URI "https://$DOMAIN/auth/callback" /etc/chimii/backend.env
set_env CHIMII_VCS_INTEGRATION_ENABLED true /etc/chimii/backend.env
set_env CHIMII_VCS_SECRET_KEY "$vcs_secret" /etc/chimii/backend.env
chmod 600 /etc/chimii/backend.env /etc/chimii/jwt-secret /etc/chimii/vcs-secret /etc/chimii/postgres-password
ln -sfn /etc/chimii/backend.env "$REMOTE_ROOT/.env"

{
  printf '%s\n' 'NODE_ENV=production' 'HOSTNAME=127.0.0.1' "PORT=$WEB_PORT" "REMOTE_API_URL=http://127.0.0.1:$BACKEND_PORT" 'NEXT_TELEMETRY_DISABLED=1'
} > /etc/chimii/web.env
chmod 600 /etc/chimii/web.env

cat > /etc/systemd/system/chimii-backend.service <<UNIT
[Unit]
Description=Chimii Backend
Wants=network-online.target
After=network-online.target postgresql.service redis-server.service
Requires=postgresql.service redis-server.service

[Service]
Type=simple
User=chimii
Group=chimii
WorkingDirectory=$REMOTE_ROOT/current-backend
EnvironmentFile=/etc/chimii/backend.env
ExecStart=$REMOTE_ROOT/current-backend/server
Restart=on-failure
RestartSec=5s
TimeoutStopSec=60s
UMask=0027
LimitNOFILE=65535
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=full
ProtectHome=true
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true
RestrictSUIDSGID=true

[Install]
WantedBy=multi-user.target
UNIT

cat > /etc/systemd/system/chimii-web.service <<UNIT
[Unit]
Description=Chimii Web
Wants=network-online.target
After=network-online.target chimii-backend.service
Requires=chimii-backend.service

[Service]
Type=simple
User=chimii
Group=chimii
WorkingDirectory=$REMOTE_ROOT/current-web
EnvironmentFile=/etc/chimii/web.env
ExecStart=/usr/bin/node apps/web/server.js
Restart=on-failure
RestartSec=5s
TimeoutStopSec=45s
UMask=0027
LimitNOFILE=65535
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=full
ProtectHome=true
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true
RestrictSUIDSGID=true

[Install]
WantedBy=multi-user.target
UNIT

cat > /etc/nginx/snippets/chimii-proxy.conf <<NGINX
client_max_body_size 100m;
proxy_set_header Host \$host;
proxy_set_header X-Real-IP \$remote_addr;
proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
proxy_set_header X-Forwarded-Host \$host;
proxy_set_header X-Forwarded-Proto \$scheme;
add_header X-Content-Type-Options nosniff always;
add_header Referrer-Policy strict-origin-when-cross-origin always;
location ^~ /.well-known/acme-challenge/ {
  root /var/www/chimii-acme;
  default_type text/plain;
  try_files \$uri =404;
}
location ~ ^/(health|healthz|readyz)\$ { proxy_pass http://127.0.0.1:$BACKEND_PORT; }
location /api/ {
  proxy_http_version 1.1;
  proxy_connect_timeout 10s;
  proxy_read_timeout 300s;
  proxy_send_timeout 300s;
  proxy_pass http://127.0.0.1:$BACKEND_PORT;
}
location /uploads/ { proxy_pass http://127.0.0.1:$BACKEND_PORT; }
location = /ws {
  proxy_http_version 1.1;
  proxy_set_header Upgrade \$http_upgrade;
  proxy_set_header Connection "upgrade";
  proxy_read_timeout 86400s;
  proxy_send_timeout 86400s;
  proxy_pass http://127.0.0.1:$BACKEND_PORT;
}
location / {
  proxy_http_version 1.1;
  proxy_connect_timeout 10s;
  proxy_read_timeout 300s;
  proxy_pass http://127.0.0.1:$WEB_PORT;
}
NGINX

site=/etc/nginx/sites-available/chimii.com.conf
if [[ -f "/etc/letsencrypt/live/$DOMAIN/fullchain.pem" ]]; then
  cat > "$site" <<NGINX
server {
  listen 80;
  listen [::]:80;
  server_name $DOMAIN;
  location ^~ /.well-known/acme-challenge/ {
    root /var/www/chimii-acme;
    default_type text/plain;
    try_files \$uri =404;
  }
  location / { return 301 https://$DOMAIN\$request_uri; }
}
server {
  listen 443 ssl http2;
  listen [::]:443 ssl http2;
  server_name $DOMAIN;
  ssl_certificate /etc/letsencrypt/live/$DOMAIN/fullchain.pem;
  ssl_certificate_key /etc/letsencrypt/live/$DOMAIN/privkey.pem;
  ssl_protocols TLSv1.2 TLSv1.3;
  ssl_session_timeout 1d;
  ssl_session_cache shared:chimii_tls:10m;
  ssl_session_tickets off;
  include /etc/nginx/snippets/chimii-proxy.conf;
}
NGINX
else
  cat > "$site" <<NGINX
server {
  listen 80;
  listen [::]:80;
  server_name $DOMAIN;
  include /etc/nginx/snippets/chimii-proxy.conf;
}
NGINX
fi
ln -sfn "$site" /etc/nginx/sites-enabled/chimii.com.conf
nginx -t

db_url="$(sed -n 's/^DATABASE_URL=//p' /etc/chimii/backend.env)"
(cd "$backend_release" && runuser -u chimii -- env DATABASE_URL="$db_url" ./migrate up)

ln -sfn "$backend_release" "$REMOTE_ROOT/current-backend"
ln -sfn "$web_release" "$REMOTE_ROOT/current-web"
systemctl daemon-reload
systemctl enable --now chimii-backend.service chimii-web.service >/dev/null
for _ in $(seq 1 45); do curl -fsS "http://127.0.0.1:$BACKEND_PORT/healthz" >/dev/null 2>&1 && break; sleep 1; done
curl -fsS "http://127.0.0.1:$BACKEND_PORT/healthz" >/dev/null
for _ in $(seq 1 45); do curl -fsS "http://127.0.0.1:$WEB_PORT/" >/dev/null 2>&1 && break; sleep 1; done
curl -fsS "http://127.0.0.1:$WEB_PORT/" >/dev/null
systemctl reload nginx
curl -fsS -H "Host: $DOMAIN" http://127.0.0.1/healthz >/dev/null

if [[ "$KEEP_BUILD_INPUT" != true ]]; then
  rm -rf -- "$build_dir" "$incoming"
fi
REMOTE
  ok "release activated: $RELEASE_ID"
}

verify_deployment() {
  log "verify services, listeners, database, and HTTP routes"
  ssh_run "DOMAIN='$DOMAIN' REMOTE_ROOT='$REMOTE_ROOT' BACKEND_PORT='$BACKEND_PORT' WEB_PORT='$WEB_PORT' bash -s" <<'REMOTE'
set -euo pipefail
systemctl is-active --quiet postgresql redis-server nginx chimii-backend chimii-web
systemctl is-enabled --quiet postgresql redis-server nginx chimii-backend chimii-web
curl -fsS "http://127.0.0.1:$BACKEND_PORT/healthz"
curl -fsS -o /dev/null "http://127.0.0.1:$WEB_PORT/"
curl -fsS -o /dev/null -H "Host: $DOMAIN" http://127.0.0.1/api/config
curl -fsS -o /dev/null -H "Host: $DOMAIN" http://127.0.0.1/login
curl -fsS -o /dev/null -H "Host: $DOMAIN" http://127.0.0.1/logo.svg
ss -lnt | grep -Eq "127.0.0.1:$BACKEND_PORT"
ss -lnt | grep -Eq "127.0.0.1:$WEB_PORT"
ss -lnt | grep -Eq '127\.0\.0\.1:6379'
redis-cli -h 127.0.0.1 ping | grep -Fx PONG >/dev/null
runuser -u postgres -- psql -d chimii -Atc "SELECT count(*) FROM schema_migrations" >/dev/null
test "$(stat -c %a /etc/chimii/backend.env)" = 600
test "$(stat -c %a /etc/chimii/web.env)" = 600
printf 'backend=%s\nweb=%s\n' "$(readlink -f "$REMOTE_ROOT/current-backend")" "$(readlink -f "$REMOTE_ROOT/current-web")"
REMOTE

  if curl -fsS --max-time 10 --resolve "$DOMAIN:80:$SERVER_IP" "http://$DOMAIN/healthz" >/dev/null; then
    ok "external HTTP route reaches sshmd"
  else
    warn "external HTTP route failed; check firewall/security group"
  fi
  ok "deployment verification passed"
}

enable_tls() {
  [[ -n "${TLS_EMAIL:-}" ]] || die "TLS_EMAIL is required for the tls action"
  log "issue and install Let's Encrypt certificate for $DOMAIN"
  ssh_run "DOMAIN='$DOMAIN' SERVER_IP='$SERVER_IP' TLS_EMAIL='$TLS_EMAIL' bash -s" <<'REMOTE'
set -euo pipefail
apt-get update -qq
DEBIAN_FRONTEND=noninteractive apt-get install -y -qq certbot

acme_root=/var/www/chimii-acme
install -d -m 0755 "$acme_root/.well-known/acme-challenge"
probe_name="chimii-acme-probe-$$"
probe_value="chimii-acme-ok-$$"
printf '%s' "$probe_value" > "$acme_root/.well-known/acme-challenge/$probe_name"
trap 'rm -f "$acme_root/.well-known/acme-challenge/$probe_name"' EXIT
actual="$(curl -fsSL --max-time 20 -H 'Cache-Control: no-cache' "http://$DOMAIN/.well-known/acme-challenge/$probe_name")"
[[ "$actual" == "$probe_value" ]] || { echo "public ACME route for $DOMAIN does not reach this host" >&2; exit 1; }

certbot certonly --webroot \
  --webroot-path "$acme_root" \
  --non-interactive \
  --agree-tos \
  --no-eff-email \
  --email "$TLS_EMAIL" \
  --preferred-challenges http \
  --key-type ecdsa \
  -d "$DOMAIN"

site=/etc/nginx/sites-available/chimii.com.conf
cat > "$site" <<NGINX
server {
  listen 80;
  listen [::]:80;
  server_name $DOMAIN;
  location ^~ /.well-known/acme-challenge/ {
    root /var/www/chimii-acme;
    default_type text/plain;
    try_files \$uri =404;
  }
  location / { return 301 https://$DOMAIN\$request_uri; }
}
server {
  listen 443 ssl http2;
  listen [::]:443 ssl http2;
  server_name $DOMAIN;
  ssl_certificate /etc/letsencrypt/live/$DOMAIN/fullchain.pem;
  ssl_certificate_key /etc/letsencrypt/live/$DOMAIN/privkey.pem;
  ssl_protocols TLSv1.2 TLSv1.3;
  ssl_session_timeout 1d;
  ssl_session_cache shared:chimii_tls:10m;
  ssl_session_tickets off;
  include /etc/nginx/snippets/chimii-proxy.conf;
}
NGINX

install -d -m 0755 /etc/letsencrypt/renewal-hooks/deploy
cat > /etc/letsencrypt/renewal-hooks/deploy/reload-nginx <<'HOOK'
#!/bin/sh
set -eu
nginx -t
systemctl reload nginx
HOOK
chmod 755 /etc/letsencrypt/renewal-hooks/deploy/reload-nginx
nginx -t
systemctl reload nginx
systemctl enable --now certbot.timer >/dev/null 2>&1 || true
curl -fsS --resolve "$DOMAIN:443:$SERVER_IP" "https://$DOMAIN/healthz" >/dev/null
REMOTE
  ok "HTTPS enabled for $DOMAIN"
}

validate_config
cd "$ROOT_DIR"

case "$ACTION" in
  deploy|all)
    preflight
    provision_remote
    build_local_inputs
    upload_inputs
    build_web_remote
    install_and_activate
    verify_deployment
    ;;
  verify)
    preflight
    verify_deployment
    ;;
  tls)
    preflight
    enable_tls
    ;;
  *)
    die "usage: $0 [deploy|verify|tls]"
    ;;
esac
