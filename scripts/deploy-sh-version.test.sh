#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/deploy-sh.sh"
fixture_dir="$(mktemp -d)"
trap 'rm -rf -- "$fixture_dir"' EXIT
fixture_root="$fixture_dir/repo with spaces"
mkdir -p "$fixture_root/scripts"
cp "$SCRIPT_DIR/check-release-version.sh" "$fixture_root/scripts/"
printf '{"version":"0.3.0"}\n' > "$fixture_root/package.json"
git -C "$fixture_root" init -q -b main
fixture_git() {
  git -C "$fixture_root" -c user.name=DeployTest -c user.email=deploy-test@example.invalid \
    -c commit.gpgsign=false -c tag.gpgSign=false -c core.hooksPath=/dev/null "$@"
}
fixture_git add .
fixture_git commit -qm initial

assert_version() {
  (
    ROOT_DIR="$fixture_root"
    resolve_deploy_version >/dev/null
    [[ "$VERSION" == "$1" ]]
    [[ "$SOURCE_COMMIT" == "$(fixture_git rev-parse HEAD)" ]]
  )
  [[ -z "$(fixture_git status --porcelain)" ]]
}
reject_version() {
  if (ROOT_DIR="$fixture_root"; resolve_deploy_version) > "$fixture_dir/output" 2>&1; then
    echo "accepted unsafe deployment source: $1" >&2; exit 1
  fi
  grep -F "$1" "$fixture_dir/output" >/dev/null
}

# A clean untagged commit needs no tags, remote, version bump, or Git mutation.
initial_commit="$(fixture_git rev-parse HEAD)"
assert_version "v0.3.0-$(fixture_git rev-parse --short=7 HEAD)"
assert_version "v0.3.0-$(fixture_git rev-parse --short=7 HEAD)"
[[ "$(fixture_git rev-parse HEAD)" == "$initial_commit" && -z "$(fixture_git tag --list)" ]]

# Both lightweight and annotated stable tags are supported.
fixture_git tag v0.3.0
assert_version v0.3.0
fixture_git tag -d v0.3.0 >/dev/null
fixture_git tag -a v0.3.0 -m release
assert_version v0.3.0
fixture_git tag v0.2.9
reject_version 'tag=v0.2.9, package.json=0.3.0'
fixture_git tag -d v0.2.9 >/dev/null

# New commits do not reuse an ancestor's release label.
fixture_git commit --allow-empty -qm next
commit_version="v0.3.0-$(fixture_git rev-parse --short=7 HEAD)"
assert_version "$commit_version"
fixture_git tag checkpoint
assert_version "$commit_version"
for invalid_tag in v0.3.0-rc.1 v0.3.0-dirty v0.3; do
  fixture_git tag "$invalid_tag"
  reject_version 'requires a stable release tag'
  fixture_git tag -d "$invalid_tag" >/dev/null
done

printf 'untracked\n' > "$fixture_root/new.txt"
reject_version 'requires a clean worktree'
grep -F 'new.txt' "$fixture_dir/output" >/dev/null
fixture_git add new.txt
reject_version 'requires a clean worktree'
fixture_git commit -qm tracked
printf 'modified\n' >> "$fixture_root/new.txt"
reject_version 'requires a clean worktree'
fixture_git add new.txt
fixture_git commit -qm modified
fixture_git switch -qc feature
reject_version 'requires branch main'
fixture_git switch -q main
fixture_git switch -q --detach
reject_version 'requires branch main'
fixture_git switch -q main
printf '{"version":"0.3.0-rc.1"}\n' > "$fixture_root/package.json"
fixture_git add package.json
fixture_git commit -qm prerelease
reject_version 'root package version must be stable'

# The command refuses dirty input before opening any SSH connection.
printf '{"version":"0.3.0"}\n' > "$fixture_root/package.json"
fixture_git add package.json
fixture_git commit -qm stable
printf 'untracked\n' > "$fixture_root/blocked.txt"
if (
  ROOT_DIR="$fixture_root"
  ACTION=deploy
  cleanup() { :; }
  acquire_local_lock() { :; }
  verify_target() { echo unexpected-ssh > "$fixture_dir/ssh"; }
  deploy_daily() { echo unexpected-deploy > "$fixture_dir/deploy"; }
  main
) > "$fixture_dir/blocked" 2>&1; then
  echo 'dirty command unexpectedly succeeded' >&2; exit 1
fi
grep -F 'requires a clean worktree' "$fixture_dir/blocked" >/dev/null
[[ ! -f "$fixture_dir/ssh" && ! -f "$fixture_dir/deploy" ]]

echo 'deployment-version tests passed: commit builds, tags, mismatches, dirty files, branches, pre-SSH rejection'
