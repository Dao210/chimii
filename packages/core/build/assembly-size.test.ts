import { describe, expect, it } from "vitest";
import { EMPTY_BUILD_CREATION } from "./schemas";
import { buildOccupancySize } from "./assembly-size";

describe("catalog model dimensions", () => {
  const part = { id: "beam", name: "Beam", category: "brick", ldraw_id: "3001.dat", studs_x: 4, studs_z: 2, plates_y: 3, quantity: 0 };
  const plan = { ...EMPTY_BUILD_CREATION.build_plan, parts: { beam: part }, placements: [
    { id: "a", part_id: "beam", color: 4, x: -2, y: 0, z: 1, rotation: 90, step: 1, module: "left" },
    { id: "b", part_id: "beam", color: 4, x: 5, y: 3, z: -3, rotation: 0, step: 2, module: "right" },
  ] };
  it("uses all real placements including translated and rotated objects", () => {
    expect(buildOccupancySize(plan)).toEqual({ width: 11, depth: 8, height: 6 });
  });
  it("does not report invented dimensions for incomplete catalogs", () => {
    expect(buildOccupancySize({ ...plan, parts: {} })).toBeNull();
    expect(buildOccupancySize({ ...plan, placements: [] })).toBeNull();
    expect(buildOccupancySize({ ...plan, parts: { beam: { ...part, plates_y: 0 } } })).toBeNull();
  });
});
