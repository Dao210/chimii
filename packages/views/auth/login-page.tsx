"use client";

import { useState, useEffect, useCallback, useRef, type ReactNode } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { Input } from "@chimii/ui/components/ui/input";
import { Button } from "@chimii/ui/components/ui/button";
import { Label } from "@chimii/ui/components/ui/label";
import {
  InputOTP,
  InputOTPGroup,
  InputOTPSlot,
} from "@chimii/ui/components/ui/input-otp";
import { useAuthStore } from "@chimii/core/auth";
import { useConfigStore } from "@chimii/core/config";
import { workspaceKeys } from "@chimii/core/workspace/queries";
import { api } from "@chimii/core/api";
import type { User } from "@chimii/core/types";
import { Cog, DraftingCompass, Lightbulb, Loader2 } from "lucide-react";
import { useT } from "../i18n";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface GoogleAuthConfig {
  clientId: string;
  redirectUri: string;
  /** Opaque state passed through Google OAuth (e.g. "platform:desktop"). */
  state?: string;
}

interface CliCallbackConfig {
  /** Validated localhost callback URL */
  url: string;
  /** Opaque state to pass back to CLI */
  state: string;
}

interface LoginPageProps {
  /** Logo element rendered above the title */
  logo?: ReactNode;
  /** Called after successful login. The workspace list is seeded into React
   *  Query before this fires, so the caller can compute a destination URL. */
  onSuccess: () => void;
  /** Google OAuth config. Omit to disable Google login. */
  google?: GoogleAuthConfig;
  /** CLI callback config for authorizing CLI tools. */
  cliCallback?: CliCallbackConfig;
  /** Called after a token is obtained (e.g. to set cookies). */
  onTokenObtained?: () => void;
  /** Override Google login handler (e.g. desktop opens browser externally). When provided, renders the Google button even if `google` config is omitted. */
  onGoogleLogin?: () => void;
  /** Slot rendered at the bottom of the sign-in card, below the
   *  Google button. The web shell uses it for a "Prefer the desktop
   *  app?" prompt; desktop omits it (a download prompt inside the app
   *  would be absurd). */
  extra?: ReactNode;
}

function errorStatus(error: unknown): number | undefined {
  if (!error || typeof error !== "object" || !("status" in error)) {
    return undefined;
  }
  return typeof error.status === "number" ? error.status : undefined;
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

export function redirectToCliCallback(url: string, token: string, state: string) {
  const separator = url.includes("?") ? "&" : "?";
  window.location.href = `${url}${separator}token=${encodeURIComponent(token)}&state=${encodeURIComponent(state)}`;
}

/**
 * Validate that a CLI callback URL points to a safe host over HTTP.
 * Allows localhost and private/LAN IPs (RFC 1918) to support self-hosted setups
 * on local VMs while blocking arbitrary public hosts.
 */
export function validateCliCallback(cliCallback: string): boolean {
  try {
    const cbUrl = new URL(cliCallback);
    if (cbUrl.protocol !== "http:") return false;
    const h = cbUrl.hostname;
    if (h === "localhost" || h === "127.0.0.1") return true;
    // Allow RFC 1918 private IPs: 10.x.x.x, 172.16-31.x.x, 192.168.x.x
    if (/^10\./.test(h)) return true;
    if (/^172\.(1[6-9]|2\d|3[01])\./.test(h)) return true;
    if (/^192\.168\./.test(h)) return true;
    return false;
  } catch {
    return false;
  }
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function LoginPage({
  logo,
  onSuccess,
  google,
  cliCallback,
  onTokenObtained,
  onGoogleLogin,
  extra,
}: LoginPageProps) {
  const { t } = useT("auth");
  const qc = useQueryClient();
  const allowSignup = useConfigStore((state) => state.allowSignup);
  const [step, setStep] = useState<"email" | "code" | "cli_confirm">("email");
  const [email, setEmail] = useState("");
  const [code, setCode] = useState("");
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [loading, setLoading] = useState(false);
  const [resending, setResending] = useState(false);
  const [cooldown, setCooldown] = useState(0);
  const [existingUser, setExistingUser] = useState<User | null>(null);
  const emailInputRef = useRef<HTMLInputElement>(null);
  const otpAreaRef = useRef<HTMLDivElement>(null);
  // Tracks how the existing session was detected so handleCliAuthorize
  // uses the matching token source (cookie → issueCliToken, localStorage → direct).
  const authSourceRef = useRef<"cookie" | "localStorage">("cookie");

  // Check for existing session when CLI callback is present.
  // Prioritises cookie auth (= current browser session) to avoid authorising
  // the CLI with a stale or mismatched localStorage token.
  useEffect(() => {
    if (!cliCallback) return;

    // Ensure no stale bearer token interferes — we want to test the cookie first.
    api.setToken(null);

    api
      .getMe()
      .then((user) => {
        authSourceRef.current = "cookie";
        setExistingUser(user);
        setStep("cli_confirm");
      })
      .catch(() => {
        // Cookie auth failed — fall back to localStorage token
        const token = localStorage.getItem("chimii_token");
        if (!token) return;

        api.setToken(token);
        api
          .getMe()
          .then((user) => {
            authSourceRef.current = "localStorage";
            setExistingUser(user);
            setStep("cli_confirm");
          })
          .catch(() => {
            api.setToken(null);
            localStorage.removeItem("chimii_token");
          });
      });
  }, [cliCallback]);

  // Cooldown timer for resend
  useEffect(() => {
    if (cooldown <= 0) return;
    const timer = setTimeout(() => setCooldown((c) => c - 1), 1000);
    return () => clearTimeout(timer);
  }, [cooldown]);

  // Autofocus is useful on desktop, but opening the mobile keyboard before the
  // user has oriented themselves makes the auth surface feel cramped.
  useEffect(() => {
    if (loading) return;
    const isDesktop =
      typeof window.matchMedia !== "function" ||
      window.matchMedia("(min-width: 768px)").matches;
    if (!isDesktop) return;
    if (step === "email") emailInputRef.current?.focus();
    if (step === "code") {
      otpAreaRef.current
        ?.querySelector<HTMLInputElement>("input[data-input-otp]")
        ?.focus();
    }
  }, [loading, step]);

  const sendCodeError = useCallback(
    (err: unknown, resend = false) => {
      const status = errorStatus(err);
      if (status === 429) return t(($) => $.errors.rate_limited);
      if (status === 403) return t(($) => $.errors.signup_disabled);
      return resend
        ? t(($) => $.errors.resend_failed)
        : `${t(($) => $.errors.send_failed)} ${t(($) => $.errors.server_unreachable)}`;
    },
    [t],
  );

  const handleSendCode = useCallback(
    async (e?: React.FormEvent) => {
      e?.preventDefault();
      const normalizedEmail = email.trim();
      if (!normalizedEmail) {
        setError(t(($) => $.common.email_required));
        return;
      }
      setLoading(true);
      setError("");
      setNotice("");
      setEmail(normalizedEmail);
      try {
        await useAuthStore.getState().sendCode(normalizedEmail);
        setStep("code");
        setCode("");
        setCooldown(60);
      } catch (err) {
        setError(sendCodeError(err));
      } finally {
        setLoading(false);
      }
    },
    [email, sendCodeError, t],
  );

  const handleVerify = useCallback(
    async (value: string) => {
      if (value.length !== 6 || loading) return;
      setLoading(true);
      setError("");
      setNotice("");
      try {
        if (cliCallback) {
          // CLI path: get token directly for the redirect URL
          const { token } = await api.verifyCode(email, value);
          localStorage.setItem("chimii_token", token);
          api.setToken(token);
          onTokenObtained?.();
          redirectToCliCallback(cliCallback.url, token, cliCallback.state);
          return;
        }

        // Normal path: seed the workspace list into the Query cache so the
        // caller's onSuccess can read it synchronously to compute a destination
        // URL (first workspace's slug, or /workspaces/new for zero-workspace
        // users).
        await useAuthStore.getState().verifyCode(email, value);
        const wsList = await api.listWorkspaces();
        qc.setQueryData(workspaceKeys.list(), wsList);
        onTokenObtained?.();
        onSuccess();
      } catch (err) {
        setError(t(($) => $.errors.code_invalid));
        setCode("");
        setLoading(false);
      }
    },
    [email, loading, onSuccess, cliCallback, onTokenObtained, qc, t],
  );

  const handleResend = async () => {
    if (cooldown > 0 || resending || loading) return;
    setResending(true);
    setError("");
    setNotice("");
    try {
      await useAuthStore.getState().sendCode(email);
      setCooldown(60);
      setNotice(t(($) => $.verify.resent));
    } catch (err) {
      setError(sendCodeError(err, true));
    } finally {
      setResending(false);
    }
  };

  const handleCliAuthorize = async () => {
    if (!cliCallback) return;
    setLoading(true);

    try {
      let token: string;

      if (authSourceRef.current === "localStorage") {
        // Session was detected via localStorage — reuse that token directly.
        const stored = localStorage.getItem("chimii_token");
        if (!stored) throw new Error("token missing");
        token = stored;
      } else {
        // Session was detected via cookie — obtain a bearer token from the server.
        const res = await api.issueCliToken();
        token = res.token;
      }

      onTokenObtained?.();
      redirectToCliCallback(cliCallback.url, token, cliCallback.state);
    } catch {
      setError(t(($) => $.errors.cli_auth_failed));
      setExistingUser(null);
      setStep("email");
      setLoading(false);
    }
  };

  const handleChangeEmail = () => {
    setStep("email");
    setCode("");
    setError("");
    setNotice("");
  };

  const handleGoogleLogin = () => {
    if (onGoogleLogin) {
      onGoogleLogin();
      return;
    }
    if (!google) return;
    const params = new URLSearchParams({
      client_id: google.clientId,
      redirect_uri: google.redirectUri,
      response_type: "code",
      scope: "openid email profile",
      access_type: "offline",
      prompt: "select_account",
    });
    if (google.state) params.set("state", google.state);
    window.location.href = `https://accounts.google.com/o/oauth2/v2/auth?${params}`;
  };

  const primaryButtonClassName =
    "h-12 w-full rounded-full border-2 border-[#12130f] bg-[#f05a3f] px-5 font-extrabold text-white shadow-[0_4px_0_#b93b29] transition-[transform,box-shadow,background-color] hover:bg-[#e6533a] active:translate-y-0.5 active:shadow-[0_2px_0_#b93b29] focus-visible:border-[#12130f] focus-visible:ring-[#f05a3f]/35 dark:bg-[#f05a3f] dark:text-white dark:hover:bg-[#e6533a]";

  const storySteps = [
    {
      label: t(($) => $.story.imagine),
      Icon: Lightbulb,
      color: "bg-[#f6c84a]",
    },
    {
      label: t(($) => $.story.build),
      Icon: DraftingCompass,
      color: "bg-[#4b79d8] text-white",
    },
    {
      label: t(($) => $.story.life),
      Icon: Cog,
      color: "bg-[#35a87d] text-white",
    },
  ];

  return (
    <section
      className="chimii-auth-surface relative flex min-h-0 w-full flex-1 overflow-y-auto bg-[#f5f0e6] text-[#12130f]"
      aria-labelledby="auth-title"
    >
      <div className="relative grid min-h-full w-full lg:grid-cols-[minmax(0,0.92fr)_minmax(430px,0.72fr)]">
        <aside className="relative hidden overflow-hidden border-r-2 border-[#12130f] bg-[#f8df78] lg:flex lg:min-h-[640px] lg:flex-col lg:justify-between lg:p-12 xl:p-16">
          <div className="chimii-invention-grid pointer-events-none absolute inset-0 opacity-45" />
          <div className="pointer-events-none absolute -left-14 top-24 size-36 rotate-12 rounded-[30px] border-[16px] border-[#f05a3f]/35" />
          <div className="pointer-events-none absolute -right-8 bottom-20 size-28 rounded-full border-[14px] border-[#35a87d]/35" />

          <div className="relative max-w-[620px]">
            <p className="text-[11px] font-black uppercase tracking-[0.22em] text-[#12130f]/48">
              {t(($) => $.story.eyebrow)}
            </p>
            <h2 className="mt-5 text-balance font-serif text-[3.5rem] leading-[0.98] tracking-[-0.045em] xl:text-[4.6rem]">
              {t(($) => $.story.title)}
            </h2>
            <p className="mt-6 max-w-[560px] text-base font-medium leading-7 text-[#12130f]/62 xl:text-lg xl:leading-8">
              {t(($) => $.story.description)}
            </p>
          </div>

          <div className="relative mt-12 grid max-w-[680px] grid-cols-3 gap-3">
            {storySteps.map(({ label, Icon, color }, index) => (
              <div
                key={label}
                className="rounded-[22px] border-2 border-[#12130f] bg-[#fffdf7] p-3 shadow-[4px_5px_0_rgba(18,19,15,0.18)] xl:p-4"
              >
                <div
                  className={`flex aspect-[1.35] items-center justify-center rounded-[16px] border-2 border-[#12130f] ${color}`}
                >
                  <Icon className="size-8 stroke-[1.7] xl:size-10" aria-hidden="true" />
                </div>
                <p className="mt-3 text-[10px] font-black tracking-[0.13em] text-[#12130f]/42">
                  0{index + 1}
                </p>
                <p className="mt-1 text-sm font-extrabold leading-tight text-[#12130f] xl:text-base">
                  {label}
                </p>
              </div>
            ))}
          </div>
        </aside>

        <div className="relative flex min-w-0 items-center justify-center overflow-hidden bg-[#f5f0e6] px-5 py-10 sm:px-8 sm:py-14 lg:bg-[#fffdf7] lg:px-10 lg:py-16">
          <div className="chimii-invention-grid pointer-events-none absolute inset-0 opacity-25 lg:opacity-15" />
          <div className="pointer-events-none absolute -right-8 top-12 size-24 rounded-full border-[12px] border-[#4b79d8]/20" />
          <div className="pointer-events-none absolute -bottom-7 left-8 h-10 w-28 -rotate-6 rounded-full bg-[#e78ab2]/25" />

          <div className="animate-auth-panel-enter relative w-full max-w-[440px]">
            {logo && (
              <div className="mb-5 flex justify-center [--chimii-brand-surface:#f5f0e6] lg:[--chimii-brand-surface:#fffdf7]">
                {logo}
              </div>
            )}

            <div
              className="w-full rounded-[26px] border-2 border-[#12130f] bg-[#fffdf7] p-5 shadow-[7px_8px_0_rgba(18,19,15,0.14)] sm:p-8"
              data-auth-step={step}
            >
              {step === "cli_confirm" && existingUser ? (
                <>
                  <div className="text-center">
                    <p className="text-[10px] font-black uppercase tracking-[0.18em] text-[#4b79d8]">
                      {t(($) => $.cli.eyebrow)}
                    </p>
                    <h1
                      id="auth-title"
                      className="mt-3 text-balance text-3xl font-black leading-tight tracking-[-0.035em]"
                    >
                      {t(($) => $.cli.title)}
                    </h1>
                    <p className="mt-3 text-sm leading-6 text-[#12130f]/58">
                      {t(($) => $.cli.description, {
                        email: existingUser.email,
                      })}
                    </p>
                  </div>
                  <div className="mt-7 flex flex-col gap-3">
                    <Button
                      onClick={handleCliAuthorize}
                      disabled={loading}
                      aria-busy={loading}
                      className={primaryButtonClassName}
                    >
                      {loading && <Loader2 className="size-4 animate-spin" aria-hidden="true" />}
                      {loading
                        ? t(($) => $.cli.authorizing)
                        : t(($) => $.cli.authorize)}
                    </Button>
                    <Button
                      variant="ghost"
                      className="h-11 w-full rounded-full text-[#12130f]/62 hover:bg-[#12130f]/6 hover:text-[#12130f] dark:text-[#12130f]/62 dark:hover:bg-[#12130f]/6 dark:hover:text-[#12130f]"
                      onClick={() => {
                        setExistingUser(null);
                        handleChangeEmail();
                      }}
                    >
                      {t(($) => $.cli.different_account)}
                    </Button>
                  </div>
                </>
              ) : step === "code" ? (
                <>
                  <div className="text-center">
                    <p className="text-[10px] font-black uppercase tracking-[0.18em] text-[#4b79d8]">
                      {t(($) => $.verify.eyebrow)}
                    </p>
                    <h1
                      id="auth-title"
                      className="mt-3 text-balance text-3xl font-black leading-tight tracking-[-0.035em]"
                    >
                      {t(($) => $.verify.title)}
                    </h1>
                    <p className="mt-3 break-words text-sm leading-6 text-[#12130f]/58">
                      {t(($) => $.verify.description, { email })}
                    </p>
                  </div>

                  <div
                    ref={otpAreaRef}
                    className="mt-7 flex flex-col items-center gap-4"
                  >
                    <InputOTP
                      aria-label={t(($) => $.verify.code_label)}
                      aria-invalid={Boolean(error)}
                      aria-describedby="auth-message"
                      autoComplete="one-time-code"
                      inputMode="numeric"
                      pattern="[0-9]*"
                      maxLength={6}
                      value={code}
                      onChange={(value) => {
                        setCode(value);
                        setError("");
                        setNotice("");
                      }}
                      onComplete={handleVerify}
                      disabled={loading}
                    >
                      <InputOTPGroup aria-invalid={Boolean(error)}>
                        {[0, 1, 2, 3, 4, 5].map((index) => (
                          <InputOTPSlot
                            key={index}
                            index={index}
                            className="size-9 border-[#12130f]/28 bg-white text-lg font-extrabold text-[#12130f] min-[360px]:size-11 sm:size-12 sm:text-xl dark:bg-white dark:text-[#12130f]"
                          />
                        ))}
                      </InputOTPGroup>
                    </InputOTP>

                    {loading && (
                      <p className="flex items-center gap-2 text-sm font-semibold text-[#12130f]/58" aria-live="polite">
                        <Loader2 className="size-4 animate-spin" aria-hidden="true" />
                        {t(($) => $.verify.verifying)}
                      </p>
                    )}

                    <div
                      id="auth-message"
                      className="flex min-h-6 items-center text-center"
                      aria-live="polite"
                      aria-atomic="true"
                    >
                      {error ? (
                        <p className="text-sm font-semibold text-[#b3261e]">{error}</p>
                      ) : notice ? (
                        <p className="text-sm font-semibold text-[#237a58]">{notice}</p>
                      ) : null}
                    </div>

                    <button
                      type="button"
                      onClick={handleResend}
                      disabled={cooldown > 0 || resending || loading}
                      className="min-h-11 rounded-full px-4 text-sm font-bold text-[#12130f] underline decoration-[#12130f]/25 underline-offset-4 transition-colors hover:decoration-[#12130f] focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[#12130f] disabled:cursor-not-allowed disabled:text-[#12130f]/38 disabled:no-underline"
                    >
                      {resending
                        ? t(($) => $.verify.resending)
                        : cooldown > 0
                          ? t(($) => $.verify.resend_cooldown, {
                              seconds: cooldown,
                            })
                          : t(($) => $.verify.resend)}
                    </button>
                    <Button
                      type="button"
                      variant="ghost"
                      className="h-11 w-full rounded-full text-[#12130f]/62 hover:bg-[#12130f]/6 hover:text-[#12130f] dark:text-[#12130f]/62 dark:hover:bg-[#12130f]/6 dark:hover:text-[#12130f]"
                      onClick={handleChangeEmail}
                      disabled={loading}
                    >
                      {t(($) => $.verify.change_email)}
                    </Button>
                  </div>
                </>
              ) : (
                <>
                  <div className="text-center">
                    <p className="text-[10px] font-black uppercase tracking-[0.18em] text-[#f05a3f]">
                      {t(($) => $.signin.eyebrow)}
                    </p>
                    <h1
                      id="auth-title"
                      className="mt-3 text-balance text-3xl font-black leading-tight tracking-[-0.035em] sm:text-[2rem]"
                    >
                      {allowSignup
                        ? t(($) => $.signin.title)
                        : t(($) => $.signin.title_existing)}
                    </h1>
                    <p className="mt-3 text-sm leading-6 text-[#12130f]/58">
                      {allowSignup
                        ? t(($) => $.signin.description)
                        : t(($) => $.signin.description_existing)}
                    </p>
                  </div>

                  <form
                    id="login-form"
                    onSubmit={handleSendCode}
                    className="mt-7 space-y-4"
                    noValidate={false}
                  >
                    <div className="space-y-2">
                      <Label
                        htmlFor="login-email"
                        className="font-bold text-[#12130f]"
                      >
                        {t(($) => $.common.email)}
                      </Label>
                      <Input
                        ref={emailInputRef}
                        id="login-email"
                        name="email"
                        type="email"
                        inputMode="email"
                        autoComplete="email"
                        spellCheck={false}
                        placeholder={t(($) => $.common.email_placeholder)}
                        value={email}
                        onChange={(event) => {
                          setEmail(event.target.value);
                          setError("");
                        }}
                        aria-invalid={Boolean(error)}
                        aria-describedby="auth-message"
                        className="h-12 rounded-[14px] border-2 border-[#12130f]/22 bg-white px-4 text-base text-[#12130f] placeholder:text-[#12130f]/34 focus-visible:border-[#4b79d8] focus-visible:ring-[#4b79d8]/20 dark:border-[#12130f]/22 dark:bg-white dark:text-[#12130f] dark:placeholder:text-[#12130f]/34 md:text-base"
                        required
                      />
                    </div>

                    <div
                      id="auth-message"
                      className="flex min-h-6 items-center"
                      aria-live="polite"
                      aria-atomic="true"
                    >
                      {error ? (
                        <p className="text-sm font-semibold text-[#b3261e]">{error}</p>
                      ) : null}
                    </div>

                    <Button
                      type="submit"
                      className={primaryButtonClassName}
                      disabled={loading}
                      aria-busy={loading}
                    >
                      {loading && <Loader2 className="size-4 animate-spin" aria-hidden="true" />}
                      {loading
                        ? t(($) => $.signin.sending)
                        : t(($) => $.signin.continue)}
                    </Button>

                    {(google || onGoogleLogin) && (
                      <>
                        <div className="flex items-center gap-3 py-1" aria-hidden="true">
                          <span className="h-px flex-1 bg-[#12130f]/14" />
                          <span className="text-[11px] font-bold uppercase tracking-[0.14em] text-[#12130f]/38">
                            {t(($) => $.signin.or)}
                          </span>
                          <span className="h-px flex-1 bg-[#12130f]/14" />
                        </div>
                        <Button
                          type="button"
                          variant="outline"
                          className="h-12 w-full rounded-full border-2 border-[#12130f]/24 bg-white font-bold text-[#12130f] hover:border-[#12130f]/45 hover:bg-[#12130f]/4 hover:text-[#12130f] dark:border-[#12130f]/24 dark:bg-white dark:text-[#12130f] dark:hover:bg-[#12130f]/4"
                          onClick={handleGoogleLogin}
                          disabled={loading}
                        >
                          <svg
                            className="mr-1 size-4"
                            viewBox="0 0 24 24"
                            aria-hidden="true"
                          >
                            <path d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92a5.06 5.06 0 0 1-2.2 3.32v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.1z" fill="#4285F4" />
                            <path d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" fill="#34A853" />
                            <path d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z" fill="#FBBC05" />
                            <path d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" fill="#EA4335" />
                          </svg>
                          {t(($) => $.signin.google)}
                        </Button>
                      </>
                    )}
                  </form>
                </>
              )}
            </div>

            {extra && <div className="mt-5 w-full text-center">{extra}</div>}
          </div>
        </div>
      </div>
    </section>
  );
}
