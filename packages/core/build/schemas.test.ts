import { describe, expect, it } from "vitest";
import { parseWithFallback } from "../api/schema";
import {
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
