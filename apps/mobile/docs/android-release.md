# Android APK

The mobile app uses Expo SDK 55 / React Native 0.83.10 with Hermes. Android builds use Expo prebuild and Gradle directly. EAS and store accounts are not required for installing the APK.

Production API and web URL: `https://chimii.com`. Package: `ai.chimii.mobile`. The default APK targets ARM64 devices running Android 7.0 (API 24) or newer. Staging uses its own package and `.env.staging`.

## Local build

Install Node 22, pnpm from the root package manifest, JDK 17 and Android SDK. Set `JAVA_HOME` and `ANDROID_HOME`. Install SDK platform 36, build-tools 36.0.0, NDK 27.1.12297006 and CMake 3.22.1.

Keep the release key outside version control. The local build reads `apps/mobile/.signing/release.env`, if present, or these environment variables:

- `ANDROID_KEYSTORE_PATH`: absolute path to the private keystore.
- `ANDROID_KEYSTORE_PASSWORD`: keystore password.
- `ANDROID_KEY_ALIAS`: key alias.
- `ANDROID_KEY_PASSWORD`: key password.

Back up the keystore and its passwords securely. Future updates must use the same key and a higher `ANDROID_VERSION_CODE`. Never regenerate the key for an update.

```bash
pnpm install --frozen-lockfile
pnpm -C apps/mobile typecheck
pnpm -C apps/mobile lint
pnpm -C apps/mobile test
ANDROID_VERSION_CODE=1 pnpm -C apps/mobile android:release
```

The script regenerates Android configuration, bundles JavaScript into a Release APK, checks its signature, and writes the APK and SHA-256 to `apps/mobile/artifacts/`. It refuses to build without release credentials. `ANDROID_ARCHITECTURES` can override the default `arm64-v8a` for a different device.

## Manual GitHub build

The `Android APK` workflow is manual (`.github/workflows/mobile-android.yml`). Configure `ANDROID_KEYSTORE_BASE64`, `ANDROID_KEYSTORE_PASSWORD`, `ANDROID_KEY_ALIAS`, and `ANDROID_KEY_PASSWORD` as repository secrets using the same local release key. Supply an increasing version code and choose production or staging. The workflow validates mobile, builds the signed APK, and retains it as an artifact for 14 days. It does not publish a store release.

## Acceptance

Check signature, package/version, embedded Hermes bundle and disabled debugging. Install with `adb install -r path/to.apk`, launch with Metro stopped, and verify login, workspace switching, tasks, chat, attachment download, Android Back, menu selection, keyboard, reconnect, and foreground resume. A local typecheck or JavaScript export does not establish native startup or authenticated production acceptance.

The first Android build follows Chimii's current seven-status API and single active chat task. It does not request Multica-only status catalogs, subscription summaries or unsupported WebSocket events.
