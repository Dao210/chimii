// @vitest-environment jsdom
import type { ReactNode } from "react";
import { createElement } from "react";
import { act, cleanup, renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useBuildConversation } from "./use-build-conversation";

const mocks = vi.hoisted(() => ({
  read: vi.fn(),
  send: vi.fn(),
  cancel: vi.fn(),
}));
vi.mock("../api", () => ({
  api: {
    getBuildConversation: mocks.read,
    sendBuildMessage: mocks.send,
    cancelBuildSession: mocks.cancel,
  },
}));
vi.mock("../hooks", () => ({ useWorkspaceId: () => "ws-1" }));
const session = {
  id: "run-1",
  conversation_id: "run-1",
  kind: "circuit",
  status: "completed",
  revision: 4,
  prompt: "Light",
  answers: {},
  created_at: "now",
  updated_at: "now",
};
function mount(kind: "brick" | "circuit") {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return renderHook(() => useBuildConversation("ws-1", kind, "run-1"), {
    wrapper: ({ children }: { children: ReactNode }) =>
      createElement(QueryClientProvider, { client }, children),
  });
}
beforeEach(() => {
  vi.resetAllMocks();
  mocks.read.mockResolvedValue({
    id: "run-1",
    session,
    result: null,
    messages: [],
    next_cursor: "",
  });
  mocks.send.mockResolvedValue({ ...session, id: "run-2" });
});
afterEach(cleanup);
describe("domain-specific creation turns", () => {
  it("preserves the circuit kit revision and refuses a foreign source", async () => {
    const { result } = mount("circuit");
    await waitFor(() => expect(result.current.session).toBeDefined());
    const circuit = {
      kit_id: "boson",
      catalog_version: "v2",
      inventory_revision: 7,
    };
    await act(() =>
      result.current.sendMessage("Button light", {
        circuit,
        source: { kind: "brick", id: "foreign", hash: "wrong" },
      }),
    );
    expect(mocks.send).toHaveBeenCalledWith(
      expect.objectContaining({
        kind: "circuit",
        circuit,
        expected_session_id: "run-1",
        expected_revision: 4,
      }),
      "run-1",
    );
    expect(mocks.send.mock.calls[0]?.[0].source_creation_id).toBeUndefined();
  });
  it("retains an expired conversation while allowing a new generation", async () => {
    mocks.read.mockResolvedValue({
      id: "run-1",
      session: { ...session, status: "generating", expired: true },
      result: { ...session, circuit_creation_id: "original" },
      messages: [],
      next_cursor: "",
    });
    const { result } = mount("circuit");
    await waitFor(() => expect(result.current.resultId).toBe("original"));
    expect(result.current.isWorking).toBe(false);
    await act(() => result.current.sendMessage("Change the title"));
    expect(mocks.send).toHaveBeenCalledTimes(1);
  });
  it("keeps a mixed historical conversation readable without silently changing domains", async () => {
    mocks.read.mockResolvedValue({
      id: "run-1",
      session,
      messages: [{ metadata: { creation_id: "old-brick" } }],
      next_cursor: "",
    });
    const { result } = mount("circuit");
    await waitFor(() => expect(result.current.readOnly).toBe(true));
    expect(result.current.messages).toHaveLength(1);
    await act(() => result.current.sendMessage("Continue"));
    expect(mocks.send).not.toHaveBeenCalled();
  });
});
