import { LocaleProvider } from "@/features/landing/i18n";
import {
  InventionFooter,
  InventionHeader,
} from "@/features/landing/components/invention-landing";
import {
  instrumentSerif,
  notoSerifSC,
} from "@/features/landing/fonts";
import { getRequestLocale } from "@/lib/request-locale";

export default async function LoginLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const initialLocale = await getRequestLocale();

  return (
    <div
      className={`${instrumentSerif.variable} ${notoSerifSC.variable} landing-light h-full overflow-x-hidden overflow-y-auto bg-[#f5f0e6]`}
    >
      <LocaleProvider initialLocale={initialLocale}>
        <div className="flex min-h-full flex-col">
          <InventionHeader auth />
          <main className="flex min-h-[calc(100svh-72px)] flex-1">
            {children}
          </main>
          <InventionFooter />
        </div>
      </LocaleProvider>
    </div>
  );
}
