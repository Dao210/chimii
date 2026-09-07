#!/usr/bin/env bash
# Produce an iPhoneOS IPA for personal re-signing; no Apple credentials needed.
set -euo pipefail
cd "$(dirname "$0")/.."
export APP_ENV="${APP_ENV:-production}"
case "$APP_ENV" in production|staging) ;; *) echo 'APP_ENV must be production or staging' >&2; exit 1 ;; esac
export IOS_BUILD_NUMBER="${IOS_BUILD_NUMBER:-1}"
[[ "$IOS_BUILD_NUMBER" =~ ^[1-9][0-9]*$ ]] || { echo 'IOS_BUILD_NUMBER must be positive' >&2; exit 1; }
export NODE_ENV=production EXPO_NO_DOTENV=1 EXPO_NO_TELEMETRY=1 CI=1
# Explicit dotenv keeps local overrides out of both native generation and Metro.
if [[ "${CHIMII_IOS_ENV_LOADED:-}" != 1 ]]; then
  export CHIMII_IOS_ENV_LOADED=1
  exec pnpm exec dotenv -o -e ".env.$APP_ENV" -- bash "$PWD/scripts/ios-unsigned.sh"
fi
xcodebuild -version
command -v pod >/dev/null || { echo 'CocoaPods is required' >&2; exit 1; }
export NODE_BINARY="$(command -v node)"
unset SKIP_BUNDLING
node scripts/build-maker-renderer.mjs
bash scripts/ios-run.sh --prebuild-only
mkdir -p artifacts/ios/logs
(cd ios && pod install) 2>&1 | tee artifacts/ios/logs/pods.log
workspace="$(node -e 'const fs=require("fs"); const w=fs.readdirSync("ios").filter(p=>p.endsWith(".xcworkspace")); if(w.length!==1) throw new Error("Expected one app workspace"); process.stdout.write(w[0]);')"
scheme="${workspace%.xcworkspace}"
work_dir="$(mktemp -d "${RUNNER_TEMP:-${TMPDIR:-/tmp}}/chimii-ios.XXXXXX")"
trap 'rm -rf "$work_dir"' EXIT
xcodebuild -workspace "ios/$workspace" -scheme "$scheme" \
  -configuration Release -sdk iphoneos -destination 'generic/platform=iOS' \
  -archivePath "$work_dir/Chimii.xcarchive" -derivedDataPath "$work_dir/DerivedData" \
  -jobs 3 archive CODE_SIGNING_ALLOWED=NO CODE_SIGNING_REQUIRED=NO \
  CODE_SIGN_IDENTITY= DEVELOPMENT_TEAM= COMPILER_INDEX_STORE_ENABLE=NO \
  2>&1 | tee artifacts/ios/logs/xcodebuild.log
app="$work_dir/Chimii.xcarchive/Products/Applications/$scheme.app"
[[ -d "$app" ]] || { echo 'Archive did not contain the application' >&2; exit 1; }
version="$(node -p 'require("./package.json").version')"
filename="chimii-ios-$APP_ENV-$version-$IOS_BUILD_NUMBER-unsigned.ipa"
ipa="$work_dir/$filename"
mkdir -p "$work_dir/Payload"
ditto "$app" "$work_dir/Payload/$scheme.app"
ditto -c -k --norsrc --keepParent "$work_dir/Payload" "$ipa"
python3 scripts/verify-ios-ipa.py "$ipa"
mv "$ipa" "${ipa%.ipa}.checks.json" artifacts/ios/
ipa="artifacts/ios/$filename"
(cd artifacts/ios && shasum -a 256 "$(basename "$ipa")") > "$ipa.sha256"
echo "Unsigned IPA (requires personal signing): $PWD/$ipa"
