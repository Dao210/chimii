import type { BuildPlan } from "./types";

// These are catalog occupancy dimensions, not measured physical tolerances.
export function buildOccupancySize(plan: Pick<BuildPlan, "placements" | "parts">) {
  if (plan.placements.length === 0) return null;
  let minX = Infinity, minY = Infinity, minZ = Infinity;
  let maxX = -Infinity, maxY = -Infinity, maxZ = -Infinity;
  for (const placement of plan.placements) {
    const part = plan.parts[placement.part_id];
    if (!part || ![part.studs_x, part.studs_z, part.plates_y].every(n => Number.isInteger(n) && n > 0)) return null;
    const rotated = placement.rotation % 180 !== 0;
    minX = Math.min(minX, placement.x); minY = Math.min(minY, placement.y); minZ = Math.min(minZ, placement.z);
    maxX = Math.max(maxX, placement.x + (rotated ? part.studs_z : part.studs_x));
    maxZ = Math.max(maxZ, placement.z + (rotated ? part.studs_x : part.studs_z));
    maxY = Math.max(maxY, placement.y + part.plates_y);
  }
  return { width: maxX - minX, depth: maxZ - minZ, height: maxY - minY };
}
