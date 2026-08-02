"use client";

import { LandingHeader } from "./landing-header";
import { LandingHero } from "./landing-hero";
import { LiveDemoSection } from "./live-demo-section";
import { ComparisonSection } from "./comparison-section";
import { FeaturesSection } from "./features-section";
import { TransitionSection } from "./transition-section";
import { WorkflowSection } from "./workflow-section";
import { FounderSection } from "./founder-section";
import { WaitlistSection } from "./waitlist-section";
import { FAQSection } from "./faq-section";
import { LandingFooter } from "./landing-footer";

/**
 * ChimiiLanding — Enterprise AI Workstation marketing page.
 *
 * Flow (inspired by meridianos.ai):
 *  1. Header (dark, absolute over hero)
 *  2. Hero — bold headline + stats band + trusted-by
 *  3. LiveDemo — "演示中" LIVE panel showing agent workflow
 *  4. Comparison — past vs present two-column
 *  5. Features — Product 01 (teammates / autonomous / skills / runtimes)
 *  6. Transition — single-line bridge from one product to the next
 *  7. Workflow — Product 02 (6-step enterprise AI workflow)
 *  8. Founder — #founder letter
 *  9. Waitlist — pre-footer CTA
 * 10. FAQ — questions & answers
 * 11. Footer
 */
export function ChimiiLanding() {
  return (
    <>
      <LandingHeader variant="light" />
      <LandingHero />

      <LiveDemoSection />
      <ComparisonSection />
      <FeaturesSection />
      <TransitionSection />
      <WorkflowSection />
      <FounderSection />
      <WaitlistSection />
      <FAQSection />
      <LandingFooter />
    </>
  );
}
