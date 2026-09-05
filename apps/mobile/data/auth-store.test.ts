import { beforeEach, describe, expect, it, vi } from "vitest";
const mocks = vi.hoisted(() => ({
  resetSession: vi.fn(), getToken: vi.fn(), clearToken: vi.fn(), persistToken: vi.fn(), setToken: vi.fn(), getMe: vi.fn(),
  restoreSlug: vi.fn(), clearWorkspace: vi.fn(), cancelQueries: vi.fn(), clearQueries: vi.fn(),
}));
vi.mock("./session-state", () => ({ resetSessionState: mocks.resetSession }));
vi.mock("./api", () => ({ api: { setToken: mocks.setToken, getMe: mocks.getMe }, ApiError: class ApiError extends Error { status = 401; } }));
vi.mock("./secure-storage", () => ({ getToken: mocks.getToken, clearToken: mocks.clearToken, setToken: mocks.persistToken }));
vi.mock("./workspace-store", () => ({ useWorkspaceStore: { getState: () => ({ restoreSlug: mocks.restoreSlug, clear: mocks.clearWorkspace }) } }));
vi.mock("./query-client", () => ({ queryClient: { cancelQueries: mocks.cancelQueries, clear: mocks.clearQueries } }));
import { useAuthStore } from "./auth-store";

describe("native session lifecycle", () => {
  beforeEach(() => {
    vi.resetAllMocks();
    useAuthStore.setState({ user: null, isLoading: true, initializationError: null });
  });
  it("leaves loading and offers retry when secure storage fails", async () => {
    mocks.getToken.mockRejectedValue(new Error("Storage unavailable"));
    await useAuthStore.getState().initialize();
    expect(useAuthStore.getState().isLoading).toBe(false);
    expect(useAuthStore.getState().initializationError).toBeTruthy();
  });
  it("preserves credentials on a network failure and restores the session on retry", async () => {
    mocks.getToken.mockResolvedValue("saved-token");
    mocks.getMe.mockRejectedValueOnce(new Error("Offline")).mockResolvedValueOnce({ id: "user-1" });
    await useAuthStore.getState().initialize();
    expect(mocks.clearToken).not.toHaveBeenCalled();
    expect(useAuthStore.getState().initializationError).toBeTruthy();
    await useAuthStore.getState().initialize();
    expect(useAuthStore.getState().user?.id).toBe("user-1");
    expect(useAuthStore.getState().initializationError).toBeNull();
  });
  it("clears account caches even when token deletion fails", async () => {
    mocks.clearToken.mockRejectedValue(new Error("Storage unavailable"));
    await expect(useAuthStore.getState().logout()).rejects.toThrow();
    expect(useAuthStore.getState().user).toBeNull();
    expect(mocks.setToken).toHaveBeenCalledWith(null);
    expect(mocks.cancelQueries).toHaveBeenCalled();
    expect(mocks.clearQueries).toHaveBeenCalled();
    expect(mocks.clearWorkspace).toHaveBeenCalled();
    expect(mocks.resetSession).toHaveBeenCalled();
  });
});
