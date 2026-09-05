import type { ExpoConfig, ConfigContext } from "expo/config";

/**
 * Dynamic Expo config — replaces app.json so we can read APP_ENV at runtime
 * and switch bundleIdentifier / display name for dev / staging / production.
 *
 * APP_ENV is set by package.json scripts:
 *   - dev          → APP_ENV unset (treated as "development")
 *   - dev:staging  → APP_ENV=staging
 *   - dev:prod / android:release → APP_ENV=production
 */
export default ({ config }: ConfigContext): ExpoConfig => {
  const env = process.env.APP_ENV ?? "development";
  const isProd = env === "production";
  const isStaging = env === "staging";

  return {
    ...config,
    name: isProd
      ? "Chimii"
      : isStaging
        ? "Chimii (Staging)"
        : "Chimii (Dev)",
    slug: "chimii-mobile",
    version: "0.2.0",
    orientation: "portrait",
    userInterfaceStyle: "automatic",
    scheme: "chimii",
    // 1024x1024 source shared with the desktop client
    // (apps/desktop/build/icon.png). Expo prebuild generates every required
    // iOS icon size from this single PNG.
    icon: "./assets/icon.png",
    android: {
      package: isProd
        ? "ai.chimii.mobile"
        : isStaging
          ? "ai.chimii.mobile.staging"
          : "ai.chimii.mobile.dev",
      versionCode: Number(process.env.ANDROID_VERSION_CODE ?? "2"),
      softwareKeyboardLayoutMode: "resize",
      // The release app does not use the development overlay window.
      blockedPermissions: isProd ? ["android.permission.SYSTEM_ALERT_WINDOW"] : [],
    },
    ios: {
      // Expo keeps the top-level portrait policy for iPhone while adding all
      // iPad orientations required for multitasking when tablet support is on.
      supportsTablet: true,
      // Pins DEVELOPMENT_TEAM on every prebuild. Leaving it unset is the normal
      // path — `expo run:ios` then resolves a signing identity from the Keychain
      // itself, which is right when the Apple ID owns exactly one team. With
      // several (a personal team plus an employer's) it takes the *first*
      // identity found whenever the terminal is non-interactive, writes that
      // choice into the generated ios/, and never clears it again: prebuild only
      // writes DEVELOPMENT_TEAM when a value is present, so a project pinned to
      // the wrong team stays wrong until ios/ is deleted. Setting this re-applies
      // the intended team on every `scripts/ios-run.sh` run, which also repairs
      // an already-mispinned checkout.
      appleTeamId: process.env.EXPO_APPLE_TEAM_ID,
      // Per-variant bundle id overrides exist for one reason: an Apple ID
      // can only sign bundle prefixes it owns, so contributors not on the
      // Chimii Apple Developer team (and external users self-building a
      // personal copy against production) need to swap to a reverse-domain
      // they control. Each variant has its own `_<VARIANT>` suffix and is
      // only read inside that variant's branch — a generic
      // `EXPO_BUNDLE_IDENTIFIER` would leak across variants (Expo CLI
      // auto-loads `.env.<mode>.local` regardless of APP_ENV) and collapse
      // dev / staging / prod onto a single id.
      bundleIdentifier: isProd
        ? (process.env.EXPO_BUNDLE_IDENTIFIER_PROD ?? "ai.chimii.mobile")
        : isStaging
          ? "ai.chimii.mobile.staging"
          : (process.env.EXPO_BUNDLE_IDENTIFIER_DEV ?? "ai.chimii.mobile.dev"),
    },
    plugins: [
      "./plugins/with-android-signing.cjs",
      "expo-router",
      "expo-secure-store",
      "expo-sharing",
      "expo-image",
      "@react-native-community/datetimepicker",
      "react-native-enriched-markdown",
      [
        "expo-image-picker",
        {
          // iOS NSPhotoLibraryUsageDescription. Without this string in
          // Info.plist, calling launchImageLibraryAsync hard-crashes on
          // iOS 14+. Camera + microphone are disabled — we only ever read
          // from the existing photo library.
          photosPermission:
            "Allow Chimii to access your photos to attach images to issues and comments.",
          cameraPermission: false,
          microphonePermission: false,
        },
      ],
      [
        "expo-build-properties",
        {
          android: { usesCleartextTraffic: !isProd },
          ios: {
            buildReactNativeFromSource: true,
          },
        },
      ],
    ],
    extra: { APP_ENV: env },
  };
};
