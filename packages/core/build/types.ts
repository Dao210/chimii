export interface BuildPartSpec {
  id: string;
  name: string;
  ldraw_id: string;
  studs_x: number;
  studs_z: number;
  plates_y: number;
  quantity: number;
  origin_y_offset_ldu?: number;
  origin_center_z_offset_ldu?: number;
}

export interface BuildPlacement {
  id: string;
  part_id: string;
  color: number;
  x: number;
  y: number;
  z: number;
  rotation: number;
  step: number;
  module: string;
}

export interface BuildConnection {
  id: string;
  a_placement_id: string;
  b_placement_id: string;
  kind: string;
}

export interface BuildStep {
  number: number;
  added_placement_ids: string[];
  camera_preset: string;
  instruction_key: string;
}

export interface BuildValidationIssue {
  code: string;
  message: string;
  placement_id?: string;
}

export interface BuildValidationReport {
  buildable: boolean;
  issues: BuildValidationIssue[];
  part_count: number;
  step_count: number;
  used_parts: Record<string, number>;
}

export interface BuildPlan {
  version: number;
  kit_id: string;
  catalog_version: string;
  module_library_version: string;
  compiler_version: string;
  validator_version: string;
  title: string;
  prompt: string;
  archetype: string;
  placements: BuildPlacement[];
  connections: BuildConnection[];
  steps: BuildStep[];
  parts: Record<string, BuildPartSpec>;
  validation: BuildValidationReport;
  content_hash: string;
  generated_at: string;
}

export interface BuildRecipe {
  version: number;
  archetype: string;
  title: string;
  prompt: string;
  palette: number[];
  features: string[];
  metadata: Record<string, string>;
}

export interface BuildQuestion {
  id: string;
  prompt: string;
  options: string[];
}

export type BuildSessionStatus =
  | "clarifying"
  | "queued"
  | "generating"
  | "completed"
  | "failed";

export interface BuildSession {
  id: string;
  prompt: string;
  status: BuildSessionStatus;
  question?: BuildQuestion;
  answers: Record<string, string>;
  creation_id?: string;
  error?: string;
  created_at: string;
  updated_at: string;
}

export interface BuildCreation {
  id: string;
  session_id: string;
  title: string;
  prompt: string;
  archetype: string;
  recipe: BuildRecipe;
  build_plan: BuildPlan;
  validation: BuildValidationReport;
  created_at: string;
}

export interface BuildCreationList {
  creations: BuildCreation[];
}
