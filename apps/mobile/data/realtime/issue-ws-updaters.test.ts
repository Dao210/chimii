import { QueryClient } from "@tanstack/react-query";
import { describe, expect, it, vi } from "vitest";
import type { Issue, TimelineEntry } from "@chimii/core/types";
import { EMPTY_ISSUE_FALLBACK } from "../schemas";
import { issueKeys } from "../queries/issue-keys";
import { patchIssueDetail, patchMyIssuesList, patchIssuesList, patchIssueLabels, replaceCommentTimelineEntry, removeCommentCascade, invalidateIssueAfterReconnect } from "./issue-ws-updaters";

const ws = "ws-1", id = "issue-1";
const issue: Issue = { ...EMPTY_ISSUE_FALLBACK, id, title: "Before" };
const comment: TimelineEntry = { type: "comment", id: "comment-1", actor_type: "member", actor_id: "member-1", created_at: "2026-01-01", content: "Before", reactions: [] };

describe("current Chimii realtime protocol", () => {
  it("patches unversioned snapshots without unnecessary refetches", () => {
    const qc = new QueryClient();
    const invalidate = vi.spyOn(qc, "invalidateQueries");
    qc.setQueryData(issueKeys.detail(ws,id), issue);
    qc.setQueryData(issueKeys.list(ws), [issue]);
    const patch = { id, title: "After", status: "done" as const };
    patchIssueDetail(qc, ws, patch);
    patchIssuesList(qc, ws, patch);
    patchMyIssuesList(qc, ws, patch);
    expect(qc.getQueryData(issueKeys.detail(ws,id))).toMatchObject(patch);
    expect(qc.getQueryData(issueKeys.list(ws))).toMatchObject([patch]);
    expect(invalidate).not.toHaveBeenCalled();
  });
  it("updates labels in detail and loaded lists while preserving other fields", () => {
    const qc = new QueryClient();
    qc.setQueryData(issueKeys.detail(ws,id), issue);
    qc.setQueryData(issueKeys.list(ws), [issue]);
    patchIssueLabels(qc, ws, id, []);
    expect(qc.getQueryData(issueKeys.detail(ws,id))).toMatchObject({ title: "Before", labels: [] });
    expect(qc.getQueryData(issueKeys.list(ws))).toMatchObject([{ labels: [] }]);
  });
  it("updates comment content and removes descendant replies on deletion", () => {
    const qc = new QueryClient(), key = issueKeys.timeline(ws,id);
    qc.setQueryData(key, [comment, {...comment, id: "reply", parent_id: comment.id}]);
    replaceCommentTimelineEntry(qc, ws, id, {...comment, content: "After"});
    expect(qc.getQueryData<TimelineEntry[]>(key)?.[0]?.content).toBe("After");
    removeCommentCascade(qc, ws, id, comment.id);
    expect(qc.getQueryData(key)).toEqual([]);
  });
  it("revalidates attachments and task data after reconnect", () => {
    const qc = new QueryClient(), invalidate = vi.spyOn(qc, "invalidateQueries");
    invalidateIssueAfterReconnect(qc, ws, id);
    expect(invalidate.mock.calls.map(([f]) => f?.queryKey)).toEqual([
      issueKeys.detail(ws,id), issueKeys.timeline(ws,id), issueKeys.attachments(ws,id), issueKeys.activeTasks(ws,id), issueKeys.tasks(ws,id),
    ]);
  });
});
