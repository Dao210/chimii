"use client";

import { useEffect, useRef, useState } from "react";
import { cn } from "@chimii/ui/lib/utils";
import { useLocale } from "../i18n";

/**
 * LiveDemoSection — Sword Health 风格的浅色 LIVE 演示面板。
 *
 * Layout:
 *  - 浅色面板 + 模拟浏览器 chrome
 *  - Left column: 5 workflow steps that highlight in sequence
 *  - Right column: visual that swaps between Analysis chart / Locked targets / Draft email / Reply card
 *  - Bottom: standby line + status pill
 *
 * The auto-cycling animation pauses when the user hovers the panel and resumes on
 * leave so visitors can read the content without being interrupted.
 */
export function LiveDemoSection() {
  const { t } = useLocale();
  const demo = t.liveDemo;

  // 4 visual states — analysis / locked / drafting / reply.
  // Mapped to the 5 workflow steps so each step shows the most relevant visual.
  const visualForStep = ["analysis", "analysis", "locked", "drafting", "reply"] as const;
  const [activeStep, setActiveStep] = useState(0);
  const [paused, setPaused] = useState(false);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    if (paused) return;
    timerRef.current = setTimeout(() => {
      setActiveStep((s) => (s + 1) % demo.steps.length);
    }, 2800);
    return () => {
      if (timerRef.current) clearTimeout(timerRef.current);
    };
  }, [activeStep, paused, demo.steps.length]);

  const activeVisual = visualForStep[activeStep] ?? "analysis";

  return (
    <section
      id="live-demo"
      className="relative overflow-hidden bg-[#f6f4f1] py-24 text-[#0a0d12] sm:py-32 lg:py-40"
    >
      <div className="mx-auto max-w-[1320px] px-4 sm:px-6 lg:px-8">
        <div className="max-w-[760px]">
          <p className="text-[11px] font-semibold uppercase tracking-[0.22em] text-[#0a0d12]/44">
            {demo.badge}
          </p>
          <h2 className="mt-4 text-[2.6rem] font-semibold leading-[1.05] tracking-[-0.03em] sm:text-[3.4rem] lg:text-[4.2rem]">
            {demo.title}
          </h2>
          <p className="mt-5 max-w-[560px] text-[15px] leading-7 text-[#0a0d12]/60 sm:text-[16px]">
            {demo.subtitle}
          </p>
        </div>

        <div
          className="mt-14 overflow-hidden rounded-2xl border border-[#0a0d12]/8 bg-white shadow-[0_30px_80px_rgba(10,13,18,0.08)]"
          onMouseEnter={() => setPaused(true)}
          onMouseLeave={() => setPaused(false)}
        >
          {/* Browser chrome */}
          <div className="flex items-center gap-3 border-b border-[#0a0d12]/6 bg-[#fafaf9] px-5 py-3">
            <div className="flex items-center gap-1.5">
              <span className="size-2.5 rounded-full bg-[#ff5f57]" />
              <span className="size-2.5 rounded-full bg-[#febc2e]" />
              <span className="size-2.5 rounded-full bg-[#28c840]" />
            </div>
            <div className="mx-auto flex items-center gap-2 rounded-md bg-[#0a0d12]/5 px-3 py-1 text-[11px] text-[#0a0d12]/56">
              <span className="size-1.5 rounded-full bg-emerald-500 animate-pulse" />
              {demo.browserBar}
            </div>
          </div>

          {/* Body — two columns on lg, stacked on smaller screens */}
          <div className="grid lg:grid-cols-[minmax(0,360px)_1fr]">
            {/* Left: workflow steps */}
            <div className="border-b border-[#0a0d12]/6 p-5 sm:p-6 lg:border-b-0 lg:border-r">
              <ol className="space-y-1">
                {demo.steps.map((step, i) => {
                  const isActive = i === activeStep;
                  const isDone = i < activeStep;
                  return (
                    <li key={step.id}>
                      <button
                        type="button"
                        onClick={() => setActiveStep(i)}
                        className={cn(
                          "flex w-full items-start gap-3 rounded-lg px-3 py-3 text-left transition-colors",
                          isActive
                            ? "bg-[#0a0d12]/[0.04]"
                            : "hover:bg-[#0a0d12]/[0.02]",
                        )}
                      >
                        <StepIndicator
                          state={isActive ? "active" : isDone ? "done" : "idle"}
                          index={i}
                        />
                        <div className="min-w-0 flex-1">
                          <div
                            className={cn(
                              "text-[13px] font-semibold leading-tight sm:text-[14px]",
                              isActive ? "text-[#0a0d12]" : "text-[#0a0d12]/72",
                            )}
                          >
                            {step.title}
                          </div>
                          <div className="mt-1 text-[12px] leading-snug text-[#0a0d12]/48 sm:text-[13px]">
                            {step.detail}
                          </div>
                        </div>
                      </button>
                    </li>
                  );
                })}
              </ol>

              <div className="mt-5 flex items-center gap-2 border-t border-[#0a0d12]/6 px-3 pt-4 text-[11px] text-[#0a0d12]/44">
                <span className="size-1.5 rounded-full bg-emerald-500/80" />
                {demo.standby}
              </div>
            </div>

            {/* Right: visual panel that swaps based on active step */}
            <div className="relative min-h-[420px] bg-gradient-to-br from-[#0a0d12]/[0.02] to-transparent p-5 sm:p-8">
              <VisualPanel
                visual={activeVisual}
                demo={demo}
              />
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}

function StepIndicator({
  state,
  index,
}: {
  state: "idle" | "active" | "done";
  index: number;
}) {
  if (state === "done") {
    return (
      <span className="mt-0.5 flex size-5 shrink-0 items-center justify-center rounded-full bg-emerald-500/15 text-emerald-700">
        <svg viewBox="0 0 12 12" className="size-3" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
          <path d="M2.5 6.5l2.5 2.5 4.5-5" />
        </svg>
      </span>
    );
  }
  if (state === "active") {
    return (
      <span className="relative mt-0.5 flex size-5 shrink-0 items-center justify-center">
        <span className="absolute inset-0 rounded-full bg-emerald-500/25 animate-ping" />
        <span className="relative size-2 rounded-full bg-emerald-500" />
      </span>
    );
  }
  return (
    <span className="mt-0.5 flex size-5 shrink-0 items-center justify-center rounded-full border border-[#0a0d12]/15 text-[10px] font-medium tabular-nums text-[#0a0d12]/40">
      {index + 1}
    </span>
  );
}

type VisualState = "analysis" | "locked" | "drafting" | "reply";

function VisualPanel({
  visual,
  demo,
}: {
  visual: VisualState;
  demo: ReturnType<typeof useLocale>["t"]["liveDemo"];
}) {
  return (
    <div className="relative h-full">
      <div key={visual} className="animate-[fadeIn_240ms_ease-out]">
        {visual === "analysis" && <AnalysisVisual demo={demo} />}
        {visual === "locked" && <LockedVisual demo={demo} />}
        {visual === "drafting" && <DraftingVisual demo={demo} />}
        {visual === "reply" && <ReplyVisual demo={demo} />}
      </div>
      <style jsx>{`
        @keyframes fadeIn {
          from { opacity: 0; transform: translateY(4px); }
          to   { opacity: 1; transform: translateY(0); }
        }
      `}</style>
    </div>
  );
}

function PanelLabel({ children }: { children: React.ReactNode }) {
  return (
    <div className="mb-3 flex items-center gap-2 text-[11px] font-semibold uppercase tracking-[0.14em] text-[#0a0d12]/44">
      <span className="size-1 rounded-full bg-emerald-500/70" />
      {children}
    </div>
  );
}

function AnalysisVisual({
  demo,
}: {
  demo: ReturnType<typeof useLocale>["t"]["liveDemo"];
}) {
  const max = Math.max(...demo.analysisItems.map((i) => i.percent));
  return (
    <div>
      <PanelLabel>{demo.analyzing}</PanelLabel>
      <h3 className="text-[1.6rem] font-semibold leading-tight tracking-[-0.02em] text-[#0a0d12] sm:text-[1.9rem]">
        MUL-18 · API Error Handling
      </h3>
      <div className="mt-5 space-y-3">
        {demo.analysisItems.map((item, i) => (
          <div key={i}>
            <div className="mb-1 flex items-baseline justify-between text-[12px] text-[#0a0d12]/64 sm:text-[13px]">
              <span>{item.label}</span>
              <span className="font-semibold tabular-nums text-[#0a0d12]">
                {item.percent}%
              </span>
            </div>
            <div className="h-1.5 overflow-hidden rounded-full bg-[#0a0d12]/8">
              <div
                className="h-full rounded-full bg-gradient-to-r from-emerald-500/80 to-emerald-400 transition-[width] duration-700"
                style={{ width: `${(item.percent / max) * 100}%` }}
              />
            </div>
          </div>
        ))}
      </div>
      <p
        className="mt-6 text-[12px] leading-relaxed text-[#0a0d12]/56 sm:text-[13px]"
        dangerouslySetInnerHTML={{ __html: demo.analysisNote }}
      />
    </div>
  );
}

function LockedVisual({
  demo,
}: {
  demo: ReturnType<typeof useLocale>["t"]["liveDemo"];
}) {
  return (
    <div>
      <PanelLabel>{demo.locked}</PanelLabel>
      <h3 className="text-[1.6rem] font-semibold leading-tight tracking-[-0.02em] text-[#0a0d12] sm:text-[1.9rem]">
        {demo.drafting.replace("EN · ", "")}
      </h3>
      <ul className="mt-5 space-y-2.5">
        {demo.lockedTargets.map((target, i) => (
          <li
            key={i}
            className="flex items-center gap-3 rounded-lg border border-[#0a0d12]/8 bg-[#0a0d12]/[0.02] px-3 py-2.5"
          >
            <span className="flex size-9 shrink-0 items-center justify-center rounded-full bg-[#0a0d12]/8 text-[11px] font-semibold text-[#0a0d12]/80">
              {target.initials}
            </span>
            <div className="min-w-0 flex-1">
              <div className="text-[13px] font-semibold text-[#0a0d12] sm:text-[14px]">
                {target.name}
              </div>
              <div className="text-[11px] text-[#0a0d12]/48 sm:text-[12px]">
                {target.meta}
              </div>
            </div>
            <span className="shrink-0 text-[11px] font-medium text-emerald-700 sm:text-[12px]">
              {target.tag}
            </span>
          </li>
        ))}
      </ul>
    </div>
  );
}

function DraftingVisual({
  demo,
}: {
  demo: ReturnType<typeof useLocale>["t"]["liveDemo"];
}) {
  return (
    <div>
      <PanelLabel>{demo.drafting}</PanelLabel>
      <div className="rounded-lg border border-[#0a0d12]/8 bg-[#0a0d12]/[0.02] p-4 sm:p-5">
        <div className="text-[11px] font-mono text-[#0a0d12]/44">
          {demo.draftHeader}
        </div>
        <p className="mt-3 text-[13px] leading-relaxed text-[#0a0d12]/80 sm:text-[14px]">
          {demo.draftBody}
        </p>
      </div>
    </div>
  );
}

function ReplyVisual({
  demo,
}: {
  demo: ReturnType<typeof useLocale>["t"]["liveDemo"];
}) {
  return (
    <div>
      <PanelLabel>{demo.replyTitle}</PanelLabel>
      <div className="rounded-lg border border-emerald-500/20 bg-emerald-50/60 p-4 sm:p-5">
        <div className="flex items-center gap-2">
          <span className="flex size-7 items-center justify-center rounded-full bg-emerald-500/15 text-[10px] font-semibold text-emerald-700">
            AR
          </span>
          <span className="text-[12px] text-[#0a0d12]/72 sm:text-[13px]">
            {demo.replyMeta}
          </span>
        </div>
        <p className="mt-3 text-[13px] leading-relaxed text-[#0a0d12]/88 sm:text-[14px]">
          {demo.replyBody}
        </p>
        <div className="mt-4 flex items-center justify-between border-t border-[#0a0d12]/8 pt-3 text-[11px] sm:text-[12px]">
          <span className="font-medium text-emerald-700">{demo.replyTag}</span>
          <span className="text-[#0a0d12]/40">{demo.replyFooter}</span>
        </div>
      </div>
    </div>
  );
}
