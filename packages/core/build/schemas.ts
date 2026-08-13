import { z } from "zod";
import type {
  BrickInventory,
  BuildCatalog,
  BuildCatalogPartPage,
  LDrawCatalogSyncStatus,
  BuildCreation,
  BuildCreationList,
  BuildSession,
} from "./types";

const PartSpecSchema = z.looseObject({
  id: z.string(),
  name: z.string(),
  category: z.string().catch("brick"),
  popularity_rank: z.number().int().positive().optional(),
  certification_level: z.enum(["asset_only", "basic", "advanced", "certified"]).optional(),
  auto_build_eligible: z.boolean().catch(false).optional(),
  inventory_eligible: z.boolean().catch(false).optional(),
  geometry_profile: z.string().optional(),
  ldraw_id: z.string(),
  ldraw_status: z.string().optional(),
  license: z.string().optional(),
  studs_x: z.number().int().positive(),
  studs_z: z.number().int().positive(),
  plates_y: z.number().int().positive(),
  quantity: z.number().int().nonnegative(),
  origin_y_offset_ldu: z.number().int().optional(),
  origin_center_z_offset_ldu: z.number().int().optional(),
});

const CatalogBrowserPartSpecSchema = PartSpecSchema.extend({
  studs_x: z.number().int().nonnegative(),
  studs_z: z.number().int().nonnegative(),
  plates_y: z.number().int().nonnegative(),
});

const InventoryItemSchema = z.looseObject({
  part_id: z.string(),
  color: z.number().int(),
  quantity: z.number().int().positive(),
});

const InventoryItemsSchema = z
  .array(InventoryItemSchema)
  .nullish()
  .transform((items) => items ?? []);

const CatalogSourceSchema = z
  .looseObject({
    release: z.string(),
    archive_sha256: z.string(),
    source_url: z.string(),
  })
  .nullish()
  .transform((source) => source ?? undefined);

export const BrickInventorySchema = z.looseObject({
  configured: z.boolean(),
  catalog_version: z.string(),
  revision: z.number().int().nonnegative(),
  // Older servers encoded an empty Go slice as JSON null. Normalize that
  // wire-compatible shape so existing creations remain readable.
  items: InventoryItemsSchema,
  updated_at: z.string().optional(),
});

const BrickInventorySnapshotSchema = BrickInventorySchema.extend({
  content_hash: z.string(),
});

export const BuildCatalogSchema = z.looseObject({
  catalog_version: z.string(),
  catalog_source: CatalogSourceSchema,
  parts: z.array(PartSpecSchema),
  colors: z.array(z.looseObject({
    code: z.number().int(),
    name: z.string(),
    hex: z.string().regex(/^#[0-9a-fA-F]{6}$/),
  })),
});

export const BuildCatalogPartPageSchema = z.looseObject({
  kit_id: z.string(),
  kit_version: z.number().int().positive(),
  kit_name: z.string(),
  catalog_version: z.string(),
  profile_total: z.number().int().positive(),
  filtered_total: z.number().int().nonnegative(),
  categories: z.array(z.string()).catch([]),
  parts: z.array(CatalogBrowserPartSpecSchema),
  next_cursor: z.string().optional(),
});

export const LDrawCatalogSyncStatusSchema = z.looseObject({
  enabled: z.boolean(),
  can_manage: z.boolean(),
  kit_id: z.string(),
  catalog_version: z.string(),
  target_part_count: z.number().int().positive(),
  stored_part_count: z.number().int().nonnegative(),
  progress_part_count: z.number().int().nonnegative(),
  status: z.enum(["idle", "queued", "running", "completed", "failed"]),
  error: z.string().optional(),
  created_at: z.string().optional(),
  started_at: z.string().optional(),
  completed_at: z.string().optional(),
  updated_at: z.string().optional(),
});

const PlacementSchema = z.looseObject({
  id: z.string(),
  part_id: z.string(),
  color: z.number().int(),
  x: z.number().int(),
  y: z.number().int(),
  z: z.number().int(),
  rotation: z.number().int(),
  step: z.number().int().positive(),
  module: z.string(),
});

export const BuildValidationSchema = z.looseObject({
  buildable: z.boolean(),
  // A successful historical validation has no issues and was encoded as
  // null by Go. Consumers always receive an array after this boundary.
  issues: z
    .array(z.looseObject({
      code: z.string(),
      message: z.string(),
      placement_id: z.string().optional(),
    }))
    .nullish()
    .transform((issues) => issues ?? []),
  part_count: z.number().int().nonnegative(),
  step_count: z.number().int().nonnegative(),
  used_parts: z.record(z.string(), z.number().int().nonnegative()),
});

const BuildPlanSchema = z.looseObject({
  version: z.number().int().positive(),
  kit_id: z.string(),
  catalog_version: z.string(),
  module_library_version: z.string(),
  compiler_version: z.string(),
  validator_version: z.string(),
  title: z.string(),
  prompt: z.string(),
  archetype: z.string(),
  placements: z.array(PlacementSchema),
  connections: z.array(z.looseObject({
    id: z.string(),
    a_placement_id: z.string(),
    b_placement_id: z.string(),
    kind: z.string(),
  })),
  steps: z.array(z.looseObject({
    number: z.number().int().positive(),
    added_placement_ids: z.array(z.string()),
    camera_preset: z.string(),
    instruction_key: z.string(),
  })),
  parts: z.record(z.string(), PartSpecSchema),
  validation: BuildValidationSchema,
  inventory: BrickInventorySnapshotSchema.optional(),
  content_hash: z.string(),
  generated_at: z.string(),
});

const BuildRecipeSchema = z.looseObject({
  version: z.number().int().positive(),
  archetype: z.string(),
  title: z.string(),
  prompt: z.string(),
  palette: z.array(z.number().int()),
  features: z.array(z.string()),
  metadata: z.record(z.string(), z.string()),
});

export const BuildSessionSchema = z.looseObject({
  id: z.string(),
  prompt: z.string(),
  status: z.enum(["clarifying", "queued", "generating", "completed", "failed"]),
  question: z.looseObject({ id: z.string(), prompt: z.string(), options: z.array(z.string()) }).optional(),
  answers: z.record(z.string(), z.string()),
  creation_id: z.string().optional(),
  error: z.string().optional(),
  created_at: z.string(),
  updated_at: z.string(),
});

export const BuildCreationSchema = z.looseObject({
  id: z.string(),
  session_id: z.string(),
  title: z.string(),
  prompt: z.string(),
  archetype: z.string(),
  recipe: BuildRecipeSchema,
  build_plan: BuildPlanSchema,
  validation: BuildValidationSchema,
  created_at: z.string(),
});

export const BuildCreationListSchema = z.looseObject({ creations: z.array(BuildCreationSchema) });

export const EMPTY_BUILD_SESSION: BuildSession = {
  id: "",
  prompt: "",
  status: "failed",
  answers: {},
  error: "The build response could not be read.",
  created_at: "",
  updated_at: "",
};

const emptyValidation = { buildable: false, issues: [], part_count: 0, step_count: 0, used_parts: {} };

export const EMPTY_BUILD_CREATION: BuildCreation = {
  id: "",
  session_id: "",
  title: "",
  prompt: "",
  archetype: "",
  recipe: { version: 1, archetype: "", title: "", prompt: "", palette: [], features: [], metadata: {} },
  build_plan: {
    version: 1, kit_id: "", catalog_version: "", module_library_version: "", compiler_version: "", validator_version: "",
    title: "", prompt: "", archetype: "", placements: [], connections: [], steps: [], parts: {}, validation: emptyValidation,
    content_hash: "", generated_at: "",
  },
  validation: emptyValidation,
  created_at: "",
};

export const EMPTY_BUILD_CREATIONS: BuildCreationList = { creations: [] };

export const EMPTY_BUILD_CATALOG: BuildCatalog = { catalog_version: "", parts: [], colors: [] };

export const EMPTY_BUILD_CATALOG_PART_PAGE: BuildCatalogPartPage = {
  kit_id: "",
  kit_version: 0,
  kit_name: "",
  catalog_version: "",
  profile_total: 0,
  filtered_total: 0,
  categories: [],
  parts: [],
};

export const EMPTY_LDRAW_CATALOG_SYNC_STATUS: LDrawCatalogSyncStatus = {
  enabled: false,
  can_manage: false,
  kit_id: "",
  catalog_version: "",
  target_part_count: 0,
  stored_part_count: 0,
  progress_part_count: 0,
  status: "idle",
};

export const EMPTY_BRICK_INVENTORY: BrickInventory = {
  configured: false,
  catalog_version: "",
  revision: 0,
  items: [],
};
