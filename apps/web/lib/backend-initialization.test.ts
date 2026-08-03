import { describe, expect, it } from "vitest";
import { shouldInitializeBackend } from "./backend-initialization";

describe("shouldInitializeBackend", () => {
  it.each([
    "/",
    "/about",
    "/changelog",
    "/contact-sales",
    "/download",
    "/homepage",
  ])("keeps the public marketing route %s backend-independent", (pathname) => {
    expect(shouldInitializeBackend(pathname)).toBe(false);
  });

  it("normalizes trailing slashes on public routes", () => {
    expect(shouldInitializeBackend("/download/")).toBe(false);
  });

  it.each([
    "/login",
    "/onboarding",
    "/auth/callback",
    "/acme/issues",
    "/slack/bind",
  ])("initializes the backend for the app route %s", (pathname) => {
    expect(shouldInitializeBackend(pathname)).toBe(true);
  });
});
