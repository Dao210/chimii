/** Only the configured API origin may receive the native session token. */
export function attachmentHeaders(uri: string, apiUrl: string, token: string | null, slug: string | null): Record<string, string> {
  if (!token) return {};
  try {
    const target = new URL(uri);
    const base = new URL(apiUrl);
    if (target.origin !== base.origin || !target.pathname.startsWith("/api/attachments/") || target.username || target.password) return {};
    return { Authorization: `Bearer ${token}`, ...(slug ? { "X-Workspace-Slug": slug } : {}) };
  } catch { return {}; }
}
