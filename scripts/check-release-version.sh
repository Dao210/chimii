#!/usr/bin/env bash
# Check a proposed tag, or HEAD's exact tag, against the root release version.
# This is read-only and does not create tags or establish deployment readiness.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
fail() { printf '[release-version] ERROR %s\n' "$*" >&2; exit 1; }

(( $# <= 1 )) || fail 'usage: bash scripts/check-release-version.sh [vX.Y.Z]'
command -v node >/dev/null 2>&1 || fail 'node is required to read package.json'

if (( $# == 1 )); then
  release_tag="$1"
else
  release_tag="$(git -C "$ROOT_DIR" describe --tags --exact-match HEAD 2>/dev/null)" || fail 'HEAD has no exact release tag; pass a proposed tag to check before tagging'
fi

[[ "$release_tag" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$ && "$release_tag" != *-dirty* ]] || fail "invalid release tag: $release_tag"
package_version="$(node -p 'require(process.argv[1]).version' "$ROOT_DIR/package.json")" || fail 'could not read root package.json version'
[[ "$release_tag" == "v$package_version" ]] || fail "release tag and root package version must match: tag=$release_tag, package.json=$package_version, expected_tag=v$package_version"

printf '[release-version] OK tag=%s package.json=%s\n' "$release_tag" "$package_version"
