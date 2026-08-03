"use client";

import { useState } from "react";
import { ArrowRight, Download, Loader2 } from "lucide-react";
import { Button, buttonVariants } from "@chimii/ui/components/ui/button";
import { ChimiiInventorBrand } from "@chimii/ui/components/common/chimii-inventor-brand";
import { cn } from "@chimii/ui/lib/utils";
import { DragStrip } from "@chimii/views/platform";
import { InventionLoop } from "../components/invention-loop";
import { useT } from "../../i18n";

const primaryCtaClassName =
  "h-12 rounded-full border-2 border-[#12130f] bg-[#f05a3f] px-6 font-extrabold text-white shadow-[0_5px_0_#b93b29] transition-[transform,box-shadow,background-color] hover:translate-y-0.5 hover:bg-[#e6533a] hover:shadow-[0_3px_0_#b93b29] active:translate-y-1 active:shadow-[0_1px_0_#b93b29] focus-visible:border-[#12130f] focus-visible:ring-[#f05a3f]/35 dark:bg-[#f05a3f] dark:text-white dark:hover:bg-[#e6533a]";

const secondaryCtaClassName =
  "h-12 rounded-full border-2 border-[#12130f]/24 bg-[#fffdf7]/75 px-5 font-bold text-[#12130f] hover:border-[#12130f]/45 hover:bg-[#fffdf7] hover:text-[#12130f] dark:border-[#12130f]/24 dark:bg-[#fffdf7]/75 dark:text-[#12130f] dark:hover:bg-[#fffdf7]";

/**
 * Step 0 — the one-shot product intro shown on every onboarding
 * entry (which-step-are-you-on is not persisted). Returning users
 * who are already onboarded never reach this screen; they're gated
 * out earlier by `!hasOnboarded`.
 *
 * Layout: two-column invention-studio hero on lg+, single column below.
 * Left = brand + editorial story + CTA; right = one morphing object that
 * moves from the Chimii mark through issue, agent hand-off, and shared
 * board states. The right column is decorative and hidden below lg so
 * the headline and actions stay focused on narrow viewports.
 *
 * `onSkip`, when provided, renders a secondary ghost CTA that marks
 * onboarding complete server-side and sends the user straight to
 * their existing workspace. OnboardingFlow only passes it when the
 * user has ≥ 1 workspace — without that, skipping lands in limbo.
 *
 * `isWeb` flips two things when true: the subheading acknowledges
 * that web users have an extra runtime step (so "3 minutes" stops
 * being a lie), and a "Download Desktop" secondary CTA surfaces
 * before the user has invested in questionnaire / workspace. Desktop
 * bundles a daemon, so the same prompt would be noise there.
 */
export function StepWelcome({
  onNext,
  onSkip,
  isWeb = false,
}: {
  onNext: () => void | Promise<void>;
  onSkip?: () => void | Promise<void>;
  isWeb?: boolean;
}) {
  const { t } = useT("onboarding");
  // Tracks which button is mid-flight so we can show a per-button
  // spinner and disable both while one is in progress.
  const [pending, setPending] = useState<"next" | "skip" | null>(null);

  const handleNext = async () => {
    if (pending) return;
    setPending("next");
    try {
      await onNext();
    } finally {
      setPending(null);
    }
  };

  const handleSkip = async () => {
    if (pending || !onSkip) return;
    setPending("skip");
    try {
      await onSkip();
    } finally {
      setPending(null);
    }
  };

  return (
    <div className="animate-onboarding-enter grid h-full min-h-[640px] grid-cols-1 bg-[#f5f0e6] text-[#12130f] lg:grid-cols-[minmax(0,1.12fr)_minmax(430px,0.88fr)]">
      {/* Left — restrained prose surface. */}
      <section className="relative flex min-w-0 flex-col overflow-hidden">
        <div className="chimii-invention-grid pointer-events-none absolute inset-0 opacity-20" />
        <div className="pointer-events-none absolute -left-12 bottom-16 size-28 rotate-12 rounded-[28px] border-[14px] border-[#f6c84a]/28" />
        <DragStrip />
        <div className="relative flex flex-1 flex-col justify-center px-6 pb-8 pt-2 sm:px-10 md:px-16 lg:px-16 xl:px-24">
          <div className="flex w-full max-w-[590px] flex-col gap-6 xl:gap-7">
            <div className="flex flex-col items-start gap-4">
              <ChimiiInventorBrand className="[--chimii-brand-surface:#f5f0e6]" />
              <span className="rounded-full border border-[#12130f]/14 bg-[#fffdf7]/65 px-3 py-1.5 text-[10px] font-black uppercase tracking-[0.2em] text-[#12130f]/48">
                {t(($) => $.welcome.wordmark)}
              </span>
            </div>

            <h1 className="text-balance font-serif text-[3.25rem] font-medium leading-[0.98] tracking-[-0.045em] sm:text-[3.8rem] lg:text-[3.65rem] xl:text-[4.3rem]">
              {t(($) => $.welcome.headline_line1)}
              <br />
              {t(($) => $.welcome.headline_line2)}
              <br />
              <em className="italic text-[#f05a3f]">
                {t(($) => $.welcome.headline_emphasis)}
              </em>
            </h1>

            <div className="flex flex-col gap-4">
              <p className="max-w-[560px] text-lg font-medium leading-8 text-[#12130f]/72">
                {t(($) => $.welcome.lede)}
              </p>
              <p className="max-w-[540px] text-sm leading-6 text-[#12130f]/48">
                {isWeb
                  ? t(($) => $.welcome.lede_web)
                  : t(($) => $.welcome.lede_desktop)}
              </p>
            </div>

            <div className="flex flex-wrap items-center gap-3">
              {isWeb ? (
                <>
                  <Button
                    size="lg"
                    className={primaryCtaClassName}
                    onClick={handleNext}
                    disabled={pending !== null}
                  >
                    {pending === "next" && (
                      <Loader2 className="h-4 w-4 animate-spin" />
                    )}
                    {t(($) => $.welcome.continue_on_web)}
                    <ArrowRight className="h-4 w-4" />
                  </Button>
                  {/* Download remains a real link so middle-click, copy-link,
                      and opening in a new tab all work. It is deliberately
                      secondary here: the user already entered onboarding and
                      the runtime step offers the platform choice again. */}
                  <a
                    href="/download"
                    target="_blank"
                    rel="noopener noreferrer"
                    className={cn(
                      buttonVariants({ variant: "outline", size: "lg" }),
                      secondaryCtaClassName,
                    )}
                  >
                    <Download className="h-4 w-4" />
                    {t(($) => $.welcome.download_desktop)}
                  </a>
                </>
              ) : (
                <Button
                  size="lg"
                  className={primaryCtaClassName}
                  onClick={handleNext}
                  disabled={pending !== null}
                >
                  {pending === "next" && (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  )}
                  {t(($) => $.welcome.start_exploring)}
                  <ArrowRight className="h-4 w-4" />
                </Button>
              )}
              {onSkip && (
                <Button
                  size="lg"
                  variant="ghost"
                  className="h-12 rounded-full px-5 font-bold text-[#12130f]/58 hover:bg-[#12130f]/6 hover:text-[#12130f] dark:text-[#12130f]/58 dark:hover:bg-[#12130f]/6 dark:hover:text-[#12130f]"
                  onClick={handleSkip}
                  disabled={pending !== null}
                >
                  {pending === "skip" && (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  )}
                  {t(($) => $.welcome.skip_existing)}
                </Button>
              )}
            </div>
          </div>
        </div>
      </section>

      {/* Right — one continuous product story, hidden below lg. */}
      <aside className="relative hidden overflow-hidden border-l-2 border-[#12130f] bg-[#c9def7] lg:flex lg:min-h-0 lg:flex-col">
        <div className="chimii-invention-grid pointer-events-none absolute inset-0 opacity-45" />
        <div className="pointer-events-none absolute -right-12 top-20 size-36 rounded-full border-[16px] border-[#35a87d]/30" />
        <DragStrip />
        <div className="relative flex min-h-0 flex-1 flex-col items-center justify-center gap-7 px-8 py-10 xl:px-12">
          <InventionLoop />
          <p className="max-w-[440px] text-balance text-center font-serif text-[15px] font-medium italic leading-snug text-[#12130f]/58">
            {t(($) => $.welcome.illustration_caption)}
          </p>
        </div>
      </aside>
    </div>
  );
}
