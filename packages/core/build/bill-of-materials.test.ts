import { describe, expect, it } from "vitest";
import { EMPTY_BUILD_CREATION } from "./schemas";
import { buildBillOfMaterials, buildMaterialsCSV } from "./bill-of-materials";

describe("build bill of materials", () => {
  it("counts the actual model by part and color, with advisory shortages", () => {
    const plan = { ...EMPTY_BUILD_CREATION.build_plan, inventory_compared: true, missing_parts: [{ part_id: "brick", color: 4, quantity: 1 }], placements: [4, 4, 15].map((color, i) => ({ id: `p${i}`, part_id: "brick", color, x: 0, y: i, z: 0, rotation: 0, step: i + 1, module: "body" })) };
    expect(buildBillOfMaterials(plan)).toEqual([
      { partId: "brick", ldrawId: "", name: "brick", color: 4, quantity: 2, missing: 1 },
      { partId: "brick", ldrawId: "", name: "brick", color: 15, quantity: 1, missing: 0 },
    ]);
    expect(buildBillOfMaterials({ ...plan, inventory_compared: false })[0]?.missing).toBeUndefined();
    expect(buildBillOfMaterials(plan, { configured: true, catalog_version: "v", revision: 2, items: [{ part_id: "brick", color: 4, quantity: 5 }] })[0]?.missing).toBe(0);
    expect(buildBillOfMaterials(plan, { configured: false, catalog_version: "v", revision: 3, items: [] })[0]?.missing).toBeUndefined();
  });
  it("exports readable CSV and neutralizes spreadsheet formulas", () => {
    const csv = buildMaterialsCSV([{ partId: "p", ldrawId: "3001.dat", name: '=HYPERLINK("x")', color: 4, quantity: 2 }]);
    expect(csv).toContain('"\'=HYPERLINK(""x"")"');
    expect(csv).toContain('"3001.dat"');
  });
});
