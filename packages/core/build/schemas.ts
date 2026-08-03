import { z } from "zod";
import type { BuildCreation, BuildCreationList, BuildSession } from "./types";

const PartSpecSchema = z.looseObject({
  id: z.string(),
  name: z.string(),
  ldraw_id: z.string(),
  studs_x: z.number().int().positive(),
  studs_z: z.number().int().positive(),
  plates_y: z.number().int().positive(),
  quantity: z.number().int().nonnegative(),
  origin_y_offset_ldu: z.number().int().optional(),
  origin_center_z_offset_ldu: z.number().int().optional(),
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
  issues: z.array(z.looseObject({
    code: z.string(),
    message: z.string(),
    placement_id: z.string().optional(),
  })),
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
