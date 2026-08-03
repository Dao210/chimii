import { LocaleProvider } from "@/features/landing/i18n";
import {
  instrumentSerif,
  notoSerifSC,
} from "@/features/landing/fonts";
import { getRequestLocale } from "@/lib/request-locale";

const jsonLd = {
  "@context": "https://schema.org",
  "@graph": [
    {
      "@type": "Organization",
      name: "CHIMII 奇觅",
      url: "https://www.chimii.ai",
      sameAs: ["https://github.com/chimii-ai/chimii"],
    },
    {
      "@type": "Product",
      name: "CHIMII 奇觅发明家",
      category: "Educational invention kit",
      description:
        "An AI invention kit that helps kids turn ideas into real moving creations.",
    },
  ],
};

export default async function LandingLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const initialLocale = await getRequestLocale();

  return (
    <>
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{ __html: JSON.stringify(jsonLd) }}
      />
      <div className={`${instrumentSerif.variable} ${notoSerifSC.variable} landing-light h-full overflow-x-hidden overflow-y-auto bg-white`}>
        <LocaleProvider initialLocale={initialLocale}>{children}</LocaleProvider>
      </div>
    </>
  );
}
