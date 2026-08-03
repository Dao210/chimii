import type { StorageAdapter } from "../types/storage";

const CHILD_MODE_COOKIE = "chimii_child_mode=1";

/**
 * Detect the restricted credential without reading the HttpOnly Web token.
 * The readable cookie is a boolean mode marker only; Desktop can inspect its
 * explicit token transport.
 */
export function hasChildCredential(
  storage: StorageAdapter,
  cookieAuth?: boolean,
): boolean {
  if (!cookieAuth) {
    return storage.getItem("chimii_token")?.startsWith("mch_") === true;
  }
  return (
    typeof document !== "undefined" &&
    document.cookie
      .split(";")
      .some((value) => value.trim() === CHILD_MODE_COOKIE)
  );
}
