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
  vi.clearAllMocks();
  const { catalog, creation } = circuitFixture();
  mockApi.getCircuitCatalog.mockResolvedValue(catalog);
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
    fireEvent.click(screen.getByText("Check or change quantities"));
    fireEvent.change(screen.getByRole("spinbutton", { name: /B1/ }), {
      target: { value: "0" },
    });
    expect(start).toBeDisabled();
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
    fireEvent.click(screen.getByRole("checkbox"));
    fireEvent.click(worked);
    expect(
      await screen.findByText(/You reported that it worked/),
    ).toBeVisible();
    expect(screen.getByText(/physical testing not completed/)).toBeVisible();
  });
});
