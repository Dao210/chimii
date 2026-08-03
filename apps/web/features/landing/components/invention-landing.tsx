"use client";

import type { LucideIcon } from "lucide-react";
import { useState } from "react";
import {
  ArrowRight,
  Bot,
  CarFront,
  Castle,
  ChevronDown,
  Cog,
  DraftingCompass,
  Languages,
  Laugh,
  Lightbulb,
  Puzzle,
  RotateCcw,
  ShieldCheck,
  Sparkles,
} from "lucide-react";
import Link from "next/link";
import { useAuthStore } from "@chimii/core/auth";
import { ChimiiInventorBrand } from "@chimii/ui/components/common/chimii-inventor-brand";
import { cn } from "@chimii/ui/lib/utils";
import { localeLabels, locales, useLocale } from "../i18n";
import { useDashboardCtaHref } from "../utils/use-dashboard-cta";
import { inventionCopy } from "./invention-copy";

const sectionLabelClassName =
  "text-[11px] font-bold uppercase tracking-[0.22em] text-[#12130f]/48 sm:text-xs";

const parentValueIcons: LucideIcon[] = [
  Lightbulb,
  Puzzle,
  DraftingCompass,
  ShieldCheck,
  Sparkles,
];

const possibilityIcons: LucideIcon[] = [Bot, Castle, Cog, CarFront, Laugh];

const possibilityColors = [
  "bg-[#f05a3f]",
  "bg-[#f6c84a]",
  "bg-[#35a87d]",
  "bg-[#4b79d8]",
  "bg-[#e78ab2]",
];

export function InventionLanding() {
  const { locale } = useLocale();
  const copy = inventionCopy[locale];
  const user = useAuthStore((state) => state.user);
  const ctaHref = useDashboardCtaHref();

  return (
    <div className="min-h-full bg-[#f5f0e6] text-[#12130f] selection:bg-[#f6c84a] selection:text-[#12130f]">
      <InventionHeader
        ctaHref={ctaHref}
        ctaLabel={user ? copy.header.dashboard : copy.header.cta}
      />
      <main>
        <HeroSection ctaHref={ctaHref} />
        <FlowSection />
        <PossibilitiesSection />
        <ParentsSection />
        <ResponsibleAiSection />
        <FinalCtaSection ctaHref={ctaHref} />
      </main>
      <InventionFooter />
    </div>
  );
}

export function InventionHeader({
  ctaHref,
  ctaLabel,
  auth = false,
}: {
  ctaHref?: string;
  ctaLabel?: string;
  auth?: boolean;
}) {
  const { locale } = useLocale();
  const copy = inventionCopy[locale];

  return (
    <header className="sticky top-0 z-50 border-b border-[#12130f]/10 bg-[#f5f0e6]/92 backdrop-blur-xl">
      <div className="mx-auto flex h-[72px] max-w-[1440px] items-center justify-between gap-4 px-4 sm:px-6 lg:px-10">
        <BrandLockup />

        <nav
          aria-label={copy.navigation.aria}
          className="hidden items-center gap-7 lg:flex"
        >
          <a
            className="invention-nav-link"
            href={`${auth ? "/" : ""}#how-it-works`}
          >
            {copy.navigation.how}
          </a>
          <a
            className="invention-nav-link"
            href={`${auth ? "/" : ""}#possibilities`}
          >
            {copy.navigation.make}
          </a>
          <a
            className="invention-nav-link"
            href={`${auth ? "/" : ""}#for-parents`}
          >
            {copy.navigation.parents}
          </a>
        </nav>

        <div className="flex items-center gap-2 sm:gap-3">
          <LocaleSelect compact={auth} />
          <Link
            href={auth ? "/" : (ctaHref ?? "/login")}
            className="inline-flex min-h-10 items-center justify-center rounded-full bg-[#12130f] px-4 text-xs font-bold text-white transition-transform duration-200 hover:-translate-y-0.5 hover:bg-[#2a2c25] focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[#12130f] sm:px-5 sm:text-sm"
          >
            {auth ? (
              <>
                <span className="hidden min-[380px]:inline">
                  {copy.header.backHome}
                </span>
                <span className="min-[380px]:hidden">
                  {copy.header.backHomeShort}
                </span>
              </>
            ) : (
              ctaLabel
            )}
          </Link>
        </div>
      </div>
    </header>
  );
}

function LocaleSelect({
  inverse = false,
  compact = false,
}: {
  inverse?: boolean;
  compact?: boolean;
}) {
  const { locale, setLocale } = useLocale();
  const copy = inventionCopy[locale];
  const [isOpen, setIsOpen] = useState(false);

  return (
    <div className="relative">
      <button
        type="button"
        onClick={() => setIsOpen((open) => !open)}
        aria-expanded={isOpen}
        aria-haspopup="menu"
        aria-label={copy.header.language}
        className={cn(
          "flex min-h-10 items-center gap-2 rounded-full border px-3 text-xs font-bold transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 sm:text-sm",
          inverse
            ? "border-white/20 text-white/72 focus-visible:outline-white"
            : "border-[#12130f]/14 bg-white/45 text-[#12130f]/70 hover:bg-white/70 focus-visible:outline-[#12130f]",
        )}
      >
        <Languages className="size-4" aria-hidden />
        <span className={cn(compact && "max-[379px]:hidden")}>
          {localeLabels[locale]}
        </span>
        <ChevronDown
          className={cn(
            "size-3 transition-transform",
            isOpen && "rotate-180",
          )}
          aria-hidden
        />
      </button>

      {isOpen ? (
        <div
          role="menu"
          aria-label={copy.header.language}
          className={cn(
            "absolute right-0 top-[calc(100%+8px)] z-50 min-w-32 overflow-hidden rounded-[16px] border p-1.5 shadow-[0_16px_40px_rgba(18,19,15,0.18)]",
            inverse
              ? "border-white/12 bg-[#25261f] text-white"
              : "border-[#12130f]/12 bg-[#fffdf7] text-[#12130f]",
          )}
        >
          {locales.map((item) => (
            <button
              type="button"
              role="menuitem"
              key={item}
              onClick={() => {
                setLocale(item);
                setIsOpen(false);
              }}
              className={cn(
                "flex min-h-9 w-full items-center justify-between rounded-[10px] px-3 text-left text-xs font-bold transition-colors",
                inverse ? "hover:bg-white/8" : "hover:bg-[#12130f]/6",
                item === locale &&
                  (inverse ? "bg-white/8" : "bg-[#f6c84a]/45"),
              )}
            >
              {localeLabels[item]}
              {item === locale ? <span aria-hidden>•</span> : null}
            </button>
          ))}
        </div>
      ) : null}
    </div>
  );
}

function BrandLockup({ inverse = false }: { inverse?: boolean }) {
  return (
    <Link
      href="/"
      className="group flex shrink-0 items-center gap-2.5 rounded-sm focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-current"
      aria-label="CHIMII 奇觅"
    >
      <ChimiiInventorBrand
        inverse={inverse}
        className={cn(
          inverse
            ? "[--chimii-brand-surface:#11120f]"
            : "[--chimii-brand-surface:#f5f0e6]",
        )}
      />
    </Link>
  );
}

function HeroSection({ ctaHref }: { ctaHref: string }) {
  const { locale } = useLocale();
  const copy = inventionCopy[locale];

  return (
    <section className="relative overflow-hidden border-b border-[#12130f]/10">
      <HeroBackdrop />
      <div className="relative mx-auto max-w-[1440px] px-4 pb-16 pt-16 sm:px-6 sm:pb-24 sm:pt-24 lg:px-10 lg:pb-28 lg:pt-28">
        <div className="mx-auto max-w-[1040px] text-center">
          <p className={sectionLabelClassName}>{copy.hero.eyebrow}</p>
          <h1
            className={cn(
              "mx-auto mt-5 max-w-[1020px] text-balance font-[family-name:var(--font-serif)] text-[3.25rem] leading-[0.94] tracking-[-0.045em] sm:mt-7 sm:text-[5rem] lg:text-[6.6rem]",
              locale === "zh-Hans" &&
                "font-[family-name:var(--font-serif-zh)] text-[3rem] leading-[1.08] tracking-[-0.055em] sm:text-[4.6rem] lg:text-[5.8rem]",
            )}
          >
            {copy.hero.title}
          </h1>
          <p className="mx-auto mt-7 max-w-[740px] text-balance text-base leading-7 text-[#12130f]/64 sm:text-lg sm:leading-8">
            {copy.hero.body}
          </p>

          <div className="mt-9 flex flex-col items-center justify-center gap-3 sm:flex-row">
            <Link
              href={ctaHref}
              className="group inline-flex min-h-12 w-full items-center justify-center gap-2 rounded-full bg-[#f05a3f] px-6 text-sm font-extrabold text-white shadow-[0_6px_0_#b93b29] transition-[transform,box-shadow] duration-200 hover:translate-y-0.5 hover:shadow-[0_3px_0_#b93b29] focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-[#f05a3f] sm:w-auto"
            >
              {copy.hero.cta}
              <ArrowRight
                className="size-4 transition-transform group-hover:translate-x-0.5"
                aria-hidden
              />
            </Link>
            <a
              href="#how-it-works"
              className="inline-flex min-h-12 w-full items-center justify-center rounded-full border border-[#12130f]/18 bg-white/45 px-6 text-sm font-bold transition-colors hover:bg-white/80 focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-[#12130f] sm:w-auto"
            >
              {copy.hero.secondaryCta}
            </a>
          </div>

          <p className="mt-7 text-xs font-semibold leading-5 text-[#12130f]/46 sm:text-sm">
            {copy.hero.note}
          </p>
        </div>

        <HeroStudio />
      </div>
    </section>
  );
}

function HeroBackdrop() {
  return (
    <div className="pointer-events-none absolute inset-0" aria-hidden>
      <div className="invention-grid absolute inset-0 opacity-45" />
      <div className="absolute -left-16 top-32 size-40 rotate-12 rounded-[34px] border-[18px] border-[#f6c84a]/45" />
      <div className="absolute -right-10 top-12 size-28 rounded-full border-[14px] border-[#35a87d]/35" />
      <div className="absolute bottom-20 right-[12%] h-8 w-28 -rotate-12 rounded-full bg-[#e78ab2]/25" />
    </div>
  );
}

function HeroStudio() {
  const { locale } = useLocale();
  const copy = inventionCopy[locale];

  return (
    <div className="relative mx-auto mt-14 max-w-[1180px] sm:mt-20">
      <div className="absolute -left-3 -top-3 hidden rotate-[-7deg] rounded-full border border-[#12130f]/12 bg-[#f6c84a] px-4 py-2 text-[10px] font-black tracking-[0.15em] shadow-[0_4px_0_rgba(18,19,15,0.14)] sm:block">
        {copy.hero.studioLabel}
      </div>
      <div className="overflow-hidden rounded-[28px] border-2 border-[#12130f] bg-[#fffdf7] shadow-[0_16px_0_rgba(18,19,15,0.12)] sm:rounded-[38px]">
        <div className="flex h-12 items-center justify-between border-b-2 border-[#12130f] px-4 sm:px-6">
          <div className="flex gap-1.5" aria-hidden>
            <span className="size-2.5 rounded-full bg-[#f05a3f]" />
            <span className="size-2.5 rounded-full bg-[#f6c84a]" />
            <span className="size-2.5 rounded-full bg-[#35a87d]" />
          </div>
          <span className="text-[9px] font-black tracking-[0.18em] text-[#12130f]/42 sm:text-[10px]">
            CHIMII INVENTION STUDIO
          </span>
        </div>

        <div className="grid lg:grid-cols-[1fr_52px_1.08fr_52px_1fr]">
          <StudioPanel
            label={copy.hero.ideaLabel}
            className="bg-[#f8df78]"
          >
            <div className="relative mx-auto flex min-h-52 max-w-[290px] items-center justify-center sm:min-h-60">
              <div className="absolute left-1 top-2 size-16 rotate-[-8deg] rounded-[18px] border-2 border-[#12130f] bg-[#e78ab2]" />
              <div className="relative rounded-[24px] border-2 border-[#12130f] bg-[#fffdf7] p-5 text-left text-lg font-bold leading-snug shadow-[6px_7px_0_#12130f] sm:p-6 sm:text-xl">
                “{copy.hero.ideaPrompt}”
                <span className="absolute -bottom-[17px] right-8 block size-8 rotate-45 border-b-2 border-r-2 border-[#12130f] bg-[#fffdf7]" />
              </div>
            </div>
          </StudioPanel>

          <StudioArrow />

          <StudioPanel
            label={copy.hero.planLabel}
            className="bg-[#c9def7]"
          >
            <BlueprintPet />
            <p className="mt-3 text-center text-xs font-bold text-[#12130f]/56 sm:text-sm">
              {copy.hero.planNote}
            </p>
          </StudioPanel>

          <StudioArrow />

          <StudioPanel
            label={copy.hero.creationLabel}
            className="bg-[#f4a794]"
          >
            <RobotPet />
            <p className="mt-3 text-center text-xs font-bold text-[#12130f]/56 sm:text-sm">
              {copy.hero.creationNote}
            </p>
          </StudioPanel>
        </div>
      </div>
    </div>
  );
}

function StudioPanel({
  label,
  className,
  children,
}: {
  label: string;
  className: string;
  children: React.ReactNode;
}) {
  return (
    <div className={cn("min-w-0 p-5 sm:p-7 lg:p-6", className)}>
      <p className="text-center text-[10px] font-black uppercase tracking-[0.16em] text-[#12130f]/55 sm:text-xs">
        {label}
      </p>
      {children}
    </div>
  );
}

function StudioArrow() {
  return (
    <div className="flex h-10 items-center justify-center border-y-2 border-[#12130f] bg-[#fffdf7] lg:h-auto lg:border-x-2 lg:border-y-0">
      <ArrowRight className="size-5 rotate-90 lg:rotate-0" aria-hidden />
    </div>
  );
}

function BlueprintPet() {
  return (
    <div className="mx-auto mt-4 flex min-h-44 max-w-[300px] items-center justify-center rounded-[20px] border-2 border-[#12130f] bg-[#3767bb] p-4 shadow-[5px_6px_0_rgba(18,19,15,0.22)] sm:min-h-52">
      <svg
        viewBox="0 0 260 170"
        className="w-full text-white"
        fill="none"
        stroke="currentColor"
        strokeLinecap="round"
        strokeLinejoin="round"
        aria-hidden
      >
        <path d="M15 25H245M15 65H245M15 105H245M15 145H245M50 12V158M90 12V158M130 12V158M170 12V158M210 12V158" opacity=".18" />
        <path d="M74 62h102l18 45H56l18-45Z" strokeWidth="3" />
        <path d="M90 62 76 36l32 11M170 62l14-26-32 11" strokeWidth="3" />
        <path d="M72 107 55 145M104 107l-8 38M156 107l8 38M188 107l17 38" strokeWidth="5" />
        <path d="m56 76-32-18M196 76l34-22 12 10" strokeWidth="4" />
        <circle cx="103" cy="79" r="7" strokeWidth="3" />
        <circle cx="153" cy="79" r="7" strokeWidth="3" />
        <path d="m121 94 8 5 8-5" strokeWidth="3" />
        <path d="M36 152h32M82 152h30M150 152h30M193 152h32" strokeWidth="3" />
        <path d="M28 28h38M194 28h38" opacity=".7" />
        <path d="m39 22-11 6 11 6M221 22l11 6-11 6" opacity=".7" />
      </svg>
    </div>
  );
}

function RobotPet() {
  return (
    <div
      className="relative mx-auto mt-4 min-h-44 max-w-[300px] sm:min-h-52"
      aria-hidden
    >
      <div className="absolute left-1/2 top-6 h-28 w-40 -translate-x-1/2 rounded-[28px] border-[3px] border-[#12130f] bg-[#f6c84a] shadow-[5px_6px_0_#12130f]">
        <div className="absolute -left-5 -top-5 size-14 rotate-[-18deg] rounded-[16px] border-[3px] border-[#12130f] bg-[#4b79d8]" />
        <div className="absolute -right-5 -top-5 size-14 rotate-[18deg] rounded-[16px] border-[3px] border-[#12130f] bg-[#4b79d8]" />
        <div className="absolute left-5 top-7 size-7 rounded-full border-[3px] border-[#12130f] bg-white">
          <span className="absolute left-2 top-2 size-2 rounded-full bg-[#12130f]" />
        </div>
        <div className="absolute right-5 top-7 size-7 rounded-full border-[3px] border-[#12130f] bg-white">
          <span className="absolute left-2 top-2 size-2 rounded-full bg-[#12130f]" />
        </div>
        <div className="absolute bottom-5 left-1/2 h-4 w-10 -translate-x-1/2 rounded-b-full border-b-[3px] border-[#12130f]" />
      </div>
      <div className="absolute bottom-3 left-1/2 h-16 w-52 -translate-x-1/2 rounded-[24px] border-[3px] border-[#12130f] bg-[#35a87d] shadow-[5px_6px_0_#12130f]">
        <span className="absolute -bottom-5 left-6 size-9 rounded-full border-[3px] border-[#12130f] bg-[#fffdf7]" />
        <span className="absolute -bottom-5 right-6 size-9 rounded-full border-[3px] border-[#12130f] bg-[#fffdf7]" />
        <span className="absolute left-1/2 top-4 h-6 w-11 -translate-x-1/2 rounded-full border-[3px] border-[#12130f] bg-[#f05a3f]" />
      </div>
      <div className="invention-tail absolute -right-1 bottom-10 h-5 w-16 origin-left rotate-[-24deg] rounded-full border-[3px] border-[#12130f] bg-[#4b79d8]" />
    </div>
  );
}

function FlowSection() {
  const { locale } = useLocale();
  const copy = inventionCopy[locale];

  return (
    <section
      id="how-it-works"
      className="scroll-mt-24 bg-[#fffdf7] py-20 sm:py-28 lg:py-36"
    >
      <div className="mx-auto max-w-[1320px] px-4 sm:px-6 lg:px-10">
        <div className="max-w-[820px]">
          <p className={sectionLabelClassName}>{copy.flow.eyebrow}</p>
          <h2 className="mt-4 text-balance text-[2.55rem] font-black leading-[1.02] tracking-[-0.04em] sm:text-[4rem] lg:text-[5rem]">
            {copy.flow.title}
          </h2>
          <p className="mt-6 max-w-[700px] text-base leading-7 text-[#12130f]/62 sm:text-lg sm:leading-8">
            {copy.flow.body}
          </p>
        </div>

        <div className="mt-12 grid gap-5 lg:mt-16 lg:grid-cols-3">
          {copy.flow.steps.map((step, index) => (
            <article
              key={step.kicker}
              className={cn(
                "group relative overflow-hidden rounded-[28px] border-2 border-[#12130f] p-6 shadow-[0_8px_0_rgba(18,19,15,0.12)] transition-transform duration-300 hover:-translate-y-1 sm:p-8",
                index === 0 && "bg-[#f8df78]",
                index === 1 && "bg-[#c9def7]",
                index === 2 && "bg-[#b9ddce]",
              )}
            >
              <FlowIllustration index={index} />
              <p className="mt-8 text-[11px] font-black tracking-[0.16em] text-[#12130f]/48 sm:text-xs">
                {step.kicker}
              </p>
              <h3 className="mt-3 text-2xl font-black leading-tight tracking-[-0.025em] sm:text-[1.7rem]">
                {step.title}
              </h3>
              <p className="mt-4 text-sm leading-6 text-[#12130f]/62 sm:text-[15px] sm:leading-7">
                {step.body}
              </p>
            </article>
          ))}
        </div>
      </div>
    </section>
  );
}

function FlowIllustration({ index }: { index: number }) {
  if (index === 0) {
    return (
      <div className="relative flex h-44 items-center justify-center rounded-[20px] border-2 border-[#12130f] bg-[#fffdf7]">
        <span className="absolute left-5 top-5 size-6 rotate-12 rounded-[6px] bg-[#4b79d8]" />
        <span className="absolute bottom-5 right-5 size-8 rounded-full bg-[#f05a3f]" />
        <Lightbulb className="size-20 stroke-[1.5]" aria-hidden />
        <span className="absolute right-[30%] top-[28%] size-3 rounded-full bg-[#f6c84a]" />
      </div>
    );
  }

  if (index === 1) {
    return (
      <div className="invention-blueprint relative flex h-44 items-center justify-center overflow-hidden rounded-[20px] border-2 border-[#12130f] bg-[#3767bb] text-white">
        <DraftingCompass className="size-20 stroke-[1.4]" aria-hidden />
        <span className="absolute left-5 top-5 rounded-full border border-white/45 px-2 py-1 text-[9px] font-bold tracking-wider">
          STEP 04 / 12
        </span>
        <span className="absolute bottom-6 right-7 h-px w-20 bg-white/70" />
      </div>
    );
  }

  return (
    <div className="relative flex h-44 items-center justify-center overflow-hidden rounded-[20px] border-2 border-[#12130f] bg-[#fffdf7]">
      <Cog className="invention-gear size-24 stroke-[1.35] text-[#35a87d]" aria-hidden />
      <RotateCcw className="absolute size-11 stroke-[1.5] text-[#f05a3f]" aria-hidden />
      <span className="absolute right-6 top-6 size-4 rounded-full bg-[#f6c84a]" />
      <span className="absolute bottom-6 left-7 h-4 w-16 rounded-full bg-[#4b79d8]" />
    </div>
  );
}

function PossibilitiesSection() {
  const { locale } = useLocale();
  const copy = inventionCopy[locale];

  return (
    <section
      id="possibilities"
      className="scroll-mt-24 border-y-2 border-[#12130f] bg-[#f05a3f] py-20 sm:py-28 lg:py-32"
    >
      <div className="mx-auto max-w-[1320px] px-4 sm:px-6 lg:px-10">
        <div className="grid gap-8 lg:grid-cols-[0.8fr_1.2fr] lg:items-end">
          <div>
            <p className="text-[11px] font-black uppercase tracking-[0.22em] text-white/64 sm:text-xs">
              {copy.possibilities.eyebrow}
            </p>
            <h2 className="mt-4 max-w-[680px] text-balance text-[2.65rem] font-black leading-[1.02] tracking-[-0.04em] text-white sm:text-[4rem] lg:text-[4.7rem]">
              {copy.possibilities.title}
            </h2>
          </div>
          <p className="max-w-[560px] text-base leading-7 text-white/72 lg:justify-self-end lg:text-lg lg:leading-8">
            {copy.possibilities.body}
          </p>
        </div>

        <div className="mt-12 grid gap-3 sm:grid-cols-2 lg:mt-16 lg:grid-cols-5">
          {copy.possibilities.items.map((item, index) => {
            const Icon = possibilityIcons[index] ?? Sparkles;
            return (
              <article
                key={item.title}
                className="group rounded-[24px] border-2 border-[#12130f] bg-[#fffdf7] p-4 shadow-[5px_6px_0_#12130f] transition-transform duration-300 hover:-translate-y-1 sm:p-5"
              >
                <div
                  className={cn(
                    "flex aspect-[1.25] items-center justify-center rounded-[18px] border-2 border-[#12130f]",
                    possibilityColors[index],
                  )}
                >
                  <Icon
                    className="size-16 stroke-[1.5] text-white transition-transform duration-300 group-hover:scale-110 group-hover:rotate-[-4deg]"
                    aria-hidden
                  />
                </div>
                <h3 className="mt-5 text-lg font-black leading-tight">
                  {item.title}
                </h3>
                <p className="mt-2 text-sm leading-5 text-[#12130f]/56">
                  {item.note}
                </p>
              </article>
            );
          })}
        </div>
      </div>
    </section>
  );
}

function ParentsSection() {
  const { locale } = useLocale();
  const copy = inventionCopy[locale];

  return (
    <section
      id="for-parents"
      className="scroll-mt-24 bg-[#11120f] py-20 text-white sm:py-28 lg:py-36"
    >
      <div className="mx-auto max-w-[1320px] px-4 sm:px-6 lg:px-10">
        <div className="grid gap-8 lg:grid-cols-[1fr_0.62fr] lg:items-end">
          <div>
            <p className="text-[11px] font-bold uppercase tracking-[0.22em] text-[#f6c84a] sm:text-xs">
              {copy.parents.eyebrow}
            </p>
            <h2 className="mt-4 max-w-[860px] text-balance text-[2.65rem] font-black leading-[1.02] tracking-[-0.04em] sm:text-[4rem] lg:text-[5rem]">
              {copy.parents.title}
            </h2>
          </div>
          <p className="max-w-[530px] text-base leading-7 text-white/54 lg:justify-self-end lg:text-lg lg:leading-8">
            {copy.parents.body}
          </p>
        </div>

        <div className="mt-12 grid gap-px overflow-hidden rounded-[28px] border border-white/15 bg-white/15 sm:grid-cols-2 lg:mt-16 lg:grid-cols-5">
          {copy.parents.values.map((value, index) => {
            const Icon = parentValueIcons[index] ?? Sparkles;
            return (
              <article key={value.title} className="bg-[#11120f] p-6 sm:p-7">
                <span className="flex size-11 items-center justify-center rounded-[14px] bg-[#f6c84a] text-[#11120f]">
                  <Icon className="size-5 stroke-2" aria-hidden />
                </span>
                <h3 className="mt-8 text-lg font-bold leading-tight">
                  {value.title}
                </h3>
                <p className="mt-3 text-sm leading-6 text-white/48">
                  {value.note}
                </p>
              </article>
            );
          })}
        </div>

        <div className="mt-6 grid overflow-hidden rounded-[28px] border-2 border-[#f6c84a] bg-[#f6c84a] text-[#11120f] lg:grid-cols-[0.9fr_1.1fr]">
          <div className="flex items-center border-b-2 border-[#11120f] p-7 sm:p-10 lg:border-b-0 lg:border-r-2">
            <h3 className="text-balance text-3xl font-black leading-[1.05] tracking-[-0.035em] sm:text-[2.65rem]">
              {copy.parents.screenTitle}
            </h3>
          </div>
          <div className="invention-dots flex items-center bg-[#fffdf7] p-7 sm:p-10">
            <p className="max-w-[660px] text-base font-semibold leading-7 text-[#11120f]/66 sm:text-lg sm:leading-8">
              {copy.parents.screenBody}
            </p>
          </div>
        </div>
      </div>
    </section>
  );
}

function ResponsibleAiSection() {
  const { locale } = useLocale();
  const copy = inventionCopy[locale];

  return (
    <section className="overflow-hidden bg-[#c9def7] py-20 sm:py-28 lg:py-36">
      <div className="mx-auto grid max-w-[1320px] gap-12 px-4 sm:px-6 lg:grid-cols-[0.9fr_1.1fr] lg:items-center lg:gap-20 lg:px-10">
        <div className="relative mx-auto w-full max-w-[560px]">
          <div className="relative aspect-square overflow-hidden rounded-full border-2 border-[#12130f] bg-[#fffdf7] shadow-[10px_12px_0_rgba(18,19,15,0.14)]">
            <div className="invention-grid absolute inset-0 opacity-55" />
            <div className="absolute left-[10%] top-[30%] flex size-[34%] items-center justify-center rounded-full border-2 border-[#12130f] bg-[#f6c84a]">
              <span className="text-center text-[10px] font-black uppercase tracking-[0.12em] sm:text-xs">
                {copy.ai.childRole}
              </span>
            </div>
            <div className="absolute right-[10%] top-[30%] flex size-[34%] items-center justify-center rounded-full border-2 border-[#12130f] bg-[#4b79d8] p-3 text-white">
              <span className="text-center text-[10px] font-black uppercase tracking-[0.12em] sm:text-xs">
                {copy.ai.aiRole}
              </span>
            </div>
            <div className="absolute left-1/2 top-1/2 flex size-[19%] -translate-x-1/2 -translate-y-1/2 items-center justify-center rounded-[22px] border-2 border-[#12130f] bg-[#f05a3f] text-white shadow-[4px_5px_0_#12130f]">
              <Sparkles className="size-8 sm:size-11" aria-hidden />
            </div>
          </div>
          <span className="absolute -bottom-3 right-[12%] rotate-3 rounded-full border-2 border-[#12130f] bg-[#35a87d] px-4 py-2 text-[10px] font-black tracking-[0.12em] text-white shadow-[4px_5px_0_#12130f] sm:text-xs">
            HUMAN-MADE WITH AI
          </span>
        </div>

        <div>
          <p className={sectionLabelClassName}>{copy.ai.eyebrow}</p>
          <h2 className="mt-4 text-balance text-[2.7rem] font-black leading-[1.01] tracking-[-0.04em] sm:text-[4.1rem] lg:text-[5.2rem]">
            {copy.ai.title}
          </h2>
          <p className="mt-7 max-w-[680px] text-base font-medium leading-7 text-[#12130f]/62 sm:text-lg sm:leading-8">
            {copy.ai.body}
          </p>
        </div>
      </div>
    </section>
  );
}

function FinalCtaSection({ ctaHref }: { ctaHref: string }) {
  const { locale } = useLocale();
  const copy = inventionCopy[locale];

  return (
    <section className="relative overflow-hidden border-y-2 border-[#12130f] bg-[#f6c84a] py-20 sm:py-28 lg:py-32">
      <div className="invention-grid pointer-events-none absolute inset-0 opacity-30" />
      <div className="relative mx-auto max-w-[920px] px-4 text-center sm:px-6">
        <p className={sectionLabelClassName}>{copy.cta.eyebrow}</p>
        <h2 className="mt-4 text-balance text-[3rem] font-black leading-[0.98] tracking-[-0.045em] sm:text-[4.6rem] lg:text-[5.7rem]">
          {copy.cta.title}
        </h2>
        <p className="mx-auto mt-6 max-w-[620px] text-base font-medium leading-7 text-[#12130f]/62 sm:text-lg">
          {copy.cta.body}
        </p>
        <Link
          href={ctaHref}
          className="group mt-9 inline-flex min-h-13 items-center justify-center gap-2 rounded-full border-2 border-[#12130f] bg-[#12130f] px-7 text-sm font-black text-white shadow-[0_6px_0_rgba(18,19,15,0.25)] transition-[transform,box-shadow] hover:translate-y-0.5 hover:shadow-[0_3px_0_rgba(18,19,15,0.25)] focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-[#12130f] sm:text-base"
        >
          {copy.cta.button}
          <ArrowRight
            className="size-4 transition-transform group-hover:translate-x-1"
            aria-hidden
          />
        </Link>
      </div>
    </section>
  );
}

export function InventionFooter() {
  const { locale } = useLocale();
  const copy = inventionCopy[locale];

  return (
    <footer className="bg-[#11120f] text-white">
      <div className="mx-auto flex max-w-[1440px] flex-col gap-8 px-4 py-10 sm:px-6 md:flex-row md:items-end md:justify-between lg:px-10 lg:py-14">
        <div>
          <BrandLockup inverse />
          <p className="mt-4 max-w-[440px] text-sm leading-6 text-white/46">
            {copy.footer.tagline}
          </p>
        </div>
        <div className="flex flex-col items-start gap-4 md:items-end">
          <LocaleSelect inverse />
          <p className="text-xs text-white/32">
            {copy.footer.copyright.replace(
              "{year}",
              String(new Date().getFullYear()),
            )}
          </p>
        </div>
      </div>
    </footer>
  );
}
