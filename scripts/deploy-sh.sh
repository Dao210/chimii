#!/usr/bin/env bash
# Production deployment for the Tencent Cloud host configured as SSH alias `sh`.
#
# Usage: scripts/deploy-sh.sh [deploy|plan|verify|config|rollback|bootstrap]
# No argument deploys both applications, migrating the existing database in place.
# Cloudflare forwards public HTTPS to Caddy :32443; applications stay on loopback
# :8080/:3000. Integration secrets remain in /etc/chimii/secrets/runtime.env.
# Bootstrap is a guarded, ONE-TIME replacement of the retired installation.

set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=lib/deploy-sh-runtime.sh
source "$ROOT_DIR/scripts/lib/deploy-sh-runtime.sh"
ACTION="${1:-deploy}"

SSH_HOST="${SSH_HOST:-sh}"
EXPECTED_SERVER_IP="${EXPECTED_SERVER_IP:-106.54.235.89}"
REMOTE_ROOT="${REMOTE_ROOT:-/opt/chimii}"
APP_USER="${APP_USER:-chimii}"
DB_NAME="${DB_NAME:-chimii}"
DB_USER="${DB_USER:-chimii_app}"
PRIMARY_DOMAIN="${PRIMARY_DOMAIN:-chimii.com}"
BACKEND_PORT="${BACKEND_PORT:-8080}"
WEB_PORT="${WEB_PORT:-3000}"
CANDIDATE_BACKEND_PORT="${CANDIDATE_BACKEND_PORT:-18080}"
CANDIDATE_WEB_PORT="${CANDIDATE_WEB_PORT:-13000}"
PNPM_VERSION="${PNPM_VERSION:-10.28.2}"
KEEP_RELEASES="${KEEP_RELEASES:-2}"
PUBLIC_ROUTE="${PUBLIC_ROUTE:-true}"
ORIGIN_HTTPS_PORT="${ORIGIN_HTTPS_PORT:-32443}"
TLS_CERT_FILE="${TLS_CERT_FILE:-/etc/caddy/certs/chimii.com.pem}"
TLS_KEY_FILE="${TLS_KEY_FILE:-/etc/caddy/certs/chimii.com.key}"
ALLOW_SIGNUP="${ALLOW_SIGNUP:-true}"
SKIP_LOCAL_CHECKS="${SKIP_LOCAL_CHECKS:-false}"
FORCE_DEPLOY="${FORCE_DEPLOY:-false}"
LDRAW_ARCHIVE="${LDRAW_ARCHIVE:-}"

SSH_ARGS=(
  -o BatchMode=yes
  -o ConnectTimeout=15
  -o ServerAliveInterval=30
  -o ServerAliveCountMax=6
  -o StrictHostKeyChecking=yes
)

BUILD_TMP=""
LOCAL_LOCK_DIR=""
REMOTE_LOCK_HELD=false
REMOTE_LOCK_TOKEN=""
RELEASE_ID=""
VERSION=""
SOURCE_COMMIT=""
REMOTE_UPLOAD=""
CATALOG_RELEASE=""
CATALOG_SHA256=""
CATALOG_PART_COUNT=""
CATALOG_VERSION=""

log() { printf '\033[36m[chimii-sh] %s\033[0m\n' "$*"; }
ok() { printf '\033[32m[chimii-sh] OK %s\033[0m\n' "$*"; }
warn() { printf '\033[33m[chimii-sh] WARN %s\033[0m\n' "$*"; }
die() { printf '\033[31m[chimii-sh] ERROR %s\033[0m\n' "$*" >&2; exit 1; }

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "missing local command: $1"
}

remote_user() {
  ssh "${SSH_ARGS[@]}" "$SSH_HOST" "$@"
}

remote_root() {
  local assignments="${1:-}"
  ssh "${SSH_ARGS[@]}" "$SSH_HOST" "sudo -n env $assignments bash -s"
}

scp_push() {
  scp "${SSH_ARGS[@]}" "$@" "$SSH_HOST:$REMOTE_UPLOAD/"
}

sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

cleanup() {
  local rc=$?
  set +e
  cleanup_candidate_units
  release_remote_lock
  if [[ -n "$BUILD_TMP" && -d "$BUILD_TMP" ]]; then
    rm -rf -- "$BUILD_TMP"
  fi
  if [[ -n "$LOCAL_LOCK_DIR" && -d "$LOCAL_LOCK_DIR" ]]; then
    rmdir "$LOCAL_LOCK_DIR" 2>/dev/null || true
  fi
  return "$rc"
}

cleanup_candidate_units() {
  [[ -n "$RELEASE_ID" && ( "$ACTION" == deploy || "$ACTION" == bootstrap ) ]] || return 0
  remote_root "RELEASE_ID='$RELEASE_ID'" <<'REMOTE' >/dev/null 2>&1 || true
set -eu
systemctl stop "chimii-candidate-web-${RELEASE_ID//./-}.service" "chimii-candidate-backend-${RELEASE_ID//./-}.service" 2>/dev/null || true
REMOTE
  [[ "$ACTION" != deploy ]] || daily_remote cleanup-candidates >/dev/null 2>&1 || true
}

release_remote_lock() {
  [[ "$REMOTE_LOCK_HELD" == true && -n "$REMOTE_LOCK_TOKEN" ]] || return 0
  remote_root "REMOTE_ROOT='$REMOTE_ROOT' LOCK_TOKEN='$REMOTE_LOCK_TOKEN' RELEASE_ID='$RELEASE_ID'" <<'REMOTE' >/dev/null 2>&1 || true
set -eu
lock_dir="$REMOTE_ROOT/.deploy-lock"
if [[ -f "$lock_dir/token" && "$(cat "$lock_dir/token")" == "$LOCK_TOKEN" ]]; then
  unit="chimii-sh-web-build-${RELEASE_ID//./-}.service"
  if [[ -n "$RELEASE_ID" ]] && { systemctl is-active --quiet "$unit" || systemctl is-active --quiet "chimii-sh-cutover-${RELEASE_ID//./-}.service"; }; then
    exit 0
  fi
  # A failed migration requires operator recovery; keep its lock and backup.
  [[ ! -f "$REMOTE_ROOT/state/deployments/$RELEASE_ID/needs-recovery" ]] || exit 0
  rm -f -- "$lock_dir/token" "$lock_dir/started-at"
  rmdir "$lock_dir" 2>/dev/null || true
fi
REMOTE
  REMOTE_LOCK_HELD=false
}

validate_config() {
  [[ "$SSH_HOST" =~ ^[A-Za-z0-9_.@-]+$ ]] || die "invalid SSH_HOST"
  [[ "$EXPECTED_SERVER_IP" =~ ^[0-9A-Fa-f:.]+$ ]] || die "invalid EXPECTED_SERVER_IP"
  [[ "$REMOTE_ROOT" =~ ^/[A-Za-z0-9._/-]+$ ]] || die "invalid REMOTE_ROOT"
  [[ "$REMOTE_ROOT" == /opt/chimii ]] || die "this host-specific entrypoint requires /opt/chimii"
  [[ "$APP_USER" =~ ^[a-z_][a-z0-9_-]*$ ]] || die "invalid APP_USER"
  [[ "$DB_NAME" =~ ^[a-z_][a-z0-9_]*$ ]] || die "invalid DB_NAME"
  [[ "$DB_USER" =~ ^[a-z_][a-z0-9_]*$ ]] || die "invalid DB_USER"
  [[ "$PRIMARY_DOMAIN" =~ ^[A-Za-z0-9.-]+$ ]] || die "invalid PRIMARY_DOMAIN"
  [[ "$BACKEND_PORT" =~ ^[0-9]+$ ]] || die "invalid BACKEND_PORT"
  [[ "$WEB_PORT" =~ ^[0-9]+$ ]] || die "invalid WEB_PORT"
  [[ "$CANDIDATE_BACKEND_PORT" =~ ^[0-9]+$ ]] || die "invalid CANDIDATE_BACKEND_PORT"
  [[ "$CANDIDATE_WEB_PORT" =~ ^[0-9]+$ ]] || die "invalid CANDIDATE_WEB_PORT"
  local deploy_port
  for deploy_port in "$BACKEND_PORT" "$WEB_PORT" "$CANDIDATE_BACKEND_PORT" "$CANDIDATE_WEB_PORT" "$ORIGIN_HTTPS_PORT"; do
    [[ "$deploy_port" =~ ^[1-9][0-9]{0,4}$ ]] && (( deploy_port <= 65535 )) || die "invalid port"
  done
  [[ "$(printf '%s\n' "$BACKEND_PORT" "$WEB_PORT" "$CANDIDATE_BACKEND_PORT" "$CANDIDATE_WEB_PORT" "$ORIGIN_HTTPS_PORT" | sort -u | wc -l | tr -d ' ')" == 5 ]] || die "deployment ports must be distinct"
  [[ "$TLS_CERT_FILE" =~ ^/etc/caddy/certs/[A-Za-z0-9._-]+$ ]] || die "invalid TLS_CERT_FILE"
  [[ "$TLS_KEY_FILE" =~ ^/etc/caddy/certs/[A-Za-z0-9._-]+$ ]] || die "invalid TLS_KEY_FILE"
  [[ "$PNPM_VERSION" =~ ^[0-9]+([.][0-9]+){2}$ ]] || die "invalid PNPM_VERSION"
  [[ "$KEEP_RELEASES" =~ ^[1-9][0-9]*$ ]] || die "invalid KEEP_RELEASES"
  [[ "$PUBLIC_ROUTE" == true || "$PUBLIC_ROUTE" == false ]] || die "PUBLIC_ROUTE must be true or false"
  [[ "$ALLOW_SIGNUP" == true || "$ALLOW_SIGNUP" == false ]] || die "ALLOW_SIGNUP must be true or false"
  [[ "$SKIP_LOCAL_CHECKS" == true || "$SKIP_LOCAL_CHECKS" == false ]] || die "SKIP_LOCAL_CHECKS must be true or false"
  [[ "$FORCE_DEPLOY" == true || "$FORCE_DEPLOY" == false ]] || die "FORCE_DEPLOY must be true or false"
  [[ -z "$LDRAW_ARCHIVE" || "$LDRAW_ARCHIVE" == /* ]] || die "LDRAW_ARCHIVE must be an absolute local path"
  [[ "$BACKEND_PORT" != "$CANDIDATE_BACKEND_PORT" ]] || die "candidate backend port must differ"
  [[ "$WEB_PORT" != "$CANDIDATE_WEB_PORT" ]] || die "candidate Web port must differ"
}

render_caddy_block() {
  [[ "$PUBLIC_ROUTE" == true ]] || return 0
  sh_render_caddy
}

daily_remote() {
  local remote_action="$1" assignments="" setting value
  # Only validated, non-secret deployment coordinates cross the SSH command line.
  for setting in REMOTE_ROOT RELEASE_ID VERSION SOURCE_COMMIT APP_USER DB_NAME DB_USER PRIMARY_DOMAIN BACKEND_PORT WEB_PORT CANDIDATE_BACKEND_PORT CANDIDATE_WEB_PORT ORIGIN_HTTPS_PORT TLS_CERT_FILE TLS_KEY_FILE KEEP_RELEASES; do
    printf -v value '%q' "${!setting}"
    assignments+="$setting=$value "
  done
  printf -v value '%q' "$REMOTE_LOCK_TOKEN"
  remote_root "$assignments LOCK_TOKEN=$value SH_REMOTE_ACTION=$remote_action" < "$ROOT_DIR/scripts/lib/deploy-sh-runtime.sh"
}

verify_target() {
  local resolved
  require_cmd ssh
  resolved="$(ssh -G "$SSH_HOST" 2>/dev/null | awk '$1 == "hostname" {print $2; exit}')"
  [[ "$resolved" == "$EXPECTED_SERVER_IP" ]] || die "SSH alias $SSH_HOST resolves to $resolved, expected $EXPECTED_SERVER_IP"
  remote_user 'set -eu
    test "$(hostname)" = VM-0-8-ubuntu
    test "$(uname -m)" = x86_64
    . /etc/os-release
    test "$ID" = ubuntu
    test "$VERSION_ID" = 24.04
    sudo -n true
    printf "host=%s os=%s arch=%s\n" "$(hostname)" "$PRETTY_NAME" "$(uname -m)"'
}

acquire_local_lock() {
  local lock_name
  lock_name="${SSH_HOST//[^A-Za-z0-9_.-]/_}"
  LOCAL_LOCK_DIR="${TMPDIR:-/tmp}/chimii-sh-deploy-$lock_name.lock"
  mkdir "$LOCAL_LOCK_DIR" 2>/dev/null || die "another deployment holds $LOCAL_LOCK_DIR"
}

acquire_remote_lock() {
  local token="$1"
  [[ "$token" =~ ^[0-9A-Za-z._-]+$ ]] || die "invalid remote lock token"
  remote_root "REMOTE_ROOT='$REMOTE_ROOT' LOCK_TOKEN='$token'" <<'REMOTE'
set -euo pipefail
lock_dir="$REMOTE_ROOT/.deploy-lock"
now="$(date +%s)"
if ! mkdir "$lock_dir" 2>/dev/null; then
  if [[ -f "$lock_dir/token" && "$(cat "$lock_dir/token")" == "$LOCK_TOKEN" ]]; then
    echo "resuming deployment lock for $LOCK_TOKEN"
    exit 0
  fi
  started=0
  [[ -f "$lock_dir/started-at" ]] && started="$(cat "$lock_dir/started-at")"
  if [[ "$started" =~ ^[0-9]+$ ]] && (( now - started > 7200 )); then
    echo "stale deployment lock detected; inspect running deployment/build processes before removing $lock_dir" >&2
  else
    echo "another deployment holds $lock_dir (token: $(cat "$lock_dir/token" 2>/dev/null || echo unknown))" >&2
  fi
  exit 1
fi
printf '%s\n' "$LOCK_TOKEN" > "$lock_dir/token"
printf '%s\n' "$now" > "$lock_dir/started-at"
chmod 600 "$lock_dir/token" "$lock_dir/started-at"
REMOTE
  REMOTE_LOCK_TOKEN="$token"
  REMOTE_LOCK_HELD=true
}

init_release() {
  require_cmd git
  require_cmd node
  local catalog_lock="$ROOT_DIR/server/internal/ldrawsync/catalog.lock.json"
  local catalog_manifest="$ROOT_DIR/server/internal/ldrawsync/starter-kit-1000.json"
  SOURCE_COMMIT="$(git -C "$ROOT_DIR" rev-parse HEAD)"
  VERSION="$(git -C "$ROOT_DIR" describe --tags --exact-match HEAD 2>/dev/null || true)"
  [[ -n "$VERSION" && "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]] || die "production deployment requires HEAD at an exact vX.Y.Z tag"
  [[ "$(git -C "$ROOT_DIR" branch --show-current)" == main ]] || die "production deployment requires branch main"
  [[ -z "$(git -C "$ROOT_DIR" status --porcelain)" ]] || die "production deployment requires a clean worktree"
  [[ "$VERSION" == "v$(node -p 'require("./package.json").version')" ]] || die "release tag and root package version must match"
  CATALOG_RELEASE="$(node -e 'const x=require(process.argv[1]); process.stdout.write(x.release)' "$catalog_lock")"
  CATALOG_SHA256="$(node -e 'const x=require(process.argv[1]); process.stdout.write(x.archive_sha256)' "$catalog_lock")"
  CATALOG_PART_COUNT="$(node -e 'const x=require(process.argv[1]); process.stdout.write(String(x.part_count))' "$catalog_manifest")"
  [[ "$CATALOG_RELEASE" =~ ^[0-9]{4}-[0-9]{2}(-[0-9]{2})?$ ]] || die "invalid embedded LDraw release"
  [[ "$CATALOG_SHA256" =~ ^[0-9a-f]{64}$ ]] || die "invalid embedded LDraw SHA-256"
  [[ "$CATALOG_PART_COUNT" =~ ^[1-9][0-9]*$ ]] || die "invalid embedded LDraw part count"
  CATALOG_VERSION="ldraw-official-$CATALOG_RELEASE-${CATALOG_SHA256:0:12}"
  if [[ -n "$LDRAW_ARCHIVE" ]]; then
    [[ -f "$LDRAW_ARCHIVE" ]] || die "LDRAW_ARCHIVE does not exist: $LDRAW_ARCHIVE"
    [[ "$(sha256_file "$LDRAW_ARCHIVE")" = "$CATALOG_SHA256" ]] || die "LDRAW_ARCHIVE does not match the embedded catalog SHA-256"
  fi
  RELEASE_ID="${RELEASE_ID_OVERRIDE:-$(date -u +%Y%m%dT%H%M%SZ)-${VERSION#v}-${SOURCE_COMMIT:0:12}}"
  [[ "$RELEASE_ID" =~ ^[0-9A-Za-z._-]+$ ]] || die "invalid release id"
  REMOTE_UPLOAD="/home/ubuntu/.cache/chimii-deploy/$RELEASE_ID"
}

build_backend() {
  require_cmd go
  require_cmd tar
  require_cmd gzip
  BUILD_TMP="$(mktemp -d "${TMPDIR:-/tmp}/chimii-sh-deploy.XXXXXX")"
  install -d "$BUILD_TMP/backend/migrations"
  log "cross-compiling Linux amd64 binaries"
  (
    cd "$ROOT_DIR/server"
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w -X main.version=$VERSION -X main.commit=${SOURCE_COMMIT:0:12}" -o "$BUILD_TMP/backend/server" ./cmd/server &
    p1=$!
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o "$BUILD_TMP/backend/migrate" ./cmd/migrate &
    p2=$!
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w -X main.version=$VERSION -X main.commit=${SOURCE_COMMIT:0:12}" -o "$BUILD_TMP/backend/chimii" ./cmd/chimii &
    p3=$!
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o "$BUILD_TMP/backend/ldraw_catalog_sync" ./cmd/ldraw_catalog_sync &
    p4=$!
    status=0
    wait "$p1" || status=$?
    wait "$p2" || status=$?
    wait "$p3" || status=$?
    wait "$p4" || status=$?
    (( status == 0 ))
  )
  cp "$ROOT_DIR"/server/migrations/*.sql "$BUILD_TMP/backend/migrations/"
  printf 'release=%s\ncommit=%s\nbuilt_at=%s\n' "$RELEASE_ID" "$SOURCE_COMMIT" "$(date -u +%Y-%m-%dT%H:%M:%SZ)" > "$BUILD_TMP/backend/manifest"
  COPYFILE_DISABLE=1 tar --no-xattrs -cf "$BUILD_TMP/backend.tar" -C "$BUILD_TMP" backend
  gzip -n "$BUILD_TMP/backend.tar"
}

package_web() {
  require_cmd corepack
  if [[ "$SKIP_LOCAL_CHECKS" == false ]]; then
    log "running Web workspace typecheck"
    (cd "$ROOT_DIR" && corepack pnpm exec turbo typecheck --filter=@chimii/web...)
  fi
  log "packaging Web source"
  COPYFILE_DISABLE=1 tar --no-xattrs -cf "$BUILD_TMP/web-source.tar" \
    --exclude='.env' --exclude='.env.*' --exclude='.envrc' \
    --exclude='.DS_Store' --exclude='node_modules' --exclude='.next' \
    --exclude='.turbo' --exclude='*.tsbuildinfo' --exclude='test-results' \
    -C "$ROOT_DIR" \
    package.json pnpm-lock.yaml pnpm-workspace.yaml turbo.json .npmrc \
    apps/web packages/core packages/ui packages/views packages/tsconfig packages/eslint-config
  gzip -n "$BUILD_TMP/web-source.tar"
}

upload_artifacts() {
  local backend_sha web_sha runtime_sha ldraw_sha=""
  backend_sha="$(sha256_file "$BUILD_TMP/backend.tar.gz")"
  web_sha="$(sha256_file "$BUILD_TMP/web-source.tar.gz")"
  runtime_sha="$(sha256_file "$ROOT_DIR/scripts/lib/deploy-sh-runtime.sh")"
  [[ -z "$LDRAW_ARCHIVE" ]] || ldraw_sha="$CATALOG_SHA256"
  remote_user "install -d -m 0700 '$REMOTE_UPLOAD'"
  scp_push "$BUILD_TMP/backend.tar.gz" "$BUILD_TMP/web-source.tar.gz" "$ROOT_DIR/scripts/lib/deploy-sh-runtime.sh"
  if [[ -n "$LDRAW_ARCHIVE" ]]; then
    scp "${SSH_ARGS[@]}" "$LDRAW_ARCHIVE" "$SSH_HOST:$REMOTE_UPLOAD/ldraw-complete.zip"
  fi
  remote_root "REMOTE_UPLOAD='$REMOTE_UPLOAD' REMOTE_ROOT='$REMOTE_ROOT' RELEASE_ID='$RELEASE_ID' BACKEND_SHA='$backend_sha' WEB_SHA='$web_sha' LDRAW_SHA='$ldraw_sha' RUNTIME_SHA='$runtime_sha'" <<'REMOTE'
set -euo pipefail
incoming="$REMOTE_ROOT/incoming/$RELEASE_ID"
install -d -m 0700 "$incoming"
test "$(sha256sum "$REMOTE_UPLOAD/backend.tar.gz" | awk '{print $1}')" = "$BACKEND_SHA"
test "$(sha256sum "$REMOTE_UPLOAD/web-source.tar.gz" | awk '{print $1}')" = "$WEB_SHA"
install -m 0600 "$REMOTE_UPLOAD/backend.tar.gz" "$incoming/backend.tar.gz"
install -m 0600 "$REMOTE_UPLOAD/web-source.tar.gz" "$incoming/web-source.tar.gz"
test "$(sha256sum "$REMOTE_UPLOAD/deploy-sh-runtime.sh" | awk '{print $1}')" = "$RUNTIME_SHA"
install -m 0700 "$REMOTE_UPLOAD/deploy-sh-runtime.sh" "$incoming/deploy-sh-runtime.sh"
if [[ -n "$LDRAW_SHA" ]]; then
  test "$(sha256sum "$REMOTE_UPLOAD/ldraw-complete.zip" | awk '{print $1}')" = "$LDRAW_SHA"
  install -m 0600 "$REMOTE_UPLOAD/ldraw-complete.zip" "$incoming/ldraw-complete.zip"
fi
rm -rf -- "$REMOTE_UPLOAD"
REMOTE
}

setup_remote() {
  remote_root "REMOTE_ROOT='$REMOTE_ROOT' APP_USER='$APP_USER' DB_USER='$DB_USER'" <<'REMOTE'
set -euo pipefail
export DEBIAN_FRONTEND=noninteractive
test "$(uname -m)" = x86_64
available_kb="$(df -Pk / | awk 'NR==2 {print $4}')"
(( available_kb >= 8 * 1024 * 1024 )) || { echo "at least 8 GiB free disk is required" >&2; exit 1; }

if ! command -v redis-server >/dev/null 2>&1; then
  apt-get update -qq
  apt-get install -y -qq redis-server
fi
marker='# >>> chimii-sh >>>'
if ! grep -Fq "$marker" /etc/redis/redis.conf; then
  cat >> /etc/redis/redis.conf <<'CONF'
# >>> chimii-sh >>>
bind 127.0.0.1 -::1
protected-mode yes
port 6379
maxmemory 128mb
maxmemory-policy allkeys-lru
# <<< chimii-sh <<<
CONF
fi
systemctl enable --now redis-server >/dev/null
redis-cli -h 127.0.0.1 ping | grep -Fx PONG >/dev/null

if ! id "$APP_USER" >/dev/null 2>&1; then
  useradd --system --home-dir /var/lib/chimii --create-home --shell /usr/sbin/nologin "$APP_USER"
fi
install -d -m 0755 "$REMOTE_ROOT" "$REMOTE_ROOT/releases" "$REMOTE_ROOT/releases/backend" "$REMOTE_ROOT/releases/web" "$REMOTE_ROOT/releases/manifests"
install -d -m 0700 "$REMOTE_ROOT/incoming" "$REMOTE_ROOT/state" "$REMOTE_ROOT/state/legacy"
install -d -o ubuntu -g ubuntu -m 0755 "$REMOTE_ROOT/builder" "$REMOTE_ROOT/builder/workspace" "$REMOTE_ROOT/builder/status"
install -d -o "$APP_USER" -g "$APP_USER" -m 0750 /var/lib/chimii /var/lib/chimii/uploads
install -d -o "$APP_USER" -g "$APP_USER" -m 0750 "$REMOTE_ROOT/cache" "$REMOTE_ROOT/cache/ldraw"
install -d -m 0700 /etc/chimii

umask 077
test -s /etc/chimii/v2-postgres-password || openssl rand -hex 24 > /etc/chimii/v2-postgres-password
test -s /etc/chimii/v2-jwt-secret || openssl rand -hex 32 > /etc/chimii/v2-jwt-secret
test -s /etc/chimii/v2-vcs-secret || openssl rand -hex 32 > /etc/chimii/v2-vcs-secret
db_password="$(cat /etc/chimii/v2-postgres-password)"
if ! runuser -u postgres -- psql -d postgres -X -Atqc "SELECT 1 FROM pg_roles WHERE rolname='$DB_USER'" | grep -qx 1; then
  runuser -u postgres -- psql -d postgres -X -v ON_ERROR_STOP=1 -c "CREATE ROLE $DB_USER LOGIN PASSWORD '$db_password'" >/dev/null
else
  runuser -u postgres -- psql -d postgres -X -v ON_ERROR_STOP=1 -c "ALTER ROLE $DB_USER PASSWORD '$db_password'" >/dev/null
fi
chmod 600 /etc/chimii/v2-postgres-password /etc/chimii/v2-jwt-secret /etc/chimii/v2-vcs-secret
REMOTE
}

install_backend_release() {
  remote_root "REMOTE_ROOT='$REMOTE_ROOT' RELEASE_ID='$RELEASE_ID' APP_USER='$APP_USER'" <<'REMOTE'
set -euo pipefail
incoming="$REMOTE_ROOT/incoming/$RELEASE_ID"
release="$REMOTE_ROOT/releases/backend/$RELEASE_ID"
if [[ ! -d "$release" ]]; then
  install -d -m 0755 "$release"
  tar -xzf "$incoming/backend.tar.gz" -C "$release" --strip-components=1
fi
for binary in server migrate chimii ldraw_catalog_sync; do
  test -f "$release/$binary"
  chmod 0755 "$release/$binary"
done
chown -R "$APP_USER:$APP_USER" "$release"
REMOTE
}

build_web_remote() {
  remote_root "REMOTE_ROOT='$REMOTE_ROOT' RELEASE_ID='$RELEASE_ID' PNPM_VERSION='$PNPM_VERSION' VERSION='$VERSION'" <<'REMOTE'
set -euo pipefail
incoming="$REMOTE_ROOT/incoming/$RELEASE_ID"
workspace="$REMOTE_ROOT/builder/workspace"
status_dir="$REMOTE_ROOT/builder/status"
runner="$REMOTE_ROOT/builder/run-web-build"
status_file="$status_dir/$RELEASE_ID.exit-code"
unit="chimii-sh-web-build-${RELEASE_ID//./-}"

if [[ -f "$status_file" && "$(cat "$status_file")" = 0 && -f "$workspace/apps/web/.next/standalone/apps/web/server.js" ]]; then
  exit 0
fi

rm -rf -- "$workspace/apps/web" "$workspace/packages/core" "$workspace/packages/ui" "$workspace/packages/views" "$workspace/packages/tsconfig" "$workspace/packages/eslint-config" "$workspace/.next-cache-hold"
rm -f -- "$workspace/package.json" "$workspace/pnpm-lock.yaml" "$workspace/pnpm-workspace.yaml" "$workspace/turbo.json" "$workspace/.npmrc"
tar -xzf "$incoming/web-source.tar.gz" -C "$workspace"
chown -R ubuntu:ubuntu "$workspace" "$status_dir"

cat > "$runner" <<'RUNNER'
#!/usr/bin/env bash
set -uo pipefail
release_id="$1"
version="$2"
remote_root="$3"
workspace="$remote_root/builder/workspace"
status_file="$remote_root/builder/status/$release_id.exit-code"
tmp="$status_file.tmp"
set +e
cd "$workspace"
export COREPACK_ENABLE_DOWNLOAD_PROMPT=0
corepack pnpm --version | grep -Fx '10.28.2' && \
corepack pnpm install --frozen-lockfile --prefer-offline --filter @chimii/web... && \
/usr/bin/env STANDALONE=true NEXT_PUBLIC_APP_VERSION="$version" NEXT_TELEMETRY_DISABLED=1 CHIMII_LOW_MEMORY_BUILD=true CHIMII_DEPLOY_TYPECHECKED=true NODE_OPTIONS=--max-old-space-size=1536 COREPACK_ENABLE_DOWNLOAD_PROMPT=0 corepack pnpm --filter @chimii/web build
rc=$?
printf '%s\n' "$rc" > "$tmp"
mv "$tmp" "$status_file"
exit "$rc"
RUNNER
sed -i "s/grep -Fx '10.28.2'/grep -Fx '$PNPM_VERSION'/" "$runner"
chmod 0755 "$runner"
chown ubuntu:ubuntu "$runner"
rm -f -- "$status_file" "$status_file.tmp"
systemctl reset-failed "$unit.service" 2>/dev/null || true
systemd-run --quiet --unit="$unit" --property=Type=exec --property=User=ubuntu --property="WorkingDirectory=$workspace" --property=Nice=5 --property=IOSchedulingClass=best-effort --property=IOSchedulingPriority=6 "$runner" "$RELEASE_ID" "$VERSION" "$REMOTE_ROOT"
REMOTE

  local deadline status failures=0
  deadline=$(( $(date +%s) + 5400 ))
  while (( $(date +%s) < deadline )); do
    status="$(remote_user "sudo -n bash -c 'file=\"$REMOTE_ROOT/builder/status/$RELEASE_ID.exit-code\"; unit=\"chimii-sh-web-build-${RELEASE_ID//./-}.service\"; if test -f \"\$file\"; then printf done:; cat \"\$file\"; elif systemctl is-active --quiet \"\$unit\"; then echo running; else echo lost; fi'" 2>/dev/null || true)"
    case "$status" in
      done:0) break ;;
      done:*) remote_user "sudo -n journalctl -u 'chimii-sh-web-build-${RELEASE_ID//./-}.service' -n 160 --no-pager"; die "remote Web build failed" ;;
      running) failures=0 ;;
      lost) failures=$((failures + 1)); (( failures < 3 )) || die "remote Web build unit disappeared" ;;
    esac
    sleep 30
  done
  (( $(date +%s) < deadline )) || die "remote Web build timed out"

  remote_root "REMOTE_ROOT='$REMOTE_ROOT' RELEASE_ID='$RELEASE_ID' APP_USER='$APP_USER'" <<'REMOTE'
set -euo pipefail
workspace="$REMOTE_ROOT/builder/workspace"
release="$REMOTE_ROOT/releases/web/$RELEASE_ID"
test -f "$workspace/apps/web/.next/standalone/apps/web/server.js"
if [[ ! -d "$release" ]]; then
  install -d -m 0755 "$release"
  cp -a "$workspace/apps/web/.next/standalone/." "$release/"
  install -d -m 0755 "$release/apps/web/.next"
  cp -a "$workspace/apps/web/.next/static" "$release/apps/web/.next/static"
  cp -a "$workspace/apps/web/public" "$release/apps/web/public"
  printf 'release=%s\n' "$RELEASE_ID" > "$release/manifest"
fi
chown -R "$APP_USER:$APP_USER" "$release"
REMOTE
}

prepare_candidate() {
  local candidate_db="chimii_candidate_$(printf '%s' "$RELEASE_ID" | cut -c1-16 | tr -cd '0-9')"
  [[ "$candidate_db" =~ ^[a-z_][a-z0-9_]*$ ]] || die "invalid candidate database name"
  remote_root "DEPLOY_KIND='$ACTION' REMOTE_ROOT='$REMOTE_ROOT' RELEASE_ID='$RELEASE_ID' VERSION='$VERSION' APP_USER='$APP_USER' DB_NAME='$DB_NAME' DB_USER='$DB_USER' CANDIDATE_DB='$candidate_db' PRIMARY_DOMAIN='$PRIMARY_DOMAIN' CANDIDATE_BACKEND_PORT='$CANDIDATE_BACKEND_PORT' CANDIDATE_WEB_PORT='$CANDIDATE_WEB_PORT' ALLOW_SIGNUP='$ALLOW_SIGNUP' CATALOG_RELEASE='$CATALOG_RELEASE' CATALOG_SHA256='$CATALOG_SHA256' CATALOG_PART_COUNT='$CATALOG_PART_COUNT' CATALOG_VERSION='$CATALOG_VERSION'" <<'REMOTE'
set -euo pipefail
backend="$REMOTE_ROOT/releases/backend/$RELEASE_ID"
web="$REMOTE_ROOT/releases/web/$RELEASE_ID"
db_password="$(cat /etc/chimii/v2-postgres-password)"
jwt_secret="$(cat /etc/chimii/v2-jwt-secret)"
vcs_secret="$(cat /etc/chimii/v2-vcs-secret)"

if ! runuser -u postgres -- psql -d postgres -X -Atqc "SELECT 1 FROM pg_database WHERE datname='$CANDIDATE_DB'" | grep -qx 1; then
  runuser -u postgres -- createdb -O "$DB_USER" "$CANDIDATE_DB"
fi
candidate_url="postgres://$DB_USER:$db_password@127.0.0.1:5432/$CANDIDATE_DB?sslmode=disable"

umask 077
# systemd-run's EnvironmentFile property takes one filename. Merge server-only
# secrets first, then override all production routing and storage coordinates.
if [[ -f /etc/chimii/secrets/runtime.env ]]; then
  cp /etc/chimii/secrets/runtime.env /etc/chimii/v2-candidate-backend.env
  printf '\n' >> /etc/chimii/v2-candidate-backend.env
else
  : > /etc/chimii/v2-candidate-backend.env
fi
cat >> /etc/chimii/v2-candidate-backend.env <<EOF
APP_ENV=production
DATABASE_URL=$candidate_url
DATABASE_MAX_CONNS=10
DATABASE_MIN_CONNS=2
REDIS_URL=
PORT=$CANDIDATE_BACKEND_PORT
CHIMII_BIND_HOST=127.0.0.1
BACKEND_PORT=$CANDIDATE_BACKEND_PORT
FRONTEND_PORT=$CANDIDATE_WEB_PORT
FRONTEND_ORIGIN=https://$PRIMARY_DOMAIN
CORS_ALLOWED_ORIGINS=https://$PRIMARY_DOMAIN
CHIMII_APP_URL=https://$PRIMARY_DOMAIN
CHIMII_PUBLIC_URL=https://$PRIMARY_DOMAIN
CHIMII_SERVER_URL=wss://$PRIMARY_DOMAIN/ws
CHIMII_TRUSTED_PROXIES=127.0.0.1/32
RATE_LIMIT_TRUSTED_PROXIES=127.0.0.1/32
ALLOW_SIGNUP=$ALLOW_SIGNUP
COOKIE_DOMAIN=
JWT_SECRET=$jwt_secret
LOCAL_UPLOAD_DIR=/var/lib/chimii/candidate-uploads
LOCAL_UPLOAD_BASE_URL=https://$PRIMARY_DOMAIN
GOOGLE_REDIRECT_URI=https://$PRIMARY_DOMAIN/auth/callback
CHIMII_VCS_INTEGRATION_ENABLED=true
CHIMII_LDRAW_CATALOG_SYNC_ENABLED=false
CHIMII_VCS_SECRET_KEY=$vcs_secret
EOF
cat > /etc/chimii/v2-candidate-web.env <<EOF
NODE_ENV=production
HOSTNAME=127.0.0.1
PORT=$CANDIDATE_WEB_PORT
REMOTE_API_URL=http://127.0.0.1:$CANDIDATE_BACKEND_PORT
NEXT_TELEMETRY_DISABLED=1
EOF
chmod 600 /etc/chimii/v2-candidate-backend.env /etc/chimii/v2-candidate-web.env

runuser -u "$APP_USER" -- env DATABASE_URL="$candidate_url" "$backend/migrate" up

catalog_fingerprint() {
  local database="$1"
  runuser -u postgres -- psql -d "$database" -X -Atqc "
COPY (
  SELECT row_kind || E'\\t' || row_data
  FROM (
    SELECT '1-release' AS row_kind,
           concat_ws(E'\\t', id, catalog_version, release, source_url, archive_sha256, status,
                     COALESCE(release_json, '<NULL>'), part_count) AS row_data
      FROM ldraw_catalog_release
    UNION ALL
    SELECT '2-revision',
           concat_ws(E'\\t', id, catalog_version, part_id, revision, kind, ldraw_sha256,
                     storage_backend, COALESCE(content_sha256, '<NULL>'), content_type, storage_key,
                     COALESCE(encode(digest(payload, 'sha256'), 'hex'), '<NULL>'),
                     payload_size_bytes, payload_format)
      FROM ldraw_part_revision
    UNION ALL
    SELECT '3-definition',
           concat_ws(E'\\t', part_key, name, category, popularity_rank, certification_level,
                     auto_build_eligible, geometry_profile, studs_x, studs_z, plates_y,
                     default_quantity, has_top_studs, has_bottom_receptors)
      FROM part_definition
    UNION ALL
    SELECT '4-catalog-revision',
           concat_ws(E'\\t', catalog_version, part_key, ldraw_part_id, ldraw_sha256,
                     COALESCE(content_sha256, '<NULL>'), semantic_version, origin_y_offset_ldu,
                     origin_center_z_offset_ldu, bounds_json, connections_json, occupancy_json)
      FROM part_catalog_revision
    UNION ALL
    SELECT '5-kit',
           concat_ws(E'\\t', kit_id, version, name, description, source_manifest_kit_id,
                     catalog_version, part_count, status)
      FROM kit_profile
    UNION ALL
    SELECT '6-kit-part',
           concat_ws(E'\\t', kit_id, kit_version, catalog_version, part_key,
                     popularity_rank, default_quantity, enabled)
      FROM kit_profile_part
  ) AS rows
  ORDER BY row_kind, row_data
) TO STDOUT
" | sha256sum | awk '{print $1}'
}

sync_args=()
uploaded_archive="$REMOTE_ROOT/incoming/$RELEASE_ID/ldraw-complete.zip"
if [[ -f "$uploaded_archive" ]]; then
  cached_archive="$REMOTE_ROOT/cache/ldraw/$CATALOG_SHA256.zip"
  install -o "$APP_USER" -g "$APP_USER" -m 0400 "$uploaded_archive" "$cached_archive"
  sync_args+=(--archive "$cached_archive")
fi
if [[ "$DEPLOY_KIND" == deploy ]] || ! runuser -u "$APP_USER" -- env DATABASE_URL="$candidate_url" "$backend/ldraw_catalog_sync" "${sync_args[@]}"; then
  echo "validating and copying the exact pinned catalog from $DB_NAME" >&2
  expected_counts="1|$CATALOG_PART_COUNT|$CATALOG_PART_COUNT|$CATALOG_PART_COUNT|1|100|0|0"
  source_counts="$(runuser -u postgres -- psql -d "$DB_NAME" -X -Atqc "
SELECT
  (SELECT count(*) FROM ldraw_catalog_release WHERE catalog_version='$CATALOG_VERSION' AND release='$CATALOG_RELEASE' AND archive_sha256='$CATALOG_SHA256' AND status='active') || '|' ||
  (SELECT count(*) FROM ldraw_part_revision WHERE catalog_version='$CATALOG_VERSION') || '|' ||
  (SELECT count(*) FROM part_definition) || '|' ||
  (SELECT count(*) FROM part_catalog_revision WHERE catalog_version='$CATALOG_VERSION') || '|' ||
  (SELECT count(*) FROM kit_profile WHERE kit_id='chimii-starter-100' AND catalog_version='$CATALOG_VERSION' AND status='active' AND part_count=100) || '|' ||
  (SELECT count(*) FROM kit_profile_part WHERE kit_id='chimii-starter-100' AND catalog_version='$CATALOG_VERSION' AND enabled) || '|' ||
  (SELECT count(*) FROM ldraw_part_revision WHERE catalog_version='$CATALOG_VERSION' AND payload IS NULL) || '|' ||
  (SELECT count(*) FROM ldraw_part_revision WHERE catalog_version='$CATALOG_VERSION' AND encode(digest(payload, 'sha256'), 'hex') IS DISTINCT FROM content_sha256)
")"
  [[ "$source_counts" = "$expected_counts" ]] || {
    echo "legacy catalog validation failed: got $source_counts, expected $expected_counts" >&2
    exit 1
  }
  source_fingerprint="$(catalog_fingerprint "$DB_NAME")"
  transfer="$REMOTE_ROOT/state/catalog-transfer-$RELEASE_ID.dump"
  runuser -u postgres -- pg_dump -Fc --data-only --no-owner -d "$DB_NAME" \
    -t ldraw_catalog_release -t ldraw_part_revision -t part_definition \
    -t part_catalog_revision -t kit_profile -t kit_profile_part > "$transfer"
  runuser -u postgres -- psql -d "$CANDIDATE_DB" -X -v ON_ERROR_STOP=1 -c \
    'TRUNCATE ldraw_catalog_release, ldraw_part_revision, part_definition, part_catalog_revision, kit_profile, kit_profile_part' >/dev/null
  runuser -u postgres -- pg_restore --exit-on-error --data-only --no-owner -d "$CANDIDATE_DB" < "$transfer"
  rm -f -- "$transfer"
  candidate_fingerprint="$(catalog_fingerprint "$CANDIDATE_DB")"
  [[ "$candidate_fingerprint" = "$source_fingerprint" ]] || {
    echo "catalog fingerprint mismatch after candidate import" >&2
    exit 1
  }
fi

expected="$(mktemp)"
applied="$(mktemp)"
trap 'rm -f "$expected" "$applied"' EXIT
find "$backend/migrations" -maxdepth 1 -type f -name '*.up.sql' -printf '%f\n' | sed 's/\.up\.sql$//' | sort > "$expected"
runuser -u postgres -- psql -d "$CANDIDATE_DB" -X -Atqc 'SELECT version FROM schema_migrations ORDER BY version' > "$applied"
diff -u "$expected" "$applied"

printf 'RELEASE_ID=%s\nCANDIDATE_DB=%s\n' "$RELEASE_ID" "$CANDIDATE_DB" > "$REMOTE_ROOT/state/candidate.env"
chmod 600 "$REMOTE_ROOT/state/candidate.env"

backend_unit="chimii-candidate-backend-${RELEASE_ID//./-}"
web_unit="chimii-candidate-web-${RELEASE_ID//./-}"
systemctl stop "$backend_unit.service" "$web_unit.service" 2>/dev/null || true
systemctl reset-failed "$backend_unit.service" "$web_unit.service" 2>/dev/null || true
install -d -o "$APP_USER" -g "$APP_USER" -m 0700 /var/lib/chimii/candidate-uploads
systemd-run --quiet --unit="$backend_unit" --property=Type=simple --property="User=$APP_USER" --property="Group=$APP_USER" --property="WorkingDirectory=$backend" --property=EnvironmentFile=/etc/chimii/v2-candidate-backend.env --property=IPAddressDeny=any --property=IPAddressAllow=localhost "$backend/server"
for _ in $(seq 1 90); do curl -fsS "http://127.0.0.1:$CANDIDATE_BACKEND_PORT/readyz" >/dev/null 2>&1 && break; sleep 1; done
curl -fsS "http://127.0.0.1:$CANDIDATE_BACKEND_PORT/readyz"

systemd-run --quiet --unit="$web_unit" --property=Type=simple --property="User=$APP_USER" --property="Group=$APP_USER" --property="WorkingDirectory=$web" --property=EnvironmentFile=/etc/chimii/v2-candidate-web.env /usr/bin/node apps/web/server.js
for _ in $(seq 1 90); do curl -fsS "http://127.0.0.1:$CANDIDATE_WEB_PORT/" >/dev/null 2>&1 && break; sleep 1; done
curl -fsS -o /dev/null "http://127.0.0.1:$CANDIDATE_WEB_PORT/"
curl -fsS "http://127.0.0.1:$CANDIDATE_BACKEND_PORT/api/config" | grep -F "\"server_version\":\"$VERSION\"" >/dev/null
runuser -u postgres -- psql -d "$CANDIDATE_DB" -X -Atqc "SELECT count(*) FROM ldraw_catalog_release WHERE status='active'" | grep -Fx 1 >/dev/null
runuser -u postgres -- psql -d "$CANDIDATE_DB" -X -Atqc "SELECT count(*) FROM kit_profile WHERE status='active' AND kit_id='chimii-starter-100' AND part_count=100" | grep -Fx 1 >/dev/null
REMOTE
  ok "candidate release and database verified"
}

write_final_units_and_env() {
  remote_root "REMOTE_ROOT='$REMOTE_ROOT' LOCK_TOKEN='$REMOTE_LOCK_TOKEN' APP_USER='$APP_USER' DB_NAME='$DB_NAME' DB_USER='$DB_USER' PRIMARY_DOMAIN='$PRIMARY_DOMAIN' BACKEND_PORT='$BACKEND_PORT' WEB_PORT='$WEB_PORT' ALLOW_SIGNUP='$ALLOW_SIGNUP'" <<'REMOTE'
set -euo pipefail
test "$(cat "$REMOTE_ROOT/.deploy-lock/token")" = "$LOCK_TOKEN"
db_password="$(cat /etc/chimii/v2-postgres-password)"
jwt_secret="$(cat /etc/chimii/v2-jwt-secret)"
vcs_secret="$(cat /etc/chimii/v2-vcs-secret)"
umask 077
cat > /etc/chimii/backend.env <<EOF
APP_ENV=production
DATABASE_URL=postgres://$DB_USER:$db_password@127.0.0.1:5432/$DB_NAME?sslmode=disable
DATABASE_MAX_CONNS=10
DATABASE_MIN_CONNS=2
REDIS_URL=redis://127.0.0.1:6379/0
PORT=$BACKEND_PORT
CHIMII_BIND_HOST=127.0.0.1
BACKEND_PORT=$BACKEND_PORT
FRONTEND_PORT=$WEB_PORT
FRONTEND_ORIGIN=https://$PRIMARY_DOMAIN
CORS_ALLOWED_ORIGINS=https://$PRIMARY_DOMAIN
CHIMII_APP_URL=https://$PRIMARY_DOMAIN
CHIMII_PUBLIC_URL=https://$PRIMARY_DOMAIN
CHIMII_SERVER_URL=wss://$PRIMARY_DOMAIN/ws
CHIMII_TRUSTED_PROXIES=127.0.0.1/32
RATE_LIMIT_TRUSTED_PROXIES=127.0.0.1/32
ALLOW_SIGNUP=$ALLOW_SIGNUP
COOKIE_DOMAIN=
JWT_SECRET=$jwt_secret
LOCAL_UPLOAD_DIR=/var/lib/chimii/uploads
LOCAL_UPLOAD_BASE_URL=https://$PRIMARY_DOMAIN
GOOGLE_REDIRECT_URI=https://$PRIMARY_DOMAIN/auth/callback
CHIMII_VCS_INTEGRATION_ENABLED=true
CHIMII_LDRAW_CATALOG_SYNC_ENABLED=true
CHIMII_VCS_SECRET_KEY=$vcs_secret
EOF
cat > /etc/chimii/web.env <<EOF
NODE_ENV=production
HOSTNAME=127.0.0.1
PORT=$WEB_PORT
REMOTE_API_URL=http://127.0.0.1:$BACKEND_PORT
NEXT_TELEMETRY_DISABLED=1
EOF
chmod 600 /etc/chimii/backend.env /etc/chimii/web.env

cat > /etc/systemd/system/chimii-backend.service <<EOF
[Unit]
Description=Chimii Backend
Wants=network-online.target
After=network-online.target postgresql.service redis-server.service
Requires=postgresql.service redis-server.service

[Service]
Type=simple
User=$APP_USER
Group=$APP_USER
WorkingDirectory=$REMOTE_ROOT/current-backend
EnvironmentFile=/etc/chimii/backend.env
EnvironmentFile=-/etc/chimii/secrets/runtime.env
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
EOF
cat > /etc/systemd/system/chimii-web.service <<EOF
[Unit]
Description=Chimii Web
Wants=network-online.target
After=network-online.target chimii-backend.service
Requires=chimii-backend.service

[Service]
Type=simple
User=$APP_USER
Group=$APP_USER
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
EOF
systemctl daemon-reload
REMOTE
}

backup_legacy() {
  remote_root "REMOTE_ROOT='$REMOTE_ROOT' LOCK_TOKEN='$REMOTE_LOCK_TOKEN' RELEASE_ID='$RELEASE_ID' DB_NAME='$DB_NAME'" <<'REMOTE'
set -euo pipefail
test "$(cat "$REMOTE_ROOT/.deploy-lock/token")" = "$LOCK_TOKEN"
backup="$REMOTE_ROOT/state/legacy/$RELEASE_ID"
if [[ -f "$backup/backup.complete" ]]; then exit 0; fi
install -d -m 0700 "$backup" "$backup/systemd"
runuser -u postgres -- pg_dump -Fc -d "$DB_NAME" > "$backup/$DB_NAME.dump"
pg_restore --list "$backup/$DB_NAME.dump" >/dev/null
sha256sum "$backup/$DB_NAME.dump" > "$backup/$DB_NAME.dump.sha256"
cp -a /etc/chimii "$backup/etc-chimii"
cp -a /etc/caddy/Caddyfile "$backup/Caddyfile"
for unit in chimii-backend.service chimii-web.service chimii-daemon.service chimii-public-sales-runtime.service chimii-public-sales-egress.service; do
  [[ -f "/etc/systemd/system/$unit" ]] && cp -a "/etc/systemd/system/$unit" "$backup/systemd/$unit"
  systemctl is-active "$unit" > "$backup/systemd/$unit.active" 2>/dev/null || true
  systemctl is-enabled "$unit" > "$backup/systemd/$unit.enabled" 2>/dev/null || true
done
for link in current current-backend current-web server-current web-current daemon-current chimii-cli; do
  if [[ -L "$REMOTE_ROOT/$link" ]]; then
    readlink -f "$REMOTE_ROOT/$link" > "$backup/$link.link"
  elif [[ ! -e "$REMOTE_ROOT/$link" ]]; then
    touch "$backup/$link.absent"
  fi
done
if [[ -f "$REMOTE_ROOT/.env" && ! -L "$REMOTE_ROOT/.env" ]]; then
  cp -a "$REMOTE_ROOT/.env" "$backup/root.env"
fi
touch "$backup/backup.complete"
chmod -R go-rwx "$backup"
REMOTE
  ok "legacy database, configuration, units, and release pointers backed up"
}

configure_caddy() {
  local apply_caddy="${1:-true}"
  [[ "$apply_caddy" == true || "$apply_caddy" == false ]] || die "invalid Caddy apply mode"
  [[ "$PUBLIC_ROUTE" == true ]] || die "this production entrypoint requires PUBLIC_ROUTE=true"
  if [[ "$apply_caddy" == true ]]; then daily_remote config-apply; else daily_remote config-prepare; fi
}

stop_retired_services() {
  remote_root "REMOTE_ROOT='$REMOTE_ROOT' LOCK_TOKEN='$REMOTE_LOCK_TOKEN' RELEASE_ID='$RELEASE_ID'" <<'REMOTE'
set -euo pipefail
test "$(cat "$REMOTE_ROOT/.deploy-lock/token")" = "$LOCK_TOKEN"
candidate_backend="chimii-candidate-backend-${RELEASE_ID//./-}.service"
candidate_web="chimii-candidate-web-${RELEASE_ID//./-}.service"
systemctl stop "$candidate_web" "$candidate_backend" 2>/dev/null || true
systemctl stop chimii-daemon.service chimii-public-sales-runtime.service chimii-public-sales-egress.service chimii-web.service chimii-backend.service 2>/dev/null || true
systemctl disable chimii-daemon.service chimii-public-sales-runtime.service chimii-public-sales-egress.service >/dev/null 2>&1 || true

while IFS= read -r pid; do
  [[ "$pid" =~ ^[0-9]+$ ]] || continue
  cwd="$(readlink "/proc/$pid/cwd" 2>/dev/null || true)"
  cwd="${cwd% (deleted)}"
  process_pwd="$(tr '\0' '\n' < "/proc/$pid/environ" 2>/dev/null | sed -n 's/^PWD=//p' | head -n 1 || true)"
  if [[ "$cwd" == "$REMOTE_ROOT"/* || "$process_pwd" == "$REMOTE_ROOT"/* ]]; then
    kill -TERM "$pid"
    for _ in $(seq 1 20); do kill -0 "$pid" 2>/dev/null || break; sleep 1; done
    kill -0 "$pid" 2>/dev/null && kill -KILL "$pid" || true
  else
    echo "refusing to stop unexpected listener on retired port 3100: pid=$pid cwd=$cwd pwd=$process_pwd" >&2
    exit 1
  fi
done < <(ss -ltnp | sed -nE 's/.*127\.0\.0\.1:3100.*pid=([0-9]+).*/\1/p')
if ss -lnt | grep -Eq '127\.0\.0\.1:(3100|9443)'; then
  echo 'retired listener remains on port 3100 or 9443' >&2
  exit 1
fi
REMOTE
}

switch_databases() {
  remote_root "REMOTE_ROOT='$REMOTE_ROOT' LOCK_TOKEN='$REMOTE_LOCK_TOKEN' RELEASE_ID='$RELEASE_ID' DB_NAME='$DB_NAME'" <<'REMOTE'
set -euo pipefail
test "$(cat "$REMOTE_ROOT/.deploy-lock/token")" = "$LOCK_TOKEN"
expected_release_id="$RELEASE_ID"
. "$REMOTE_ROOT/state/candidate.env"
[[ "$expected_release_id" = "$RELEASE_ID" ]]
legacy_db="chimii_legacy_$(printf '%s' "$RELEASE_ID" | cut -c1-16 | tr -cd '0-9')"
[[ "$CANDIDATE_DB" =~ ^chimii_candidate_[0-9]+$ ]]
[[ "$legacy_db" =~ ^chimii_legacy_[0-9]+$ ]]
printf 'LEGACY_DB=%s\nCANDIDATE_DB=%s\nNEW_DB=%s\nSWITCH_COMPLETE=false\n' "$legacy_db" "$CANDIDATE_DB" "$DB_NAME" > "$REMOTE_ROOT/state/legacy/$RELEASE_ID/databases.env"
chmod 600 "$REMOTE_ROOT/state/legacy/$RELEASE_ID/databases.env"
runuser -u postgres -- psql -d postgres -X -v ON_ERROR_STOP=1 -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname IN ('$DB_NAME', '$CANDIDATE_DB') AND pid <> pg_backend_pid()" >/dev/null
runuser -u postgres -- psql -d postgres -X -v ON_ERROR_STOP=1 -c "ALTER DATABASE $DB_NAME RENAME TO $legacy_db"
runuser -u postgres -- psql -d postgres -X -v ON_ERROR_STOP=1 -c "ALTER DATABASE $CANDIDATE_DB RENAME TO $DB_NAME"
sed -i 's/^SWITCH_COMPLETE=false$/SWITCH_COMPLETE=true/' "$REMOTE_ROOT/state/legacy/$RELEASE_ID/databases.env"
REMOTE
}

activate_release() {
  remote_root "REMOTE_ROOT='$REMOTE_ROOT' LOCK_TOKEN='$REMOTE_LOCK_TOKEN' RELEASE_ID='$RELEASE_ID' VERSION='$VERSION' SOURCE_COMMIT='$SOURCE_COMMIT' KEEP_RELEASES='$KEEP_RELEASES' BACKEND_PORT='$BACKEND_PORT' WEB_PORT='$WEB_PORT'" <<'REMOTE'
set -euo pipefail
test "$(cat "$REMOTE_ROOT/.deploy-lock/token")" = "$LOCK_TOKEN"
backend="$REMOTE_ROOT/releases/backend/$RELEASE_ID"
web="$REMOTE_ROOT/releases/web/$RELEASE_ID"
test -x "$backend/server"
test -f "$web/apps/web/server.js"
old_backend="$(readlink -f "$REMOTE_ROOT/current-backend" 2>/dev/null || true)"
old_web="$(readlink -f "$REMOTE_ROOT/current-web" 2>/dev/null || true)"
[[ -n "$old_backend" ]] && printf '%s\n' "$old_backend" > "$REMOTE_ROOT/state/previous-backend"
[[ -n "$old_web" ]] && printf '%s\n' "$old_web" > "$REMOTE_ROOT/state/previous-web"
ln -sfn "$backend" "$REMOTE_ROOT/current-backend"
ln -sfn "$web" "$REMOTE_ROOT/current-web"
for legacy_link in current server-current web-current daemon-current chimii-cli backend-current; do
  [[ -L "$REMOTE_ROOT/$legacy_link" ]] && rm -f -- "$REMOTE_ROOT/$legacy_link"
done
rm -f "$REMOTE_ROOT/.env"
ln -s /etc/chimii/backend.env "$REMOTE_ROOT/.env"
printf 'RELEASE_ID=%s\nVERSION=%s\nSOURCE_COMMIT=%s\nBACKEND=%s\nWEB=%s\nDEPLOYED_AT=%s\n' "$RELEASE_ID" "$VERSION" "$SOURCE_COMMIT" "$backend" "$web" "$(date -u +%Y-%m-%dT%H:%M:%SZ)" > "$REMOTE_ROOT/state/current.env"
chmod 600 "$REMOTE_ROOT/state/current.env"
cp "$REMOTE_ROOT/state/current.env" "$REMOTE_ROOT/releases/manifests/$RELEASE_ID.env"
systemctl enable chimii-backend.service chimii-web.service >/dev/null
systemctl restart chimii-backend.service
for _ in $(seq 1 90); do curl -fsS "http://127.0.0.1:$BACKEND_PORT/readyz" >/dev/null 2>&1 && break; sleep 1; done
curl -fsS "http://127.0.0.1:$BACKEND_PORT/readyz" >/dev/null
systemctl restart chimii-web.service
for _ in $(seq 1 90); do curl -fsS "http://127.0.0.1:$WEB_PORT/" >/dev/null 2>&1 && break; sleep 1; done
curl -fsS "http://127.0.0.1:$WEB_PORT/" >/dev/null
rm -rf -- "$REMOTE_ROOT/incoming/$RELEASE_ID"
REMOTE
}

verify_deployment() {
  remote_root "REMOTE_ROOT='$REMOTE_ROOT' DB_NAME='$DB_NAME' BACKEND_PORT='$BACKEND_PORT' WEB_PORT='$WEB_PORT' PUBLIC_ROUTE='$PUBLIC_ROUTE' PRIMARY_DOMAIN='$PRIMARY_DOMAIN'" <<'REMOTE'
set -euo pipefail
systemctl is-active --quiet postgresql redis-server caddy chimii-backend chimii-web
systemctl is-enabled --quiet postgresql redis-server caddy chimii-backend chimii-web
curl -fsS "http://127.0.0.1:$BACKEND_PORT/readyz" | grep -F '"status":"ok"' >/dev/null
curl -fsS -o /dev/null "http://127.0.0.1:$WEB_PORT/"
expected_version="$(sed -n 's/^VERSION=//p' "$REMOTE_ROOT/state/current.env")"
[[ "$expected_version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]
curl -fsS "http://127.0.0.1:$BACKEND_PORT/api/config" | grep -F "\"server_version\":\"$expected_version\"" >/dev/null
ss -lnt | grep -Eq "127.0.0.1:$BACKEND_PORT"
ss -lnt | grep -Eq "127.0.0.1:$WEB_PORT"
ss -lnt | grep -Eq '127\.0\.0\.1:6379'
if ss -lnt | grep -Eq '127\.0\.0\.1:(3100|9443)'; then
  echo 'retired listener remains on port 3100 or 9443' >&2
  exit 1
fi
redis-cli -h 127.0.0.1 ping | grep -Fx PONG >/dev/null
runuser -u postgres -- psql -d "$DB_NAME" -X -Atqc "SELECT count(*) FROM ldraw_catalog_release WHERE status='active'" | grep -Fx 1 >/dev/null
runuser -u postgres -- psql -d "$DB_NAME" -X -Atqc "SELECT count(*) FROM kit_profile WHERE status='active' AND part_count=100" | grep -Fx 1 >/dev/null
test "$(stat -c %a /etc/chimii/backend.env)" = 600
test "$(stat -c %a /etc/chimii/web.env)" = 600
caddy validate --config /etc/caddy/Caddyfile >/dev/null
if [[ "$PUBLIC_ROUTE" = false ]]; then
  if grep -Eq '^[[:space:]]*(chimii\.com|ai\.52tuan\.com|aurocreator\.com|www\.aurocreator\.com)[[:space:]]*([,:{]|$)' /etc/caddy/Caddyfile; then
    echo 'an unfiled or retired public route is still present in Caddy' >&2
    exit 1
  fi
fi
printf 'release=%s\nversion=%s\nbackend=%s\nweb=%s\ndatabase=%s\npublic_route=%s\n' \
  "$(sed -n 's/^RELEASE_ID=//p' "$REMOTE_ROOT/state/current.env")" \
  "$expected_version" \
  "$(readlink -f "$REMOTE_ROOT/current-backend")" \
  "$(readlink -f "$REMOTE_ROOT/current-web")" "$DB_NAME" "$PUBLIC_ROUTE"
REMOTE
  daily_remote origin-check
  daily_remote firewall-check
  local checked_version
  checked_version="$(remote_user "sudo -n sed -n 's/^VERSION=//p' '$REMOTE_ROOT/state/current.env'")"
  VERSION="$checked_version" daily_remote public-check
}

deploy_daily() {
  init_release
  daily_remote preflight
  acquire_remote_lock "$RELEASE_ID"
  local deployed_commit
  deployed_commit="$(remote_user "sudo -n sed -n 's/^SOURCE_COMMIT=//p' '$REMOTE_ROOT/state/current.env'")"
  if [[ "$deployed_commit" == "$SOURCE_COMMIT" && "$FORCE_DEPLOY" == false ]]; then
    verify_deployment
    ok "this commit is already deployed; verification passed"
    return 0
  fi
  if [[ "$SKIP_LOCAL_CHECKS" == false ]]; then
    bash "$ROOT_DIR/scripts/deploy-sh.test.sh"
    bash "$ROOT_DIR/scripts/test-go.sh"
  fi
  build_backend
  package_web
  upload_artifacts
  install_backend_release
  build_web_remote
  daily_remote rehearse
  # Smoke-test an empty business database, so cloned jobs cannot contact providers.
  prepare_candidate
  cleanup_candidate_units
  daily_remote start-cutover
  local job_state deadline
  deadline=$(( $(date +%s) + 3600 ))
  while (( $(date +%s) < deadline )); do
    job_state="$(remote_user "sudo -n bash -c 'file=$REMOTE_ROOT/state/deployments/$RELEASE_ID/exit-code; if test -f \"\$file\"; then printf done:; cat \"\$file\"; elif systemctl is-active --quiet chimii-sh-cutover-${RELEASE_ID//./-}.service; then echo running; else echo unknown; fi'" 2>/dev/null || true)"
    case "$job_state" in
      done:0) break ;;
      done:*) die "cutover failed; inspect $REMOTE_ROOT/state/deployments/$RELEASE_ID and journalctl -u chimii-sh-cutover-${RELEASE_ID//./-}" ;;
      *) log "cutover $RELEASE_ID: ${job_state:-SSH unavailable}; server continues independently" ;;
    esac
    sleep 15
  done
  [[ "$job_state" == done:0 ]] || die "cutover status uncertain; server retains the lock until it finishes"
  verify_deployment
  daily_remote prune
  ok "deployed $VERSION to https://$PRIMARY_DOMAIN (origin $ORIGIN_HTTPS_PORT)"
}

replace_legacy() {
  [[ "${CONFIRM_REPLACE_LEGACY:-}" == replace-chimii ]] || die "set CONFIRM_REPLACE_LEGACY=replace-chimii for the one-time replacement"
  remote_user "sudo -n test ! -f '$REMOTE_ROOT/state/current.env' -a ! -f '$REMOTE_ROOT/state/bootstrap.complete'" || die "bootstrap already completed; use scripts/deploy-sh.sh for daily deployments"
  init_release
  acquire_remote_lock "$RELEASE_ID"
  build_backend
  package_web
  upload_artifacts
  setup_remote
  install_backend_release
  build_web_remote
  prepare_candidate
  configure_caddy false
  backup_legacy
  local rc=0
  set +e
  stop_retired_services || rc=$?
  (( rc != 0 )) || switch_databases || rc=$?
  (( rc != 0 )) || write_final_units_and_env || rc=$?
  (( rc != 0 )) || activate_release || rc=$?
  (( rc != 0 )) || configure_caddy || rc=$?
  (( rc != 0 )) || verify_deployment || rc=$?
  set -e
  if (( rc != 0 )); then
    warn "cutover failed; restoring the legacy database and services"
    rollback_legacy "$RELEASE_ID" || warn "automatic legacy rollback also failed; inspect $REMOTE_ROOT/state/legacy/$RELEASE_ID"
    return "$rc"
  fi
  ok "legacy installation replaced; rollback snapshot retained under $REMOTE_ROOT/state/legacy/$RELEASE_ID"
  remote_user "sudo -n touch '$REMOTE_ROOT/state/bootstrap.complete'"
}

rollback_legacy() {
  local target_release="${1:-}"
  [[ "$target_release" =~ ^[0-9A-Za-z._-]+$ ]] || die "a safe legacy release id is required"
  remote_root "REMOTE_ROOT='$REMOTE_ROOT' RELEASE_ID='$target_release' DB_NAME='$DB_NAME'" <<'REMOTE'
set -euo pipefail
backup="$REMOTE_ROOT/state/legacy/$RELEASE_ID"
test -f "$backup/backup.complete"
systemctl stop chimii-web.service chimii-backend.service 2>/dev/null || true

if [[ -f "$backup/databases.env" ]]; then
  . "$backup/databases.env"
  [[ "$LEGACY_DB" =~ ^chimii_legacy_[0-9]+$ ]]
  [[ "$CANDIDATE_DB" =~ ^chimii_candidate_[0-9]+$ ]]
  failed_db="chimii_failed_$(date -u +%Y%m%dT%H%M%S)"
  legacy_exists="$(runuser -u postgres -- psql -d postgres -X -Atqc "SELECT 1 FROM pg_database WHERE datname='$LEGACY_DB'")"
  if [[ "$legacy_exists" = 1 ]]; then
    runuser -u postgres -- psql -d postgres -X -v ON_ERROR_STOP=1 -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname IN ('$DB_NAME', '$LEGACY_DB', '$CANDIDATE_DB') AND pid <> pg_backend_pid()" >/dev/null
    if runuser -u postgres -- psql -d postgres -X -Atqc "SELECT 1 FROM pg_database WHERE datname='$DB_NAME'" | grep -qx 1; then
      runuser -u postgres -- psql -d postgres -X -v ON_ERROR_STOP=1 -c "ALTER DATABASE $DB_NAME RENAME TO $failed_db"
    fi
    runuser -u postgres -- psql -d postgres -X -v ON_ERROR_STOP=1 -c "ALTER DATABASE $LEGACY_DB RENAME TO $DB_NAME"
  fi
fi

failed_config="/etc/chimii.failed.$(date -u +%Y%m%dT%H%M%S)"
[[ -d /etc/chimii ]] && mv /etc/chimii "$failed_config"
cp -a "$backup/etc-chimii" /etc/chimii
rm -f "$REMOTE_ROOT/.env"
if [[ -f "$backup/root.env" ]]; then
  cp -a "$backup/root.env" "$REMOTE_ROOT/.env"
fi
for link in current current-backend current-web server-current web-current daemon-current chimii-cli; do
  if [[ -f "$backup/$link.link" ]]; then
    ln -sfn "$(cat "$backup/$link.link")" "$REMOTE_ROOT/$link"
  elif [[ -f "$backup/$link.absent" ]]; then
    rm -f "$REMOTE_ROOT/$link"
  fi
done
install -m 0644 "$backup/Caddyfile" /etc/caddy/Caddyfile
for file in "$backup"/systemd/*.service; do
  [[ -f "$file" ]] || continue
  install -m 0644 "$file" "/etc/systemd/system/${file##*/}"
done
systemctl daemon-reload
caddy validate --config /etc/caddy/Caddyfile
systemctl reload caddy
for unit in chimii-backend.service chimii-web.service chimii-daemon.service chimii-public-sales-egress.service chimii-public-sales-runtime.service; do
  if grep -Fxq enabled "$backup/systemd/$unit.enabled" 2>/dev/null; then
    systemctl enable "$unit" >/dev/null 2>&1 || true
  else
    systemctl disable "$unit" >/dev/null 2>&1 || true
  fi
  if grep -Fxq active "$backup/systemd/$unit.active" 2>/dev/null; then
    systemctl start "$unit"
  else
    systemctl stop "$unit" 2>/dev/null || true
  fi
done
curl -fsS --max-time 15 http://127.0.0.1:8080/readyz >/dev/null
REMOTE
}

plan() {
  verify_target
  remote_user "sudo -n bash -c 'printf \"disk: \"; df -h / | tail -n 1; printf \"memory: \"; free -h | sed -n \"2p\"; printf \"services:\\n\"; systemctl is-active caddy postgresql chimii-backend chimii-web chimii-daemon chimii-public-sales-runtime chimii-public-sales-egress 2>/dev/null || true; printf \"ports:\\n\"; ss -lnt | grep -E \"127.0.0.1:(3000|3100|8080|9443|13000|18080)|:80 |:443 \" || true'"
  printf 'target=%s root=%s database=%s backend=%s web=%s domain=%s origin_https_port=%s public_route=%s\n' "$SSH_HOST" "$REMOTE_ROOT" "$DB_NAME" "$BACKEND_PORT" "$WEB_PORT" "$PRIMARY_DOMAIN" "$ORIGIN_HTTPS_PORT" "$PUBLIC_ROUTE"
  daily_remote preflight
}

main() {
  trap cleanup EXIT
  validate_config
  cd "$ROOT_DIR"
  case "$ACTION" in
    plan) plan ;;
    deploy)
      acquire_local_lock
      verify_target
      deploy_daily
      ;;
    bootstrap)
      acquire_local_lock
      verify_target
      replace_legacy
      ;;
    verify)
      verify_target
      verify_deployment
      ;;
    config)
      acquire_local_lock
      verify_target
      RELEASE_ID="config-$(date -u +%Y%m%dT%H%M%SZ)"
      acquire_remote_lock "$RELEASE_ID"
      configure_caddy
      verify_deployment
      ;;
    rollback)
      acquire_local_lock
      verify_target
      RELEASE_ID="rollback-$(date -u +%Y%m%dT%H%M%SZ)"
      acquire_remote_lock "$RELEASE_ID"
      daily_remote rollback
      verify_deployment
      ;;
    rollback-legacy)
      acquire_local_lock
      verify_target
      [[ "${CONFIRM_ROLLBACK_LEGACY:-}" == rollback-chimii ]] || die "set CONFIRM_ROLLBACK_LEGACY=rollback-chimii"
      target_release="${LEGACY_RELEASE_ID:-$(remote_user "sudo -n bash -c 'find $REMOTE_ROOT/state/legacy -mindepth 1 -maxdepth 1 -type d -printf \"%f\\n\" | sort | tail -n 1'")}"
      RELEASE_ID="rollback-$(date -u +%Y%m%dT%H%M%SZ)"
      acquire_remote_lock "$RELEASE_ID"
      rollback_legacy "$target_release"
      ;;
    help|--help|-h) printf 'usage: %s [deploy|plan|verify|config|rollback|bootstrap|rollback-legacy]\nNo argument deploys the current tagged main checkout to sh.\n' "$0" ;;
    *) die "usage: $0 [deploy|plan|verify|config|rollback|bootstrap|rollback-legacy]" ;;
  esac
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  main "$@"
fi
