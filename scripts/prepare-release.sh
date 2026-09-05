#!/usr/bin/env bash
# The only replacement exception is the explicitly approved v0.3.0 recovery.
set -euo pipefail

fail() { printf '[release-prepare] ERROR %s\n' "$*" >&2; exit 1; }
mode="${1:-}"
[[ "$mode" == inspect || "$mode" == verify-backup || "$mode" == delete ]] || fail 'expected inspect, verify-backup, or delete'
[[ "${GITHUB_ACTIONS:-}" == true && "${GITHUB_REPOSITORY:-}" == Dao210/chimii ]] || fail 'only the canonical GitHub Actions repository is allowed'
[[ "${GITHUB_REF:-}" == "refs/tags/${GITHUB_REF_NAME:-}" && "${GITHUB_REF_NAME:-}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$ && "$GITHUB_REF_NAME" != *-dirty* ]] || fail 'expected a release tag push'
[[ "${GITHUB_SHA:-}" =~ ^[0-9a-f]{40}$ && "${GITHUB_RUN_ID:-}" =~ ^[0-9]+$ ]] || fail 'missing workflow identity'
[[ -n "${GH_TOKEN:-}" && -n "${RUNNER_TEMP:-}" && -n "${GITHUB_OUTPUT:-}" ]] || fail 'missing runner configuration'
backup_dir="$RUNNER_TEMP/release-recovery"
api_root="repos/$GITHUB_REPOSITORY"
old_release_id=383132584
old_run_id=33944152071
old_sha=fc2f6538502d9f651118493731f4fd22d2da2a22
mkdir -p "$backup_dir"
scratch_dir="$(mktemp -d)"
trap 'rm -rf -- "$scratch_dir"' EXIT

# Only an actual HTTP 404 means no release; authentication/network errors fail.
if ! gh api "$api_root/releases/tags/$GITHUB_REF_NAME" > "$scratch_dir/release.json" 2> "$scratch_dir/error"; then
  if [[ "$mode" == inspect ]] && grep -q 'HTTP 404' "$scratch_dir/error"; then
    printf 'replace=false\n' >> "$GITHUB_OUTPUT"
    echo '[release-prepare] No existing release; normal publication is allowed.'
    exit 0
  fi
  fail 'could not read the release (or it disappeared during recovery)'
fi

[[ "$GITHUB_REF_NAME" == v0.3.0 && "$GITHUB_SHA" != "$old_sha" && "$GITHUB_RUN_ID" != "$old_run_id" ]] || fail 'replacement is not approved for this workflow'
jq -e --argjson id "$old_release_id" '.id == $id and .tag_name == "v0.3.0" and .immutable == false' "$scratch_dir/release.json" >/dev/null || fail 'existing release is not the approved mutable release'

# Confirm the tag still names this run, including annotated tags.
gh api "$api_root/git/ref/tags/$GITHUB_REF_NAME" > "$scratch_dir/ref.json"
ref_type="$(jq -r '.object.type' "$scratch_dir/ref.json")"
ref_sha="$(jq -r '.object.sha' "$scratch_dir/ref.json")"
if [[ "$ref_type" == tag ]]; then
  gh api "$api_root/git/tags/$ref_sha" > "$scratch_dir/ref.json"
  ref_type="$(jq -r '.object.type' "$scratch_dir/ref.json")"
  ref_sha="$(jq -r '.object.sha' "$scratch_dir/ref.json")"
fi
[[ "$ref_type" == commit && "$ref_sha" == "$GITHUB_SHA" ]] || fail 'tag changed while recovery was running'

gh api "$api_root/actions/runs/$old_run_id" > "$scratch_dir/old-run.json"
jq -e --arg sha "$old_sha" --argjson id "$old_run_id" '.id == $id and .head_sha == $sha and .head_branch == "v0.3.0" and .path == ".github/workflows/release.yml" and .status == "completed" and .repository.full_name == "Dao210/chimii"' "$scratch_dir/old-run.json" >/dev/null || fail 'old publisher is not the expected completed run'
gh api "$api_root/actions/workflows/release.yml/runs?branch=v0.3.0&per_page=100" > "$scratch_dir/runs.json"
jq -e --argjson current "$GITHUB_RUN_ID" '.total_count == (.workflow_runs | length) and all(.workflow_runs[]; .id == $current or .status == "completed")' "$scratch_dir/runs.json" >/dev/null || fail 'another publisher is active or the run list is incomplete'

if [[ "$mode" == inspect ]]; then
  cp "$scratch_dir/release.json" "$backup_dir/release.json"
  cp "$scratch_dir/old-run.json" "$backup_dir/old-run.json"
  printf 'replace=true\n' >> "$GITHUB_OUTPUT"
  echo '[release-prepare] Approved old release identified; back up every asset before deletion.'
  exit 0
fi

[[ -f "$backup_dir/release.json" ]] || fail 'release backup is missing'
# Download counters may change while backing up, but asset bytes/IDs must not.
fingerprint='[.id, .tag_name, (.assets | map({id, name, size, digest}) | sort_by(.id))]'
[[ "$(jq -c "$fingerprint" "$backup_dir/release.json")" == "$(jq -c "$fingerprint" "$scratch_dir/release.json")" ]] || fail 'release assets changed after inspection'
jq -e '.assets | length > 0 and all(.[]; (.name | test("^[A-Za-z0-9][A-Za-z0-9._-]*$")) and (.digest | test("^sha256:[a-f0-9]{64}$")))' "$backup_dir/release.json" >/dev/null || fail 'assets lack safe filenames or SHA-256 digests'
while IFS=$'\t' read -r name size digest; do
  asset="$backup_dir/assets/$name"
  [[ -f "$asset" && ! -L "$asset" ]] || fail "backup asset missing: $name"
  [[ "$(wc -c < "$asset" | tr -d '[:space:]')" == "$size" ]] || fail "backup size mismatch: $name"
  actual_digest="$(shasum -a 256 "$asset")"
  [[ "${actual_digest%% *}" == "${digest#sha256:}" ]] || fail "backup checksum mismatch: $name"
done < <(jq -r '.assets[] | [.name, .size, .digest] | @tsv' "$backup_dir/release.json")

if [[ "$mode" == verify-backup ]]; then
  echo '[release-prepare] Every original asset passed size and SHA-256 verification.'
  exit 0
fi

[[ "${RECOVERY_ARTIFACT_ID:-}" =~ ^[0-9]+$ ]] || fail 'uploaded backup artifact receipt is missing'
gh api "$api_root/actions/artifacts/$RECOVERY_ARTIFACT_ID" > "$scratch_dir/artifact.json"
jq -e --argjson run "$GITHUB_RUN_ID" --arg name "v0.3.0-original-release-$GITHUB_RUN_ID-${GITHUB_RUN_ATTEMPT:-1}" '.expired == false and .size_in_bytes > 0 and .name == $name and .workflow_run.id == $run' "$scratch_dir/artifact.json" >/dev/null || fail 'backup artifact is not available for this run'
gh api --method DELETE "$api_root/releases/$old_release_id"
printf '[release-prepare] Removed old release %s; original assets are recoverable from Actions artifact %s. The Git tag was retained.\n' "$old_release_id" "$RECOVERY_ARTIFACT_ID"
