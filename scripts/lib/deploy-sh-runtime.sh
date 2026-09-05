#!/usr/bin/env bash
# Shared rendering and root-only remote operations for deploy-sh.sh.
set -Eeuo pipefail

sh_fail() { echo "[chimii-sh] $*" >&2; exit 1; }
sh_db() { runuser -u postgres -- psql -X -v ON_ERROR_STOP=1 -d "$DB_NAME" "$@"; }
sh_schema() { sh_db -Atqc 'SELECT version FROM schema_migrations ORDER BY version'; }
sh_expected_schema() { find "$1/migrations" -maxdepth 1 -name '*.up.sql' -printf '%f\n' | sed 's/\.up\.sql$//' | LC_ALL=C sort; }
sh_lock() { test "$(cat "$REMOTE_ROOT/.deploy-lock/token")" = "$LOCK_TOKEN"; }
sh_atomic_link() {
  ln -s "$1" "$2.next-$RELEASE_ID"
  mv -Tf "$2.next-$RELEASE_ID" "$2"
}

sh_render_caddy() {
  cat <<EOF
# BEGIN CHIMII-SH MANAGED
https://$PRIMARY_DOMAIN:$ORIGIN_HTTPS_PORT {
	tls $TLS_CERT_FILE $TLS_KEY_FILE
	encode zstd gzip
	@backend path /api/* /auth/* /uploads/* /ws /health /healthz /readyz
	handle @backend {
		reverse_proxy 127.0.0.1:$BACKEND_PORT {
			flush_interval -1
			transport http {
				read_timeout 24h
				write_timeout 24h
			}
		}
	}
	handle {
		reverse_proxy 127.0.0.1:$WEB_PORT
	}
}
# END CHIMII-SH MANAGED
EOF
}

sh_strip_caddy() {
  # Only remove this application's owned blocks. Preserve all unrelated sites.
  PRIMARY_DOMAIN="$PRIMARY_DOMAIN" ORIGIN_HTTPS_PORT="$ORIGIN_HTTPS_PORT" perl -0777 -pe '
    s/^# BEGIN CHIMII-SH MANAGED.*?^# END CHIMII-SH MANAGED\n?//msg;
    s/^# Cloudflare proxies public HTTPS 443 to this origin port\.\n//mg;
    my $domain = quotemeta($ENV{PRIMARY_DOMAIN});
    my $port = quotemeta($ENV{ORIGIN_HTTPS_PORT});
    s/^(?:https:\/\/)?$domain(?::$port)?\s*\{.*?^\}\n?//msg;
  ' "$1"
}

sh_cert_check() {
  test -s "$TLS_CERT_FILE" && test -s "$TLS_KEY_FILE"
  runuser -u caddy -- test -r "$TLS_CERT_FILE"
  runuser -u caddy -- test -r "$TLS_KEY_FILE"
  test "$(stat -c %a "$TLS_KEY_FILE")" = 600
  openssl x509 -in "$TLS_CERT_FILE" -noout -checkend 604800 >/dev/null
  openssl x509 -in "$TLS_CERT_FILE" -noout -checkhost "$PRIMARY_DOMAIN" | grep -F 'does match certificate' >/dev/null
  local cert_key private_key
  cert_key="$(openssl x509 -in "$TLS_CERT_FILE" -pubkey -noout | openssl pkey -pubin -outform DER | sha256sum)"
  private_key="$(openssl pkey -in "$TLS_KEY_FILE" -pubout -outform DER | sha256sum)"
  test "$cert_key" = "$private_key"
}

sh_firewall_check() (
  # Verify the existing policy against Cloudflare's published address lists.
  # This command audits rules; it does not widen server access.
  local check_dir
  check_dir="$(mktemp -d)"
  trap 'rm -rf -- "$check_dir"' EXIT
  curl -fsS --max-time 15 https://www.cloudflare.com/ips-v4 > "$check_dir/v4"
  curl -fsS --max-time 15 https://www.cloudflare.com/ips-v6 > "$check_dir/v6"
  ufw status > "$check_dir/ufw"
  grep -Fx 'Status: active' "$check_dir/ufw" >/dev/null
  awk -v port="$ORIGIN_HTTPS_PORT/tcp" 'NR==FNR {allowed[$1]=1; next} $1==port && /ALLOW/ {src=($2=="(v6)" ? $4 : $3); if (!allowed[src]) bad=1; found++} END{exit(bad || found==0)}' <(awk '{print}' "$check_dir/v4" "$check_dir/v6") "$check_dir/ufw"
  # A global incoming allow policy would bypass the per-port allowlist.
  ufw status verbose | grep -F 'deny (incoming)' >/dev/null
  rm -rf -- "$check_dir"
)

sh_config_prepare() {
  sh_lock
  sh_cert_check
  install -d -m 0700 "$REMOTE_ROOT/state/deployments/$RELEASE_ID"
  local staged="$REMOTE_ROOT/state/deployments/$RELEASE_ID/Caddyfile.next"
  sh_strip_caddy /etc/caddy/Caddyfile > "$staged"
  sh_render_caddy >> "$staged"
  caddy validate --adapter caddyfile --config "$staged" >/dev/null
}

sh_origin_check() {
  local served expected
  expected="$(openssl x509 -in "$TLS_CERT_FILE" -noout -fingerprint -sha256)"
  served="$(timeout 10 openssl s_client -connect "127.0.0.1:$ORIGIN_HTTPS_PORT" -servername "$PRIMARY_DOMAIN" </dev/null 2>/dev/null | openssl x509 -noout -fingerprint -sha256)"
  test "$served" = "$expected" || return 1
  # The served certificate is pinned above. Origin CA is not a browser root.
  curl -kfsS --max-time 15 --resolve "$PRIMARY_DOMAIN:$ORIGIN_HTTPS_PORT:127.0.0.1" "https://$PRIMARY_DOMAIN:$ORIGIN_HTTPS_PORT/readyz" | grep -F '"status":"ok"' >/dev/null
}

sh_config_apply() {
  sh_config_prepare
  local backup="$REMOTE_ROOT/state/deployments/$RELEASE_ID/Caddyfile.before"
  cp -a /etc/caddy/Caddyfile "$backup"
  install -m 0644 "$REMOTE_ROOT/state/deployments/$RELEASE_ID/Caddyfile.next" /etc/caddy/Caddyfile
  if ! systemctl reload caddy || ! (set -e; sh_origin_check); then
    install -m 0644 "$backup" /etc/caddy/Caddyfile
    systemctl reload caddy
    sh_fail 'Caddy change failed; previous configuration restored'
  fi
}

sh_preflight() {
  test "$REMOTE_ROOT" = /opt/chimii
  for cmd in caddy node corepack pg_dump pg_restore psql systemd-run curl openssl jq perl; do command -v "$cmd" >/dev/null; done
  test -L "$REMOTE_ROOT/current-backend" && test -L "$REMOTE_ROOT/current-web"
  test -f "$REMOTE_ROOT/state/current.env"
  test -d "$REMOTE_ROOT/builder/workspace"
  systemctl is-active --quiet postgresql redis-server caddy chimii-backend chimii-web
  local bytes free_kb
  bytes="$(sh_db -Atqc 'SELECT pg_database_size(current_database())')"
  free_kb="$(df -Pk "$REMOTE_ROOT" | awk 'NR==2 {print $4}')"
  (( free_kb > 3 * 1024 * 1024 + bytes / 1024 * 4 )) || sh_fail 'insufficient disk for builds, backups and migration rehearsal'
  for env_file in /etc/chimii/backend.env /etc/chimii/web.env /etc/chimii/secrets/runtime.env; do
    test -s "$env_file" && test "$(stat -c %a "$env_file")" = 600
  done
  sh_cert_check
  test "$(stat -c %a /etc/chimii/v2-postgres-password)" = 600
  for port in "$CANDIDATE_BACKEND_PORT" "$CANDIDATE_WEB_PORT"; do
    if ss -ltnH | awk '{print $4}' | grep -Eq ":$port$"; then sh_fail "candidate port $port is occupied"; fi
  done
  sh_db -Atqc 'SELECT current_database()' | grep -Fx "$DB_NAME" >/dev/null
  printf 'production_database_bytes=%s disk_free_kb=%s\n' "$bytes" "$free_kb"
}

sh_migrate() {
  local database="$1" backend="$2" db_password
  db_password="$(cat /etc/chimii/v2-postgres-password)"
  # Pass the password only via the environment; never include it in a command log.
  (cd "$backend"; runuser -u "$APP_USER" -- env DATABASE_URL="postgres://$DB_USER:$db_password@127.0.0.1:5432/$database?sslmode=disable" PGOPTIONS='-c lock_timeout=15s -c statement_timeout=600s' ./migrate up)
}

sh_rehearse() {
  sh_lock
  local rehearsal_db="chimii_rehearsal_${RELEASE_ID:0:16}"
  rehearsal_db="${rehearsal_db//[TZ]/}"
  [[ "$rehearsal_db" =~ ^chimii_rehearsal_[0-9]{14}$ ]]
  local rehearsal_dir="$REMOTE_ROOT/state/deployments/$RELEASE_ID"
  install -d -m 0700 "$rehearsal_dir"
  local old_backend migration_file
  old_backend="$(readlink -f "$REMOTE_ROOT/current-backend")"
  while IFS= read -r migration_file; do
    cmp "$old_backend/migrations/$migration_file.up.sql" "$REMOTE_ROOT/releases/backend/$RELEASE_ID/migrations/$migration_file.up.sql" || sh_fail "an applied migration was modified or removed: $migration_file"
  done < <(sh_schema)
  # This clone is only used by the migration executable: no workers or providers.
  runuser -u postgres -- pg_dump -Fc -d "$DB_NAME" > "$rehearsal_dir/rehearsal.dump"
  runuser -u postgres -- createdb -O "$DB_USER" "$rehearsal_db"
  printf '%s\n' "$rehearsal_db" > "$rehearsal_dir/rehearsal-db"
  runuser -u postgres -- pg_restore --exit-on-error -d "$rehearsal_db" < "$rehearsal_dir/rehearsal.dump"
  sh_migrate "$rehearsal_db" "$REMOTE_ROOT/releases/backend/$RELEASE_ID"
  sh_expected_schema "$REMOTE_ROOT/releases/backend/$RELEASE_ID" > "$rehearsal_dir/expected-schema"
  runuser -u postgres -- psql -X -d "$rehearsal_db" -Atqc 'SELECT version FROM schema_migrations ORDER BY version' > "$rehearsal_dir/rehearsed-schema"
  diff -u "$rehearsal_dir/expected-schema" "$rehearsal_dir/rehearsed-schema"
  runuser -u postgres -- dropdb "$rehearsal_db"
  rm -f "$rehearsal_dir/rehearsal-db" "$rehearsal_dir/rehearsal.dump"
}

sh_app_check() {
  local expected_version="$1" ready=false
  for _ in $(seq 1 60); do
    if curl -fsS --max-time 3 "http://127.0.0.1:$BACKEND_PORT/readyz" 2>/dev/null | grep -F '"status":"ok"' >/dev/null; then ready=true; break; fi
    sleep 1
  done
  test "$ready" = true || return 1
  curl -fsS --max-time 10 "http://127.0.0.1:$BACKEND_PORT/api/config" | jq -e --arg v "$expected_version" '.server_version == $v' >/dev/null || return 1
  curl -fsS --retry 10 --retry-connrefused --retry-delay 1 --max-time 10 "http://127.0.0.1:$WEB_PORT/login" >/dev/null || return 1
}

sh_public_check() {
  local expected_version="$1" route_name
  for route_name in / /login /readyz /api/config; do
    test "$(curl -fsS --retry 2 --retry-delay 2 --connect-timeout 5 --max-time 15 "https://$PRIMARY_DOMAIN$route_name" -o /dev/null -w '%{http_code}')" = 200
  done
  curl -fsS --max-time 15 "https://$PRIMARY_DOMAIN/api/config" | jq -e --arg v "$expected_version" '.server_version == $v' >/dev/null
}

sh_start_apps() {
  systemctl start chimii-backend.service || return 1
  systemctl start chimii-web.service || return 1
}

sh_restore_apps() {
  systemctl stop chimii-web.service chimii-backend.service || return 1
  sh_atomic_link "$(cat "$deploy_dir/backend.before")" "$REMOTE_ROOT/current-backend" || return 1
  sh_atomic_link "$(cat "$deploy_dir/web.before")" "$REMOTE_ROOT/current-web" || return 1
  cp -a "$deploy_dir/current.before" "$REMOTE_ROOT/state/current.env" || return 1
  sh_start_apps || return 1
  sh_app_check "$(sed -n 's/^VERSION=//p' "$deploy_dir/current.before")" || return 1
  install -m 0644 "$deploy_dir/Caddyfile.before" /etc/caddy/Caddyfile || return 1
  systemctl reload caddy || return 1
}

sh_finish_cutover() {
  local rc=$?
  trap - EXIT
  set +e
  if (( rc != 0 )); then
    echo "cutover failed (stage=$deploy_stage); backup=$deploy_dir" >&2
    # Migrations run outside transactions. A partial failure cannot be inferred
    # safe from schema_migrations. Never restart old code over an unknown schema.
    if [[ "$migration_started" == true && "$schema_changed" == true ]]; then
      systemctl stop chimii-web.service chimii-backend.service
      install -m 0644 "$deploy_dir/Caddyfile.maintenance" /etc/caddy/Caddyfile
      systemctl reload caddy
      touch "$deploy_dir/needs-recovery"
      echo 'manual database recovery required; maintenance and deployment lock retained' >&2
    elif [[ "$traffic_paused" == true ]]; then
      if (set -e; sh_restore_apps); then
        echo 'previous application release restored' >&2
      else
        touch "$deploy_dir/needs-recovery"
      fi
    fi
  fi
  printf '%s\n' "$rc" > "$deploy_dir/exit-code"
  if [[ ! -f "$deploy_dir/needs-recovery" ]] && [[ "$(cat "$REMOTE_ROOT/.deploy-lock/token" 2>/dev/null)" == "$LOCK_TOKEN" ]]; then
    rm -f "$REMOTE_ROOT/.deploy-lock/token" "$REMOTE_ROOT/.deploy-lock/started-at"
    rmdir "$REMOTE_ROOT/.deploy-lock"
  fi
  exit "$rc"
}

sh_cutover() {
  sh_lock
  deploy_dir="$REMOTE_ROOT/state/deployments/$RELEASE_ID"
  deploy_stage=preflight
  traffic_paused=false
  migration_started=false
  schema_changed=false
  install -d -m 0700 "$deploy_dir"
  trap sh_finish_cutover EXIT
  sh_preflight
  readlink -f "$REMOTE_ROOT/current-backend" > "$deploy_dir/backend.before"
  readlink -f "$REMOTE_ROOT/current-web" > "$deploy_dir/web.before"
  cp -a "$REMOTE_ROOT/state/current.env" "$deploy_dir/current.before"
  cp -a /etc/caddy/Caddyfile "$deploy_dir/Caddyfile.before"
  cp -a /etc/chimii "$deploy_dir/etc-chimii"
  sh_db -Atqc 'SELECT oid FROM pg_database WHERE datname=current_database()' > "$deploy_dir/database-oid.before"
  sh_schema > "$deploy_dir/schema.before"
  sh_expected_schema "$REMOTE_ROOT/releases/backend/$RELEASE_ID" > "$deploy_dir/schema.expected"
  if ! cmp -s "$deploy_dir/schema.before" "$deploy_dir/schema.expected"; then schema_changed=true; fi
  sh_config_prepare
  # An existing drop-in has the same purpose; repeated EnvironmentFile entries
  # are harmless, and the runtime file remains the final override.
  install -d -m 0755 /etc/systemd/system/chimii-backend.service.d
  printf '%s\n' '[Service]' 'EnvironmentFile=-/etc/chimii/secrets/runtime.env' > /etc/systemd/system/chimii-backend.service.d/10-runtime-secrets.conf
  systemctl daemon-reload
  sh_strip_caddy /etc/caddy/Caddyfile > "$deploy_dir/Caddyfile.maintenance"
  cat >> "$deploy_dir/Caddyfile.maintenance" <<EOF
https://$PRIMARY_DOMAIN:$ORIGIN_HTTPS_PORT {
	tls $TLS_CERT_FILE $TLS_KEY_FILE
	header Retry-After 60
	respond "Chimii deployment in progress" 503
}
EOF
  caddy validate --adapter caddyfile --config "$deploy_dir/Caddyfile.maintenance" >/dev/null
  deploy_stage=maintenance
  traffic_paused=true
  install -m 0644 "$deploy_dir/Caddyfile.maintenance" /etc/caddy/Caddyfile
  systemctl reload caddy
  systemctl stop chimii-web.service chimii-backend.service
  # A dedicated app role is required: other writers make a stopped-app backup unsafe.
  sh_db -Atqc "SELECT count(*) FROM pg_stat_activity WHERE datname=current_database() AND usename='$DB_USER'" | grep -Fx 0 >/dev/null
  deploy_stage=backup
  runuser -u postgres -- pg_dump -Fc -d "$DB_NAME" > "$deploy_dir/$DB_NAME.dump"
  pg_restore --list "$deploy_dir/$DB_NAME.dump" > "$deploy_dir/backup.list"
  sha256sum "$deploy_dir/$DB_NAME.dump" > "$deploy_dir/$DB_NAME.dump.sha256"
  deploy_stage=migration
  migration_started=true
  sh_migrate "$DB_NAME" "$REMOTE_ROOT/releases/backend/$RELEASE_ID"
  sh_schema > "$deploy_dir/schema.after"
  sh_db -Atqc 'SELECT oid FROM pg_database WHERE datname=current_database()' > "$deploy_dir/database-oid.after"
  cmp "$deploy_dir/database-oid.before" "$deploy_dir/database-oid.after"
  diff -u "$deploy_dir/schema.expected" "$deploy_dir/schema.after"
  deploy_stage=activation
  sh_atomic_link "$REMOTE_ROOT/releases/backend/$RELEASE_ID" "$REMOTE_ROOT/current-backend"
  sh_atomic_link "$REMOTE_ROOT/releases/web/$RELEASE_ID" "$REMOTE_ROOT/current-web"
  sh_start_apps
  sh_app_check "$VERSION"
  deploy_stage=public-verification
  install -m 0644 "$deploy_dir/Caddyfile.next" /etc/caddy/Caddyfile
  systemctl reload caddy
  sh_origin_check
  sh_public_check "$VERSION"
  deploy_stage=record
  cp -a "$deploy_dir/current.before" "$REMOTE_ROOT/state/previous.env"
  printf 'RELEASE_ID=%s\nVERSION=%s\nSOURCE_COMMIT=%s\nBACKEND=%s\nWEB=%s\nDEPLOYED_AT=%s\n' "$RELEASE_ID" "$VERSION" "$SOURCE_COMMIT" "$REMOTE_ROOT/releases/backend/$RELEASE_ID" "$REMOTE_ROOT/releases/web/$RELEASE_ID" "$(date -u +%FT%TZ)" > "$deploy_dir/current.next"
  install -m 0600 "$deploy_dir/current.next" "$REMOTE_ROOT/state/current.env"
  cp -a "$REMOTE_ROOT/state/current.env" "$REMOTE_ROOT/releases/manifests/$RELEASE_ID.env"
  touch "$REMOTE_ROOT/state/bootstrap.complete"
  deploy_stage=complete
  echo "release=$RELEASE_ID version=$VERSION database=$DB_NAME origin_port=$ORIGIN_HTTPS_PORT"
}

sh_start_cutover() {
  sh_lock
  local job_dir="$REMOTE_ROOT/state/deployments/$RELEASE_ID" setting
  install -d -m 0700 "$job_dir"
  install -m 0700 "$REMOTE_ROOT/incoming/$RELEASE_ID/deploy-sh-runtime.sh" "$job_dir/runner.sh"
  : > "$job_dir/runner.env"
  for setting in REMOTE_ROOT RELEASE_ID LOCK_TOKEN VERSION SOURCE_COMMIT DB_NAME DB_USER APP_USER PRIMARY_DOMAIN BACKEND_PORT WEB_PORT CANDIDATE_BACKEND_PORT CANDIDATE_WEB_PORT ORIGIN_HTTPS_PORT TLS_CERT_FILE TLS_KEY_FILE; do
    printf '%s=%s\n' "$setting" "${!setting}" >> "$job_dir/runner.env"
  done
  chmod 0600 "$job_dir/runner.env"
  systemd-run --quiet --unit="chimii-sh-cutover-${RELEASE_ID//./-}" --property=Type=exec --property="EnvironmentFile=$job_dir/runner.env" /bin/bash "$job_dir/runner.sh" cutover
}

sh_rollback() {
  sh_lock
  local previous="$REMOTE_ROOT/state/previous.env" target_backend target_web target_version
  test -f "$previous" || sh_fail 'no previous daily release recorded'
  target_backend="$(sed -n 's/^BACKEND=//p' "$previous")"
  target_web="$(sed -n 's/^WEB=//p' "$previous")"
  target_version="$(sed -n 's/^VERSION=//p' "$previous")"
  [[ "$target_backend" == "$REMOTE_ROOT/releases/backend/"* && "$target_web" == "$REMOTE_ROOT/releases/web/"* ]]
  test -x "$target_backend/server" && test -f "$target_web/apps/web/server.js"
  # Application rollback is allowed only when both releases have the same schema.
  diff -u <(sh_schema) <(sh_expected_schema "$target_backend") || sh_fail 'schema changed; an application-only rollback is unsafe'
  deploy_dir="$REMOTE_ROOT/state/deployments/$RELEASE_ID"
  install -d -m 0700 "$deploy_dir"
  readlink -f "$REMOTE_ROOT/current-backend" > "$deploy_dir/backend.before"
  readlink -f "$REMOTE_ROOT/current-web" > "$deploy_dir/web.before"
  cp -a "$REMOTE_ROOT/state/current.env" "$deploy_dir/current.before"
  cp -a /etc/caddy/Caddyfile "$deploy_dir/Caddyfile.before"
  traffic_paused=true; migration_started=false; schema_changed=false; deploy_stage=rollback
  trap sh_finish_cutover EXIT
  systemctl stop chimii-web.service chimii-backend.service
  sh_atomic_link "$target_backend" "$REMOTE_ROOT/current-backend"
  sh_atomic_link "$target_web" "$REMOTE_ROOT/current-web"
  sh_start_apps
  sh_app_check "$target_version"
  sh_origin_check
  sh_public_check "$target_version"
  cp -a "$previous" "$REMOTE_ROOT/state/current.env"
  cp -a "$deploy_dir/current.before" "$previous"
}

sh_cleanup_candidates() {
  # Only this invocation's timestamp-named databases may be removed.
  local suffix="${RELEASE_ID:0:16}" candidate_name
  suffix="${suffix//[TZ]/}"
  [[ "$suffix" =~ ^[0-9]{14}$ ]] || return 0
  systemctl stop "chimii-candidate-web-${RELEASE_ID//./-}.service" "chimii-candidate-backend-${RELEASE_ID//./-}.service" 2>/dev/null || true
  for candidate_name in "chimii_candidate_$suffix" "chimii_rehearsal_$suffix"; do
    runuser -u postgres -- dropdb --if-exists "$candidate_name"
  done
}

sh_prune() {
  test "$REMOTE_ROOT" = /opt/chimii
  local component candidate name kept protected
  for component in backend web; do
    kept=0
    while IFS= read -r candidate; do
      name="${candidate##*/}"
      [[ "$name" =~ ^[0-9]{8}T[0-9]{6}Z-[0-9]+\.[0-9]+\.[0-9]+-[0-9a-f]{12}$ ]] || continue
      test -f "$REMOTE_ROOT/releases/manifests/$name.env" || continue
      kept=$((kept + 1))
      (( kept > KEEP_RELEASES )) || continue
      [[ "$candidate" != "$(readlink -f "$REMOTE_ROOT/current-$component")" ]] || continue
      if grep -Fqx "${component^^}=$candidate" "$REMOTE_ROOT/state/previous.env"; then continue; fi
      protected="$(find "$REMOTE_ROOT/state/legacy" -maxdepth 2 -name '*.link' -exec grep -lFx "$candidate" {} \;)"
      [[ -z "$protected" ]] || continue
      rm -rf -- "$candidate"
      echo "pruned release artifacts: $candidate (database backups retained)"
    done < <(find "$REMOTE_ROOT/releases/$component" -mindepth 1 -maxdepth 1 -type d | LC_ALL=C sort -r)
  done
  [[ "$RELEASE_ID" =~ ^[0-9]{8}T[0-9]{6}Z-[0-9]+\.[0-9]+\.[0-9]+-[0-9a-f]{12}$ ]] || return 0
  rm -rf -- "$REMOTE_ROOT/incoming/$RELEASE_ID"
}

if [[ "${BASH_SOURCE[0]:-}" == "$0" || "$0" == bash && -n "${SH_REMOTE_ACTION:-}" ]]; then
  umask 077
  case "${SH_REMOTE_ACTION:-${1:-}}" in
    preflight) sh_preflight ;;
    rehearse) sh_rehearse ;;
    config-prepare) sh_config_prepare ;;
    config-apply) sh_config_apply ;;
    origin-check) sh_cert_check; sh_origin_check ;;
    public-check) sh_public_check "$VERSION" ;;
    firewall-check) sh_firewall_check ;;
    start-cutover) sh_start_cutover ;;
    cutover) sh_cutover ;;
    rollback) sh_rollback ;;
    cleanup-candidates) sh_cleanup_candidates ;;
    prune) sh_prune ;;
    *) sh_fail 'invalid remote deployment action' ;;
  esac
fi
