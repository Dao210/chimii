import type { ReactNode } from "react";
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ApiError } from "@chimii/core/api";
import { circuitFixture } from "./test-fixture";
import { renderWithI18n } from "../test/i18n";

const { mockApi, push } = vi.hoisted(() => ({
  mockApi: {
    getCircuitCatalog: vi.fn(),
    listCircuitKits: vi.fn(),
    getCircuitInventory: vi.fn(),
    saveCircuitInventory: vi.fn(),
    listCircuitTrials: vi.fn(),
    createCircuitTrial: vi.fn(),
    listCircuitCreations: vi.fn(),
    getCircuitCreation: vi.fn(),
    createCircuit: vi.fn(),
    updateCircuitProgress: vi.fn(),
  },
  push: vi.fn(),
}));
vi.mock("@chimii/core/api", async (importOriginal) => ({
  ...(await importOriginal<object>()),
  api: mockApi,
}));
vi.mock("@chimii/core/hooks", () => ({ useWorkspaceId: () => "ws-1" }));
vi.mock("@chimii/core/paths", () => ({
  useWorkspacePaths: () => ({
    build: () => "/acme/build",
    circuit: () => "/acme/circuit",
    circuitDetail: (id: string) => `/acme/circuit/${id}`,
  }),
}));
vi.mock("../navigation", () => ({
  useNavigation: () => ({ push }),
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

import { CircuitPage, CircuitDetailPage } from "./circuit-page";

function renderPage(detail = false) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return renderWithI18n(
    <QueryClientProvider client={client}>
      {detail ? <CircuitDetailPage creationId="circuit-1" /> : <CircuitPage />}
    </QueryClientProvider>,
  );
}
beforeEach(() => {
  vi.resetAllMocks();
  const { catalog, creation } = circuitFixture();
  mockApi.getCircuitCatalog.mockResolvedValue(catalog);
  mockApi.listCircuitKits.mockResolvedValue({
    kits: [
      {
        kit_id: catalog.catalog.kit_id,
        name: catalog.catalog.name,
        version: catalog.catalog.version,
        connection_system: "snap",
      },
    ],
  });
  const inventory = {
    kit_id: catalog.catalog.kit_id,
    catalog_version: catalog.catalog.version,
    quantities: creation.document.inventory,
    confirmed: true,
    revision: 1,
    can_edit: true,
    updated_at: "2026-09-05",
  };
  mockApi.getCircuitInventory.mockResolvedValue(inventory);
  mockApi.saveCircuitInventory.mockImplementation(async (_id, input) => ({
    ...inventory,
    quantities: input.quantities,
    revision: input.expected_revision + 1,
  }));
  mockApi.listCircuitTrials.mockResolvedValue({ trials: [] });
  mockApi.listCircuitCreations.mockResolvedValue({ creations: [] });
  mockApi.getCircuitCreation.mockResolvedValue(creation);
});
afterEach(cleanup);

describe("electronic construction flow", () => {
  it("offers the documented projects without AI and blocks missing parts", async () => {
    renderPage();
    const start = await screen.findByRole("button", {
      name: "Start this project",
    });
    expect(start).toBeEnabled();
    expect(
      screen.getByRole("button", { name: "Match my idea" }),
    ).toBeDisabled();
    fireEvent.click(screen.getByText("Edit saved quantities"));
    fireEvent.change(screen.getByRole("spinbutton", { name: /B1/ }), {
      target: { value: "0" },
    });
    expect(start).toBeEnabled();
    fireEvent.click(screen.getByRole("checkbox"));
    fireEvent.click(screen.getByRole("button", { name: "Save parts box" }));
    await waitFor(() => expect(start).toBeDisabled());
    expect(screen.getByText("Some parts are missing")).toBeVisible();
  });

  it("waits for persistence before navigation and retains the request ID on retry", async () => {
    mockApi.createCircuit
      .mockRejectedValueOnce(new Error("offline"))
      .mockResolvedValueOnce(circuitFixture().creation);
    renderPage();
    fireEvent.click(
      await screen.findByRole("button", { name: "Start this project" }),
    );
    await screen.findByRole("alert");
    expect(push).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "Start this project" }));
    await waitFor(() =>
      expect(push).toHaveBeenCalledWith("/acme/circuit/circuit-1"),
    );
    expect(mockApi.createCircuit.mock.calls[0]![0].client_request_id).toBe(
      mockApi.createCircuit.mock.calls[1]![0].client_request_id,
    );
  });

  it("keeps the current assembly step when the save fails", async () => {
    mockApi.updateCircuitProgress.mockRejectedValue(
      new ApiError("conflict", 409, "Conflict", {
        code: "circuit_progress_conflict",
      }),
    );
    renderPage(true);
    fireEvent.click(
      await screen.findByRole("button", { name: "Done, next step" }),
    );
    await screen.findByRole("alert");
    expect(screen.getByText("STEP 1 / 8")).toBeVisible();
    expect(mockApi.updateCircuitProgress).toHaveBeenCalledWith("circuit-1", {
      current_step: 1,
      observation: "not_tried",
      expected_revision: 0,
    });
  });

  it("shows every radio component in the overview and preserves layer information", async () => {
    mockApi.getCircuitCreation.mockResolvedValue(
      circuitFixture("fm-radio").creation,
    );
    renderPage(true);
    fireEvent.click(
      await screen.findByRole("button", { name: "Complete layout" }),
    );
    expect(
      screen.getByTestId("circuit-board").querySelectorAll("[data-placement]"),
    ).toHaveLength(23);
    fireEvent.change(
      screen.getByRole("combobox", { name: "Highlight a layer" }),
      { target: { value: "3" } },
    );
    expect(
      screen
        .getByTestId("circuit-board")
        .querySelectorAll('[data-layer="1"][opacity="0.18"]').length,
    ).toBeGreaterThan(0);
  });

  it("keeps family observations separate from platform hardware testing", async () => {
    const { creation } = circuitFixture();
    creation.current_step = 7;
    mockApi.getCircuitCreation.mockResolvedValue(creation);
    mockApi.updateCircuitProgress.mockResolvedValue({
      ...creation,
      observation: "worked",
      progress_revision: 1,
    });
    renderPage(true);
    const worked = await screen.findByRole("button", { name: "It worked!" });
    expect(worked).toBeDisabled();
    fireEvent.click(
      screen.getByRole("checkbox", { name: /An adult and I checked/ }),
    );
    fireEvent.click(worked);
    expect(
      await screen.findByText(/You reported that it worked/),
    ).toBeVisible();
    expect(screen.getByText(/physical testing not completed/)).toBeVisible();
  });
  it("requires a saved physical inventory and keeps an unsaved draft on failure", async () => {
    const { catalog } = circuitFixture();
    mockApi.getCircuitInventory.mockResolvedValue({
      kit_id: catalog.catalog.kit_id,
      catalog_version: catalog.catalog.version,
      quantities: Object.fromEntries(
        catalog.catalog.parts.map((p) => [p.id, 0]),
      ),
      confirmed: false,
      revision: 0,
      can_edit: true,
      updated_at: "",
    });
    mockApi.saveCircuitInventory.mockRejectedValueOnce(new Error("offline"));
    renderPage();
    const start = await screen.findByRole("button", {
      name: "Start this project",
    });
    expect(start).toBeDisabled();
    fireEvent.click(screen.getByRole("button", { name: /I have this kit/ }));
    expect(
      screen.getByRole("button", { name: "Save parts box" }),
    ).toBeDisabled();
    fireEvent.click(
      screen.getByRole("checkbox", { name: /I checked the model/ }),
    );
    fireEvent.click(screen.getByRole("button", { name: "Save parts box" }));
    await screen.findByRole("alert");
    expect(start).toBeDisabled();
    expect(screen.getByRole("spinbutton", { name: /B1/ })).toHaveValue(
      catalog.catalog.parts.find((p) => p.id === "B1")!.quantity,
    );
    expect(push).not.toHaveBeenCalled();
  });

  it("leaves a child with a read-only family box", async () => {
    const { catalog, creation } = circuitFixture();
    mockApi.getCircuitInventory.mockResolvedValue({
      kit_id: catalog.catalog.kit_id,
      catalog_version: catalog.catalog.version,
      quantities: creation.document.inventory,
      confirmed: true,
      revision: 1,
      can_edit: false,
      updated_at: "",
    });
    renderPage();
    expect(
      await screen.findByRole("button", { name: "Start this project" }),
    ).toBeEnabled();
    expect(
      screen.queryByRole("button", { name: "Edit saved quantities" }),
    ).not.toBeInTheDocument();
    expect(screen.getByText(/A parent can update the parts box/)).toBeVisible();
  });

  it("renders physical module markings and cable endpoints without snap layer controls", async () => {
    mockApi.getCircuitCreation.mockResolvedValue(
      circuitFixture("boson-motion-sound-fan").creation,
    );
    renderPage(true);
    fireEvent.click(
      await screen.findByRole("button", { name: "Complete layout" }),
    );
    expect(
      screen.queryByRole("combobox", { name: "Highlight a layer" }),
    ).not.toBeInTheDocument();
    const board = screen.getByTestId("circuit-module-board");
    expect(board.querySelectorAll("[data-connection-id]")).toHaveLength(5);
    expect(board.textContent).toContain("i13");
    expect(board.textContent).toContain("f1");
    expect(screen.getByText(/physical testing not completed/)).toBeVisible();
  });

  it("retains a failed trial report and its retry ID without promoting evidence", async () => {
    const { creation } = circuitFixture();
    creation.current_step = 7;
    mockApi.getCircuitCreation.mockResolvedValue(creation);
    mockApi.createCircuitTrial
      .mockRejectedValueOnce(new Error("offline"))
      .mockResolvedValueOnce({ id: "trial-1" });
    renderPage(true);
    fireEvent.click(await screen.findByText("Physical build notes"));
    fireEvent.change(
      screen.getByRole("textbox", {
        name: "Model and markings on your actual kit",
      }),
      { target: { value: "SC-500 A" } },
    );
    fireEvent.change(screen.getByRole("textbox", { name: /What happened/ }), {
      target: { value: "Lamp lit" },
    });
    fireEvent.click(
      screen.getByRole("checkbox", { name: /An adult checked the assembly/ }),
    );
    fireEvent.click(screen.getByRole("button", { name: "Save this trial" }));
    await screen.findByRole("alert");
    expect(screen.getByRole("textbox", { name: /What happened/ })).toHaveValue(
      "Lamp lit",
    );
    fireEvent.click(screen.getByRole("button", { name: "Save this trial" }));
    await screen.findByText("Trial saved as a family report.");
    expect(mockApi.createCircuitTrial.mock.calls[0]![1].client_request_id).toBe(
      mockApi.createCircuitTrial.mock.calls[1]![1].client_request_id,
    );
    expect(screen.getByText(/physical testing not completed/)).toBeVisible();
    expect(mockApi.updateCircuitProgress).not.toHaveBeenCalled();
  });
});
