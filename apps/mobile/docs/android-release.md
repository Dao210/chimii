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
ANDROID_VERSION_CODE=2 pnpm -C apps/mobile android:release
```

Metro transformation caches include the app environment and public service URLs. The release script invalidates generated autolinking metadata so changing app IDs cannot carry a previous environment into the next package.

The script regenerates the local 3D renderer and versioned thumbnail assets, regenerates Android configuration, bundles JavaScript into a Release APK, checks its signature, and writes the APK and SHA-256 to `apps/mobile/artifacts/`. It refuses to build without release credentials. `ANDROID_ARCHITECTURES` can override the default `arm64-v8a` for a different device.

## Manual GitHub build

The `Android APK` workflow is manual (`.github/workflows/mobile-android.yml`). Configure `ANDROID_KEYSTORE_BASE64`, `ANDROID_KEYSTORE_PASSWORD`, `ANDROID_KEY_ALIAS`, and `ANDROID_KEY_PASSWORD` as repository secrets using the same local release key. Supply an increasing version code and choose production or staging. The workflow validates mobile, builds the signed APK, and retains it as an artifact for 14 days. It does not publish a store release.

## Acceptance

Check signature, package/version, embedded Hermes bundle and disabled debugging. Install with `adb install -r path/to.apk`, launch with Metro stopped, and verify login, the Build/Circuit/Block/Creations tab order, inventory confirmation, revision conflicts, native creation details, 3D rotation, step saving, circuit result feedback, Android Back, keyboard, reconnect, and foreground resume. Collaboration remains available from the account entry. A local typecheck or JavaScript export does not establish native startup or authenticated production acceptance.

The first Android build follows Chimii's current seven-status API and single active chat task. It does not request Multica-only status catalogs, subscription summaries or unsupported WebSocket events.


## Maker experience (0.2.0)

- Native Expo Router pages own navigation, forms, lists and guidance. The four tabs stay mounted so switching preserves input and scroll position; account and collaboration are secondary stack routes.
- Mobile React Query owns server data. Detail reads use creation IDs independently of the recent summary lists (60 builds / 50 circuits). Inventory and progress writes use expected revisions and only advance after the canonical response. Conflicts keep local inventory edits for review.
- Mobile Zustand owns scoped drafts. Short ideas and Build session/request pointers survive cold starts in SecureStore. Unsaved inventory edits and Circuit request signatures remain in memory. Saved inventory, creations and progress are re-fetched from the server. There is no offline write queue.
- Circuit diagrams use native SVG. Zoomed diagrams scroll using native views, including large BOSON layouts. Physical test observations are explicit family feedback, separate from generated connection checks.
- Build details lazily load a local WebView containing only the bundled Three renderer. Native authenticated requests provide versioned GLB bytes. No account token, page navigation or external network access is exposed to the renderer. Matching starter GLBs and thumbnails reuse the existing generated LDraw assets. Lists use images rather than live 3D contexts. Leaving the screen or backgrounding the app unmounts the renderer; unused model data expires after one minute.
- Collaboration sockets and presence prefetch start when the account/collaboration area is first visited, keeping maker startup lightweight.

For repeatable UI acceptance, a separate `APP_ENV` can point at a local fixture service. Non-production packages allow HTTP for local development. Production uses HTTPS and disables cleartext traffic. Fixture acceptance checks native behavior and data contracts; it does not verify a production account or physically test a construction.
