#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
export APP_ENV="${APP_ENV:-production}"
node scripts/build-maker-renderer.mjs
export NODE_ENV=production
export EXPO_NO_DOTENV=1
if [[ -f .signing/release.env && -z "${ANDROID_KEYSTORE_PATH:-}" ]]; then
  set -a
  source .signing/release.env
  set +a
fi
for name in ANDROID_KEYSTORE_PATH ANDROID_KEYSTORE_PASSWORD ANDROID_KEY_ALIAS ANDROID_KEY_PASSWORD; do
  [[ -n "${!name:-}" ]] || { echo "Missing $name. See docs/android-release.md." >&2; exit 1; }
done
[[ -f "$ANDROID_KEYSTORE_PATH" ]] || { echo 'Release keystore does not exist' >&2; exit 1; }
export ANDROID_VERSION_CODE="${ANDROID_VERSION_CODE:-2}"
[[ "$ANDROID_VERSION_CODE" =~ ^[1-9][0-9]*$ ]] || { echo 'ANDROID_VERSION_CODE must be positive' >&2; exit 1; }
# Explicit dotenv prevents a developer's local Expo environment entering a release.
pnpm exec dotenv -o -e ".env.$APP_ENV" -- expo prebuild --platform android --no-install
pnpm exec dotenv -o -e ".env.$APP_ENV" -- bash -c 'cd android && ./gradlew --no-daemon --max-workers=4 assembleRelease -PreactNativeArchitectures="${ANDROID_ARCHITECTURES:-arm64-v8a}"'
mkdir -p artifacts
apk="artifacts/chimii-${APP_ENV}-${ANDROID_VERSION_CODE}.apk"
cp android/app/build/outputs/apk/release/app-release.apk "$apk"
sdk="${ANDROID_HOME:-${ANDROID_SDK_ROOT:-}}"
"$sdk/build-tools/36.0.0/apksigner" verify --verbose --print-certs "$apk"
(cd artifacts && shasum -a 256 "$(basename "$apk")") > "$apk.sha256"
echo "APK: $PWD/$apk"
