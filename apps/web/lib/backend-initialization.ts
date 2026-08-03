const PUBLIC_MARKETING_PATHS = new Set([
  "/",
  "/about",
  "/changelog",
  "/contact-sales",
  "/download",
  "/homepage",
]);

/**
 * Public marketing pages must render without a running Chimii backend. App,
 * auth, and integration routes still initialize config, auth, and realtime
 * services as soon as they mount.
 */
export function shouldInitializeBackend(pathname: string): boolean {
  const normalizedPath = pathname.replace(/\/+$/, "") || "/";
  return !PUBLIC_MARKETING_PATHS.has(normalizedPath);
}
