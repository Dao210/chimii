import type { ReactNode } from "react";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithI18n } from "../../test/i18n";

const {
  createSession,
  getCreation,
  getSession,
  submitAnswer,
  cancelBuild,
} = vi.hoisted(() => ({
  createSession: vi.fn(),
  getCreation: vi.fn(),
  getSession: vi.fn(),
  submitAnswer: vi.fn(),
  cancelBuild: vi.fn(),
}));

vi.mock("@chimii/core/hooks", () => ({
  useWorkspaceId: () => "workspace-1",
}));

vi.mock("@chimii/core/paths", () => ({
  useWorkspacePaths: () => ({ creations: () => "/acme/creations" }),
}));

vi.mock("@chimii/core/config", () => ({
  useConfigStore: (selector: (state: { buildAvailable: boolean; buildConfigLoaded: boolean }) => unknown) =>
    selector({ buildAvailable: true, buildConfigLoaded: true }),
}));

vi.mock("@chimii/core/utils", () => ({
  generateUUID: () => "019fc501-46b3-7a22-ae7e-940f2b90678b",
}));

vi.mock("@chimii/core/build", () => ({
  buildSessionOptions: (_workspaceId: string, id: string) => ({
    queryKey: ["build", "session", id],
    queryFn: () => getSession(id),
    enabled: id.length > 0,
  }),
  buildCreationOptions: (_workspaceId: string, id: string) => ({
    queryKey: ["build", "creation", id],
    queryFn: () => getCreation(id),
    enabled: id.length > 0,
  }),
  useCreateBuildSession: () => ({ mutateAsync: createSession, isPending: false }),
  useSubmitBuildAnswers: () => ({ mutateAsync: submitAnswer, isPending: false }),
  useCancelBuildSession: () => ({ mutateAsync: cancelBuild, isPending: false }),
}));

vi.mock("../../navigation", () => ({
  AppLink: ({ children, href, className }: { children: ReactNode; href: string; className?: string }) =>
    <a href={href} className={className}>{children}</a>,
}));

vi.mock("./child-mode-controls", () => ({
  ChildModeLauncher: () => null,
}));

vi.mock("./build-result", () => ({
  BuildResult: () => <div>build result</div>,
}));

vi.mock("motion/react", () => ({
  motion: {
    div: ({ children, className }: { children: ReactNode; className?: string }) =>
      <div className={className}>{children}</div>,
    section: ({ children, className }: { children: ReactNode; className?: string }) =>
      <section className={className}>{children}</section>,
  },
}));

import { BuildPage } from "./build-page";

const completedSession = {
  id: "session-1",
  prompt: "会跑的月球车",
  status: "completed" as const,
  answers: {},
  creation_id: "creation-1",
  created_at: "2026-08-09T00:00:00Z",
  updated_at: "2026-08-09T00:00:01Z",
};

function renderPage() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  const rendered = renderWithI18n(
    <QueryClientProvider client={queryClient}>
      <BuildPage />
    </QueryClientProvider>,
    { locale: "zh-Hans" },
  );
  return { ...rendered, queryClient };
}

describe("BuildPage creation handoff", () => {
  beforeEach(() => {
    submitAnswer.mockReset(); cancelBuild.mockReset();
    createSession.mockReset();
    getCreation.mockReset();
    getSession.mockReset();
    createSession.mockResolvedValue(completedSession);
    getSession.mockResolvedValue(completedSession);
  });

  it("shows the recoverable error state when a completed creation resolves without an id", async () => {
    getCreation.mockResolvedValue({ id: "" });
    renderPage();

    fireEvent.change(screen.getByPlaceholderText(/例如：我想做一辆/), {
      target: { value: completedSession.prompt },
    });
    fireEvent.click(screen.getByRole("button", { name: "开始创造" }));

    await waitFor(() => {
      expect(screen.getByText("方案已经造好，正在重新取回")).toBeInTheDocument();
    });
    expect(screen.queryByText("正在展开你的搭建方案…")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "再取一次" })).toBeInTheDocument();
  });
});


describe("BuildPage clarification", () => {
  const session = { ...completedSession, status: "clarifying", creation_id: undefined, revision: 2, summary: "一只蓝色小狗", question: { id: "q2", prompt: "耳朵需要多长？", options: ["长耳朵"], choices: [{ id: "long", label: "长耳朵" }], allow_free_text: true } };
  it("submits the stable option ID and current revision", async () => {
    createSession.mockResolvedValue(session); getSession.mockResolvedValue(session);
    renderPage();
    fireEvent.change(screen.getByPlaceholderText(/例如：我想做一辆/), { target: { value: "蓝色小狗" } });
    fireEvent.click(screen.getByRole("button", { name: "开始创造" }));
    await screen.findByText("耳朵需要多长？");
    expect(screen.getByText("一只蓝色小狗")).toBeInTheDocument();
    submitAnswer.mockResolvedValue({ ...session, status: "queued" });
    fireEvent.click(screen.getByRole("button", { name: "长耳朵" }));
    await waitFor(() => expect(submitAnswer).toHaveBeenCalledWith({ sessionId: "session-1", revision: 2, answers: { q2: "long" } }));
  });
  it("allows free text when there are no suggested choices", async () => {
    const free = { ...session, question: { ...session.question, choices: [], options: [] } };
    createSession.mockResolvedValue(free); getSession.mockResolvedValue(free); submitAnswer.mockResolvedValue({ ...free, status: "queued" });
    renderPage();
    fireEvent.change(screen.getByPlaceholderText(/例如：我想做一辆/), { target: { value: "蓝色小狗" } });
    fireEvent.click(screen.getByRole("button", { name: "开始创造" }));
    const input = await screen.findByLabelText("也可以用自己的话补充");
    fireEvent.change(input, { target: { value: "耳朵短一点" } });
    fireEvent.click(screen.getByRole("button", { name: "继续创造" }));
    await waitFor(() => expect(submitAnswer).toHaveBeenCalledWith({ sessionId: "session-1", revision: 2, answers: { q2: "耳朵短一点" } }));
  });
  it("keeps the original idea when editing after cancellation", async () => {
    createSession.mockResolvedValue(session); getSession.mockResolvedValue(session); cancelBuild.mockResolvedValue({ ...session, status: "failed" });
    renderPage();
    fireEvent.change(screen.getByPlaceholderText(/例如：我想做一辆/), { target: { value: "蓝色小狗" } });
    fireEvent.click(screen.getByRole("button", { name: "开始创造" }));
    fireEvent.click(await screen.findByRole("button", { name: "修改想法" }));
    await waitFor(() => expect(cancelBuild).toHaveBeenCalledWith({ sessionId: "session-1", revision: 2 }));
    expect(await screen.findByPlaceholderText(/例如：我想做一辆/)).toHaveValue(session.prompt);
  });
});
