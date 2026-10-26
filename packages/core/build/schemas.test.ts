import { describe, expect, it } from "vitest";
import { parseWithFallback } from "../api/schema";
import {
  BuildSessionSchema,
  BuildSceneSpecSchema,
  BuildCreationSchema,
  EMPTY_BUILD_CREATION,
  EMPTY_BUILD_SESSION,
  EMPTY_LDRAW_CATALOG_SYNC_STATUS,
  LDrawCatalogSyncStatusSchema,
} from "./schemas";

const endpoint = { endpoint: "GET /api/build/catalog/sync" };

describe("scene creation boundary", () => {
  const scene = { version: 1, brief: "a figure", objects: [{ id: "main", label: "figure", family: "figure", position: { x: 0, y: 0, z: 0 }, rotation: 0, modules: [{ id: "base", label: "base", kind: "solid", position: { x: 0, y: 0, z: 0 }, size: { x: 8, y: 6, z: 8 }, rotation: 0, color: 71 }] }] };
  it("keeps typed ceiling access openings and rejects malformed geometry", () => {
    const opening = { x: 2, z: 3, width: 2, depth: 12 };
    const withAccess = { ...scene, objects: [{ ...scene.objects[0], modules: [{ ...scene.objects[0]!.modules[0], kind: "storey", ceiling_openings: [opening] }] }] };
    expect(BuildSceneSpecSchema.parse(withAccess).objects[0]?.modules[0]?.ceiling_openings).toEqual([opening]);
    expect(BuildSceneSpecSchema.safeParse({ ...withAccess, objects: [{ ...withAccess.objects[0], modules: [{ ...withAccess.objects[0]!.modules[0], ceiling_openings: [{ ...opening, depth: "12" }] }] }] }).success).toBe(false);
  });
  it("keeps a scene document and tolerates malformed optional evidence without losing a saved model", () => {
    const creation = { ...EMPTY_BUILD_CREATION, build_plan: { ...EMPTY_BUILD_CREATION.build_plan, document: { version: 2, scene, design_hash: "hash", parent_creation_id: "parent", solver: { status: "feasible", visited: 1, target_cells: 1, matched_cells: 1 } }, inventory_compared: true, missing_parts: [{ part_id: "p", color: 4, quantity: 3 }] } };
    const parsed = BuildCreationSchema.parse(creation);
    expect(parsed.build_plan.document?.scene?.objects[0]?.family).toBe("figure");
    expect(parsed.build_plan.document?.parent_creation_id).toBe("parent");
    expect(parsed.build_plan.missing_parts?.[0]?.quantity).toBe(3);
    const drifted = BuildCreationSchema.parse({ ...creation, build_plan: { ...creation.build_plan, missing_parts: [{ quantity: "3" }], inventory_compared: "yes", validation: { ...creation.build_plan.validation, evidence: { physical: true } } } });
    expect(drifted.build_plan.document?.scene).toEqual(scene);
    expect(drifted.build_plan.missing_parts).toBeUndefined();
    expect(drifted.build_plan.inventory_compared).toBeUndefined();
    expect(drifted.build_plan.validation.evidence).toBeUndefined();
  });
});

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
  const session = {
    id: "s",
    prompt: "dog",
    status: "clarifying",
    answers: {},
    created_at: "",
    updated_at: "",
  };
  it("normalizes null legacy modules in Go scene recipes without accepting malformed arrays", () => {
    const recipe = { version: 4, archetype: "fox", title: "Fox", prompt: "fox", palette: [], features: [], metadata: {}, modules: null };
    expect(BuildCreationSchema.parse({ ...EMPTY_BUILD_CREATION, recipe }).recipe.modules).toEqual([]);
    expect(BuildCreationSchema.safeParse({ ...EMPTY_BUILD_CREATION, recipe: { ...recipe, modules: "invalid" } }).success).toBe(false);
  });
  it("keeps a new timeout code while rejecting malformed failure responses", () => {
    const parsed = BuildSessionSchema.parse({
      ...session,
      status: "failed",
      error: "BUILD_PLANNER_TIMEOUT",
    });
    expect(parsed.error).toBe("BUILD_PLANNER_TIMEOUT");
    expect(
      BuildSessionSchema.safeParse({ ...parsed, error: { message: "timeout" } })
        .success,
    ).toBe(false);
  });
  it("supports old labels and new versioned free-text questions", () => {
    const old = BuildSessionSchema.parse({
      ...session,
      question: { id: "movement", prompt: "Move?", options: ["Run"] },
    });
    expect(old.question?.options).toEqual(["Run"]);
    const next = BuildSessionSchema.parse({
      ...session,
      revision: 2,
      phase: "planning",
      question: {
        id: "q2",
        prompt: "Which animal?",
        choices: [],
        allow_free_text: true,
      },
    });
    expect(next.question?.allow_free_text).toBe(true);
    expect(next.question?.options).toEqual([]);
  });
  it("discards malformed optional choices without losing the session", () => {
    const parsed = BuildSessionSchema.parse({
      ...session,
      question: {
        id: "q1",
        prompt: "Which?",
        options: ["Dog"],
        choices: [{ label: "Dog" }],
        allow_free_text: "yes",
      },
    });
    expect(parsed.id).toBe("s");
    expect(parsed.question?.choices).toBeUndefined();
    expect(parsed.question?.allow_free_text).toBeUndefined();
  });
  it("fails closed on a malformed revision or question", () => {
    const parsed = parseWithFallback(
      { ...session, revision: "wrong", question: { prompt: "Which?" } },
      BuildSessionSchema,
      EMPTY_BUILD_SESSION,
      { endpoint: "GET /api/build/sessions/{id}" },
    );
    expect(parsed.status).toBe("failed");
  });
});

describe("build progress boundary", () => {
  it("preserves independent scene evidence and drops malformed optional checks", async () => {
    const { BuildValidationSchema } = await import("./schemas");
    const evidence = {
      objects: 1, modules: 2, geometry: "checked", steps: "checked", semantic: "model_reviewed_not_proven", physical: "not_tested", retail_colors: "not_verified", requirements: [],
      interfaces: [{ object: "main", interface: "shoulder", a: "body", b: "arm", connections: ["stud-1"], shared_parts: [] }],
      visual: { renderer: "catalog-glb-views-v1", content_hash: "model-hash", views: ["front", "back", "left", "right", "top", "isometric"].map(direction => ({ direction, sha256: "a".repeat(64) })) },
      load_estimate: { status: "not_evaluated", model: "reference-vertical-equilibrium-v1", mass_source: "https://arxiv.org/html/2402.10711v2", clutch_per_stud_n: 0.98, physical: "not_tested", reason: "unknown part mass" },
    };
    const report = { buildable: true, issues: null, part_count: 2, step_count: 2, used_parts: {}, evidence };
    const parsed = BuildValidationSchema.parse(report);
    expect(parsed.evidence?.visual?.views).toHaveLength(6);
    expect(parsed.evidence?.load_estimate?.status).toBe("not_evaluated");
    expect(parsed.evidence?.interfaces?.[0]?.interface).toBe("shoulder");
    const malformed = BuildValidationSchema.parse({ ...report, evidence: { ...evidence, visual: { ...evidence.visual, views: [] }, load_estimate: { ...evidence.load_estimate, status: "passed" }, interfaces: "wrong" } });
    expect(malformed.evidence?.semantic).toBe("model_reviewed_not_proven");
    expect(malformed.evidence?.visual).toBeUndefined();
    expect(malformed.evidence?.load_estimate).toBeUndefined();
    expect(malformed.evidence?.interfaces).toBeUndefined();
    expect(BuildValidationSchema.parse({ ...report, evidence: undefined }).evidence).toBeUndefined();
  });
  it("rejects malformed or out-of-range progress instead of claiming completion", async () => {
    const { BuildProgressSchema, BuildSummaryListSchema } = await import(
      "./schemas"
    );
    expect(
      BuildProgressSchema.safeParse({
        id: "c",
        current_step: 9,
        step_count: 3,
        revision: 1,
        completed_at: null,
      }).success,
    ).toBe(false);
    expect(
      BuildProgressSchema.safeParse({
        id: "c",
        current_step: 1,
        step_count: 3,
        revision: "1",
        completed_at: null,
      }).success,
    ).toBe(false);
    expect(BuildSummaryListSchema.safeParse({ creations: null }).success).toBe(
      false,
    );
  });
});

describe("conversation response boundary", () => {
  it("keeps legacy sessions compatible and rejects malformed history or artifact kinds", async () => {
    const { BuildConversationSchema, InventionListSchema } = await import(
      "./schemas"
    );
    const session = {
      id: "run",
      prompt: "idea",
      status: "completed",
      answers: {},
      created_at: "now",
      updated_at: "now",
    };
    expect(BuildSessionSchema.safeParse(session).success).toBe(true);
    expect(
      BuildConversationSchema.safeParse({
        id: "run",
        session,
        messages: [],
        next_cursor: "",
      }).success,
    ).toBe(true);
    expect(
      BuildConversationSchema.safeParse({
        id: "run",
        session,
        messages: [
          { id: "m", sequence: "1", role: "assistant", content: "done" },
        ],
        next_cursor: "",
      }).success,
    ).toBe(false);
    expect(
      InventionListSchema.safeParse({
        creations: [{ id: "x", kind: "unknown" }],
        next_cursor: "",
      }).success,
    ).toBe(false);
  });
});
