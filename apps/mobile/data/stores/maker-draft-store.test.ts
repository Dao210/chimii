import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
const storage = vi.hoisted(() => ({ getItemAsync: vi.fn(), setItemAsync: vi.fn<(key: string, value: string) => Promise<void>>().mockResolvedValue(undefined), deleteItemAsync: vi.fn(async () => {}) }));
vi.mock("expo-secure-store", () => storage);
let maker: typeof import("./maker-draft-store");
beforeEach(async () => { vi.useFakeTimers(); vi.resetModules(); vi.clearAllMocks(); storage.getItemAsync.mockResolvedValue(null); maker = await import("./maker-draft-store"); });
afterEach(() => { maker.useMakerDraftStore.getState().reset(); vi.useRealTimers(); });
describe("maker draft lifecycle", () => {
  it("persists the latest short idea and session without copying server inventory", async () => {
    await maker.restoreMakerDraft("family");
    maker.useMakerDraftStore.getState().patch("family", { prompt: "first" });
    maker.useMakerDraftStore.getState().patch("family", { prompt: "latest", sessionId: "session", bricks: { revision: 4, items: [] } });
    await vi.advanceTimersByTimeAsync(350);
    expect(storage.setItemAsync).toHaveBeenCalledTimes(1);
    expect(JSON.parse(storage.setItemAsync.mock.calls[0]![1])).toMatchObject({ prompt: "latest", sessionId: "session" });
    expect(JSON.parse(storage.setItemAsync.mock.calls[0]![1])).not.toHaveProperty("bricks");
  });
  it("does not replace typing that began while secure storage was loading", async () => {
    let resolve!: (value: string) => void;
    storage.getItemAsync.mockReturnValue(new Promise<string>(r => { resolve = r; }));
    const restoring = maker.restoreMakerDraft("family");
    maker.useMakerDraftStore.getState().patch("family", { prompt: "new idea" });
    resolve(JSON.stringify({ prompt: "old idea" })); await restoring;
    expect(maker.useMakerDraftStore.getState().drafts.family?.prompt).toBe("new idea");
  });
  it("does not revive account drafts after logout during restoration", async () => {
    let resolve!: (value: string) => void;
    storage.getItemAsync.mockReturnValue(new Promise<string>(r => { resolve = r; }));
    const restoring = maker.restoreMakerDraft("old-account");
    maker.useMakerDraftStore.getState().reset();
    resolve(JSON.stringify({ prompt: "private idea" })); await restoring;
    expect(maker.useMakerDraftStore.getState().drafts).toEqual({});
  });
});
