export interface BuildPartBounds {
  min_x?: number;
  min_y?: number;
  min_z?: number;
  max_x?: number;
  max_y?: number;
  max_z?: number;
}

export interface BuildPartConnector {
  id?: string;
  kind: string;
  x_ldu: number;
  y_ldu: number;
  z_ldu: number;
  direction: "up" | "down" | "north" | "east" | "south" | "west";
  capacity_units?: number;
}

export interface BuildPartOccupancy {
  profile?: string;
  ground_contact_profile?: "footprint" | "wheel_point";
  studs_x?: number;
  studs_z?: number;
  plates_y?: number;
}

export interface BuildPartSpec {
  id: string;
  name: string;
  category: string;
  popularity_rank?: number;
  certification_level?: "asset_only" | "basic" | "advanced" | "certified";
  auto_build_eligible?: boolean;
  inventory_eligible?: boolean;
  geometry_profile?: string;
  ldraw_id: string;
  ldraw_status?: string;
  license?: string;
  studs_x: number;
  studs_z: number;
  plates_y: number;
  quantity: number;
  has_top_studs?: boolean;
  has_bottom_receptors?: boolean;
  origin_y_offset_ldu?: number;
  origin_center_z_offset_ldu?: number;
  bounds?: BuildPartBounds;
  connectors?: BuildPartConnector[];
  occupancy?: BuildPartOccupancy;
}

export type BuildCatalogCapability = "all" | "auto_build" | "inventory" | "preview";

export interface BuildCatalogPartPage {
  kit_id: string;
  kit_version: number;
  kit_name: string;
  catalog_version: string;
  profile_total: number;
  filtered_total: number;
  categories: string[];
  parts: BuildPartSpec[];
  next_cursor?: string;
}

export interface BuildCatalogPartFilters {
  query?: string;
  category?: string;
  capability?: BuildCatalogCapability;
  limit?: number;
  cursor?: string;
}

export interface BuildCatalogColor {
  code: number;
  name: string;
  hex: string;
}

export interface BuildCatalogSource {
  release: string;
  archive_sha256: string;
  source_url: string;
}

export interface BuildCatalog {
  catalog_version: string;
  catalog_source?: BuildCatalogSource;
  parts: BuildPartSpec[];
  colors: BuildCatalogColor[];
}

export type LDrawCatalogSyncState =
  | "idle"
  | "queued"
  | "running"
  | "completed"
  | "failed";

export interface LDrawCatalogSyncStatus {
  enabled: boolean;
  can_manage: boolean;
  kit_id: string;
  catalog_version: string;
  target_part_count: number;
  stored_part_count: number;
  progress_part_count: number;
  status: LDrawCatalogSyncState;
  error?: string;
  created_at?: string;
  started_at?: string;
  completed_at?: string;
  updated_at?: string;
}

export interface BrickInventoryItem {
  part_id: string;
  color: number;
  quantity: number;
}

export interface BrickInventory {
  configured: boolean;
  catalog_version: string;
  revision: number;
  items: BrickInventoryItem[];
  updated_at?: string;
}

export interface BrickInventorySnapshot extends BrickInventory {
  content_hash: string;
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
  a_connector_ids?: string[];
  b_connector_ids?: string[];
  kind: string;
  engaged_count?: number;
  capacity_units?: number;
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
  connection_count?: number;
  minimum_stability_margin_mils?: number;
  used_parts: Record<string, number>;
}

export interface BuildPlan {
  version: number;
  kit_id: string;
  catalog_version: string;
  connector_schema_version?: number;
  physics_profile_version?: string;
  generator_version?: string;
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
  inventory?: BrickInventorySnapshot;
  content_hash: string;
  generated_at: string;
}

export interface BuildModuleInstance {
  id: string;
  kind: string;
  parent?: string;
  port?: string;
  alternative_ports?: string[];
  color: number;
}

export interface BuildRecipe {
  subject?: string;
  summary?: string;
  requirements?: string[];
  constraints?: { exact_colors: boolean; no_wheels: boolean; part_count: number; required_modules?: string[] };
  modules?: BuildModuleInstance[];
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
  choices?: { id: string; label: string }[];
  allow_free_text?: boolean;
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
  revision?: number;
  phase?: string;
  summary?: string;
  message?: string;
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
