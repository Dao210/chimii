import type { Metadata } from "next";
import { ChimiiLanding } from "@/features/landing/components/chimii-landing";
import { RedirectIfAuthenticated } from "@/features/landing/components/redirect-if-authenticated";

export const metadata: Metadata = {
  title: {
    absolute: "MeridianOS — B2B Outbound · China → Global",
  },
  description:
    "No ads, no waiting for inquiries — direct your Agent in plain language to actively lock onto overseas B2B buyers and walk each one in. From market research to reply handling, the full six-step outbound loop.",
  openGraph: {
    title: "MeridianOS — B2B Outbound · China → Global",
    description:
      "Stop waiting for inbound leads. Actively walk overseas buyers in with AI agents.",
    url: "/",
  },
  alternates: {
    canonical: "/",
  },
};

export default function LandingPage() {
  return (
    <>
      <RedirectIfAuthenticated />
      <ChimiiLanding />
    </>
  );
}
