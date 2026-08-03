#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=deploy.sh
source "$SCRIPT_DIR/deploy.sh"

assert_plan() {
  local expected_backend="$1" expected_web="$2" paths="$3"
  DEPLOY_BACKEND=false
  DEPLOY_WEB=false
  classify_changed_paths <<< "$paths"
  [[ "$DEPLOY_BACKEND" == "$expected_backend" ]] || {
    echo "expected backend=$expected_backend for: $paths" >&2
    exit 1
  }
  [[ "$DEPLOY_WEB" == "$expected_web" ]] || {
    echo "expected web=$expected_web for: $paths" >&2
    exit 1
  }
}

assert_plan true false 'server/internal/http/router.go'
assert_plan false true 'apps/web/app/page.tsx'
assert_plan false true 'packages/core/api/client.ts'
assert_plan false false $'apps/desktop/main.ts\napps/mobile/app.tsx\ndocs/guide.md'
assert_plan false false $'scripts/deploy.sh\nREADME.md'
assert_plan true true 'unclassified-root-file.txt'
assert_plan true true '__FULL__'

echo 'deploy planner tests passed'
