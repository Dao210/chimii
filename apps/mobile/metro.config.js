// Metro bundler configuration for the mobile app inside the chimii monorepo.
// Watches the entire monorepo so type-only imports from packages/core/types/*
// resolve, looks up node_modules from both project and monorepo root, and
// enables symlinks so Metro can follow pnpm's symlinked layout to transitive
// deps. Hierarchical lookup is left enabled (default) — pnpm needs it.

const { getDefaultConfig } = require("expo/metro-config");
const { withNativeWind } = require("nativewind/metro");
const path = require("path");

const projectRoot = __dirname;
const monorepoRoot = path.resolve(projectRoot, "../..");

const config = getDefaultConfig(projectRoot);
// Expo inlines public environment variables into each transformed module.
// dotenv-driven release variants must not reuse another environment's transforms.
config.cacheVersion = [
  config.cacheVersion,
  process.env.APP_ENV ?? "development",
  process.env.EXPO_PUBLIC_API_URL ?? "",
  process.env.EXPO_PUBLIC_WEB_URL ?? "",
].join("|");

config.watchFolders = [monorepoRoot];
config.resolver.nodeModulesPaths = [
  path.resolve(projectRoot, "node_modules"),
  path.resolve(monorepoRoot, "node_modules"),
];
config.resolver.unstable_enableSymlinks = true;

module.exports = withNativeWind(config, { input: "./global.css", inlineRem: 16 });
