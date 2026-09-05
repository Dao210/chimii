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

const PartBoundsSchema = z.looseObject({
  min_x: z.number().optional(),
  min_y: z.number().optional(),
  min_z: z.number().optional(),
  max_x: z.number().optional(),
  max_y: z.number().optional(),
  max_z: z.number().optional(),
});

const PartConnectorSchema = z.looseObject({
  id: z.string().optional(),
  kind: z.string(),
  x_ldu: z.number().int(),
  y_ldu: z.number().int(),
  z_ldu: z.number().int(),
  direction: z.enum(["up", "down", "north", "east", "south", "west"]),
  capacity_units: z.number().int().positive().optional(),
});

const PartOccupancySchema = z.looseObject({
  profile: z.string().optional(),
  ground_contact_profile: z.enum(["footprint", "wheel_point"]).optional(),
  studs_x: z.number().int().positive().optional(),
  studs_z: z.number().int().positive().optional(),
  plates_y: z.number().int().positive().optional(),
});

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
  has_top_studs: z.boolean().optional(),
  has_bottom_receptors: z.boolean().optional(),
  origin_y_offset_ldu: z.number().int().optional(),
  origin_center_z_offset_ldu: z.number().int().optional(),
  bounds: PartBoundsSchema.optional(),
  connectors: z.array(PartConnectorSchema).optional(),
  occupancy: PartOccupancySchema.optional(),
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
  connection_count: z.number().int().nonnegative().optional(),
  minimum_stability_margin_mils: z.number().int().optional(),
  used_parts: z.record(z.string(), z.number().int().nonnegative()),
});

const BuildPlanSchema = z.looseObject({
  version: z.number().int().positive(),
  kit_id: z.string(),
  catalog_version: z.string(),
  connector_schema_version: z.number().int().positive().optional(),
  physics_profile_version: z.string().optional(),
  generator_version: z.string().optional(),
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
    a_connector_ids: z.array(z.string()).optional(),
    b_connector_ids: z.array(z.string()).optional(),
    kind: z.string(),
    engaged_count: z.number().int().positive().optional(),
    capacity_units: z.number().int().positive().optional(),
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
  subject: z.string().optional(), summary: z.string().optional(), requirements: z.array(z.string()).optional(),
  constraints: z.looseObject({ exact_colors: z.boolean(), no_wheels: z.boolean(), part_count: z.number().int().nonnegative(), required_modules: z.array(z.string()).nullable().transform((v) => v ?? []).optional() }).optional(),
  modules: z.array(z.looseObject({ id: z.string(), kind: z.string(), parent: z.string().optional(), port: z.string().optional(), color: z.number().int(), alternative_ports: z.array(z.string()).optional() })).optional(),
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
  revision: z.number().int().positive().optional(),
  phase: z.string().optional(),
  summary: z.string().optional(),
  message: z.string().optional(),
  question: z.looseObject({
    id: z.string(), prompt: z.string(), options: z.array(z.string()).default([]),
    choices: z.array(z.object({ id: z.string(), label: z.string() })).optional().catch(undefined),
    allow_free_text: z.boolean().optional().catch(undefined),
  }).optional(),
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

export const BuildProgressSchema = z.object({
  id: z.string().min(1), current_step: z.number().int().nonnegative(),
  completed_at: z.string().nullable(), revision: z.number().int().nonnegative(),
  step_count: z.number().int().positive(),
}).refine(v => v.current_step <= v.step_count, "Invalid progress step");
export const BuildSummarySchema = z.object({
  id: z.string().min(1), title: z.string(), prompt: z.string(), archetype: z.string(),
  part_count: z.number().int().nonnegative(), step_count: z.number().int().positive(),
  created_at: z.string(), progress: BuildProgressSchema,
});
export const BuildSummaryListSchema = z.object({ creations: z.array(BuildSummarySchema) });
export type BuildProgress = z.infer<typeof BuildProgressSchema>;
export type BuildSummary = z.infer<typeof BuildSummarySchema>;
export interface BuildProgressInput { current_step: number; expected_revision: number; completed?: boolean }
