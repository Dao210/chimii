"use client";

import Link from "next/link";
import { ArrowRight, Download } from "lucide-react";
import { cn } from "@chimii/ui/lib/utils";
import { useAuthStore } from "@chimii/core/auth";
import { useLocale } from "../i18n";
import { useDashboardCtaHref } from "../utils/use-dashboard-cta";
import {
  ClaudeCodeLogo,
  CodexLogo,
  GeminiCliLogo,
  OpenClawLogo,
  OpenCodeLogo,
  heroButtonClassName,
} from "./shared";

/**
 * LandingHero — Sword Health 风格的浅色明亮 hero。
 *
 * 设计要点:
 *  - 浅色基调 (warm white + 顶部柔和光晕),sans-serif 标题
 *  - 居中布局: eyebrow → 大标题 → 副标题 → CTA → stats band
 *  - 大数字 stats 突出 (Sword Health 标志性手法)
 *  - 全宽产品截图收尾
 */
export function LandingHero() {
  const { t } = useLocale();
  const user = useAuthStore((s) => s.user);
  const ctaHref = useDashboardCtaHref();

  return (
    <div className="relative overflow-hidden bg-[#fafaf9] text-[#0a0d12]">
      <LandingBackdrop />

      <main className="relative z-10">
        <section
          id="product"
          className="mx-auto max-w-[1320px] px-4 pb-16 pt-32 sm:px-6 sm:pt-36 lg:px-8 lg:pb-24 lg:pt-44"
        >
          <div className="mx-auto max-w-[1120px] text-center">
            <p className="text-[11px] font-semibold uppercase tracking-[0.22em] text-[#0a0d12]/48 sm:text-[12px]">
              {t.hero.eyebrow}
            </p>

            <h1 className="mt-6 text-[3.65rem] font-semibold leading-[0.98] tracking-[-0.035em] text-[#0a0d12] sm:mt-7 sm:text-[4.85rem] lg:text-[6.4rem]">
              {t.hero.headlineLine1}
              <br />
              <span className="text-[#0a0d12]/56">{t.hero.headlineLine2}</span>
            </h1>

            <p className="mx-auto mt-7 max-w-[760px] text-[15px] leading-7 text-[#0a0d12]/64 sm:text-[17px]">
              {t.hero.subheading}
            </p>

            <div className="mt-8 flex flex-wrap items-center justify-center gap-3">
              <Link
                href={ctaHref}
                className={heroButtonClassName("solid", "light")}
              >
                {user ? t.header.dashboard : t.hero.cta}
              </Link>
              <Link
                href="#workflow"
                className={cn(heroButtonClassName("ghost", "light"), "group")}
              >
                {t.hero.ctaSecondary}
                <ArrowRight
                  className="size-4 transition-transform group-hover:translate-x-0.5"
                  aria-hidden
                />
              </Link>
              <Link
                href="/download"
                className="group inline-flex items-center justify-center gap-1.5 rounded-[12px] px-3 py-3 text-[14px] font-semibold text-[#0a0d12]/60 transition-colors hover:text-[#0a0d12]"
              >
                <Download className="size-4" aria-hidden />
                {t.hero.downloadDesktop}
              </Link>
            </div>

            <StatsBand />
          </div>

          <div className="mt-10 flex flex-wrap items-center justify-center gap-x-6 gap-y-3">
            <span className="text-[15px] text-[#0a0d12]/44">
              {t.hero.worksWith}
            </span>
            <div className="flex flex-wrap items-center justify-center gap-x-5 gap-y-3">
              <div className="flex items-center gap-2.5 text-[#0a0d12]/72">
                <ClaudeCodeLogo className="size-5" />
                <span className="text-[15px] font-medium">Claude Code</span>
              </div>
              <div className="flex items-center gap-2.5 text-[#0a0d12]/72">
                <CodexLogo className="size-5" />
                <span className="text-[15px] font-medium">Codex</span>
              </div>
              <div className="flex items-center gap-2.5 text-[#0a0d12]/72">
                <GeminiCliLogo className="size-5" />
                <span className="text-[15px] font-medium">Gemini CLI</span>
              </div>
              <div className="flex items-center gap-2.5 text-[#0a0d12]/72">
                <OpenClawLogo className="size-5" />
                <span className="text-[15px] font-medium">OpenClaw</span>
              </div>
              <div className="flex items-center gap-2.5 text-[#0a0d12]/72">
                <OpenCodeLogo className="size-5" />
                <span className="text-[15px] font-medium">OpenCode</span>
              </div>
            </div>
          </div>

        </section>
      </main>
    </div>
  );
}

function StatsBand() {
  const { t } = useLocale();
  return (
    <div className="mt-14 sm:mt-16">
      <div className="mx-auto grid max-w-[760px] grid-cols-3 gap-4 sm:gap-8">
        {t.stats.items.map((item, i) => (
          <div
            key={i}
            className="border-l border-[#0a0d12]/12 pl-4 text-left sm:pl-6"
          >
            <div className="text-[2.2rem] font-semibold leading-none tracking-[-0.025em] text-[#0a0d12] sm:text-[2.8rem]">
              {item.value}
            </div>
            <div className="mt-2 text-[12px] leading-tight text-[#0a0d12]/52 sm:text-[13px]">
              {item.label}
            </div>
          </div>
        ))}
      </div>
      <p className="mx-auto mt-8 max-w-[680px] text-[12px] leading-relaxed text-[#0a0d12]/40 sm:text-[13px]">
        {t.stats.trustedBy}
      </p>
    </div>
  );
}

function LandingBackdrop() {
  return (
    <div
      aria-hidden
      className="pointer-events-none absolute inset-0"
      style={{
        background:
          "radial-gradient(ellipse 70% 50% at 50% 0%, rgba(16,185,129,0.06), transparent 70%), radial-gradient(ellipse 50% 40% at 80% 10%, rgba(59,130,246,0.04), transparent 70%)",
      }}
    />
  );
}
