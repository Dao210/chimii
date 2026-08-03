import type { OnboardingStep } from "./types";

/**
 * Canonical order of the persisted onboarding steps.
 *
 * Single source of truth for "what step comes after what" — consumed
 * by the UI progress indicator to compute `index of current_step` and
 * `total step count`. Inserting, reordering, or removing a step only
 * requires changing this array; every call site that reads it updates
 * automatically.
 *
 * Intentionally excludes "welcome": welcome is a first-entry product
 * intro, not a persisted step. It doesn't show a progress indicator
 * for the same reason — users shouldn't think of reading the intro
 * as progress toward completing setup.
 *
 * Two questions are intentionally NOT steps anymore:
 *
 *   - "source" (How did you hear about Chimii?) is pure attribution
 *     data with zero user-facing payoff, so it no longer taxes the
 *     critical path. It is collected post-onboarding by the workspace
 *     source-backfill prompt, and only after agents have completed
 *     work for the user — see `needs-backfill.ts`.
 *   - The legacy "role" question is no longer shown. "about_you" now
 *     collects a child's interests in the existing `use_case` slot so
 *     personalization stays compatible without asking adult job questions.
 *
 * Runtime connection is also not a user-facing step anymore. After the
 * creation space is chosen, the flow automatically selects an online,
 * remotely provisioned runtime (with an online local runtime as fallback)
 * and then enters the space. Local runtime management remains available
 * from the workspace Runtimes area.
 */
export const ONBOARDING_STEP_ORDER: readonly OnboardingStep[] = [
  "about_you",
  "workspace",
] as const;
