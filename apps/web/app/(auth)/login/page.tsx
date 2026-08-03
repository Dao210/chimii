"use client";

import { Suspense, useEffect, useRef, useState } from "react";
import { useSearchParams, useRouter } from "next/navigation";
import { useQueryClient, type QueryClient } from "@tanstack/react-query";
import { sanitizeNextUrl, useAuthStore } from "@chimii/core/auth";
import { useConfigStore } from "@chimii/core/config";
import {
  workspaceKeys,
  workspaceListOptions,
} from "@chimii/core/workspace/queries";
import {
  paths,
  resolvePostAuthDestination,
  useHasOnboarded,
} from "@chimii/core/paths";
import { api } from "@chimii/core/api";
import type { Workspace } from "@chimii/core/types";
import { Button } from "@chimii/ui/components/ui/button";
import { Loader2 } from "lucide-react";
import { setLoggedInCookie } from "@/features/auth/auth-cookie";
import Link from "next/link";
import { LoginPage, validateCliCallback } from "@chimii/views/auth";
import { useT } from "@chimii/views/i18n";

/**
 * Pick where a logged-in user with no explicit `?next=` should land.
 * Un-onboarded users with pending invitations on their email get routed to
 * the batch /invitations page; everyone else falls through to the standard
 * resolver. A network blip on listMyInvitations is non-fatal — we fall
 * through rather than trap the user on an error screen.
 */
async function resolveLoggedInDestination(
  qc: QueryClient,
  hasOnboarded: boolean,
  workspaces: Workspace[],
): Promise<string> {
  if (!hasOnboarded) {
    try {
      const invites = await api.listMyInvitations();
      if (invites.length > 0) {
        qc.setQueryData(workspaceKeys.myInvitations(), invites);
        return paths.invitations();
      }
    } catch {
      // fall through
    }
  }
  return resolvePostAuthDestination(workspaces, hasOnboarded);
}

function LoginPageContent() {
  const router = useRouter();
  const qc = useQueryClient();
  const { t } = useT("auth");
  const googleClientId = useConfigStore((state) => state.googleClientId);
  const user = useAuthStore((s) => s.user);
  const isLoading = useAuthStore((s) => s.isLoading);
  const searchParams = useSearchParams();

  const cliCallbackRaw = searchParams.get("cli_callback");
  const cliState = searchParams.get("cli_state") || "";
  const platform = searchParams.get("platform");
  const isDesktopHandoff = platform === "desktop" && !cliCallbackRaw;
  // `next` carries a protected URL the user was originally headed to
  // (e.g. /invite/{id}). With URL-driven workspaces there is no legacy
  // global default — if `next` is absent we resolve the user's workspace and
  // enter its /build studio. Sanitize first so a crafted `?next=https://evil`
  // cannot bounce the user off-origin after a successful login.
  const nextUrl = sanitizeNextUrl(searchParams.get("next"));

  const [desktopToken, setDesktopToken] = useState<string | null>(null);
  const [desktopError, setDesktopError] = useState("");
  const hasOnboarded = useHasOnboarded();

  // Latched once auth has been observed settled as logged-out on this page.
  // Any `user` that appears afterwards came from the login form in this
  // session — not from an existing session found on arrival.
  const settledLoggedOutRef = useRef(false);

  // Already authenticated ON ARRIVAL — honor ?next= or fall back to first
  // workspace (or /onboarding if the user has none). Skip this entire path
  // when the user arrived to authorize the CLI.
  useEffect(() => {
    if (isLoading) return;
    if (!user) {
      settledLoggedOutRef.current = true;
      return;
    }
    if (cliCallbackRaw) return;
    if (isDesktopHandoff) {
      // Desktop opened the browser for login but the web session is already
      // authenticated — mint a bearer token from the cookie session and hand
      // it off via deep link instead of silently redirecting to the workspace.
      api
        .issueCliToken()
        .then(({ token }) => {
          setDesktopToken(token);
          window.location.href = `chimii://auth/callback?token=${encodeURIComponent(token)}`;
        })
        .catch(() => {
          setDesktopError(t(($) => $.web.desktop_handoff.prepare_failed));
        });
      return;
    }
    // Fresh form login (issue #5009): `user` was written by verifyCode while
    // handleVerify was still fetching the workspace list, so this effect used
    // to read the not-yet-seeded list cache and race handleSuccess with a
    // replace to /workspaces/new. handleSuccess owns post-login navigation;
    // this effect only serves visitors who arrived already authenticated.
    if (settledLoggedOutRef.current) return;
    if (nextUrl) {
      router.replace(nextUrl);
      return;
    }
    // Fetch instead of reading the cache: on a fresh page load the cache is
    // cold, and `getQueryData() ?? []` would misroute a user who does have
    // workspaces to /workspaces/new. On fetch failure fall back to [] —
    // same destination the cold-cache read produced, rather than trapping
    // the user on the login page.
    void qc
      .ensureQueryData(workspaceListOptions())
      .catch(() => [] as Workspace[])
      .then((list) => resolveLoggedInDestination(qc, hasOnboarded, list))
      .then((dest) => router.replace(dest));
  }, [
    isLoading,
    user,
    router,
    nextUrl,
    cliCallbackRaw,
    isDesktopHandoff,
    hasOnboarded,
    qc,
    t,
  ]);

  const handleSuccess = async () => {
    // Read the latest user snapshot directly — the closure's `hasOnboarded`
    // was captured before login completed and would be stale here.
    const currentUser = useAuthStore.getState().user;
    const onboarded = currentUser?.onboarded_at != null;
    if (nextUrl) {
      router.push(nextUrl);
      return;
    }
    const list = qc.getQueryData<Workspace[]>(workspaceKeys.list()) ?? [];
    router.push(await resolveLoggedInDestination(qc, onboarded, list));
  };

  // Build Google OAuth state: encode platform, next URL, and CLI callback
  // params so the callback can redirect to the right place after login.
  // CLI callback/state must survive the Google OAuth round-trip so the
  // post-login callback page can redirect the JWT back to the CLI's local
  // HTTP listener (critical for headless / WSL2 environments).
  const googleState = [
    platform === "desktop" ? "platform:desktop" : "",
    nextUrl ? `next:${nextUrl}` : "",
    cliCallbackRaw && validateCliCallback(cliCallbackRaw)
      ? `cli_callback:${encodeURIComponent(cliCallbackRaw)}`
      : "",
    cliState ? `cli_state:${encodeURIComponent(cliState)}` : "",
  ]
    .filter(Boolean)
    .join(",") || undefined;

  // While the desktop handoff is in progress (or has produced a token/error),
  // render a dedicated screen instead of flashing the login form or redirecting
  // away to a workspace page.
  if (isDesktopHandoff && user) {
    return (
      <section
        className="chimii-auth-surface relative flex min-h-[520px] w-full flex-1 items-center justify-center overflow-hidden bg-[#f5f0e6] px-5 py-12 text-[#12130f] sm:px-8"
        aria-labelledby="desktop-handoff-title"
      >
        <div className="chimii-invention-grid pointer-events-none absolute inset-0 opacity-25" />
        <div className="pointer-events-none absolute -left-8 top-16 size-28 rotate-12 rounded-[28px] border-[14px] border-[#f6c84a]/45" />
        <div className="pointer-events-none absolute -bottom-8 right-8 size-28 rounded-full border-[14px] border-[#4b79d8]/25" />

        <div className="animate-auth-panel-enter relative w-full max-w-[440px] rounded-[26px] border-2 border-[#12130f] bg-[#fffdf7] p-6 text-center shadow-[7px_8px_0_rgba(18,19,15,0.14)] sm:p-9">
          <div
            className={`mx-auto flex size-16 items-center justify-center rounded-[20px] border-2 border-[#12130f] ${
              desktopError ? "bg-[#f05a3f]" : "bg-[#f6c84a]"
            }`}
          >
            {desktopError ? (
              <span className="text-3xl font-black text-white" aria-hidden="true">
                !
              </span>
            ) : (
              <Loader2
                className="size-7 animate-spin text-[#12130f]"
                aria-hidden="true"
              />
            )}
          </div>
          <p className="mt-6 text-[10px] font-black uppercase tracking-[0.18em] text-[#4b79d8]">
            CHIMII DESKTOP
          </p>
          <h1
            id="desktop-handoff-title"
            className="mt-3 text-balance text-3xl font-black leading-tight tracking-[-0.035em]"
          >
            {desktopError
              ? t(($) => $.web.desktop_handoff.failed_title)
              : t(($) => $.web.desktop_handoff.opening_title)}
          </h1>
          <p className="mt-3 text-sm leading-6 text-[#12130f]/58" aria-live="polite">
            {desktopError ||
              (desktopToken
                ? t(($) => $.web.desktop_handoff.opening_description)
                : t(($) => $.web.desktop_handoff.preparing))}
          </p>

          {desktopToken && !desktopError && (
            <div className="mt-7">
              <Button
                className="h-12 w-full rounded-full border-2 border-[#12130f] bg-[#f05a3f] px-5 font-extrabold text-white shadow-[0_4px_0_#b93b29] hover:bg-[#e6533a] dark:bg-[#f05a3f] dark:text-white dark:hover:bg-[#e6533a]"
                onClick={() => {
                  window.location.href = `chimii://auth/callback?token=${encodeURIComponent(desktopToken)}`;
                }}
              >
                {t(($) => $.web.desktop_handoff.open_button)}
              </Button>
            </div>
          )}
        </div>
      </section>
    );
  }

  return (
    <LoginPage
      onSuccess={handleSuccess}
      google={
        googleClientId
          ? {
              clientId: googleClientId,
              redirectUri: `${window.location.origin}/auth/callback`,
              state: googleState,
            }
          : undefined
      }
      cliCallback={
        cliCallbackRaw && validateCliCallback(cliCallbackRaw)
          ? { url: cliCallbackRaw, state: cliState }
          : undefined
      }
      onTokenObtained={setLoggedInCookie}
      extra={
        <span className="text-xs text-muted-foreground">
          {t(($) => $.web.prefer_desktop)}{" "}
          <Link
            href="/download"
            className="font-medium text-foreground underline decoration-foreground/30 underline-offset-4 hover:decoration-foreground/70"
          >
            {t(($) => $.web.download)}
          </Link>
        </span>
      }
    />
  );
}

export default function Page() {
  return (
    <Suspense fallback={null}>
      <LoginPageContent />
    </Suspense>
  );
}
