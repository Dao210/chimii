#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
fixture_dir="$(mktemp -d)"
trap 'rm -rf -- "$fixture_dir"' EXIT
export GITHUB_ACTIONS=true GITHUB_REPOSITORY=Dao210/chimii
export GITHUB_REF=refs/tags/v0.3.0 GITHUB_REF_NAME=v0.3.0
export GITHUB_SHA=dc561db2c3f1d534dd30dc5a212da1d27503d398 GITHUB_RUN_ID=40000000000 GITHUB_RUN_ATTEMPT=1
export GH_TOKEN=fixture-only RUNNER_TEMP="$fixture_dir/runner" GITHUB_OUTPUT="$fixture_dir/output"
export RELEASE_FIXTURE_DIR="$fixture_dir" RELEASE_FIXTURE_CASE=normal
mkdir -p "$RUNNER_TEMP"
# Exported fake command: no real executable, account, or network is accessed.
gh() {
  if [[ "$*" == *'--method DELETE'* ]]; then
    printf '%s\n' "$*" >> "$RELEASE_FIXTURE_DIR/deletions"
    return 0
  fi
  case "$2" in
    */releases/tags/*)
      case "$RELEASE_FIXTURE_CASE" in
        missing) echo 'gh: Not Found (HTTP 404)' >&2; return 1 ;;
        auth) echo 'gh: Bad credentials (HTTP 401)' >&2; return 1 ;;
        unexpected) jq '.id = 999' "$RELEASE_FIXTURE_DIR/release.json" ;;
        immutable) jq '.immutable = true' "$RELEASE_FIXTURE_DIR/release.json" ;;
        changed) jq '.assets[0].id = 999' "$RELEASE_FIXTURE_DIR/release.json" ;;
        *) cat "$RELEASE_FIXTURE_DIR/release.json" ;;
      esac ;;
    */git/ref/tags/*)
      if [[ "$RELEASE_FIXTURE_CASE" == moved ]]; then
        printf '{"object":{"type":"commit","sha":"different"}}\n'
      else
        printf '{"object":{"type":"tag","sha":"annotation"}}\n'
      fi ;;
    */git/tags/annotation) printf '{"object":{"type":"commit","sha":"%s"}}\n' "$GITHUB_SHA" ;;
    */actions/runs/33944152071)
      jq -n --arg status "$([[ "$RELEASE_FIXTURE_CASE" == running ]] && echo in_progress || echo completed)" '{id:33944152071, head_sha:"fc2f6538502d9f651118493731f4fd22d2da2a22", head_branch:"v0.3.0", path:".github/workflows/release.yml", status:$status, repository:{full_name:"Dao210/chimii"}}' ;;
    */actions/workflows/release.yml/runs*)
      if [[ "$RELEASE_FIXTURE_CASE" == concurrent ]]; then
        echo '{"total_count":1,"workflow_runs":[{"id":999,"status":"in_progress"}]}'
      else
        echo '{"total_count":1,"workflow_runs":[{"id":33944152071,"status":"completed"}]}'
      fi ;;
    */actions/artifacts/123)
      jq -n --argjson run "$GITHUB_RUN_ID" --arg name "v0.3.0-original-release-$GITHUB_RUN_ID-1" --argjson expired "$([[ "$RELEASE_FIXTURE_CASE" == expired ]] && echo true || echo false)" '{expired:$expired, size_in_bytes:10, name:$name, workflow_run:{id:$run}}' ;;
    *) echo "unexpected fixture API call: $*" >&2; return 88 ;;
  esac
}
export -f gh
printf 'original asset\n' > "$fixture_dir/original"
digest="$(shasum -a 256 "$fixture_dir/original")"
jq -n --arg digest "sha256:${digest%% *}" '{id:383132584, tag_name:"v0.3.0", immutable:false, assets:[{id:1,name:"checksums.txt",size:15,digest:$digest}]}' > "$fixture_dir/release.json"

reject() {
  if bash "$SCRIPT_DIR/prepare-release.sh" "$1" > "$fixture_dir/log" 2>&1; then
    echo "accepted unsafe case: $RELEASE_FIXTURE_CASE / $1" >&2; exit 1
  fi
  [[ ! -f "$fixture_dir/deletions" ]]
}
RELEASE_FIXTURE_CASE=missing bash "$SCRIPT_DIR/prepare-release.sh" inspect >/dev/null
grep -Fx 'replace=false' "$GITHUB_OUTPUT" >/dev/null
for scenario in auth unexpected immutable moved running concurrent; do
  export RELEASE_FIXTURE_CASE="$scenario"
  reject inspect
done
export RELEASE_FIXTURE_CASE=normal
(export GITHUB_REPOSITORY=other/repo; reject inspect)
(export GITHUB_REF=refs/heads/main; reject inspect)
(export GITHUB_SHA=fc2f6538502d9f651118493731f4fd22d2da2a22; reject inspect)
bash "$SCRIPT_DIR/prepare-release.sh" inspect >/dev/null
grep -Fx 'replace=true' "$GITHUB_OUTPUT" >/dev/null
reject delete
mkdir -p "$RUNNER_TEMP/release-recovery/assets"
printf 'corrupted asset' > "$RUNNER_TEMP/release-recovery/assets/checksums.txt"
reject verify-backup
cp "$fixture_dir/original" "$RUNNER_TEMP/release-recovery/assets/checksums.txt"
bash "$SCRIPT_DIR/prepare-release.sh" verify-backup >/dev/null
reject delete
export RECOVERY_ARTIFACT_ID=123
for scenario in changed expired concurrent; do
  export RELEASE_FIXTURE_CASE="$scenario"
  reject delete
done
export RELEASE_FIXTURE_CASE=normal
bash "$SCRIPT_DIR/prepare-release.sh" delete >/dev/null
[[ "$(cat "$fixture_dir/deletions")" == 'api --method DELETE repos/Dao210/chimii/releases/383132584' ]]
echo 'release preparation tests passed: identity, concurrency, backup hashes, artifact receipt, exact deletion'
