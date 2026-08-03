import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { I18nProvider } from "@chimii/core/i18n/react";
import enCommon from "../../locales/en/common.json";
import enOnboarding from "../../locales/en/onboarding.json";

vi.mock("../components/invention-loop", () => ({
  InventionLoop: () => <div data-testid="invention-loop" />,
}));

import { StepWelcome } from "./step-welcome";

const TEST_RESOURCES = { en: { common: enCommon, onboarding: enOnboarding } };

function renderStep({
  isWeb = false,
  onNext = vi.fn(),
  onSkip,
}: {
  isWeb?: boolean;
  onNext?: () => void | Promise<void>;
  onSkip?: () => void | Promise<void>;
} = {}) {
  render(
    <I18nProvider locale="en" resources={TEST_RESOURCES}>
      <StepWelcome isWeb={isWeb} onNext={onNext} onSkip={onSkip} />
    </I18nProvider>,
  );
  return { onNext };
}

describe("StepWelcome", () => {
  it("renders the invention-loop story beside the welcome copy", () => {
    renderStep();

    expect(screen.getByText("CHIMII · AI INVENTION KIT")).toBeInTheDocument();
    expect(screen.getByTestId("invention-loop")).toBeInTheDocument();
    expect(
      screen.getByText("AI helps understand and plan", {
        exact: false,
      }),
    ).toBeInTheDocument();
  });

  it("keeps web continuation primary and desktop download secondary", async () => {
    const user = userEvent.setup();
    const onNext = vi.fn();
    renderStep({ isWeb: true, onNext });

    const continueButton = screen.getByRole("button", {
      name: /start inventing/i,
    });
    const downloadLink = screen.getByRole("link", {
      name: /download desktop/i,
    });

    expect(continueButton).toHaveClass("bg-[#f05a3f]");
    expect(downloadLink).toHaveAttribute("href", "/download");
    expect(downloadLink).toHaveAttribute("target", "_blank");

    await user.click(continueButton);
    expect(onNext).toHaveBeenCalledTimes(1);
  });

  it("shows the returning-user escape only when supplied", async () => {
    const user = userEvent.setup();
    const onSkip = vi.fn();
    const { unmount } = render(
      <I18nProvider locale="en" resources={TEST_RESOURCES}>
        <StepWelcome onNext={vi.fn()} onSkip={onSkip} />
      </I18nProvider>,
    );

    await user.click(
      screen.getByRole("button", { name: /open my invention space/i }),
    );
    expect(onSkip).toHaveBeenCalledTimes(1);

    unmount();
    renderStep();
    expect(
      screen.queryByRole("button", { name: /open my invention space/i }),
    ).not.toBeInTheDocument();
  });
});
