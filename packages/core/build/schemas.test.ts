import { describe, expect, it } from "vitest";
import { parseWithFallback } from "../api/schema";
import {
  BuildSessionSchema,
  EMPTY_BUILD_SESSION,
  EMPTY_LDRAW_CATALOG_SYNC_STATUS,
  LDrawCatalogSyncStatusSchema,
} from "./schemas";

const endpoint = { endpoint: "GET /api/build/catalog/sync" };

describe("LDrawCatalogSyncStatusSchema", () => {
  it("parses a running Starter Kit sync", () => {
    const parsed = LDrawCatalogSyncStatusSchema.parse({
      enabled: true,
      can_manage: true,
      kit_id: "starter-kit-1000-v1",
      catalog_version: "ldraw-official-2026-05-29-6009f2e94204",
      target_part_count: 1000,
      stored_part_count: 420,
      progress_part_count: 420,
      status: "running",
      updated_at: "2026-08-12T02:49:00Z",
    });

    expect(parsed.status).toBe("running");
    expect(parsed.progress_part_count).toBe(420);
  });

  it("fails closed when a malformed response claims completion", () => {
    const parsed = parseWithFallback(
      { status: "completed", target_part_count: "1000" },
      LDrawCatalogSyncStatusSchema,
      EMPTY_LDRAW_CATALOG_SYNC_STATUS,
      endpoint,
    );

    expect(parsed).toBe(EMPTY_LDRAW_CATALOG_SYNC_STATUS);
    expect(parsed.status).toBe("idle");
    expect(parsed.enabled).toBe(false);
  });
});


describe("BuildSessionSchema planning protocol", () => {
  const session = { id: "s", prompt: "dog", status: "clarifying", answers: {}, created_at: "", updated_at: "" };
  it("supports old labels and new versioned free-text questions", () => {
    const old = BuildSessionSchema.parse({ ...session, question: { id: "movement", prompt: "Move?", options: ["Run"] } });
    expect(old.question?.options).toEqual(["Run"]);
    const next = BuildSessionSchema.parse({ ...session, revision: 2, phase: "planning", question: { id: "q2", prompt: "Which animal?", choices: [], allow_free_text: true } });
    expect(next.question?.allow_free_text).toBe(true);
    expect(next.question?.options).toEqual([]);
  });
  it("discards malformed optional choices without losing the session", () => {
    const parsed = BuildSessionSchema.parse({ ...session, question: { id: "q1", prompt: "Which?", options: ["Dog"], choices: [{ label: "Dog" }], allow_free_text: "yes" } });
    expect(parsed.id).toBe("s"); expect(parsed.question?.choices).toBeUndefined();
    expect(parsed.question?.allow_free_text).toBeUndefined();
  });
  it("fails closed on a malformed revision or question", () => {
    const parsed = parseWithFallback({ ...session, revision: "wrong", question: { prompt: "Which?" } }, BuildSessionSchema, EMPTY_BUILD_SESSION, { endpoint: "GET /api/build/sessions/{id}" });
    expect(parsed.status).toBe("failed");
  });
});

describe("build progress boundary", () => {
  it("rejects malformed or out-of-range progress instead of claiming completion", async () => {
    const { BuildProgressSchema, BuildSummaryListSchema } = await import("./schemas");
    expect(BuildProgressSchema.safeParse({ id: "c", current_step: 9, step_count: 3, revision: 1, completed_at: null }).success).toBe(false);
    expect(BuildProgressSchema.safeParse({ id: "c", current_step: 1, step_count: 3, revision: "1", completed_at: null }).success).toBe(false);
    expect(BuildSummaryListSchema.safeParse({ creations: null }).success).toBe(false);
  });
});
