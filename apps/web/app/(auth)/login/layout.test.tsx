import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

vi.mock("@/features/landing/components/invention-landing", () => ({
  InventionHeader: ({ auth }: { auth?: boolean }) => (
    <header data-auth={String(auth)}>Marketing header</header>
  ),
  InventionFooter: () => <footer>Marketing footer</footer>,
}));

vi.mock("@/features/landing/fonts", () => ({
  instrumentSerif: { variable: "font-serif-test" },
  notoSerifSC: { variable: "font-serif-zh-test" },
}));

vi.mock("@/features/landing/i18n", () => ({
  LocaleProvider: ({ children }: { children: React.ReactNode }) => children,
}));

vi.mock("@/lib/request-locale", () => ({
  getRequestLocale: vi.fn().mockResolvedValue("en"),
}));

import LoginLayout from "./layout";

describe("LoginLayout", () => {
  it("wraps login content with the shared marketing header and footer", async () => {
    render(
      await LoginLayout({
        children: <div>Login content</div>,
      }),
    );

    expect(screen.getByRole("banner")).toHaveAttribute("data-auth", "true");
    expect(screen.getByRole("main")).toHaveTextContent("Login content");
    expect(screen.getByRole("contentinfo")).toHaveTextContent(
      "Marketing footer",
    );
  });
});
