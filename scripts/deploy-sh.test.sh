#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# Sourcing the entrypoint must never run a deployment.
source "$SCRIPT_DIR/deploy-sh.sh"

if [[ "${1:-}" == failure-fixture ]]; then
  REMOTE_ROOT="$2"
  RELEASE_ID=20260905T010000Z-0.2.8-123456abcdef
  LOCK_TOKEN="$RELEASE_ID"
  deploy_dir="$REMOTE_ROOT/state/deployments/$RELEASE_ID"
  deploy_stage=migration
  traffic_paused=true
  migration_started=true
  schema_changed="$3"
  systemctl() { printf 'systemctl %s\n' "$*" >> "$REMOTE_ROOT/events"; }
  install() { printf 'install %s\n' "$*" >> "$REMOTE_ROOT/events"; }
  sh_restore_apps() { echo restored >> "$REMOTE_ROOT/events"; }
  trap sh_finish_cutover EXIT
  exit 37
fi

bash "$SCRIPT_DIR/check-release-version.test.sh"
bash "$SCRIPT_DIR/deploy-sh-version.test.sh"
bash "$SCRIPT_DIR/deploy-sh-verify.test.sh"
validate_config
[[ "$ACTION" == deploy && "$SSH_HOST" == sh && "$REMOTE_ROOT" == /opt/chimii ]]
[[ "$DB_NAME" == chimii && "$BACKEND_PORT" == 8080 && "$WEB_PORT" == 3000 ]]
[[ "$PUBLIC_ROUTE" == true && "$ORIGIN_HTTPS_PORT" == 32443 ]]

fixture_dir="$(mktemp -d)"
trap 'rm -rf -- "$fixture_dir"' EXIT
# Existing manual 32443 + managed 443 must converge to one managed block.
printf '%s\n' \
  'other.example.com {' ' reverse_proxy 127.0.0.1:9900' '}' \
  '# BEGIN CHIMII-SH MANAGED' 'chimii.com {' ' respond "old"' '}' '# END CHIMII-SH MANAGED' \
  '# Cloudflare proxies public HTTPS 443 to this origin port.' \
  'https://chimii.com:32443 {' ' handle {' '  respond "manual"' ' }' '}' > "$fixture_dir/Caddyfile"
sh_strip_caddy "$fixture_dir/Caddyfile" > "$fixture_dir/first"
render_caddy_block >> "$fixture_dir/first"
sh_strip_caddy "$fixture_dir/first" > "$fixture_dir/second"
render_caddy_block >> "$fixture_dir/second"
cmp "$fixture_dir/first" "$fixture_dir/second"
[[ "$(grep -c '^https://chimii.com:32443 {' "$fixture_dir/second")" == 1 ]]
grep -Fx 'other.example.com {' "$fixture_dir/second" >/dev/null
grep -F 'reverse_proxy 127.0.0.1:9900' "$fixture_dir/second" >/dev/null
grep -F 'tls /etc/caddy/certs/chimii.com.pem /etc/caddy/certs/chimii.com.key' "$fixture_dir/second" >/dev/null
grep -F 'reverse_proxy 127.0.0.1:8080' "$fixture_dir/second" >/dev/null
grep -F 'reverse_proxy 127.0.0.1:3000' "$fixture_dir/second" >/dev/null
! grep -q 'respond "manual"\|respond "old"' "$fixture_dir/second"

if (BACKEND_PORT=32443; validate_config) 2>/dev/null; then echo 'accepted colliding ports' >&2; exit 1; fi
if (ORIGIN_HTTPS_PORT=99999; validate_config) 2>/dev/null; then echo 'accepted invalid port' >&2; exit 1; fi
if (REMOTE_ROOT=/; validate_config) 2>/dev/null; then echo 'accepted broad deployment root' >&2; exit 1; fi

# Recovery callers run under conditionals, where Bash disables implicit errexit.
(
  systemctl() { return 19; }
  if sh_start_apps; then exit 1; fi
)

# A partial migration retains the lock and never restarts old code.
for changed in true false; do
  fixture_root="$fixture_dir/$changed"
  fixture_release=20260905T010000Z-0.2.8-123456abcdef
  mkdir -p "$fixture_root/state/deployments/$fixture_release" "$fixture_root/.deploy-lock"
  printf '%s\n' "$fixture_release" > "$fixture_root/.deploy-lock/token"
  : > "$fixture_root/.deploy-lock/started-at"
  fixture_rc=0
  bash "$0" failure-fixture "$fixture_root" "$changed" > "$fixture_root/log" 2>&1 || fixture_rc=$?
  [[ "$fixture_rc" == 37 ]]
  [[ "$(< "$fixture_root/state/deployments/$fixture_release/exit-code")" == 37 ]]
  if [[ "$changed" == true ]]; then
    test -f "$fixture_root/state/deployments/$fixture_release/needs-recovery"
    test -f "$fixture_root/.deploy-lock/token"
    ! grep -q '^restored$' "$fixture_root/events"
  else
    grep -Fx restored "$fixture_root/events" >/dev/null
    test ! -d "$fixture_root/.deploy-lock"
  fi
done

# Default dispatch must use daily deploy, never the legacy replacement.
(
  cleanup() { :; }
  acquire_local_lock() { :; }
  resolve_deploy_version() { :; }
  verify_target() { :; }
  deploy_daily() { echo daily > "$fixture_dir/dispatch"; }
  replace_legacy() { exit 99; }
  main
)
[[ "$(< "$fixture_dir/dispatch")" == daily ]]

# An identical commit with a newly added release tag must refresh its version.
for scenario in same-version promoted-tag forced; do
  (
    SOURCE_COMMIT=same-commit
    VERSION=v0.3.0
    FORCE_DEPLOY=false
    [[ "$scenario" != forced ]] || FORCE_DEPLOY=true
    SKIP_LOCAL_CHECKS=true
    init_release() { :; }
    acquire_remote_lock() { :; }
    daily_remote() { :; }
    remote_user() {
      printf 'SOURCE_COMMIT=same-commit\n'
      if [[ "$scenario" == promoted-tag ]]; then
        printf 'VERSION=v0.3.0-1234567\n'
      else
        printf 'VERSION=v0.3.0\n'
      fi
    }
    verify_deployment() { echo verified > "$fixture_dir/$scenario"; }
    build_backend() { echo build-required > "$fixture_dir/$scenario"; exit 0; }
    deploy_daily
  )
  if [[ "$scenario" == same-version ]]; then
    [[ "$(< "$fixture_dir/$scenario")" == verified ]]
  else
    [[ "$(< "$fixture_dir/$scenario")" == build-required ]]
  fi
done

# A failed rehearsal stops before candidate startup or production cutover.
(
  SKIP_LOCAL_CHECKS=true
  SOURCE_COMMIT=new-commit
  FORCE_DEPLOY=false
  init_release() { :; }
  acquire_remote_lock() { :; }
  build_backend() { :; }
  package_web() { :; }
  upload_artifacts() { :; }
  install_backend_release() { :; }
  build_web_remote() { :; }
  remote_user() { echo previous-commit; }
  daily_remote() {
    printf '%s\n' "$1" >> "$fixture_dir/stages"
    [[ "$1" != rehearse ]] || exit 47
  }
  prepare_candidate() { echo unsafe >> "$fixture_dir/stages"; }
  export fixture_dir SKIP_LOCAL_CHECKS SOURCE_COMMIT FORCE_DEPLOY
  export -f deploy_daily init_release acquire_remote_lock build_backend package_web upload_artifacts install_backend_release build_web_remote daily_remote prepare_candidate remote_user
  bash -e -c deploy_daily
) && fixture_rc=0 || fixture_rc=$?
[[ "$fixture_rc" == 47 ]]
[[ "$(< "$fixture_dir/stages")" == $'preflight\nrehearse' ]]

printf 'deploy-sh tests passed: dispatch, Caddy convergence, ports, recovery, migration gating, version-aware no-op\n'
