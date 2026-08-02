import type { Metadata } from "next";
import { ChimiiLanding } from "@/features/landing/components/chimii-landing";

export const metadata: Metadata = {
  title: "Homepage",
  description:
    "Chimii — open-source platform that turns coding agents into real teammates. Assign tasks, track progress, compound skills.",
  openGraph: {
    title: "Chimii — Project Management for Human + Agent Teams",
    description:
      "Manage your human + agent workforce in one place.",
    url: "/homepage",
  },
  alternates: {
    canonical: "/homepage",
  },
};

export default function HomepagePage() {
  return <ChimiiLanding />;
}
