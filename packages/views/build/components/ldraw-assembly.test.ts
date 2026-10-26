import * as three from "three";
import { LDrawConditionalLineMaterial } from "three/addons/materials/LDrawConditionalLineMaterial.js";
import { describe, expect, it } from "vitest";
import type { BuildPartSpec } from "@chimii/core/build";
import { batchLDrawAssembly, updateBatchedLDraw, placeLDrawPart, disposeBatchedLDraw } from "./ldraw-assembly";
import { instantiateLDrawPart, ldrawMeshBounds, prepareLDrawTemplate } from "./ldraw-render-preset";

describe("large LDraw assemblies", () => {
  it("batches 600 surfaces, preserves geometry bounds/lines and independently highlights steps", () => {
    const template = new three.Group();
    template.add(new three.Mesh(new three.BoxGeometry(20, 8, 40), new three.MeshStandardMaterial({ name: "current" })));
    template.userData.ldrawLines = { version: 1, groups: [{ color: "edge-current", conditional: true, vertices: [-10, 4, 0, 10, 4, 0, 0, 1000, 0, 0, 1000, 5] }] };
    prepareLDrawTemplate(three, template, LDrawConditionalLineMaterial);
    const spec: BuildPartSpec = { id: "p", name: "p", category: "brick", ldraw_id: "p.dat", studs_x: 1, studs_z: 2, plates_y: 1, quantity: 0 };
    const loaded = Array.from({ length: 600 }, (_, i) => ({ template, spec, placement: { id: `p${i}`, part_id: "p", color: 4, x: i % 20, y: Math.floor(i / 20), z: 0, rotation: i % 2 ? 90 : 0, step: i + 1, module: "body" } }));
    const batched = batchLDrawAssembly(three, loaded);
    expect(batched.batches).toHaveLength(2);
    expect(batched.materials).toHaveLength(4);
    expect(batched.geometries).toHaveLength(1);
    const classic = new three.Group();
    for (const item of loaded) {
      const { part } = instantiateLDrawPart(three, template, 4);
      placeLDrawPart(three, part, item.placement, spec);
      classic.add(part);
    }
    const old = ldrawMeshBounds(three, classic), next = ldrawMeshBounds(three, batched.group);
    expect(next.min.distanceTo(old.min)).toBeLessThan(0.0001);
    expect(next.max.distanceTo(old.max)).toBeLessThan(0.0001);
    updateBatchedLDraw(three, batched, 2, new Set(["p1"]));
    const matrix = new three.Matrix4();
    batched.batches[0]!.getMatrixAt(0, matrix);
    expect(matrix.determinant()).toBeGreaterThan(0);
    batched.batches[0]!.getMatrixAt(1, matrix);
    expect(matrix.determinant()).toBe(0);
    batched.batches[1]!.getMatrixAt(1, matrix);
    expect(matrix.determinant()).toBeGreaterThan(0);
    expect(batched.instances[2]!.lines.visible).toBe(false);
    expect(batched.instances[1]!.lineSlots[0]!.line.material).toBeInstanceOf(LDrawConditionalLineMaterial);
    expect(batched.instances[0]!.lineSlots[0]!.line.material).not.toBe(batched.instances[1]!.lineSlots[0]!.line.material);
    disposeBatchedLDraw(batched);
  });
});
