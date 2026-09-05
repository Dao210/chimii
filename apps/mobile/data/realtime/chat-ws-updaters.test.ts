import { describe, expect, it, vi } from "vitest";
import { QueryClient, QueryObserver } from "@tanstack/react-query";
import type {
  ChatDonePayload,
  ChatMessage,
  ChatPendingTask,
} from "@chimii/core/types";


import {
  applyChatDoneToCache,
  promotePendingTaskToRunning,
  seedAcceptedPendingTask,
  seedPendingTaskFromQueued,
} from "./chat-ws-updaters";
import { chatKeys } from "@/data/queries/chat";

// chat-ws-updaters imports chatKeys from data/queries/chat, which transitively
// imports the native fetch client. Mock it so the Node test never loads RN
// modules — chatKeys itself is a pure key factory and needs nothing from api.
vi.mock("@/data/api", () => ({ api: {} }));

const SESSION = "session-1";

function donePayload(over: Partial<ChatDonePayload> = {}): ChatDonePayload {
  return {
    chat_session_id: SESSION,
    task_id: "task-1",
    message_id: "msg-1",
    content: "here is the chart",
    created_at: "2026-07-09T00:00:00Z",
    elapsed_ms: 1200,
    ...over,
  };
}

describe("applyChatDoneToCache", () => {
  it("patches the assistant bubble inline AND invalidates messages so bound attachments refetch", () => {
    const qc = new QueryClient();
    qc.setQueryData<ChatMessage[]>(chatKeys.messages(SESSION), []);
    const invalidate = vi.spyOn(qc, "invalidateQueries");

    applyChatDoneToCache(qc, donePayload());

    // Inline patch: the bubble lands immediately (no flicker) — but without
    // attachments, because the event payload never carries them.
    const msgs = qc.getQueryData<ChatMessage[]>(chatKeys.messages(SESSION));
    expect(msgs).toHaveLength(1);
    expect(msgs?.[0].id).toBe("msg-1");
    expect(msgs?.[0].attachments).toBeUndefined();

    // Refetch: the authoritative message list (with attachments) is pulled in.
    expect(invalidate).toHaveBeenCalledWith({
      queryKey: chatKeys.messages(SESSION),
    });
    expect(invalidate).toHaveBeenCalledWith({
      queryKey: chatKeys.pendingTask(SESSION),
    });
  });

  it("invalidates even when the payload lacks an inline message (legacy shape)", () => {
    const qc = new QueryClient();
    const invalidate = vi.spyOn(qc, "invalidateQueries");

    applyChatDoneToCache(
      qc,
      donePayload({ message_id: undefined, content: undefined, created_at: undefined }),
    );

    expect(invalidate).toHaveBeenCalledWith({
      queryKey: chatKeys.messages(SESSION),
    });
  });


  it("does not duplicate an echoed message on reconnect", () => {
    const qc = new QueryClient();
    const existing: ChatMessage = {
      id: "msg-1",
      chat_session_id: SESSION,
      role: "assistant",
      content: "here is the chart",
      task_id: "task-1",
      created_at: "2026-07-09T00:00:00Z",
    };
    qc.setQueryData<ChatMessage[]>(chatKeys.messages(SESSION), [existing]);

    applyChatDoneToCache(qc, donePayload());

    expect(qc.getQueryData<ChatMessage[]>(chatKeys.messages(SESSION))).toHaveLength(1);
  });
});

describe("pending task queue events", () => {
  it("keeps an idle accepted send out of the follow-up queue", () => {
    const qc = new QueryClient();

    seedAcceptedPendingTask(qc, {
      chat_session_id: SESSION,
      task_id: "task-1",
      created_at: "2026-07-09T00:00:00Z",
      supports_queue: true,
      queued: false,
    });

    expect(qc.getQueryData<ChatPendingTask>(chatKeys.pendingTask(SESSION))).toMatchObject({
      task_id: "task-1",
      queued_tasks: [],
    });
  });

  it("retains a placeholder head when a queued response arrives before its predecessor", () => {
    const qc = new QueryClient();
    qc.setQueryData<ChatPendingTask>(chatKeys.pendingTask(SESSION), {
      task_id: "optimistic-message-1",
      status: "queued",
      created_at: "2026-07-09T00:00:01Z",
    });

    seedAcceptedPendingTask(qc, {
      chat_session_id: SESSION,
      task_id: "task-follow-up",
      created_at: "2026-07-09T00:00:01Z",
      message_id: "message-follow-up",
      content: "follow up",
      optimistic_task_id: "optimistic-message-1",
      supports_queue: true,
      queued: true,
    });

    expect(qc.getQueryData<ChatPendingTask>(chatKeys.pendingTask(SESSION))).toEqual({
      task_id: "optimistic-message-1",
      status: "queued",
      created_at: "2026-07-09T00:00:01Z",
      supports_queue: true,
      queued_tasks: [{
        task_id: "task-follow-up",
        status: "queued",
        created_at: "2026-07-09T00:00:01Z",
        message_id: "message-follow-up",
        content: "follow up",
      }],
    });
  });

  it("replaces an optimistic follow-up without disturbing the active head", () => {
    const qc = new QueryClient();
    qc.setQueryData<ChatPendingTask>(chatKeys.pendingTask(SESSION), {
      task_id: "task-active",
      status: "running",
      created_at: "2026-07-09T00:00:00Z",
      queued_tasks: [{
        task_id: "optimistic-message-1",
        status: "queued",
        created_at: "2026-07-09T00:00:01Z",
      }],
    });

    seedAcceptedPendingTask(qc, {
      chat_session_id: SESSION,
      task_id: "task-follow-up",
      created_at: "2026-07-09T00:00:01Z",
      optimistic_task_id: "optimistic-message-1",
      supports_queue: true,
      queued: true,
    });

    expect(qc.getQueryData<ChatPendingTask>(chatKeys.pendingTask(SESSION))).toMatchObject({
      task_id: "task-active",
      status: "running",
      supports_queue: true,
      queued_tasks: [{ task_id: "task-follow-up" }],
    });
  });

  it("keeps dispatch state when the send response arrives later", () => {
    const qc = new QueryClient();
    qc.setQueryData<ChatPendingTask>(chatKeys.pendingTask(SESSION), {
      task_id: "task-1",
      status: "queued",
      created_at: "2026-07-09T00:00:00Z",
    });
    const invalidate = vi.spyOn(qc, "invalidateQueries");

    promotePendingTaskToRunning(qc, {
      task_id: "task-1",
      agent_id: "agent-1",
      issue_id: "",
      runtime_id: "runtime-1",
      chat_session_id: SESSION,
    });
    seedAcceptedPendingTask(qc, {
      chat_session_id: SESSION,
      task_id: "task-1",
      created_at: "2026-07-09T00:00:01Z",
      queued: false,
    });

    expect(qc.getQueryData<ChatPendingTask>(chatKeys.pendingTask(SESSION))).toMatchObject({
      task_id: "task-1",
      status: "running",
    });
    expect(invalidate).toHaveBeenCalledWith({
      queryKey: chatKeys.pendingTask(SESSION),
    });
  });

  it("does not replace an active head with a sparse follow-up queued event", () => {
    const qc = new QueryClient();
    const active: ChatPendingTask = {
      task_id: "task-active",
      status: "running",
      created_at: "2026-07-09T00:00:00Z",
    };
    qc.setQueryData(chatKeys.pendingTask(SESSION), active);
    const invalidate = vi.spyOn(qc, "invalidateQueries");

    seedPendingTaskFromQueued(qc, {
      task_id: "task-follow-up",
      agent_id: "agent-1",
      issue_id: "",
      chat_session_id: SESSION,
      status: "queued",
    });

    expect(qc.getQueryData(chatKeys.pendingTask(SESSION))).toEqual(active);
    expect(invalidate).toHaveBeenCalledWith({
      queryKey: chatKeys.pendingTask(SESSION),
    });
  });

  it("refetches pending state and messages when a task is dispatched", () => {
    const qc = new QueryClient();
    qc.setQueryData<ChatPendingTask>(chatKeys.pendingTask(SESSION), {
      task_id: "task-active",
      status: "running",
    });
    const invalidate = vi.spyOn(qc, "invalidateQueries");

    promotePendingTaskToRunning(qc, {
      task_id: "task-follow-up",
      agent_id: "agent-1",
      issue_id: "",
      runtime_id: "runtime-1",
      chat_session_id: SESSION,
    });

    expect(qc.getQueryData<ChatPendingTask>(
      chatKeys.pendingTask(SESSION),
    )?.task_id).toBe("task-active");
    expect(invalidate).toHaveBeenCalledWith({
      queryKey: chatKeys.pendingTask(SESSION),
    });
    expect(invalidate).toHaveBeenCalledWith({
      queryKey: chatKeys.messages(SESSION),
    });
  });
});
