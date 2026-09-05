import { afterEach, describe, expect, it, vi } from "vitest";
import { ApiClient } from "../api/client";
const api = new ApiClient("https://api.example.test");
afterEach(() => vi.unstubAllGlobals());
describe("progress API response validation", () => {
  it("rejects invalid read, write and summary responses", async () => {
    vi.stubGlobal("fetch", vi.fn().mockImplementation(async () => new Response(JSON.stringify({ id: "c", revision: "wrong", creations: null }), { status: 200, headers: { "Content-Type": "application/json" } })));
    await expect(api.getBuildProgress("c")).rejects.toThrow();
    await expect(api.updateBuildProgress("c", { current_step: 1, expected_revision: 0 })).rejects.toThrow();
    await expect(api.listBuildSummaries()).rejects.toThrow();
  });
});
