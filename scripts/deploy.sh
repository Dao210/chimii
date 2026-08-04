#!/usr/bin/env bash
# Native Linux deployment for Chimii (no Docker).
#
# Go binaries are cross-compiled locally. Only the Web workspace is uploaded
# and built on Linux so native Next.js dependencies match production. Desktop,
# mobile, and documentation applications are never packaged.
#
# First server setup:
#   ./scripts/deploy.sh setup
#
# Normal deployments:
#   ./scripts/deploy.sh auto       # deploy only changed components (default)
#   ./scripts/deploy.sh backend    # backend + migrations only
#   ./scripts/deploy.sh web        # Web only
#   ./scripts/deploy.sh config     # .env/systemd/nginx only
#   ./scripts/deploy.sh full       # backend + Web + config
#   ./scripts/deploy.sh rollback   # switch to the previous recorded release
#   ./scripts/deploy.sh verify
#
# TLS:
#   TLS_EMAIL=ops@example.com ./scripts/deploy.sh tls

set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ACTION="${1:-auto}"
[[ "$ACTION" == deploy ]] && ACTION=auto
[[ "$ACTION" == all ]] && ACTION=full

SSH_HOST="${SSH_HOST:-root@47.90.152.4}"
SSH_KEY="${SSH_KEY:-}"
DOMAIN="${DOMAIN:-chimii.com}"
SERVER_IP="${SERVER_IP:-47.90.152.4}"
REMOTE_ROOT="${REMOTE_ROOT:-/opt/chimii}"
BACKEND_PORT="${BACKEND_PORT:-18080}"
WEB_PORT="${WEB_PORT:-13000}"
PNPM_VERSION="${PNPM_VERSION:-10.28.2}"
KEEP_RELEASES="${KEEP_RELEASES:-3}"
BUILD_POLL_SECONDS="${BUILD_POLL_SECONDS:-45}"
ALLOW_SIGNUP="${ALLOW_SIGNUP:-true}"
SKIP_LOCAL_TYPECHECK="${SKIP_LOCAL_TYPECHECK:-false}"

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

DEPLOY_BACKEND=false
DEPLOY_WEB=false
DEPLOY_CONFIG=false
MUTATING=false
REMOTE_LOCK_HELD=false
LOCAL_LOCK_DIR=""
BUILD_TMP=""
REPORT_STARTED_AT="$(date +%s)"
REPORT_OUTCOME=failed
CURRENT_STAGE=""
CURRENT_STAGE_STARTED=0
STAGE_NAMES=()
STAGE_SECONDS=()
STAGE_STATUSES=()
VERSION="${VERSION:-}"
SOURCE_COMMIT=unknown
SOURCE_COMMIT_SHORT=unknown
SOURCE_FINGERPRINT=unknown
RELEASE_ID="${RELEASE_ID:-}"
REMOTE_INCOMING=""
BACKEND_SHA256=""
WEB_SHA256=""
WEB_SOURCE_SHA256=""
ENV_SHA256=""
REMOTE_SOURCE_COMMIT=""
REMOTE_SOURCE_FINGERPRINT=""
REMOTE_ENV_SHA256=""

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

sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

record_stage() {
  local name="$1" status="$2" started="$3" ended
  ended="$(date +%s)"
  STAGE_NAMES+=("$name")
  STAGE_SECONDS+=("$((ended - started))")
  STAGE_STATUSES+=("$status")
}

on_error() {
  local rc=$?
  trap - ERR
  if [[ -n "$CURRENT_STAGE" ]]; then
    record_stage "$CURRENT_STAGE" failed "$CURRENT_STAGE_STARTED"
    CURRENT_STAGE=""
  fi
  exit "$rc"
}
run_stage() {
  local name="$1"
  shift
  CURRENT_STAGE="$name"
  CURRENT_STAGE_STARTED="$(date +%s)"
  log "stage: $name"
  "$@"
  record_stage "$name" success "$CURRENT_STAGE_STARTED"
  CURRENT_STAGE=""
}

write_report() {
  local report_dir report_path report_release_id ended i comma
  report_dir="$ROOT_DIR/.chimii/deploy-reports"
  mkdir -p "$report_dir" 2>/dev/null || return 0
  report_release_id="${RELEASE_ID:-check-$(date -u +%Y%m%d-%H%M%S)}"
  report_path="$report_dir/$report_release_id.json"
  ended="$(date +%s)"
  {
    printf '{\n'
    printf '  "releaseId": "%s",\n' "$report_release_id"
    printf '  "action": "%s",\n' "$ACTION"
    printf '  "outcome": "%s",\n' "$REPORT_OUTCOME"
    printf '  "sourceCommit": "%s",\n' "$SOURCE_COMMIT"
    printf '  "sourceFingerprint": "%s",\n' "$SOURCE_FINGERPRINT"
    printf '  "components": {"backend": %s, "web": %s, "config": %s},\n' "$DEPLOY_BACKEND" "$DEPLOY_WEB" "$DEPLOY_CONFIG"
    printf '  "startedAtEpoch": %s,\n' "$REPORT_STARTED_AT"
    printf '  "finishedAtEpoch": %s,\n' "$ended"
    printf '  "totalSeconds": %s,\n' "$((ended - REPORT_STARTED_AT))"
    printf '  "artifacts": {"backendSha256": "%s", "webSha256": "%s", "webSourceSha256": "%s", "envSha256": "%s"},\n' "$BACKEND_SHA256" "$WEB_SHA256" "$WEB_SOURCE_SHA256" "$ENV_SHA256"
    printf '  "stages": ['
    comma=""
    for ((i = 0; i < ${#STAGE_NAMES[@]}; i++)); do
      printf '%s\n    {"name": "%s", "seconds": %s, "status": "%s"}' "$comma" "${STAGE_NAMES[$i]}" "${STAGE_SECONDS[$i]}" "${STAGE_STATUSES[$i]}"
      comma=,
    done
    if ((${#STAGE_NAMES[@]} > 0)); then printf '\n  '; fi
    printf ']\n}\n'
  } > "$report_path"
  printf '\033[36m[chimii] deployment report: %s\033[0m\n' "$report_path"
}

release_remote_lock() {
  [[ "$REMOTE_LOCK_HELD" == true ]] || return 0
  ssh_run "REMOTE_ROOT='$REMOTE_ROOT' LOCK_TOKEN='$RELEASE_ID' bash -s" <<'REMOTE' >/dev/null 2>&1 || true
set -eu
lock_dir="$REMOTE_ROOT/.deploy-lock"
if [[ -f "$lock_dir/token" ]] && [[ "$(cat "$lock_dir/token")" == "$LOCK_TOKEN" ]]; then
  unit="chimii-web-build-${LOCK_TOKEN//./-}.service"
  # Keep the ownership token when the client disappears during a build. The
  # same release id can resume safely; a different deployment cannot overlap.
  if systemctl is-active --quiet "$unit"; then
    exit 0
  fi
  rm -f -- "$lock_dir/token" "$lock_dir/started-at"
  rmdir "$lock_dir" 2>/dev/null || true
fi
REMOTE
  REMOTE_LOCK_HELD=false
}

cleanup() {
  local rc=$?
  set +e
  release_remote_lock
  if [[ -n "$BUILD_TMP" && -d "$BUILD_TMP" ]]; then
    rm -rf -- "$BUILD_TMP"
  fi
  if [[ -n "$LOCAL_LOCK_DIR" && -d "$LOCAL_LOCK_DIR" ]]; then
    rmdir "$LOCAL_LOCK_DIR" 2>/dev/null || true
  fi
  write_report
  return "$rc"
}
validate_config() {
  [[ "$DOMAIN" =~ ^[A-Za-z0-9.-]+$ ]] || die "invalid DOMAIN: $DOMAIN"
  [[ "$SERVER_IP" =~ ^[0-9A-Fa-f:.]+$ ]] || die "invalid SERVER_IP: $SERVER_IP"
  [[ "$BACKEND_PORT" =~ ^[0-9]+$ ]] || die "invalid BACKEND_PORT"
  [[ "$WEB_PORT" =~ ^[0-9]+$ ]] || die "invalid WEB_PORT"
  [[ "$REMOTE_ROOT" =~ ^/[A-Za-z0-9._/-]+$ ]] || die "REMOTE_ROOT must be a safe absolute path"
  [[ "$PNPM_VERSION" =~ ^[0-9]+([.][0-9]+){2}$ ]] || die "invalid PNPM_VERSION"
  [[ "$KEEP_RELEASES" =~ ^[1-9][0-9]*$ ]] || die "KEEP_RELEASES must be a positive integer"
  [[ "$BUILD_POLL_SECONDS" =~ ^[0-9]+$ ]] || die "BUILD_POLL_SECONDS must be an integer"
  (( BUILD_POLL_SECONDS >= 15 && BUILD_POLL_SECONDS <= 300 )) || die "BUILD_POLL_SECONDS must be between 15 and 300"
  [[ "$ALLOW_SIGNUP" == true || "$ALLOW_SIGNUP" == false ]] || die "ALLOW_SIGNUP must be true or false"
  [[ "$SKIP_LOCAL_TYPECHECK" == true || "$SKIP_LOCAL_TYPECHECK" == false ]] || die "SKIP_LOCAL_TYPECHECK must be true or false"
  [[ "$SSH_HOST" != sshmd ]] || die "shell aliases do not expand in scripts; set SSH_HOST to the real SSH destination"
}

acquire_local_lock() {
  local lock_name
  lock_name="${SSH_HOST//[^A-Za-z0-9_.-]/_}-${DOMAIN//[^A-Za-z0-9_.-]/_}"
  LOCAL_LOCK_DIR="${TMPDIR:-/tmp}/chimii-deploy-$lock_name.lock"
  if ! mkdir "$LOCAL_LOCK_DIR" 2>/dev/null; then
    die "another local deployment holds $LOCAL_LOCK_DIR"
  fi
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

require_remote_setup() {
  ssh_run "REMOTE_ROOT='$REMOTE_ROOT' bash -s" <<'REMOTE'
set -eu
for command in node corepack nginx redis-cli psql systemd-run; do
  command -v "$command" >/dev/null 2>&1 || { echo "missing remote command: $command; run deploy.sh setup" >&2; exit 1; }
done
test -d "$REMOTE_ROOT/releases/backend" || { echo "server is not initialized; run deploy.sh setup" >&2; exit 1; }
test -d "$REMOTE_ROOT/releases/web" || { echo "server is not initialized; run deploy.sh setup" >&2; exit 1; }
test -d "$REMOTE_ROOT/builder/workspace" || { echo "persistent builder is missing; run deploy.sh setup" >&2; exit 1; }
REMOTE
}

acquire_remote_lock() {
  ssh_run "REMOTE_ROOT='$REMOTE_ROOT' LOCK_TOKEN='$RELEASE_ID' bash -s" <<'REMOTE'
set -euo pipefail
lock_dir="$REMOTE_ROOT/.deploy-lock"
now="$(date +%s)"
if ! mkdir "$lock_dir" 2>/dev/null; then
  if [[ -f "$lock_dir/token" ]] && [[ "$(cat "$lock_dir/token")" == "$LOCK_TOKEN" ]]; then
    echo "resuming deployment lock for $LOCK_TOKEN"
    exit 0
  fi
  started=0
  [[ -f "$lock_dir/started-at" ]] && started="$(cat "$lock_dir/started-at")"
  if [[ "$started" =~ ^[0-9]+$ ]] && (( now - started > 7200 )); then
    echo "stale deployment lock detected; remove $lock_dir only after confirming no deployment is running" >&2
  else
    echo "another remote deployment holds $lock_dir (token: $(cat "$lock_dir/token" 2>/dev/null || echo unknown))" >&2
  fi
  exit 1
fi
printf '%s\n' "$LOCK_TOKEN" > "$lock_dir/token"
printf '%s\n' "$now" > "$lock_dir/started-at"
chmod 600 "$lock_dir/token" "$lock_dir/started-at"
REMOTE
  REMOTE_LOCK_HELD=true
}

compute_source_fingerprint() {
  local untracked
  {
    git -C "$ROOT_DIR" rev-parse HEAD 2>/dev/null || printf 'unknown\n'
    git -C "$ROOT_DIR" diff --binary HEAD -- . ':(exclude).env*' 2>/dev/null || true
    untracked="$(git -C "$ROOT_DIR" ls-files --others --exclude-standard 2>/dev/null || true)"
    if [[ -n "$untracked" ]]; then
      while IFS= read -r path; do
        printf '%s\0' "$path"
        [[ -f "$ROOT_DIR/$path" ]] && sha256_file "$ROOT_DIR/$path"
      done <<< "$untracked"
    fi
  } | if command -v sha256sum >/dev/null 2>&1; then sha256sum; else shasum -a 256; fi | awk '{print $1}'
}

init_release_context() {
  require_cmd git
  require_cmd node
  VERSION="${VERSION:-$(node -p "require('$ROOT_DIR/package.json').version")}"
  SOURCE_COMMIT="${COMMIT:-$(git -C "$ROOT_DIR" rev-parse HEAD 2>/dev/null || printf unknown)}"
  SOURCE_COMMIT_SHORT="$(printf '%s' "$SOURCE_COMMIT" | cut -c1-12)"
  SOURCE_FINGERPRINT="$(compute_source_fingerprint)"
  RELEASE_ID="${RELEASE_ID:-$(date -u +%Y%m%d-%H%M%S)-$VERSION-$SOURCE_COMMIT_SHORT}"
  [[ "$VERSION" =~ ^[0-9A-Za-z._+-]+$ ]] || die "invalid VERSION: $VERSION"
  [[ "$SOURCE_COMMIT" =~ ^[0-9A-Za-z._+-]+$ ]] || die "invalid COMMIT: $SOURCE_COMMIT"
  [[ "$SOURCE_FINGERPRINT" =~ ^[0-9a-f]{64}$ ]] || die "invalid source fingerprint"
  [[ "$RELEASE_ID" =~ ^[0-9A-Za-z._+-]+$ ]] || die "invalid RELEASE_ID: $RELEASE_ID"
  REMOTE_INCOMING="$REMOTE_ROOT/incoming/$RELEASE_ID"
}

read_remote_state() {
  local state line key value
  state="$(ssh_run "REMOTE_ROOT='$REMOTE_ROOT' bash -s" <<'REMOTE'
set -eu
file="$REMOTE_ROOT/state/current.env"
[[ -f "$file" ]] || exit 0
grep -E '^(SOURCE_COMMIT|SOURCE_FINGERPRINT|ENV_SHA256)=' "$file" || true
REMOTE
)"
  while IFS= read -r line; do
    key="${line%%=*}"
    value="${line#*=}"
    case "$key" in
      SOURCE_COMMIT) REMOTE_SOURCE_COMMIT="$value" ;;
      SOURCE_FINGERPRINT) REMOTE_SOURCE_FINGERPRINT="$value" ;;
      ENV_SHA256) REMOTE_ENV_SHA256="$value" ;;
    esac
  done <<< "$state"
}

collect_changed_paths() {
  local base="$1"
  if [[ -n "$base" ]] && git -C "$ROOT_DIR" cat-file -e "$base^{commit}" 2>/dev/null; then
    git -C "$ROOT_DIR" diff --name-only "$base" HEAD --
  else
    printf '__FULL__\n'
  fi
  git -C "$ROOT_DIR" diff --name-only HEAD --
  git -C "$ROOT_DIR" ls-files --others --exclude-standard
}

classify_changed_paths() {
  local path unknown=false
  DEPLOY_BACKEND=false
  DEPLOY_WEB=false
  while IFS= read -r path; do
    [[ -n "$path" ]] || continue
    case "$path" in
      __FULL__) DEPLOY_BACKEND=true; DEPLOY_WEB=true ;;
      server/*) DEPLOY_BACKEND=true ;;
      apps/web/*|packages/core/*|packages/ui/*|packages/views/*|packages/tsconfig/*|packages/eslint-config/*|package.json|pnpm-lock.yaml|pnpm-workspace.yaml|turbo.json|.npmrc)
        DEPLOY_WEB=true
        ;;
      apps/desktop/*|apps/mobile/*|apps/docs/*|docs/*|*.md|.github/*|.agents/*|.codex/*|scripts/*|.gitignore|LICENSE*)
        ;;
      *) unknown=true ;;
    esac
  done
  if [[ "$unknown" == true ]]; then
    warn "unclassified source changes detected; selecting backend and Web for safety"
    DEPLOY_BACKEND=true
    DEPLOY_WEB=true
  fi
}

local_env_path() {
  if [[ -f "$ROOT_DIR/.env" ]]; then
    printf '%s\n' "$ROOT_DIR/.env"
  else
    printf '%s\n' "$ROOT_DIR/.env.example"
  fi
}

plan_components() {
  local env_path changed_paths
  env_path="$(local_env_path)"
  ENV_SHA256="$(sha256_file "$env_path")"
  case "$ACTION" in
    auto)
      if [[ -z "$REMOTE_SOURCE_FINGERPRINT" ]]; then
        DEPLOY_BACKEND=true
        DEPLOY_WEB=true
        DEPLOY_CONFIG=true
      elif [[ "$REMOTE_SOURCE_FINGERPRINT" != "$SOURCE_FINGERPRINT" ]]; then
        changed_paths="$(collect_changed_paths "$REMOTE_SOURCE_COMMIT" | sort -u)"
        classify_changed_paths <<< "$changed_paths"
      fi
      [[ "$REMOTE_ENV_SHA256" == "$ENV_SHA256" ]] || DEPLOY_CONFIG=true
      ;;
    backend) DEPLOY_BACKEND=true ;;
    web) DEPLOY_WEB=true ;;
    config) DEPLOY_CONFIG=true ;;
    full) DEPLOY_BACKEND=true; DEPLOY_WEB=true; DEPLOY_CONFIG=true ;;
  esac
  log "plan: backend=$DEPLOY_BACKEND web=$DEPLOY_WEB config=$DEPLOY_CONFIG"
}

provision_remote() {
  log "one-time setup: PostgreSQL 17, pgvector, Redis, nginx, service user, and persistent builder"
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
(( node_major >= 22 )) || { echo "Node.js 22+ is required (found: ${node_major:-none})" >&2; exit 1; }

redis_marker='# >>> chimii >>>'
if ! grep -Fq "$redis_marker" /etc/redis/redis.conf; then
  cat >> /etc/redis/redis.conf <<'CONF'
# >>> chimii >>>
bind 127.0.0.1 -::1
protected-mode yes
port 6379
maxmemory 128mb
maxmemory-policy allkeys-lru
# <<< chimii <<<
CONF
  systemctl restart redis-server
fi

systemctl enable --now postgresql redis-server nginx >/dev/null
redis-cli -h 127.0.0.1 ping | grep -Fx PONG >/dev/null
if ! id chimii >/dev/null 2>&1; then
  useradd --system --home-dir /var/lib/chimii --create-home --shell /usr/sbin/nologin chimii
fi
install -d -m 0755 "$REMOTE_ROOT" "$REMOTE_ROOT/releases/backend" "$REMOTE_ROOT/releases/web" "$REMOTE_ROOT/releases/manifests"
install -d -m 0700 "$REMOTE_ROOT/incoming" "$REMOTE_ROOT/state" "$REMOTE_ROOT/state/config-backups"
  install -d -m 0755 "$REMOTE_ROOT/builder"
install -d -o chimii -g chimii -m 0755 /var/lib/chimii/uploads
install -d -m 0700 /etc/chimii
install -d -m 0755 /var/www/chimii-acme/.well-known/acme-challenge

if [[ ! -d "$REMOTE_ROOT/builder/workspace" ]]; then
  seed=''
  if [[ -d "$REMOTE_ROOT/build" ]]; then
    while IFS= read -r candidate; do
      if [[ -f "$candidate/apps/web/.next/BUILD_ID" && -f "$candidate/apps/web/.next/standalone/apps/web/server.js" ]]; then
        seed="$candidate"
        break
      fi
    done < <(find "$REMOTE_ROOT/build" -mindepth 1 -maxdepth 1 -type d -printf '%T@ %p\n' | sort -rn | cut -d' ' -f2-)
  fi
  if [[ -n "$seed" ]]; then
    mv "$seed" "$REMOTE_ROOT/builder/workspace"
    seed_id="${seed##*/}"
    if [[ -d "$REMOTE_ROOT/incoming/$seed_id" ]]; then
      rm -rf -- "$REMOTE_ROOT/incoming/$seed_id"
    fi
    echo "seeded persistent builder from $seed_id"
  else
    install -d -m 0755 "$REMOTE_ROOT/builder/workspace"
  fi
fi

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
for extension in vector pg_trgm btree_gin; do
  runuser -u postgres -- psql -v ON_ERROR_STOP=1 -d chimii -c "CREATE EXTENSION IF NOT EXISTS $extension" >/dev/null
done
ss -lnt | grep -Eq '127\.0\.0\.1:5432|\[::1\]:5432'
ss -lnt | grep -Eq '127\.0\.0\.1:6379'
REMOTE
  ok "remote setup completed"
}

prepare_build_tmp() {
  if [[ -z "$BUILD_TMP" ]]; then
    BUILD_TMP="$(mktemp -d "${TMPDIR:-/tmp}/chimii-deploy.XXXXXX")"
  fi
}

build_backend_local() {
  require_cmd go
  require_cmd tar
  prepare_build_tmp
  log "cross-compile Go backend for linux/amd64"
  install -d "$BUILD_TMP/backend/migrations"
  (
    cd "$ROOT_DIR/server"
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w -X main.version=$VERSION -X main.commit=$SOURCE_COMMIT_SHORT" -o "$BUILD_TMP/backend/server" ./cmd/server &
    p1=$!
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags '-s -w' -o "$BUILD_TMP/backend/migrate" ./cmd/migrate &
    p2=$!
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags '-s -w' -o "$BUILD_TMP/backend/backfill_task_usage_hourly" ./cmd/backfill_task_usage_hourly &
    p3=$!
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags '-s -w' -o "$BUILD_TMP/backend/backfill_codex_usage_cache" ./cmd/backfill_codex_usage_cache &
    p4=$!
    status=0
    wait "$p1" || status=$?
    wait "$p2" || status=$?
    wait "$p3" || status=$?
    wait "$p4" || status=$?
    (( status == 0 ))
  )
  cp "$ROOT_DIR"/server/migrations/*.sql "$BUILD_TMP/backend/migrations/"
  require_cmd gzip
  COPYFILE_DISABLE=1 tar --no-xattrs -cf "$BUILD_TMP/backend.tar" -C "$BUILD_TMP" backend
  gzip -n "$BUILD_TMP/backend.tar"
  BACKEND_SHA256="$(sha256_file "$BUILD_TMP/backend.tar.gz")"
}

typecheck_web_local() {
  [[ "$SKIP_LOCAL_TYPECHECK" == false ]] || { warn "local Web typecheck skipped by explicit override"; return 0; }
  require_cmd corepack
  log "typecheck Web and its workspace dependencies locally"
  (cd "$ROOT_DIR" && corepack pnpm exec turbo typecheck --filter=@chimii/web...)
}

package_web_source() {
  require_cmd tar
  require_cmd gzip
  prepare_build_tmp
  log "package Web source only (desktop/mobile excluded)"
  COPYFILE_DISABLE=1 tar --no-xattrs -cf "$BUILD_TMP/web-source.tar" \
    --exclude='.DS_Store' --exclude='node_modules' --exclude='.next' \
    --exclude='.turbo' --exclude='*.tsbuildinfo' --exclude='test-results' \
    -C "$ROOT_DIR" \
    package.json pnpm-lock.yaml pnpm-workspace.yaml turbo.json .npmrc \
    apps/web packages/core packages/ui packages/views packages/tsconfig packages/eslint-config
  WEB_SOURCE_SHA256="$(sha256_file "$BUILD_TMP/web-source.tar")"
  gzip -n "$BUILD_TMP/web-source.tar"
  WEB_SHA256="$(sha256_file "$BUILD_TMP/web-source.tar.gz")"
}

package_config() {
  local env_path
  prepare_build_tmp
  env_path="$(local_env_path)"
  cp "$env_path" "$BUILD_TMP/project.env"
  chmod 600 "$BUILD_TMP/project.env"
  ENV_SHA256="$(sha256_file "$BUILD_TMP/project.env")"
}

upload_inputs() {
  local files=()
  require_cmd ssh
  require_cmd scp
  ssh_run "install -d -m 0700 '$REMOTE_INCOMING'"
  [[ "$DEPLOY_BACKEND" == true ]] && files+=("$BUILD_TMP/backend.tar.gz")
  [[ "$DEPLOY_WEB" == true ]] && files+=("$BUILD_TMP/web-source.tar.gz")
  [[ "$DEPLOY_CONFIG" == true ]] && files+=("$BUILD_TMP/project.env")
  ((${#files[@]} > 0)) || return 0
  scp_push "${files[@]}"
  ssh_run "chmod 600 '$REMOTE_INCOMING/project.env' 2>/dev/null || true; sha256sum '$REMOTE_INCOMING'/*"
}

build_web_remote() {
  local deadline poll_status lost_polls=0 poll_failures=0
  log "refresh persistent Linux workspace and start a disconnect-safe Web build"
  ssh_run "RELEASE_ID='$RELEASE_ID' REMOTE_ROOT='$REMOTE_ROOT' PNPM_VERSION='$PNPM_VERSION' VERSION='$VERSION' WEB_SHA256='$WEB_SHA256' WEB_SOURCE_SHA256='$WEB_SOURCE_SHA256' LOCK_TOKEN='$RELEASE_ID' bash -s" <<'REMOTE'
set -euo pipefail
lock_dir="$REMOTE_ROOT/.deploy-lock"
[[ -f "$lock_dir/token" && "$(cat "$lock_dir/token")" == "$LOCK_TOKEN" ]] || { echo "deployment lock lost" >&2; exit 1; }
incoming="$REMOTE_ROOT/incoming/$RELEASE_ID"
workspace="$REMOTE_ROOT/builder/workspace"
cache_hold="$REMOTE_ROOT/builder/next-cache-hold"
status_dir="$REMOTE_ROOT/builder/status"
status_file="$status_dir/$RELEASE_ID.exit-code"
input_file="$status_dir/$RELEASE_ID.input-sha256"
runner="$REMOTE_ROOT/builder/run-web-build"
unit="chimii-web-build-${RELEASE_ID//./-}"
test -f "$incoming/web-source.tar.gz"
install -d -m 0755 "$workspace" "$status_dir"
actual_sha="$(sha256sum "$incoming/web-source.tar.gz" | awk '{print $1}')"
[[ "$actual_sha" == "$WEB_SHA256" ]] || { echo "uploaded Web source checksum mismatch" >&2; exit 1; }
actual_source_sha="$(gzip -dc "$incoming/web-source.tar.gz" | sha256sum | awk '{print $1}')"
[[ "$actual_source_sha" == "$WEB_SOURCE_SHA256" ]] || { echo "uploaded Web source content checksum mismatch" >&2; exit 1; }

if [[ -f "$status_file" && -f "$input_file" ]] \
  && [[ "$(cat "$status_file")" == 0 ]] \
  && [[ "$(cat "$input_file")" == "$WEB_SOURCE_SHA256" ]] \
  && [[ -f "$workspace/apps/web/.next/standalone/apps/web/server.js" ]]; then
  echo "reusing completed Web build for $RELEASE_ID"
  exit 0
fi

rm -rf -- "$cache_hold"
if [[ -d "$workspace/apps/web/.next/cache" ]]; then
  mv "$workspace/apps/web/.next/cache" "$cache_hold"
fi
rm -rf -- \
  "$workspace/apps/web" \
  "$workspace/packages/core" \
  "$workspace/packages/ui" \
  "$workspace/packages/views" \
  "$workspace/packages/tsconfig" \
  "$workspace/packages/eslint-config"
rm -f -- "$workspace/package.json" "$workspace/pnpm-lock.yaml" "$workspace/pnpm-workspace.yaml" "$workspace/turbo.json" "$workspace/.npmrc"
tar -xzf "$incoming/web-source.tar.gz" -C "$workspace"
if [[ -d "$cache_hold" ]]; then
  install -d -m 0755 "$workspace/apps/web/.next"
  mv "$cache_hold" "$workspace/apps/web/.next/cache"
fi

cd "$workspace"
export COREPACK_ENABLE_DOWNLOAD_PROMPT=0
corepack pnpm --version | grep -Fx "$PNPM_VERSION"
# pnpm's root-owned content-addressed store is already persistent on this host;
# the workspace-level node_modules is retained between builds as well.
corepack pnpm install --frozen-lockfile --prefer-offline --filter @chimii/web...

rm -f -- "$status_file" "$input_file" "$status_file.tmp"
printf '%s\n' "$WEB_SOURCE_SHA256" > "$input_file"
cat > "$runner" <<'RUNNER'
#!/usr/bin/env bash
set -uo pipefail
release_id="$1"
version="$2"
remote_root="$3"
status_file="$remote_root/builder/status/$release_id.exit-code"
status_tmp="$status_file.tmp"
set +e
/usr/bin/env \
  STANDALONE=true \
  NEXT_PUBLIC_APP_VERSION="$version" \
  NEXT_TELEMETRY_DISABLED=1 \
  CHIMII_LOW_MEMORY_BUILD=true \
  CHIMII_DEPLOY_TYPECHECKED=true \
  NODE_OPTIONS=--max-old-space-size=1536 \
  COREPACK_ENABLE_DOWNLOAD_PROMPT=0 \
  /usr/bin/corepack pnpm --filter @chimii/web build
rc=$?
printf '%s\n' "$rc" > "$status_tmp"
mv "$status_tmp" "$status_file"
exit "$rc"
RUNNER
chmod 0755 "$runner"
systemctl reset-failed "$unit.service" 2>/dev/null || true
systemd-run --quiet --unit="$unit" \
  --property=Type=exec \
  --property="WorkingDirectory=$workspace" \
  --property=Nice=5 \
  --property=IOSchedulingClass=best-effort \
  --property=IOSchedulingPriority=6 \
  "$runner" "$RELEASE_ID" "$VERSION" "$REMOTE_ROOT"
REMOTE

  # Poll with short independent SSH connections. A dropped connection cannot
  # cancel the systemd build, and a rerun with the same release id can resume.
  deadline=$(( $(date +%s) + 5400 ))
  while (( $(date +%s) < deadline )); do
    if poll_status="$(ssh_run "RELEASE_ID='$RELEASE_ID' REMOTE_ROOT='$REMOTE_ROOT' bash -s" <<'REMOTE'
set -eu
status_file="$REMOTE_ROOT/builder/status/$RELEASE_ID.exit-code"
unit="chimii-web-build-${RELEASE_ID//./-}.service"
if [[ -f "$status_file" ]]; then
  printf 'done:%s\n' "$(cat "$status_file")"
elif systemctl is-active --quiet "$unit"; then
  printf 'running\n'
else
  printf 'lost\n'
fi
REMOTE
)"; then
      if (( poll_failures > 0 )); then
        log "Web build status polling recovered after $poll_failures dropped connection(s)"
        poll_failures=0
      fi
      case "$poll_status" in
        done:0) break ;;
        done:*)
          ssh_run "journalctl -u 'chimii-web-build-${RELEASE_ID//./-}.service' -n 120 --no-pager" || true
          return 1
          ;;
        running) lost_polls=0 ;;
        lost)
          lost_polls=$((lost_polls + 1))
          if (( lost_polls >= 3 )); then
            ssh_run "journalctl -u 'chimii-web-build-${RELEASE_ID//./-}.service' -n 120 --no-pager" || true
            return 1
          fi
          ;;
        *) warn "unexpected Web build status: $poll_status" ;;
      esac
    else
      poll_failures=$((poll_failures + 1))
      if (( poll_failures == 1 || poll_failures % 3 == 0 )); then
        warn "Web build status connection dropped ($poll_failures consecutive); systemd build continues"
      fi
    fi
    sleep "$BUILD_POLL_SECONDS"
  done
  if (( $(date +%s) >= deadline )); then
    ssh_run "systemctl stop 'chimii-web-build-${RELEASE_ID//./-}.service'" || true
    return 1
  fi
  ssh_run "journalctl -u 'chimii-web-build-${RELEASE_ID//./-}.service' -n 120 --no-pager; test -f '$REMOTE_ROOT/builder/workspace/apps/web/.next/standalone/apps/web/server.js'; test -d '$REMOTE_ROOT/builder/workspace/apps/web/.next/cache'"
  ok "Linux Web build completed"
}

configure_remote() {
  log "update protected production env, systemd units, and isolated nginx vhost"
  ssh_run "RELEASE_ID='$RELEASE_ID' REMOTE_ROOT='$REMOTE_ROOT' DOMAIN='$DOMAIN' BACKEND_PORT='$BACKEND_PORT' WEB_PORT='$WEB_PORT' ALLOW_SIGNUP='$ALLOW_SIGNUP' LOCK_TOKEN='$RELEASE_ID' bash -s" <<'REMOTE'
set -Eeuo pipefail
lock_dir="$REMOTE_ROOT/.deploy-lock"
[[ -f "$lock_dir/token" && "$(cat "$lock_dir/token")" == "$LOCK_TOKEN" ]] || { echo "deployment lock lost" >&2; exit 1; }
incoming="$REMOTE_ROOT/incoming/$RELEASE_ID"
backup="$REMOTE_ROOT/state/config-backups/$RELEASE_ID"
install -d -m 0700 "$backup"

backup_file() {
  local source="$1" name="$2"
  if [[ -e "$source" || -L "$source" ]]; then cp -a "$source" "$backup/$name"; fi
}
restore_config_files() {
  set +e
  for item in backend.env web.env chimii-backend.service chimii-web.service chimii-proxy.conf chimii-site.conf; do
    [[ -e "$backup/$item" ]] || continue
    case "$item" in
      backend.env|web.env) cp -a "$backup/$item" "/etc/chimii/$item" ;;
      *.service) cp -a "$backup/$item" "/etc/systemd/system/$item" ;;
      chimii-proxy.conf) cp -a "$backup/$item" /etc/nginx/snippets/chimii-proxy.conf ;;
      chimii-site.conf) cp -a "$backup/$item" /etc/nginx/sites-available/chimii.com.conf ;;
    esac
  done
  systemctl daemon-reload
  nginx -t && systemctl reload nginx
}
on_config_error() {
  local rc=$?
  trap - ERR
  restore_config_files
  exit "$rc"
}
trap on_config_error ERR

backup_file /etc/chimii/backend.env backend.env
backup_file /etc/chimii/web.env web.env
backup_file /etc/systemd/system/chimii-backend.service chimii-backend.service
backup_file /etc/systemd/system/chimii-web.service chimii-web.service
backup_file /etc/nginx/snippets/chimii-proxy.conf chimii-proxy.conf
backup_file /etc/nginx/sites-available/chimii.com.conf chimii-site.conf

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

printf '%s\n' 'NODE_ENV=production' 'HOSTNAME=127.0.0.1' "PORT=$WEB_PORT" "REMOTE_API_URL=http://127.0.0.1:$BACKEND_PORT" 'NEXT_TELEMETRY_DISABLED=1' > /etc/chimii/web.env
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
location = /api/daemon/ws {
  proxy_http_version 1.1;
  proxy_set_header Upgrade \$http_upgrade;
  proxy_set_header Connection "upgrade";
  proxy_connect_timeout 10s;
  proxy_read_timeout 86400s;
  proxy_send_timeout 86400s;
  proxy_pass http://127.0.0.1:$BACKEND_PORT;
}
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
systemctl daemon-reload
systemctl enable chimii-backend.service chimii-web.service >/dev/null
nginx -t
systemctl reload nginx
trap - ERR
REMOTE
}

activate_release() {
  log "install immutable release, migrate, switch symlinks, and health-check"
  ssh_run "RELEASE_ID='$RELEASE_ID' REMOTE_ROOT='$REMOTE_ROOT' DOMAIN='$DOMAIN' BACKEND_PORT='$BACKEND_PORT' WEB_PORT='$WEB_PORT' DEPLOY_BACKEND='$DEPLOY_BACKEND' DEPLOY_WEB='$DEPLOY_WEB' DEPLOY_CONFIG='$DEPLOY_CONFIG' KEEP_RELEASES='$KEEP_RELEASES' SOURCE_COMMIT='$SOURCE_COMMIT' SOURCE_FINGERPRINT='$SOURCE_FINGERPRINT' BACKEND_SHA256='$BACKEND_SHA256' WEB_SHA256='$WEB_SHA256' ENV_SHA256='$ENV_SHA256' LOCK_TOKEN='$RELEASE_ID' bash -s" <<'REMOTE'
set -Eeuo pipefail
lock_dir="$REMOTE_ROOT/.deploy-lock"
[[ -f "$lock_dir/token" && "$(cat "$lock_dir/token")" == "$LOCK_TOKEN" ]] || { echo "deployment lock lost" >&2; exit 1; }
incoming="$REMOTE_ROOT/incoming/$RELEASE_ID"
workspace="$REMOTE_ROOT/builder/workspace"
backend_release="$REMOTE_ROOT/releases/backend/$RELEASE_ID"
web_release="$REMOTE_ROOT/releases/web/$RELEASE_ID"
old_backend="$(readlink -f "$REMOTE_ROOT/current-backend" 2>/dev/null || true)"
old_web="$(readlink -f "$REMOTE_ROOT/current-web" 2>/dev/null || true)"
new_backend="$old_backend"
new_web="$old_web"
config_backup="$REMOTE_ROOT/state/config-backups/$RELEASE_ID"

restore_config() {
  [[ "$DEPLOY_CONFIG" == true && -d "$config_backup" ]] || return 0
  for item in backend.env web.env chimii-backend.service chimii-web.service chimii-proxy.conf chimii-site.conf; do
    [[ -e "$config_backup/$item" ]] || continue
    case "$item" in
      backend.env|web.env) cp -a "$config_backup/$item" "/etc/chimii/$item" ;;
      *.service) cp -a "$config_backup/$item" "/etc/systemd/system/$item" ;;
      chimii-proxy.conf) cp -a "$config_backup/$item" /etc/nginx/snippets/chimii-proxy.conf ;;
      chimii-site.conf) cp -a "$config_backup/$item" /etc/nginx/sites-available/chimii.com.conf ;;
    esac
  done
  systemctl daemon-reload
  nginx -t && systemctl reload nginx
}

rollback_failed_activation() {
  local rc=$?
  trap - ERR
  set +e
  echo "activation failed; restoring previous application links" >&2
  if [[ "$DEPLOY_BACKEND" == true && -n "$old_backend" ]]; then ln -sfn "$old_backend" "$REMOTE_ROOT/current-backend"; fi
  if [[ "$DEPLOY_WEB" == true && -n "$old_web" ]]; then ln -sfn "$old_web" "$REMOTE_ROOT/current-web"; fi
  restore_config
  systemctl restart chimii-backend.service
  systemctl restart chimii-web.service
  exit "$rc"
}
trap rollback_failed_activation ERR

if [[ "$DEPLOY_BACKEND" == true ]]; then
  [[ ! -e "$backend_release" ]] || { echo "backend release already exists: $backend_release" >&2; exit 1; }
  install -d -m 0755 "$backend_release"
  tar -xzf "$incoming/backend.tar.gz" -C "$backend_release" --strip-components=1
  chmod 0755 "$backend_release/server" "$backend_release/migrate" "$backend_release/backfill_task_usage_hourly" "$backend_release/backfill_codex_usage_cache"
  chown -R chimii:chimii "$backend_release"
  test -x "$backend_release/server"
  db_url="$(sed -n 's/^DATABASE_URL=//p' /etc/chimii/backend.env)"
  (cd "$backend_release" && runuser -u chimii -- env DATABASE_URL="$db_url" ./migrate up)
  new_backend="$backend_release"
fi

if [[ "$DEPLOY_WEB" == true ]]; then
  [[ ! -e "$web_release" ]] || { echo "web release already exists: $web_release" >&2; exit 1; }
  install -d -m 0755 "$web_release"
  cp -a "$workspace/apps/web/.next/standalone/." "$web_release/"
  install -d -m 0755 "$web_release/apps/web/.next"
  cp -a "$workspace/apps/web/.next/static" "$web_release/apps/web/.next/static"
  cp -a "$workspace/apps/web/public" "$web_release/apps/web/public"
  chown -R chimii:chimii "$web_release"
  test -f "$web_release/apps/web/server.js"
  new_web="$web_release"
fi

[[ -n "$new_backend" && -n "$new_web" ]] || { echo "both backend and Web releases must exist after activation" >&2; exit 1; }
[[ "$DEPLOY_BACKEND" == false ]] || ln -sfn "$new_backend" "$REMOTE_ROOT/current-backend"
[[ "$DEPLOY_WEB" == false ]] || ln -sfn "$new_web" "$REMOTE_ROOT/current-web"

if [[ "$DEPLOY_BACKEND" == true || "$DEPLOY_CONFIG" == true ]]; then
  systemctl restart chimii-backend.service
elif ! systemctl is-active --quiet chimii-backend.service; then
  systemctl start chimii-backend.service
fi
for _ in $(seq 1 60); do curl -fsS "http://127.0.0.1:$BACKEND_PORT/healthz" >/dev/null 2>&1 && break; sleep 1; done
curl -fsS "http://127.0.0.1:$BACKEND_PORT/healthz" >/dev/null

if [[ "$DEPLOY_WEB" == true || "$DEPLOY_CONFIG" == true ]]; then
  systemctl restart chimii-web.service
elif ! systemctl is-active --quiet chimii-web.service; then
  systemctl start chimii-web.service
fi
for _ in $(seq 1 60); do curl -fsS "http://127.0.0.1:$WEB_PORT/" >/dev/null 2>&1 && break; sleep 1; done
curl -fsS "http://127.0.0.1:$WEB_PORT/" >/dev/null
curl -fsS -H "Host: $DOMAIN" http://127.0.0.1/healthz >/dev/null

state="$REMOTE_ROOT/state/current.env"
previous="$REMOTE_ROOT/state/previous.env"
if [[ -f "$state" ]]; then
  cp "$state" "$previous"
elif [[ -n "$old_backend" && -n "$old_web" ]]; then
  {
    printf 'RELEASE_ID=pre-optimized\n'
    printf 'SOURCE_COMMIT=unknown\n'
    printf 'SOURCE_FINGERPRINT=unknown\n'
    printf 'BACKEND_RELEASE=%s\n' "$old_backend"
    printf 'WEB_RELEASE=%s\n' "$old_web"
    printf 'BACKEND_SHA256=\nWEB_SHA256=\nENV_SHA256=\n'
  } > "$previous"
fi

old_backend_sha="$(sed -n 's/^BACKEND_SHA256=//p' "$state" 2>/dev/null || true)"
old_web_sha="$(sed -n 's/^WEB_SHA256=//p' "$state" 2>/dev/null || true)"
old_env_sha="$(sed -n 's/^ENV_SHA256=//p' "$state" 2>/dev/null || true)"
[[ "$DEPLOY_BACKEND" == true ]] || BACKEND_SHA256="$old_backend_sha"
[[ "$DEPLOY_WEB" == true ]] || WEB_SHA256="$old_web_sha"
[[ "$DEPLOY_CONFIG" == true ]] || ENV_SHA256="$old_env_sha"
{
  printf 'RELEASE_ID=%s\n' "$RELEASE_ID"
  printf 'SOURCE_COMMIT=%s\n' "$SOURCE_COMMIT"
  printf 'SOURCE_FINGERPRINT=%s\n' "$SOURCE_FINGERPRINT"
  printf 'BACKEND_RELEASE=%s\n' "$new_backend"
  printf 'WEB_RELEASE=%s\n' "$new_web"
  printf 'BACKEND_SHA256=%s\n' "$BACKEND_SHA256"
  printf 'WEB_SHA256=%s\n' "$WEB_SHA256"
  printf 'ENV_SHA256=%s\n' "$ENV_SHA256"
  printf 'DEPLOYED_AT=%s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
} > "$state"
chmod 600 "$state" "$previous" 2>/dev/null || true
cp "$state" "$REMOTE_ROOT/releases/manifests/$RELEASE_ID.env"

prune_component() {
  local base="$1" current="$2" previous_path="$3" index=0 entry path
  while IFS= read -r entry; do
    path="${entry#* }"
    if (( index < KEEP_RELEASES )) || [[ "$path" == "$current" || "$path" == "$previous_path" ]]; then
      index=$((index + 1))
      continue
    fi
    case "$path" in "$base"/*) rm -rf -- "$path" ;; *) echo "refusing unsafe prune target: $path" >&2; exit 1 ;; esac
    index=$((index + 1))
  done < <(find "$base" -mindepth 1 -maxdepth 1 -type d -printf '%T@ %p\n' | sort -rn)
}
previous_backend="$(sed -n 's/^BACKEND_RELEASE=//p' "$previous" 2>/dev/null || true)"
previous_web="$(sed -n 's/^WEB_RELEASE=//p' "$previous" 2>/dev/null || true)"
prune_component "$REMOTE_ROOT/releases/backend" "$new_backend" "$previous_backend"
prune_component "$REMOTE_ROOT/releases/web" "$new_web" "$previous_web"
rm -rf -- "$incoming"
trap - ERR
REMOTE
  ok "release activated: $RELEASE_ID"
}

record_noop_state() {
  ssh_run "REMOTE_ROOT='$REMOTE_ROOT' RELEASE_ID='$RELEASE_ID' SOURCE_COMMIT='$SOURCE_COMMIT' SOURCE_FINGERPRINT='$SOURCE_FINGERPRINT' ENV_SHA256='$ENV_SHA256' LOCK_TOKEN='$RELEASE_ID' bash -s" <<'REMOTE'
set -euo pipefail
lock_dir="$REMOTE_ROOT/.deploy-lock"
[[ -f "$lock_dir/token" && "$(cat "$lock_dir/token")" == "$LOCK_TOKEN" ]] || exit 1
state="$REMOTE_ROOT/state/current.env"
[[ -f "$state" ]] || exit 0
tmp="$(mktemp "$REMOTE_ROOT/state/current.XXXXXX")"
grep -Ev '^(RELEASE_ID|SOURCE_COMMIT|SOURCE_FINGERPRINT|ENV_SHA256|DEPLOYED_AT)=' "$state" > "$tmp"
printf 'RELEASE_ID=%s\nSOURCE_COMMIT=%s\nSOURCE_FINGERPRINT=%s\nENV_SHA256=%s\nDEPLOYED_AT=%s\n' \
  "$RELEASE_ID" "$SOURCE_COMMIT" "$SOURCE_FINGERPRINT" "$ENV_SHA256" "$(date -u +%Y-%m-%dT%H:%M:%SZ)" >> "$tmp"
chmod 600 "$tmp"
mv "$tmp" "$state"
REMOTE
}

rollback_release() {
  ssh_run "REMOTE_ROOT='$REMOTE_ROOT' BACKEND_PORT='$BACKEND_PORT' WEB_PORT='$WEB_PORT' LOCK_TOKEN='$RELEASE_ID' bash -s" <<'REMOTE'
set -Eeuo pipefail
lock_dir="$REMOTE_ROOT/.deploy-lock"
[[ -f "$lock_dir/token" && "$(cat "$lock_dir/token")" == "$LOCK_TOKEN" ]] || exit 1
state="$REMOTE_ROOT/state/current.env"
previous="$REMOTE_ROOT/state/previous.env"
[[ -f "$state" && -f "$previous" ]] || { echo "no recorded previous release" >&2; exit 1; }
backend="$(sed -n 's/^BACKEND_RELEASE=//p' "$previous")"
web="$(sed -n 's/^WEB_RELEASE=//p' "$previous")"
[[ -x "$backend/server" && -f "$web/apps/web/server.js" ]] || { echo "previous release artifacts are missing" >&2; exit 1; }
old_backend="$(readlink -f "$REMOTE_ROOT/current-backend")"
old_web="$(readlink -f "$REMOTE_ROOT/current-web")"
rollback_failed() {
  local rc=$?
  trap - ERR
  set +e
  ln -sfn "$old_backend" "$REMOTE_ROOT/current-backend"
  ln -sfn "$old_web" "$REMOTE_ROOT/current-web"
  systemctl restart chimii-backend chimii-web
  exit "$rc"
}
trap rollback_failed ERR
ln -sfn "$backend" "$REMOTE_ROOT/current-backend"
ln -sfn "$web" "$REMOTE_ROOT/current-web"
systemctl restart chimii-backend
for _ in $(seq 1 60); do curl -fsS "http://127.0.0.1:$BACKEND_PORT/healthz" >/dev/null 2>&1 && break; sleep 1; done
curl -fsS "http://127.0.0.1:$BACKEND_PORT/healthz" >/dev/null
systemctl restart chimii-web
for _ in $(seq 1 60); do curl -fsS "http://127.0.0.1:$WEB_PORT/" >/dev/null 2>&1 && break; sleep 1; done
curl -fsS "http://127.0.0.1:$WEB_PORT/" >/dev/null
tmp="$(mktemp "$REMOTE_ROOT/state/swap.XXXXXX")"
cp "$state" "$tmp"
cp "$previous" "$state"
mv "$tmp" "$previous"
trap - ERR
REMOTE
}

verify_deployment() {
  log "verify services, listeners, database, state, and HTTP routes"
  ssh_run "DOMAIN='$DOMAIN' REMOTE_ROOT='$REMOTE_ROOT' BACKEND_PORT='$BACKEND_PORT' WEB_PORT='$WEB_PORT' bash -s" <<'REMOTE'
set -euo pipefail
systemctl is-active --quiet postgresql redis-server nginx chimii-backend chimii-web
systemctl is-enabled --quiet postgresql redis-server nginx chimii-backend chimii-web
curl -fsS "http://127.0.0.1:$BACKEND_PORT/healthz" >/dev/null
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
test -x "$(readlink -f "$REMOTE_ROOT/current-backend")/server"
test -f "$(readlink -f "$REMOTE_ROOT/current-web")/apps/web/server.js"
printf 'backend=%s\nweb=%s\n' "$(readlink -f "$REMOTE_ROOT/current-backend")" "$(readlink -f "$REMOTE_ROOT/current-web")"
REMOTE

  if curl -fsS --max-time 15 --resolve "$DOMAIN:443:$SERVER_IP" "https://$DOMAIN/healthz" >/dev/null; then
    ok "external HTTPS route reaches sshmd"
  elif curl -fsS --max-time 10 --resolve "$DOMAIN:80:$SERVER_IP" "http://$DOMAIN/healthz" >/dev/null; then
    warn "external HTTP works but HTTPS verification failed"
  else
    warn "external route failed; check DNS, firewall, and security group"
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
certbot certonly --webroot --webroot-path "$acme_root" --non-interactive --agree-tos --no-eff-email --email "$TLS_EMAIL" --preferred-challenges http --key-type ecdsa -d "$DOMAIN"
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
ln -sfn "$site" /etc/nginx/sites-enabled/chimii.com.conf
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

deploy_selected_components() {
  if [[ "$DEPLOY_BACKEND" == false && "$DEPLOY_WEB" == false && "$DEPLOY_CONFIG" == false ]]; then
    log "no deployable changes; desktop/mobile/docs changes require no Linux deployment"
    run_stage verify verify_deployment
    run_stage record-state record_noop_state
    return 0
  fi
  [[ "$DEPLOY_BACKEND" == false ]] || run_stage build-backend build_backend_local
  if [[ "$DEPLOY_WEB" == true ]]; then
    run_stage typecheck-web typecheck_web_local
    run_stage package-web package_web_source
  fi
  [[ "$DEPLOY_CONFIG" == false ]] || run_stage package-config package_config
  run_stage upload upload_inputs
  [[ "$DEPLOY_WEB" == false ]] || run_stage build-web-linux build_web_remote
  [[ "$DEPLOY_CONFIG" == false ]] || run_stage configure configure_remote
  run_stage activate activate_release
  run_stage verify verify_deployment
}

main() {
  trap on_error ERR
  trap cleanup EXIT
  validate_config
  cd "$ROOT_DIR"
  case "$ACTION" in
    setup|auto|backend|web|config|full|rollback|tls) MUTATING=true ;;
    verify) MUTATING=false ;;
    *) die "usage: $0 [setup|auto|backend|web|config|full|rollback|verify|tls]" ;;
  esac
  [[ "$MUTATING" == false ]] || acquire_local_lock
  run_stage preflight preflight

  case "$ACTION" in
    setup)
      run_stage setup provision_remote
      init_release_context
      run_stage remote-lock acquire_remote_lock
      DEPLOY_CONFIG=true
      package_config
      run_stage upload-config upload_inputs
      run_stage configure configure_remote
      if ssh_run "test -L '$REMOTE_ROOT/current-backend' -a -L '$REMOTE_ROOT/current-web'"; then
        # Setup changes runtime configuration, but it does not deploy the local
        # source tree. Keep the source state unknown so the next auto run is full.
        SOURCE_COMMIT=unknown
        SOURCE_FINGERPRINT=unknown
        run_stage activate-config activate_release
        run_stage verify verify_deployment
      else
        warn "runtime configuration installed; deploy a first full release before verification"
      fi
      ;;
    auto|backend|web|config|full)
      init_release_context
      run_stage setup-check require_remote_setup
      run_stage remote-lock acquire_remote_lock
      read_remote_state
      plan_components
      deploy_selected_components
      ;;
    rollback)
      init_release_context
      run_stage setup-check require_remote_setup
      run_stage remote-lock acquire_remote_lock
      run_stage rollback rollback_release
      run_stage verify verify_deployment
      ;;
    verify)
      run_stage verify verify_deployment
      ;;
    tls)
      init_release_context
      run_stage tls enable_tls
      run_stage verify verify_deployment
      ;;
  esac
  REPORT_OUTCOME=success
  ok "action completed: $ACTION"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  main "$@"
fi
