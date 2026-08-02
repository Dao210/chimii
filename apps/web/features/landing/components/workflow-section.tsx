"use client";

import { cn } from "@chimii/ui/lib/utils";
import { Check } from "lucide-react";
import { useLocale } from "../i18n";

/**
 * WorkflowSection — 6-step enterprise AI workflow, inspired by meridianos.ai's
 * product 02 multi-step showcase. Each step has a code label (STEP 01 / RESEARCH_CONTEXT),
 * title, one-line detail, and a panel that surfaces a representative UI mock.
 *
 * Layout: alternating two-column rows. Left column carries the step meta
 * (code/title/detail). Right column carries the visual panel. On lg+, rows
 * alternate left/right for editorial rhythm. Below lg, everything stacks.
 */
export function WorkflowSection() {
  const { t } = useLocale();
  const wf = t.workflow;

  return (
    <section
      id="workflow"
      className="bg-[#f6f4f1] py-24 text-[#0a0d12] sm:py-32 lg:py-40"
    >
      <div className="mx-auto max-w-[1320px] px-4 sm:px-6 lg:px-8">
        <div className="max-w-[860px]">
          <p className="text-[11px] font-semibold uppercase tracking-[0.22em] text-[#0a0d12]/40">
            {wf.label}
          </p>
          <h2 className="mt-4 font-semibold text-[2.6rem] leading-[1.05] tracking-[-0.03em] sm:text-[3.4rem] lg:text-[4.2rem]">
            {wf.headline}
          </h2>
          <p className="mt-6 max-w-[640px] text-[15px] leading-7 text-[#0a0d12]/60 sm:text-[16px]">
            {wf.subheading}
          </p>
        </div>

        <ol className="mt-20 space-y-16 sm:space-y-20 lg:space-y-24">
          {wf.steps.map((step, i) => (
            <WorkflowStep key={step.id} step={step} index={i} />
          ))}
        </ol>

        <p className="mt-16 border-t border-[#0a0d12]/10 pt-8 text-[12px] leading-relaxed text-[#0a0d12]/48 sm:text-[13px]">
          <span className="font-semibold uppercase tracking-wider text-[#0a0d12]/60">
            {wf.noteLabel}:
          </span>{" "}
          {wf.note}
        </p>
      </div>
    </section>
  );
}

function WorkflowStep({
  step,
  index,
}: {
  step: ReturnType<typeof useLocale>["t"]["workflow"]["steps"][number];
  index: number;
}) {
  // Alternate row direction on lg+ for editorial rhythm.
  const flipped = index % 2 === 1;

  return (
    <li
      className="relative grid gap-6 lg:grid-cols-2 lg:gap-12"
      data-step-index={index}
    >
      {/* Connector line on lg+ */}
      <span
        aria-hidden
        className="absolute left-[19px] top-0 hidden h-full w-px bg-[#0a0d12]/8 lg:block"
      />

      {/* Left/meta column */}
      <div
        className={cn(
          "relative lg:pl-12",
          flipped && "lg:order-2 lg:pl-0 lg:pr-12",
        )}
      >
        <div className="flex items-center gap-2">
          <span
            aria-hidden
            className="absolute left-0 top-1 hidden size-10 items-center justify-center rounded-full border border-[#0a0d12]/12 bg-white text-[12px] font-semibold tabular-nums text-[#0a0d12]/56 lg:flex"
          >
            {String(index + 1).padStart(2, "0")}
          </span>
          <span className="text-[11px] font-semibold uppercase tracking-[0.18em] text-[#0a0d12]/44">
            {step.code}
          </span>
        </div>

        <h3 className="mt-3 font-semibold text-[1.9rem] leading-[1.1] tracking-[-0.02em] text-[#0a0d12] sm:text-[2.2rem] lg:text-[2.4rem]">
          {step.title}
        </h3>
        <p className="mt-4 max-w-[460px] text-[14px] leading-[1.7] text-[#0a0d12]/60 sm:text-[15px]">
          {step.detail}
        </p>
      </div>

      {/* Right/panel column */}
      <div className={cn(flipped && "lg:order-1")}>
        <WorkflowPanel step={step} />
      </div>
    </li>
  );
}

function WorkflowPanel({
  step,
}: {
  step: ReturnType<typeof useLocale>["t"]["workflow"]["steps"][number];
}) {
  const panel = step.panel;
  return (
    <div className="overflow-hidden rounded-2xl border border-[#0a0d12]/8 bg-white shadow-[0_20px_50px_rgba(10,13,18,0.06)]">
      <div className="flex items-center justify-between border-b border-[#0a0d12]/6 px-5 py-3 sm:px-6">
        <span className="text-[12px] font-semibold text-[#0a0d12] sm:text-[13px]">
          {panel.title}
        </span>
        {panel.subtitle ? (
          <span className="text-[11px] text-[#0a0d12]/52 sm:text-[12px]">
            {panel.subtitle}
          </span>
        ) : null}
      </div>
      <div className="p-5 sm:p-6">
        <p className="text-[12px] leading-relaxed text-[#0a0d12]/56 sm:text-[13px]">
          {panel.body}
        </p>
        {panel.bullets && panel.bullets.length > 0 ? (
          <ul className="mt-4 space-y-2.5">
            {panel.bullets.map((b, i) => (
              <li key={i} className="flex items-start gap-2.5">
                <span className="mt-0.5 flex size-4 shrink-0 items-center justify-center rounded-full bg-emerald-500/15 text-emerald-700">
                  <Check className="size-2.5" />
                </span>
                <span className="text-[13px] leading-[1.55] text-[#0a0d12]/76 sm:text-[14px]">
                  {b}
                </span>
              </li>
            ))}
          </ul>
        ) : null}
      </div>
    </div>
  );
}
