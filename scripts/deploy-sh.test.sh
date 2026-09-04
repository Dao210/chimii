#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=deploy-sh.sh
source "$SCRIPT_DIR/deploy-sh.sh"

validate_config
[[ "$SSH_HOST" == sh ]]
[[ "$REMOTE_ROOT" == /opt/chimii ]]
[[ "$DB_NAME" == chimii ]]
[[ "$BACKEND_PORT" == 8080 ]]
[[ "$WEB_PORT" == 3000 ]]
[[ "$PUBLIC_ROUTE" == false ]]
[[ -z "$(render_caddy_block)" ]]

PUBLIC_ROUTE=true
block="$(render_caddy_block)"
grep -F 'chimii.com {' <<< "$block" >/dev/null
grep -F 'reverse_proxy 127.0.0.1:8080' <<< "$block" >/dev/null
grep -F 'reverse_proxy 127.0.0.1:3000' <<< "$block" >/dev/null

release_id='20260904T043730Z-0.2.4-f58191142b9f'
db_suffix="$(printf '%s' "$release_id" | cut -c1-16 | tr -cd '0-9')"
[[ "$db_suffix" == 20260904043730 ]]
[[ "chimii_candidate_$db_suffix" =~ ^chimii_candidate_[0-9]+$ ]]
[[ "chimii_legacy_$db_suffix" =~ ^chimii_legacy_[0-9]+$ ]]

printf 'deploy-sh tests passed\n'
