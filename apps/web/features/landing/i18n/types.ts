import type { SupportedLocale } from "@chimii/core/i18n";

export type Locale = SupportedLocale;
export type LandingDictionaryLocale = "en" | "zh" | "ko" | "ja";

export const locales: Locale[] = ["en", "zh-Hans", "ko", "ja"];

export const localeLabels: Record<Locale, string> = {
  en: "EN",
  "zh-Hans": "\u4e2d\u6587",
  ko: "\ud55c\uad6d\uc5b4",
  ja: "\u65e5\u672c\u8a9e",
};

export function toLandingDictionaryLocale(
  locale: Locale,
): LandingDictionaryLocale {
  if (locale === "ko") return "ko";
  if (locale === "ja") return "ja";
  return locale === "zh-Hans" ? "zh" : "en";
}

export function isZhLocale(locale: Locale): boolean {
  return locale === "zh-Hans";
}

type FeatureSection = {
  label: string;
  title: string;
  description: string;
  cards: { title: string; description: string }[];
};

type FooterGroup = {
  label: string;
  links: { label: string; href: string }[];
};

export type ContactSalesOption = { value: string; label: string };

export type LandingDict = {
  header: {
    cta: string;
    dashboard: string;
    navigation: string;
    openMenu: string;
    closeMenu: string;
  };
  hero: {
    eyebrow: string;
    headlineLine1: string;
    headlineLine2: string;
    subheading: string;
    cta: string;
    ctaSecondary: string;
    downloadDesktop: string;
    talkToSales: string;
    worksWith: string;
  };
  stats: {
    items: { value: string; label: string }[];
    trustedBy: string;
  };
  liveDemo: {
    badge: string;
    title: string;
    subtitle: string;
    browserBar: string;
    steps: {
      id: string;
      title: string;
      detail: string;
    }[];
    standby: string;
    analyzing: string;
    analysisItems: { label: string; percent: number }[];
    analysisNote: string;
    locked: string;
    lockedTargets: {
      initials: string;
      name: string;
      meta: string;
      tag: string;
    }[];
    drafting: string;
    draftHeader: string;
    draftBody: string;
    replyTitle: string;
    replyMeta: string;
    replyBody: string;
    replyTag: string;
    replyFooter: string;
  };
  comparison: {
    label: string;
    headlineLine1: string;
    headlineLine2: string;
    pastLabel: string;
    presentLabel: string;
    past: { question: string; answer: string }[];
    present: { question: string; answer: string }[];
  };
  features: {
    teammates: FeatureSection;
    autonomous: FeatureSection;
    skills: FeatureSection;
    runtimes: FeatureSection;
  };
  transition: {
    headlineLine1: string;
    headlineLine2: string;
    body: string;
  };
  workflow: {
    label: string;
    headline: string;
    subheading: string;
    steps: {
      id: string;
      code: string;
      title: string;
      detail: string;
      panel: {
        title: string;
        subtitle?: string;
        body: string;
        bullets?: string[];
      };
    }[];
    noteLabel: string;
    note: string;
  };
  founder: {
    label: string;
    headlineLine1: string;
    headlineLine2: string;
    paragraphs: string[];
    punchline: string;
    name: string;
    role: string;
  };
  waitlist: {
    label: string;
    headline: string;
    body: string;
    cta: string;
    note: string;
  };
  howItWorks: {
    label: string;
    headlineMain: string;
    headlineFaded: string;
    steps: { title: string; description: string }[];
    cta: string;
    ctaGithub: string;
  };
  openSource: {
    label: string;
    headlineLine1: string;
    headlineLine2: string;
    description: string;
    cta: string;
    highlights: { title: string; description: string }[];
  };
  faq: {
    label: string;
    headline: string;
    items: { question: string; answer: string }[];
  };
  footer: {
    tagline: string;
    cta: string;
    groups: {
      product: FooterGroup;
      resources: FooterGroup;
      company: FooterGroup;
    };
    copyright: string;
  };
  about: {
    title: string;
    nameLine: {
      prefix: string;
      mul: string;
      tiplexed: string;
      i: string;
      nformationAnd: string;
      c: string;
      omputing: string;
      a: string;
      gent: string;
    };
    paragraphs: string[];
    cta: string;
  };
  download: {
    hero: {
      macArm64: {
        title: string;
        sub: string;
        primary: string;
        altZip: string;
      };
      macIntel: {
        title: string;
        sub: string;
        primary: string;
        altZip: string;
      };
      winX64: { title: string; sub: string; primary: string };
      winArm64: { title: string; sub: string; primary: string };
      linux: {
        title: string;
        sub: string;
        primary: string;
        altFormats: string;
      };
      unknown: { title: string; sub: string };
      safariMacHint: string;
      archFallbackHint: string;
    };
    allPlatforms: {
      title: string;
      macArm64Label: string;
      macX64Label: string;
      winX64Label: string;
      winArm64Label: string;
      linuxX64Label: string;
      linuxArm64Label: string;
      formatDmg: string;
      formatZip: string;
      formatExe: string;
      formatAppImage: string;
      formatDeb: string;
      formatRpm: string;
      unavailable: string;
    };
    cli: {
      title: string;
      sub: string;
      installLabel: string;
      startLabel: string;
      sshNote: string;
      copyLabel: string;
      copiedLabel: string;
    };
    cloud: { title: string; sub: string };
    footer: {
      releaseNotes: string;
      allReleases: string;
      currentVersion: string;
      versionUnavailable: string;
    };
  };
  contactSales: {
    pageTitle: string;
    pageDescription: string;
    eyebrow: string;
    title: string;
    subtitle: string;
    notice: { badge: string; body: string };
    fields: {
      firstName: string;
      lastName: string;
      businessEmail: string;
      businessEmailHint: string;
      companyName: string;
      companySize: string;
      countryRegion: string;
      useCase: string;
      goals: string;
      goalsHint: string;
      selectPlaceholder: string;
      submit: string;
      submitting: string;
    };
    companySizes: ContactSalesOption[];
    useCases: ContactSalesOption[];
    countries: string[];
    consent: {
      intro: string;
      outreach: string;
      updates: string;
      unsubscribe: string;
      submitConsent: string;
      privacyLinkLabel: string;
      privacyLinkHref: string;
    };
    success: { title: string; message: string; cta: string };
    errors: {
      generic: string;
      rateLimit: string;
      freeEmail: string;
      invalidEmail: string;
    };
  };
};
