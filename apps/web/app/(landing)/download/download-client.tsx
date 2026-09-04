"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { LandingHeader } from "@/features/landing/components/landing-header";
import { LandingFooter } from "@/features/landing/components/landing-footer";
import { DownloadHero } from "@/features/landing/components/download/hero";
import { AllPlatforms } from "@/features/landing/components/download/all-platforms";
import { CliSection } from "@/features/landing/components/download/cli-section";
import { CloudSection } from "@/features/landing/components/download/cloud-section";
import { useLocale } from "@/features/landing/i18n";
import {
  detectOS,
  type DetectResult,
} from "@/features/landing/utils/os-detect";
import type { LatestRelease } from "@/features/landing/utils/github-release";
import { CHIMII_GITHUB_RELEASES_URL } from "@chimii/core/constants/release-repository";

export function DownloadClient({ release }: { release: LatestRelease }) {
  const [detected, setDetected] = useState<DetectResult | null>(null);

  useEffect(() => {
    let cancelled = false;
    detectOS().then((result) => {
      if (cancelled) return;
      setDetected(result);
    });
    return () => {
      cancelled = true;
    };
  }, []);

  const releaseHtmlUrl = release.htmlUrl ?? CHIMII_GITHUB_RELEASES_URL;

  return (
    <>
      {/* Positioning context for the dark-variant LandingHeader —
          mirrors chimii-landing.tsx. The header is `absolute top-0
          inset-x-0`, so it anchors to this `relative` wrapper and
          scrolls off together with the dark hero below. Without the
          wrapper, `absolute` would escape to the initial containing
          block and read as fixed. */}
      <div className="relative">
        <LandingHeader variant="dark" />
        <DownloadHero
          detected={detected}
          assets={release.assets}
          releaseStatus={release.status}
        />
      </div>

      <AllPlatforms
        assets={release.assets}
        fallbackHref={CHIMII_GITHUB_RELEASES_URL}
      />
      <CliSection />
      <CloudSection />
      <VersionInfoFooter
        version={release.version}
        releaseHtmlUrl={releaseHtmlUrl}
        status={release.status}
      />
      <LandingFooter />
    </>
  );
}

function VersionInfoFooter({
  version,
  releaseHtmlUrl,
  status,
}: {
  version: string | null;
  releaseHtmlUrl: string;
  status: LatestRelease["status"];
}) {
  const { t } = useLocale();
  const d = t.download.footer;

  return (
    <section className="bg-white pb-16 text-[#0a0d12] sm:pb-20">
      <div className="mx-auto flex max-w-[920px] flex-wrap items-center gap-x-6 gap-y-2 border-t border-[#0a0d12]/8 px-4 pt-8 text-[13px] text-[#0a0d12]/60 sm:px-6 lg:px-8">
        {version ? (
          <>
            <span>
              {d.currentVersion.replace("{version}", version)}
            </span>
            <span aria-hidden className="text-[#0a0d12]/25">
              ·
            </span>
            <Link
              href={releaseHtmlUrl}
              className="underline decoration-[#0a0d12]/30 underline-offset-4 hover:text-[#0a0d12] hover:decoration-[#0a0d12]/70"
              target="_blank"
              rel="noreferrer"
            >
              {d.releaseNotes.replace("{version}", version)}
            </Link>
            <span aria-hidden className="text-[#0a0d12]/25">
              ·
            </span>
          </>
        ) : (
          <>
            <span>{releaseStatusMessage(status, d)}</span>
            <span aria-hidden className="text-[#0a0d12]/25">
              ·
            </span>
          </>
        )}
        <Link
          href={CHIMII_GITHUB_RELEASES_URL}
          className="underline decoration-[#0a0d12]/30 underline-offset-4 hover:text-[#0a0d12] hover:decoration-[#0a0d12]/70"
          target="_blank"
          rel="noreferrer"
        >
          {d.allReleases}
        </Link>
      </div>
    </section>
  );
}

function releaseStatusMessage(
  status: LatestRelease["status"],
  copy: ReturnType<typeof useLocale>["t"]["download"]["footer"],
): string {
  if (status === "no_release") return copy.noRelease;
  if (status === "incomplete") return copy.incompleteRelease;
  return copy.sourceUnavailable;
}
