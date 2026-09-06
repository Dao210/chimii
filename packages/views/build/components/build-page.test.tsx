import type { ReactNode } from "react";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithI18n } from "../../test/i18n";
import { BuildPage } from "./build-page";

const mocks = vi.hoisted(() => ({
  send: vi.fn(),
  history: vi.fn(),
  replace: vi.fn(),
  cancel: vi.fn(),
  params: "",
  available: true,
}));
vi.mock("@chimii/core/hooks", () => ({ useWorkspaceId: () => "workspace" }));
vi.mock("@chimii/core/paths", () => ({
  useWorkspacePaths: () => ({
    build: () => "/acme/build",
    circuit: () => "/acme/circuit",
    creations: () => "/acme/creations",
    creationDetail: (id: string) => `/acme/creations/${id}`,
    circuitDetail: (id: string) => `/acme/circuit/${id}`,
  }),
}));
vi.mock("@chimii/core/config", () => ({
  useConfigStore: (select: (state: unknown) => unknown) =>
    select({ buildAvailable: mocks.available, buildConfigLoaded: true }),
}));
vi.mock("../../navigation", () => ({
  useNavigation: () => ({
    pathname: "/acme/build",
    searchParams: new URLSearchParams(mocks.params),
    replace: mocks.replace,
    push: vi.fn(),
  }),
  AppLink: ({
    children,
    href,
    ...props
  }: {
    children: ReactNode;
    href: string;
  }) => (
    <a href={href} {...props}>
      {children}
    </a>
  ),
}));
vi.mock("./child-mode-controls", () => ({ ChildModeLauncher: () => null }));
vi.mock("./build-result", () => ({
  BuildResult: ({
    onDiscuss,
  }: {
    onDiscuss: (prompt: string) => Promise<boolean>;
  }) => (
    <div>
      Brick result
      <button onClick={() => void onDiscuss("Lower it, keep the doorway")}>
        Edit brick
      </button>
    </div>
  ),
}));
vi.mock("../../circuit/circuit-workbench", () => ({
  CircuitWorkbench: () => <div>Circuit result</div>,
}));
vi.mock("../../circuit/circuit-inventory", () => ({
  CircuitInventoryPanel: () => <div>Parts box</div>,
}));
vi.mock("@chimii/core/api", () => ({
  api: {
    sendBuildMessage: mocks.send,
    getBuildConversation: mocks.history,
    cancelBuildSession: mocks.cancel,
    listCircuitKits: async () => ({
      kits: [{ kit_id: "snap", name: "SC-500" }],
    }),
    getCircuitCatalog: async () => ({
      catalog: {
        kit_id: "snap",
        version: "v1",
        parts: [],
        projects: [
          {
            id: "fm-radio",
            placements: [],
            title: { en: "My radio", zh: "我的收音机" },
          },
        ],
      },
    }),
    getCircuitInventory: async () => ({
      kit_id: "snap",
      catalog_version: "v1",
      revision: 3,
      confirmed: true,
      can_edit: true,
      quantities: {},
    }),
    getBuildCreation: async () => ({
      id: "brick-1",
      build_plan: { content_hash: "source-hash" },
    }),
    getCircuitCreation: async () => ({
      id: "circuit-1",
      document: { kit_id: "snap", content_hash: "circuit-hash" },
    }),
  },
}));
let client: QueryClient;
let unmount: () => void;
function renderStudio() {
  client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  const view = renderWithI18n(
    <QueryClientProvider client={client}>
      <BuildPage />
    </QueryClientProvider>,
  );
  unmount = view.unmount;
  return view;
}
const session = {
  id: "run-1",
  conversation_id: "run-1",
  kind: "brick",
  status: "completed",
  revision: 1,
  prompt: "A house",
  answers: {},
  created_at: "now",
  updated_at: "now",
};
beforeEach(() => {
  vi.clearAllMocks();
  mocks.params = "";
  mocks.available = true;
  mocks.history.mockResolvedValue({
    id: "run-1",
    session,
    messages: [],
    next_cursor: "",
  });
  mocks.send.mockResolvedValue({ ...session, status: "queued" });
});
afterEach(() => {
  unmount?.();
  client?.clear();
});
describe("dedicated brick studio", () => {
  it("keeps a failed message and retries the exact request after a lost response", async () => {
    mocks.send.mockRejectedValueOnce(new Error("connection lost"));
    renderStudio();
    fireEvent.change(screen.getByRole("textbox"), {
      target: { value: "Build a tower" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Start creating" }));
    await screen.findByRole("button", { name: "Retry message" });
    expect(mocks.replace).not.toHaveBeenCalled();
    const original = mocks.send.mock.calls[0]?.[0];
    expect(original.client_request_id).toBeTruthy();
    expect(original.kind).toBe("brick");
    expect(original.circuit).toBeUndefined();
    fireEvent.click(screen.getByRole("button", { name: "Retry message" }));
    await waitFor(() => expect(mocks.send).toHaveBeenCalledTimes(2));
    expect(mocks.send.mock.calls[1]?.[0]).toEqual(original);
    await waitFor(() =>
      expect(mocks.replace).toHaveBeenCalledWith(
        "/acme/build?conversation=run-1",
      ),
    );
  });
  it("restores a question and binds the answer to its session and revision", async () => {
    mocks.params = "conversation=run-1";
    const question = {
      id: "q2",
      prompt: "Which shape?",
      choices: [{ id: "round", label: "Round" }],
      options: [],
      allow_free_text: true,
    };
    mocks.history.mockResolvedValue({
      id: "run-1",
      session: { ...session, status: "clarifying", revision: 2, question },
      messages: [
        {
          id: "m1",
          sequence: 1,
          session_id: "run-1",
          role: "assistant",
          kind: "question",
          content: question.prompt,
          metadata: { revision: 2 },
          created_at: "now",
        },
      ],
      next_cursor: "",
    });
    renderStudio();
    await screen.findByRole("heading", { name: "Which shape?" });
    fireEvent.click(screen.getByRole("button", { name: "Round" }));
    await waitFor(() =>
      expect(mocks.send).toHaveBeenCalledWith(
        expect.objectContaining({
          prompt: "round",
          expected_session_id: "run-1",
          expected_revision: 2,
          question_id: "q2",
        }),
        "run-1",
      ),
    );
  });
  it("keeps the original brick entry without mode controls or electronic material requests", async () => {
    renderStudio();
    await waitFor(() =>
      expect(
        screen.getByRole("button", { name: "Start creating" }),
      ).toBeVisible(),
    );
    expect(screen.queryByText("Parts box")).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Choose for me" }),
    ).not.toBeInTheDocument();
    expect(screen.queryByRole("combobox")).not.toBeInTheDocument();
  });
  it("keeps an old electronic link out of the brick planner", () => {
    mocks.params = "conversation=old-circuit&mode=circuit";
    renderStudio();
    expect(
      screen.getByRole("heading", { name: "Continue in the right studio" }),
    ).toBeVisible();
    expect(
      screen
        .getAllByRole("link", { name: "Electronic blocks studio" })
        .some(
          (link) =>
            link.getAttribute("href") ===
            "/acme/circuit?conversation=old-circuit",
        ),
    ).toBe(true);
    expect(mocks.send).not.toHaveBeenCalled();
  });
  it("uses the displayed immutable version as the source of a new run", async () => {
    mocks.params = "conversation=run-1";
    mocks.history.mockResolvedValue({
      id: "run-1",
      session: { ...session, creation_id: "brick-1" },
      messages: [],
      next_cursor: "",
    });
    renderStudio();
    await screen.findByText("Brick result");
    fireEvent.click(screen.getByRole("button", { name: "Edit brick" }));
    await waitFor(() =>
      expect(mocks.send).toHaveBeenCalledWith(
        expect.objectContaining({
          expected_session_id: "run-1",
          source_creation_id: "brick-1",
          expected_content_hash: "source-hash",
        }),
        "run-1",
      ),
    );
  });
});
