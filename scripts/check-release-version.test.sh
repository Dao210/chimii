#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
fixture_dir="$(mktemp -d)"
trap 'rm -rf -- "$fixture_dir"' EXIT
fixture_root="$fixture_dir/repo with spaces"
mkdir -p "$fixture_root/scripts"
cp "$SCRIPT_DIR/check-release-version.sh" "$fixture_root/scripts/"
check_script="$fixture_root/scripts/check-release-version.sh"

printf '{"version":"0.2.10"}\n' > "$fixture_root/package.json"
if bash "$check_script" v0.3.0 > "$fixture_dir/mismatch" 2>&1; then
  echo 'accepted mismatched release metadata' >&2; exit 1
fi
grep -F 'tag=v0.3.0, package.json=0.2.10, expected_tag=v0.2.10' "$fixture_dir/mismatch" >/dev/null

printf '{"version":"0.3.0"}\n' > "$fixture_root/package.json"
# Invocation outside the checkout must still read the repository root version.
(cd "$fixture_dir"; bash "$check_script" v0.3.0) >/dev/null
for invalid_tag in '' v0.2.10 0.3.0 v0.3 v0.3.0-dirty; do
  if bash "$check_script" "$invalid_tag" >/dev/null 2>&1; then
    echo "accepted invalid or mismatched tag: $invalid_tag" >&2; exit 1
  fi
done
printf '{"version":"0.3.0-rc.1"}\n' > "$fixture_root/package.json"
bash "$check_script" v0.3.0-rc.1 >/dev/null

printf '{"version":"0.3.0"}\n' > "$fixture_root/package.json"
git -C "$fixture_root" init -q
git -C "$fixture_root" add package.json
git -C "$fixture_root" -c user.name=ReleaseTest -c user.email=release-test@example.invalid -c commit.gpgsign=false -c core.hooksPath=/dev/null commit -qm fixture
if bash "$check_script" >/dev/null 2>&1; then
  echo 'accepted an untagged HEAD' >&2; exit 1
fi
git -C "$fixture_root" -c tag.gpgSign=false tag v0.3.0
bash "$check_script" >/dev/null
git -C "$fixture_root" -c user.name=ReleaseTest -c user.email=release-test@example.invalid -c commit.gpgsign=false -c core.hooksPath=/dev/null commit --allow-empty -qm next
if bash "$check_script" >/dev/null 2>&1; then
  echo 'accepted a tag on a previous commit' >&2; exit 1
fi

printf 'release-version tests passed: mismatch, diagnostics, paths, prereleases, exact tag\n'
