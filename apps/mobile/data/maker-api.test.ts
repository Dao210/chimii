import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
vi.hoisted(() => {
  process.env.EXPO_PUBLIC_API_URL = "https://fixture.invalid";
});
vi.mock("react-native", () => ({ Platform: { OS: "android" } }));
vi.mock("expo-constants", () => ({
  default: { expoConfig: { version: "0.2.0" } },
}));
vi.mock("./workspace-store", () => ({ getCurrentSlug: () => "family" }));
import { api, ApiError } from "./api";
const fetchMock = vi.fn();
beforeEach(() => {
  vi.stubGlobal("fetch", fetchMock);
  fetchMock.mockReset();
  api.setToken("fixture-token");
  vi.spyOn(console, "log").mockImplementation(() => {});
  vi.spyOn(console, "warn").mockImplementation(() => {});
  vi.spyOn(console, "error").mockImplementation(() => {});
});
afterEach(() => {
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});
describe("maker HTTP contracts", () => {
  it("sends authenticated workspace headers and preserves inventory revisions", async () => {
    const stock = {
      configured: true,
      catalog_version: "v",
      revision: 6,
      items: [],
      future: true,
    };
    fetchMock.mockResolvedValue(new Response(JSON.stringify(stock)));
    expect(
      (await api.saveBrickInventory({ expected_revision: 5, items: [] }))
        .revision,
    ).toBe(6);
    const [, init] = fetchMock.mock.calls[0]!;
    expect(JSON.parse(init.body)).toEqual({ expected_revision: 5, items: [] });
    expect(init.headers).toMatchObject({
      Authorization: "Bearer fixture-token",
      "X-Workspace-Slug": "family",
    });
  });
  it("fails closed for malformed detail and circuit inventory responses", async () => {
    fetchMock.mockImplementation(
      async () => new Response(JSON.stringify({ id: "incomplete" })),
    );
    await expect(api.buildCreation("old")).rejects.toThrow("作品数据不完整");
    await expect(api.circuitCreation("old")).rejects.toThrow("作品数据不完整");
    await expect(api.circuitInventory("kit")).rejects.toThrow("作品数据不完整");
  });
  it("does not swallow 409 and does not implicitly mark the last step completed", async () => {
    fetchMock.mockResolvedValue(
      new Response(JSON.stringify({ message: "revision conflict" }), {
        status: 409,
      }),
    );
    await expect(
      api.saveBuildProgress("c", { current_step: 3, expected_revision: 2 }),
    ).rejects.toBeInstanceOf(ApiError);
    expect(JSON.parse(fetchMock.mock.calls[0]![1].body)).not.toHaveProperty(
      "completed",
    );
  });
  it("forwards cancellation throughout the response body read", async () => {
    const abort = new AbortController();
    let signal: AbortSignal;
    fetchMock.mockImplementation(async (_path, init) => {
      signal = init.signal;
      return {
        ok: true,
        status: 200,
        json: () =>
          new Promise((resolve, reject) => {
            signal.addEventListener("abort", () =>
              reject(new Error("cancelled")),
            );
          }),
      };
    });
    const request = api.brickInventory({ signal: abort.signal });
    await Promise.resolve();
    abort.abort();
    await expect(request).rejects.toThrow("cancelled");
    expect(signal!.aborted).toBe(true);
  });
  it("rejects non-GLB bytes rather than passing server HTML into the renderer", async () => {
    fetchMock.mockResolvedValue(new Response("<html>failed</html>"));
    await expect(api.buildPartAsset("v", "3001.dat")).rejects.toThrow(
      "无法读取零件模型",
    );
  });
});
