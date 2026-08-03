import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { QuestionnaireAnswers } from "@chimii/core/onboarding";
import { I18nProvider } from "@chimii/core/i18n/react";
import enCommon from "../../locales/en/common.json";
import enOnboarding from "../../locales/en/onboarding.json";
import { StepAboutYou } from "./step-about-you";

const TEST_RESOURCES = { en: { common: enCommon, onboarding: enOnboarding } };

const EMPTY: QuestionnaireAnswers = {
  source: [],
  source_other: null,
  source_skipped: false,
  role: null,
  role_other: null,
  role_skipped: false,
  use_case: [],
  use_case_other: null,
  use_case_skipped: false,
  version: 2,
};

function renderStep(answers: QuestionnaireAnswers = EMPTY) {
  const onChange = vi.fn();
  const onAdvance = vi.fn();
  const onSkip = vi.fn();
  render(
    <I18nProvider locale="en" resources={TEST_RESOURCES}>
      <StepAboutYou
        answers={answers}
        onChange={onChange}
        onAdvance={onAdvance}
        onSkip={onSkip}
      />
    </I18nProvider>,
  );
  return { onChange, onAdvance, onSkip };
}

describe("StepAboutYou", () => {
  beforeEach(() => vi.restoreAllMocks());

  it("asks about a child's interests instead of an adult role", () => {
    renderStep();
    expect(screen.getByText("What you love")).toBeInTheDocument();
    expect(
      screen.getByText("What do you love making and exploring?"),
    ).toBeInTheDocument();
    expect(screen.getByText("Pick as many as you like.")).toBeInTheDocument();
    expect(
      screen.queryByText("Which best describes you?"),
    ).not.toBeInTheDocument();
  });

  it("stores an interest in the legacy use_case slot", async () => {
    const user = userEvent.setup();
    const { onChange } = renderStep();

    await user.click(
      screen.getByRole("checkbox", { name: /robots & machines/i }),
    );

    expect(onChange).toHaveBeenCalledWith({
      use_case: ["ship_code"],
      use_case_skipped: false,
    });
  });

  it("keeps Continue disabled until an interest is committed", async () => {
    const user = userEvent.setup();
    const { onAdvance } = renderStep();

    const cont = screen.getByRole("button", { name: /continue/i });
    expect(cont).toBeDisabled();
    await user.click(cont);
    expect(onAdvance).not.toHaveBeenCalled();
  });

  it("Continue clears any legacy role and advances", async () => {
    const user = userEvent.setup();
    const { onChange, onAdvance } = renderStep({
      ...EMPTY,
      role: "engineer",
      use_case: ["ship_code"],
    });

    await user.click(screen.getByRole("button", { name: /continue/i }));

    expect(onChange).toHaveBeenCalledWith({
      role: null,
      role_other: null,
      role_skipped: true,
    });
    expect(onAdvance).toHaveBeenCalledTimes(1);
  });

  it("a lone empty-text Something else does not enable Continue", () => {
    renderStep({ ...EMPTY, use_case: ["other"] });
    expect(screen.getByRole("button", { name: /continue/i })).toBeDisabled();
  });

  it("Something else with text enables Continue", () => {
    renderStep({
      ...EMPTY,
      use_case: ["other"],
      use_case_other: "music",
    });
    expect(
      screen.getByRole("button", { name: /continue/i }),
    ).toBeEnabled();
  });

  it("Skip clears legacy role and interests in one patch", async () => {
    const user = userEvent.setup();
    const { onChange, onSkip } = renderStep();

    await user.click(screen.getByRole("button", { name: /skip/i }));

    expect(onChange).toHaveBeenCalledWith({
      role: null,
      role_other: null,
      role_skipped: true,
      use_case: [],
      use_case_other: null,
      use_case_skipped: true,
    });
    expect(onSkip).toHaveBeenCalledTimes(1);
  });

  it("pre-fills multiple stored interests on re-entry", () => {
    renderStep({
      ...EMPTY,
      use_case: ["plan_research", "write_publish"],
    });

    expect(
      screen.getByRole("checkbox", { name: /science experiments/i }),
    ).toHaveAttribute("aria-checked", "true");
    expect(
      screen.getByRole("checkbox", { name: /stories & comics/i }),
    ).toHaveAttribute("aria-checked", "true");
  });
});
