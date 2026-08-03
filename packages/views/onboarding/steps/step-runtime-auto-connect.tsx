"use client";

import { useCallback, useEffect, useRef } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Cloud, Monitor, Sparkles } from "lucide-react";
import { useWSEvent } from "@chimii/core/realtime";
import {
  runtimeKeys,
  runtimeListOptions,
} from "@chimii/core/runtimes/queries";
import type { AgentRuntime } from "@chimii/core/types";
import { DragStrip } from "@chimii/views/platform";
import { useT } from "../../i18n";

const AUTO_RUNTIME_TIMEOUT_MS = 12_000;

/**
 * Pick a runtime without asking the child or parent to understand runtime
 * setup. A remotely prepared cloud runtime is the product default; an
 * already-online local runtime is a supported fallback. Offline runtimes
 * are never selected because Helper creation needs a runnable target.
 */
export function pickAutomaticRuntime(
  runtimes: readonly AgentRuntime[],
): AgentRuntime | null {
  return (
    runtimes.find(
      (runtime) =>
        runtime.status === "online" && runtime.runtime_mode === "cloud",
    ) ??
    runtimes.find(
      (runtime) =>
        runtime.status === "online" && runtime.runtime_mode === "local",
    ) ??
    null
  );
}

/**
 * Non-interactive finishing state after the creation space is chosen.
 * It watches every runtime in the workspace (not only user-owned ones),
 * so a remotely provisioned shared runtime can be selected automatically.
 */
export function StepRuntimeAutoConnect({
  wsId,
  onResolved,
}: {
  wsId: string;
  onResolved: (runtime: AgentRuntime | null) => Promise<void>;
}) {
  const { t } = useT("onboarding");
  const queryClient = useQueryClient();
  const resolvedRef = useRef(false);
  const resolvingRef = useRef(false);
  const retryTimerRef = useRef<number | null>(null);

  const { data: runtimes = [] } = useQuery({
    ...runtimeListOptions(wsId),
    refetchInterval: (query) =>
      pickAutomaticRuntime(query.state.data ?? []) ? false : 1_500,
  });

  const refreshRuntimes = useCallback(() => {
    void queryClient.invalidateQueries({ queryKey: runtimeKeys.all(wsId) });
  }, [queryClient, wsId]);
  useWSEvent("daemon:register", refreshRuntimes);

  const resolveOnce = useCallback(
    async (runtime: AgentRuntime | null) => {
      if (resolvedRef.current || resolvingRef.current) return;
      resolvingRef.current = true;
      try {
        await onResolved(runtime);
        resolvedRef.current = true;
      } catch {
        // Completion can fail transiently even though runtime discovery
        // succeeded. Keep this automatic: retry instead of adding a setup
        // button or stranding the user on the finishing screen.
        resolvingRef.current = false;
        retryTimerRef.current = window.setTimeout(
          () => void resolveOnce(runtime),
          2_000,
        );
      }
    },
    [onResolved],
  );

  useEffect(() => {
    const runtime = pickAutomaticRuntime(runtimes);
    if (runtime) void resolveOnce(runtime);
  }, [resolveOnce, runtimes]);

  useEffect(() => {
    const timeout = window.setTimeout(
      () => void resolveOnce(null),
      AUTO_RUNTIME_TIMEOUT_MS,
    );
    return () => {
      window.clearTimeout(timeout);
      if (retryTimerRef.current !== null) {
        window.clearTimeout(retryTimerRef.current);
      }
    };
  }, [resolveOnce]);

  return (
    <div className="animate-onboarding-enter flex h-full min-h-0 flex-col overflow-hidden bg-[#fffdf7] text-[#1b2722]">
      <DragStrip />
      <main className="relative flex min-h-0 flex-1 items-center justify-center overflow-hidden px-6 py-12">
        <div
          aria-hidden
          className="absolute inset-0 opacity-45 [background-image:linear-gradient(to_right,rgba(27,39,34,0.055)_1px,transparent_1px),linear-gradient(to_bottom,rgba(27,39,34,0.055)_1px,transparent_1px)] [background-size:32px_32px]"
        />
        <div
          aria-hidden
          className="absolute -left-24 top-1/3 h-64 w-64 rounded-full bg-[#f4c95d]/20 blur-3xl"
        />
        <div
          aria-hidden
          className="absolute -right-20 bottom-1/4 h-72 w-72 rounded-full bg-[#6cb7b2]/20 blur-3xl"
        />

        <section className="relative z-10 w-full max-w-[620px] text-center">
          <div className="mx-auto flex h-20 w-20 items-center justify-center rounded-[24px] border border-[#1b2722]/10 bg-white shadow-[0_18px_45px_rgba(27,39,34,0.09)]">
            <div className="relative flex h-12 w-12 items-center justify-center rounded-2xl bg-[#f05a3f] text-white">
              <Sparkles className="h-5 w-5 motion-safe:animate-pulse" />
              <span className="absolute -right-1 -top-1 h-3 w-3 rounded-full border-2 border-white bg-[#35a87d] motion-safe:animate-ping" />
            </div>
          </div>

          <div className="mt-8 text-xs font-semibold uppercase tracking-[0.14em] text-[#1b2722]/55">
            {t(($) => $.step_runtime_auto.eyebrow)}
          </div>
          <h1 className="mt-3 text-balance font-serif text-[38px] font-medium leading-[1.08] tracking-tight sm:text-[46px]">
            {t(($) => $.step_runtime_auto.headline)}
          </h1>
          <p className="mx-auto mt-5 max-w-[530px] text-[15.5px] leading-7 text-[#1b2722]/70">
            {t(($) => $.step_runtime_auto.lede)}
          </p>

          <div className="mx-auto mt-9 grid max-w-[470px] grid-cols-[1fr_auto_1fr] items-center gap-3 rounded-2xl border border-[#1b2722]/10 bg-white/80 p-4 shadow-[0_12px_38px_rgba(27,39,34,0.06)] backdrop-blur-sm">
            <div className="flex items-center justify-center gap-2 text-sm font-medium">
              <Cloud className="h-4 w-4 text-[#6c63ff]" />
              {t(($) => $.step_runtime_auto.remote_label)}
            </div>
            <div className="flex items-center gap-1.5" aria-hidden>
              <span className="h-1.5 w-1.5 rounded-full bg-[#35a87d]/35 motion-safe:animate-pulse" />
              <span className="h-1.5 w-1.5 rounded-full bg-[#35a87d]/65 motion-safe:animate-pulse [animation-delay:160ms]" />
              <span className="h-1.5 w-1.5 rounded-full bg-[#35a87d] motion-safe:animate-pulse [animation-delay:320ms]" />
            </div>
            <div className="flex items-center justify-center gap-2 text-sm font-medium text-[#1b2722]/65">
              <Sparkles className="h-4 w-4 text-[#f05a3f]" />
              {t(($) => $.step_runtime_auto.product_label)}
            </div>
          </div>

          <p className="mx-auto mt-6 flex max-w-[500px] items-center justify-center gap-2 text-xs leading-5 text-[#1b2722]/55">
            <Monitor className="h-3.5 w-3.5 shrink-0" />
            {t(($) => $.step_runtime_auto.local_note)}
          </p>
        </section>
      </main>
    </div>
  );
}
