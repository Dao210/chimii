#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/deploy-sh.sh"
fixture_dir="$(mktemp -d)"
trap 'rm -rf -- "$fixture_dir"' EXIT
REMOTE_ROOT="$fixture_dir/remote"
mkdir -p "$REMOTE_ROOT/state"

# Execute the actual remote verification body with local service fixtures.
# A separate bash preserves errexit when the test expects a failed check.
remote_root() { bash -e -s; }
remote_user() { sed -n 's/^VERSION=//p' "$REMOTE_ROOT/state/current.env"; }
daily_remote() { printf '%s\n' "$1" >> "$fixture_dir/checks"; }
systemctl() { :; }
curl() {
  case "$*" in
    */readyz) printf '{"status":"ok"}\n' ;;
    */api/config) printf '{"server_version":"%s"}\n' "$fixture_server_version" ;;
  esac
}
ss() { printf '127.0.0.1:%s\n' "$BACKEND_PORT" "$WEB_PORT" 6379; }
redis-cli() { echo PONG; }
runuser() { echo 1; }
stat() { echo 600; }
caddy() { :; }
readlink() { printf '%s\n' "$2"; }
export REMOTE_ROOT DB_NAME BACKEND_PORT WEB_PORT PUBLIC_ROUTE PRIMARY_DOMAIN fixture_dir
export -f verify_deployment remote_root remote_user daily_remote systemctl curl ss redis-cli runuser stat caddy readlink

check_verification() {
  local state_version="$1" expected_result="$3" actual_result=0
  export fixture_server_version="$2"
  printf 'RELEASE_ID=fixture\nVERSION=%s\n' "$state_version" > "$REMOTE_ROOT/state/current.env"
  : > "$fixture_dir/checks"
  bash -e -c verify_deployment > "$fixture_dir/output" 2>&1 || actual_result=$?
  if [[ "$expected_result" == pass ]]; then
    if (( actual_result != 0 )); then
      cat "$fixture_dir/output" >&2
      echo "verification rejected supported deployment version: $state_version" >&2
      exit 1
    fi
    grep -Fx "version=$state_version" "$fixture_dir/output" >/dev/null
    test "$(< "$fixture_dir/checks")" = $'origin-check\nfirewall-check\npublic-check'
  else
    (( actual_result != 0 )) || { echo "verification accepted invalid deployment: $state_version / $fixture_server_version" >&2; exit 1; }
    test ! -s "$fixture_dir/checks"
  fi
}

check_verification v0.3.0 v0.3.0 pass
check_verification v0.3.0-dffd6b9 v0.3.0-dffd6b9 pass
# Git can lengthen its requested seven-character abbreviation for uniqueness.
check_verification v0.3.0-dffd6b91 v0.3.0-dffd6b91 pass
check_verification v0.3.0-rc.1 v0.3.0-rc.1 fail
check_verification v0.3.0-invalid v0.3.0-invalid fail
check_verification v0.3.0-dffd6b9 v0.3.0-c9ecc7c fail
check_verification v0.3.0 v0.2.9 fail

echo 'deployment-verification tests passed: stable tags, commit builds, invalid versions, runtime mismatch'
