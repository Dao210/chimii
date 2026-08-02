"use client";

import Link from "next/link";
import { ArrowRight } from "lucide-react";
import { useLocale } from "../i18n";
import { useDashboardCtaHref } from "../utils/use-dashboard-cta";

/**
 * WaitlistSection — pre-footer CTA. Compact, centered. Pairs with the founder
 * letter above so visitors have a clear next step after reading the narrative.
 */
export function WaitlistSection() {
  const { t } = useLocale();
  const w = t.waitlist;
  const ctaHref = useDashboardCtaHref();

  return (
    <section
      id="waitlist"
      className="bg-white py-24 text-[#0a0d12] sm:py-32 lg:py-40"
    >
      <div className="mx-auto max-w-[760px] px-4 text-center sm:px-6 lg:px-8">
        <p className="text-[11px] font-semibold uppercase tracking-[0.22em] text-[#0a0d12]/40">
          {w.label}
        </p>

        <h2 className="mt-5 font-semibold text-[2.4rem] leading-[1.05] tracking-[-0.03em] sm:text-[3.2rem] lg:text-[3.8rem]">
          {w.headline}
        </h2>

        <p className="mx-auto mt-6 max-w-[560px] text-[15px] leading-7 text-[#0a0d12]/60 sm:text-[16px]">
          {w.body}
        </p>

        <div className="mt-10 flex flex-wrap items-center justify-center gap-4">
          <Link
            href={ctaHref}
            className="group inline-flex items-center justify-center gap-2 rounded-[12px] bg-[#0a0d12] px-6 py-3.5 text-[14px] font-semibold text-white transition-colors hover:bg-[#0a0d12]/88"
          >
            {w.cta}
            <ArrowRight className="size-4 transition-transform group-hover:translate-x-0.5" />
          </Link>
        </div>

        <p className="mt-5 text-[12px] text-[#0a0d12]/40 sm:text-[13px]">
          {w.note}
        </p>
      </div>
    </section>
  );
}
