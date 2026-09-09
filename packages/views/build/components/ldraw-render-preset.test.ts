import * as three from "three";
import { LDrawConditionalLineMaterial } from "three/addons/materials/LDrawConditionalLineMaterial.js";
import { describe, expect, it } from "vitest";
import { instantiateLDrawPart, ldrawMeshBounds, prepareLDrawTemplate, setLDrawPartHighlighted } from "./ldraw-render-preset";

function template() {
  const root = new three.Group();
  const material = new three.MeshStandardMaterial({ name: "current" });
  root.add(new three.Mesh(new three.BoxGeometry(20, 8, 20), material));
  root.userData.ldrawLines = { version: 1, groups: [
    { color: "edge-current", conditional: false, vertices: [-10, 4, 0, 10, 4, 0] },
    { color: "edge-current", conditional: true, vertices: [-10, 4, 0, 10, 4, 0, 0, 10000, 0, 0, 10000, 5] },
  ] };
  return root;
}

describe("LDraw render assets", () => {
  it("uses the original line geometry and shares buffers without sharing instance materials", () => {
    const root = prepareLDrawTemplate(three, template(), LDrawConditionalLineMaterial);
    expect(root.userData.ldrawLines).toBeUndefined();
    const a = instantiateLDrawPart(three, root, 4).part;
    const b = instantiateLDrawPart(three, root, 14).part;
    const lines = a.children.filter((child) => child instanceof three.LineSegments);
    expect(lines).toHaveLength(2);
    expect(a.children[0]!.children).toHaveLength(0);
    const conditional = lines.find((line) => line.material instanceof LDrawConditionalLineMaterial)!;
    expect(conditional.geometry.getAttribute("position").count).toBe(2);
    expect(Array.from(conditional.geometry.getAttribute("control0").array)).toEqual([0, 10000, 0, 0, 10000, 0]);
    expect(Array.from(conditional.geometry.getAttribute("direction").array)).toEqual([20, 0, 0, 20, 0, 0]);
    const second = b.children[1] as three.LineSegments;
    expect(lines[0]!.geometry).toBe(second.geometry);
    expect(lines[0]!.material).not.toBe(second.material);
    const bounds = ldrawMeshBounds(three, a);
    expect(bounds.min.toArray()).toEqual([-10, -4, -10]);
    expect(bounds.max.toArray()).toEqual([10, 4, 10]);
    setLDrawPartHighlighted(three, a, true);
    expect((a.children[0] as three.Mesh<three.BufferGeometry, three.MeshStandardMaterial>).material.emissive.getHex()).toBe(0xffd85a);
    expect((b.children[0] as three.Mesh<three.BufferGeometry, three.MeshStandardMaterial>).material.emissive.getHex()).toBe(0);
    setLDrawPartHighlighted(three, a, false);
    expect((lines[0]!.material as three.LineBasicMaterial).color.getHex()).toBe(0x333333);
    expect((lines[0]!.material as three.LineBasicMaterial).opacity).toBe(0.55);
  });

  it("retains geometry outlines for historical GLBs", () => {
    const root = template();
    delete root.userData.ldrawLines;
    const part = instantiateLDrawPart(three, prepareLDrawTemplate(three, root, LDrawConditionalLineMaterial), 4).part;
    expect(part.children[0]!.children[0]).toBeInstanceOf(three.LineSegments);
  });

  it("rejects incomplete or non-finite conditional line attributes", () => {
    const root = template();
    root.userData.ldrawLines.groups[1].vertices = [0, 0, Number.NaN];
    expect(() => prepareLDrawTemplate(three, root, LDrawConditionalLineMaterial)).toThrow("Invalid LDraw line data");
  });
});
