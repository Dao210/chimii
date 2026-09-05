import { describe, expect, it, vi } from "vitest";
import { QueryClient } from "@tanstack/react-query";
vi.mock("@/data/api", () => ({
  api: {
    buildCreation: vi.fn(async (id, opts) => ({ id, signal: opts.signal })),
    circuitCreation: vi.fn(async (id, opts) => ({ id, signal: opts.signal })),
    listBuildSummaries: vi.fn(),
    listCircuitCreations: vi.fn(),
    buildParts: vi.fn(async (_q, cursor, opts) => ({
      parts: [],
      next_cursor: cursor,
      signal: opts.signal,
    })),
  },
}));
import { api } from "@/data/api";
import {
  makerKeys,
  buildDetailOptions,
  circuitDetailOptions,
  partsOptions,
} from "./maker";
describe("native creation queries", () => {
  it("loads older deep-linked creations directly, independently of recent lists", async () => {
    const qc = new QueryClient();
    expect(
      (await qc.fetchQuery(buildDetailOptions("workspace", "older-than-sixty")))
        .id,
    ).toBe("older-than-sixty");
    expect(
      (
        await qc.fetchQuery(
          circuitDetailOptions("workspace", "older-than-fifty"),
        )
      ).id,
    ).toBe("older-than-fifty");
    expect(api.listBuildSummaries).not.toHaveBeenCalled();
    expect(api.listCircuitCreations).not.toHaveBeenCalled();
    expect(
      vi.mocked(api.buildCreation).mock.calls[0]?.[1]?.signal,
    ).toBeInstanceOf(AbortSignal);
    qc.clear();
  });
  it("scopes catalog, inventory, progress and asset caches by workspace", () => {
    for (const resource of [
      "catalog",
      "inventory",
      "progress",
      "asset",
      "circuit",
    ])
      expect(makerKeys.item("one", resource, "id")).not.toEqual(
        makerKeys.item("two", resource, "id"),
      );
  });
  it("passes search pagination and cancellation through to the API", async () => {
    const qc = new QueryClient();
    await qc.fetchInfiniteQuery(partsOptions("workspace", "wheel"));
    expect(api.buildParts).toHaveBeenCalledWith("wheel", "", {
      signal: expect.any(AbortSignal),
    });
    qc.clear();
  });
});
