import type { Metadata } from "next";
import type { SupportedLocale } from "@chimii/core/i18n";
import { ChimiiLanding } from "@/features/landing/components/chimii-landing";
import { RedirectIfAuthenticated } from "@/features/landing/components/redirect-if-authenticated";
import { getRequestLocale } from "@/lib/request-locale";

const localizedMetadata: Record<
  SupportedLocale,
  { title: string; description: string }
> = {
  en: {
    title: "CHIMII 奇觅 — Imagine it. Build it. Bring it to life.",
    description:
      "An AI invention kit that helps kids turn ideas into real moving creations.",
  },
  "zh-Hans": {
    title: "CHIMII 奇觅发明家 — 让孩子成为真实世界的创造者",
    description:
      "孩子从任意想法出发，AI 理解现有零件，生成可执行方案，并陪伴孩子完成和改造。",
  },
  ja: {
    title: "CHIMII — 想像する。つくる。命を吹き込む。",
    description:
      "子どものアイデアを、本当に動く作品へ変える AI 発明キット。",
  },
  ko: {
    title: "CHIMII — 상상하고. 만들고. 생명을 불어넣어요.",
    description:
      "아이들의 아이디어를 실제로 움직이는 창작물로 바꿔 주는 AI 발명 키트.",
  },
};

export async function generateMetadata(): Promise<Metadata> {
  const locale = await getRequestLocale();
  const metadata = localizedMetadata[locale];

  return {
    title: { absolute: metadata.title },
    description: metadata.description,
    openGraph: {
      title: metadata.title,
      description: metadata.description,
      url: "/",
    },
    alternates: { canonical: "/" },
  };
}

export default function LandingPage() {
  return (
    <>
      <RedirectIfAuthenticated />
      <ChimiiLanding />
    </>
  );
}
