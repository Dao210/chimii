import { fireEvent, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, expect, it, vi } from "vitest";
import { renderWithI18n } from "../../test/i18n";
import { BuildResult } from "./build-result";
import { EMPTY_BUILD_CREATION } from "@chimii/core/build/schemas";
const { getProgress, saveProgress } = vi.hoisted(() => ({ getProgress: vi.fn(), saveProgress: vi.fn() }));
vi.mock("@chimii/core/hooks", () => ({ useWorkspaceId: () => "w" }));
vi.mock("@chimii/core/api", () => ({ api: { getBuildProgress: getProgress, updateBuildProgress: saveProgress } }));
vi.mock("./build-model-viewer", () => ({ BuildModelViewer: () => <div>model</div> }));

describe("saved build progress", () => {
  it("leaves preview unsaved, persists building steps and requires explicit completion", async () => {
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
    let progress = { id: "c", current_step: 0, revision: 0, completed_at: null as string | null, step_count: 2 };
    getProgress.mockImplementation(async () => progress);
    saveProgress.mockImplementation(async (_id, input) => {
      progress = { ...progress, current_step: input.current_step, revision: progress.revision + 1, completed_at: input.completed ? "now" : null };
      return progress;
    });
    const view = renderWithI18n(
      <QueryClientProvider client={qc}>
        <BuildResult creation={{ ...EMPTY_BUILD_CREATION, id: "c", title: "Robot", validation: { ...EMPTY_BUILD_CREATION.validation, step_count: 2 } }} />
      </QueryClientProvider>,
    );
    try {
      await waitFor(() => expect(screen.getByRole("button", { name: "Start building" })).toBeEnabled());
      fireEvent.click(screen.getByRole("button", { name: "Go to previous step" }));
      expect(saveProgress).not.toHaveBeenCalled();
      fireEvent.click(screen.getByRole("button", { name: "Start building" }));
      await screen.findByText("Saved at step 1");
      fireEvent.click(screen.getByRole("button", { name: "Go to next step" }));
      await screen.findByText("Saved at step 2");
      expect(progress.completed_at).toBeNull();
      fireEvent.click(screen.getByRole("button", { name: "I finished building" }));
      await screen.findByText("Completed");
      expect(saveProgress).toHaveBeenLastCalledWith("c", { current_step: 2, expected_revision: 2, completed: true });
    } finally {
      view.unmount();
      qc.clear();
    }
  });
});
