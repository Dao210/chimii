import { describe, expect, it } from "vitest";
import { applyBuildDesignCommand, reduceBuildDesignHistory } from "./design-commands";
import { BuildCreationSchema, EMPTY_BUILD_CREATION } from "./schemas";
import type { BuildDesignSpec } from "./types";

const design: BuildDesignSpec = { version: 1, mode: "static", shapes: [
  { id: "dial", label: "Dial", kind: "ellipse", operation: "add", position: { x: -5, y: 0, z: -5 }, size: { x: 10, y: 6, z: 10 }, color: 15 },
  { id: "hand", label: "Hand", kind: "box", operation: "add", position: { x: 0, y: 6, z: 0 }, size: { x: 3, y: 3, z: 1 }, color: 1 },
] };

describe("design revisions", () => {
  it("edits one shape without changing the source or another feature and supports undo/redo", () => {
    const initial = { present: design, past: [], future: [] };
    const next = reduceBuildDesignHistory(initial, { type: "replace_shape", shape: { ...design.shapes[0]!, size: { x: 12, y: 6, z: 12 } } });
    expect(design.shapes[0]!.size.x).toBe(10);
    expect(next.present.shapes[1]).toStrictEqual(design.shapes[1]);
    const undone = reduceBuildDesignHistory(next, { type: "undo" });
    expect(undone.present).toBe(design);
    expect(reduceBuildDesignHistory(undone, { type: "redo" }).present).toEqual(next.present);
    expect(reduceBuildDesignHistory(undone, { type: "remove_shape", id: "hand" }).future).toEqual([]);
  });
  it("rejects unknown ids, duplicate shapes, empty designs and invalid dimensions", () => {
    expect(() => applyBuildDesignCommand(design, { type: "remove_shape", id: "missing" })).toThrow();
    expect(() => applyBuildDesignCommand(design, { type: "add_shape", shape: design.shapes[0]! })).toThrow();
    expect(() => applyBuildDesignCommand({ ...design, shapes: [design.shapes[0]!] }, { type: "remove_shape", id: "dial" })).toThrow();
    expect(() => applyBuildDesignCommand(design, { type: "replace_shape", shape: { ...design.shapes[0]!, size: { x: 0, y: 3, z: 2 } } })).toThrow();
  });
  it("keeps historical creations readable and disables editing malformed future documents", () => {
    const raw = { ...EMPTY_BUILD_CREATION, build_plan: { ...EMPTY_BUILD_CREATION.build_plan, document: { version: 1, design: { ...design, shapes: null } } } };
    const parsed = BuildCreationSchema.parse(raw);
    expect(parsed.build_plan.document).toBeUndefined();
    expect(parsed.build_plan.placements).toEqual([]);
    expect(BuildCreationSchema.parse(EMPTY_BUILD_CREATION).id).toBe("");
  });
});
