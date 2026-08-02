"use client";

import Link from "next/link";
import { useLocale } from "../i18n";
import { useDashboardCtaHref } from "../utils/use-dashboard-cta";
import { heroButtonClassName } from "./shared";

/**
 * FounderSection — "为什么是我们" founder letter, anchored at #founder so the
 * meridianos.ai/#founder URL pattern maps cleanly. Sword Health 风格的浅色
 * 布局: 大标题 + 段落叙事 + punchline 引用块 + 署名/CTA。
 */
export function FounderSection() {
  const { t } = useLocale();
  const f = t.founder;
  const ctaHref = useDashboardCtaHref();

  return (
    <section
      id="founder"
      className="relative overflow-hidden bg-[#fafaf9] py-28 text-[#0a0d12] sm:py-36 lg:py-44"
    >
      {/* Subtle backdrop accent */}
      <div
        aria-hidden
        className="pointer-events-none absolute inset-0 opacity-[0.5]"
        style={{
          background:
            "radial-gradient(ellipse 80% 60% at 50% 0%, rgba(16,185,129,0.05), transparent 70%)",
        }}
      />

      <div className="relative mx-auto max-w-[820px] px-4 sm:px-6 lg:px-8">
        <p className="text-[11px] font-semibold uppercase tracking-[0.22em] text-[#0a0d12]/44">
          {f.label}
        </p>

        <h2 className="mt-6 text-[2.6rem] font-semibold leading-[1.05] tracking-[-0.03em] sm:text-[3.6rem] lg:text-[4.4rem]">
          {f.headlineLine1}
          <br />
          <span className="text-[#0a0d12]/56">{f.headlineLine2}</span>
        </h2>

        <div className="mt-12 space-y-7">
          {f.paragraphs.map((p, i) => (
            <p
              key={i}
              className="text-[15px] leading-[1.85] text-[#0a0d12]/72 sm:text-[17px] sm:leading-[1.85]"
            >
              {p}
            </p>
          ))}
        </div>

        {/* Punchline — pulled out with a left rule for emphasis */}
        <blockquote className="mt-12 border-l-2 border-[#0a0d12]/30 pl-6 sm:pl-8">
          <p className="text-[1.4rem] font-semibold leading-[1.4] tracking-[-0.02em] text-[#0a0d12] sm:text-[1.7rem] lg:text-[1.9rem]">
            {f.punchline}
          </p>
        </blockquote>

        <div className="mt-12 flex flex-wrap items-center gap-6">
          <div>
            <div className="text-[14px] font-semibold text-[#0a0d12] sm:text-[15px]">
              {f.name}
            </div>
            <div className="text-[12px] text-[#0a0d12]/52 sm:text-[13px]">
              {f.role}
            </div>
          </div>

          <Link
            href={ctaHref}
            className={heroButtonClassName("solid", "light")}
          >
            {t.waitlist.cta}
          </Link>
        </div>
      </div>
    </section>
  );
}
