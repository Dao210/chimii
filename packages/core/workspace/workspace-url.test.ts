import { describe, expect, it } from "vitest";
import { workspaceUrlHost } from "./workspace-url";

describe("workspaceUrlHost", () => {
  it("returns the host of a full app URL", () => {
    expect(workspaceUrlHost("https://chimii.example.com")).toBe(
      "chimii.example.com",
    );
  });

  it("ignores scheme, path, and trailing slash", () => {
    expect(workspaceUrlHost("https://chimii.example.com/")).toBe(
      "chimii.example.com",
    );
    expect(workspaceUrlHost("http://chimii.example.com/app/onboarding")).toBe(
      "chimii.example.com",
    );
  });

  it("preserves a non-default port", () => {
    expect(workspaceUrlHost("https://my.host:3000")).toBe("my.host:3000");
  });

  it("accepts a bare host without a scheme", () => {
    expect(workspaceUrlHost("chimii.example.com")).toBe("chimii.example.com");
    expect(workspaceUrlHost("chimii.example.com/path")).toBe(
      "chimii.example.com",
    );
  });

  it("falls back to the brand host when no app URL is configured", () => {
    expect(workspaceUrlHost("")).toBe("chimii.ai");
    expect(workspaceUrlHost("   ")).toBe("chimii.ai");
    expect(workspaceUrlHost(null)).toBe("chimii.ai");
    expect(workspaceUrlHost(undefined)).toBe("chimii.ai");
  });
});
