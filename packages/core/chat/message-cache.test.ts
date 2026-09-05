import { QueryClient } from "@tanstack/react-query";
import { describe, expect, it, vi } from "vitest";
import { upsertChatMessageToCaches } from "./message-cache";
import { chatKeys, taskMessagesOptions, mergeTaskMessagesBySeq } from "./queries";
vi.mock("../api", () => ({ api: { listTaskMessages: vi.fn() } }));
import { api } from "../api";
import type { ChatMessage, TaskMessagePayload } from "../types";

const message: ChatMessage = { id: "m", chat_session_id: "s", role: "user", content: "hello", created_at: "2026-09-05", task_id: null };
const task = "11111111-1111-4111-8111-111111111111";
const frame = (seq: number, content = `text ${seq}`): TaskMessagePayload => ({ task_id: task, issue_id: "i", seq, type: "text", content });

describe("message cache consistency", () => {
  it("merges both send/echo arrival orders without losing attachments or duplicating a row", () => {
    const rich = { ...message, attachments: [] };
    for (const pair of [[rich, message], [message, rich]]) {
      const qc = new QueryClient();
      for (const row of pair) upsertChatMessageToCaches(qc, "s", row!, { seedIfMissing: true });
      expect(qc.getQueryData(chatKeys.messages("s"))).toEqual([rich]);
      expect(qc.getQueryData(chatKeys.messagesPage("s"))).toMatchObject({ pages: [{ messages: [rich] }] });
      const before = qc.getQueryData(chatKeys.messages("s"));
      upsertChatMessageToCaches(qc, "s", message);
      expect(qc.getQueryData(chatKeys.messages("s"))).toBe(before);
      qc.clear();
    }
  });
  it("does not seed an unopened conversation from a workspace broadcast", () => {
    const qc = new QueryClient(); upsertChatMessageToCaches(qc, "s", message);
    expect(qc.getQueryCache().getAll()).toHaveLength(0); qc.clear();
  });
  it("retains live frames when an older HTTP response completes and accepts persisted corrections", async () => {
    let resolve!: (rows: TaskMessagePayload[]) => void;
    const response = new Promise<TaskMessagePayload[]>(r => { resolve = r; });
    const spy = vi.spyOn(api, "listTaskMessages").mockReturnValue(response);
    const qc = new QueryClient();
    try {
      const fetching = qc.fetchQuery(taskMessagesOptions(task));
      qc.setQueryData(chatKeys.taskMessages(task), [frame(1, "partial"), frame(3)]);
      resolve([frame(1, "persisted"), frame(2)]);
      await fetching;
      expect(qc.getQueryData(chatKeys.taskMessages(task))).toEqual([frame(1, "persisted"), frame(2), frame(3)]);
      const before = qc.getQueryData(chatKeys.taskMessages(task));
      qc.setQueryData(chatKeys.taskMessages(task), [frame(1, "persisted"), frame(2), frame(3)]);
      expect(qc.getQueryData(chatKeys.taskMessages(task))).toBe(before);
    } finally { spy.mockRestore(); qc.clear(); }
  });
  it("deduplicates repeated seqs inside one backfill batch", () => {
    expect(mergeTaskMessagesBySeq([], [frame(2), frame(2), frame(1)])).toEqual([frame(1), frame(2)]);
  });
});
