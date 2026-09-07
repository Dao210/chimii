import { describe, expect, it } from "vitest";
import {
  makerScope,
  setBrickQuantity,
  makerError,
  quantityInput,
  guideNavigation,
  missingBuildSession,
  sameBrickQuantities,
  sameCircuitQuantities,
} from "./maker";
describe("maker data boundaries", () => {
  it("recognizes an applied inventory draft independent of row order", () => {
    const rows = [
      { part_id: "brick", color: 4, quantity: 2 },
      { part_id: "brick", color: 1, quantity: 8 },
    ];
    expect(sameBrickQuantities(rows, [...rows].reverse())).toBe(true);
    expect(sameBrickQuantities(rows, [rows[0]!])).toBe(false);
    expect(
      sameBrickQuantities(rows, [rows[0]!, { ...rows[1]!, quantity: 7 }]),
    ).toBe(false);
  });
  it("only reconciles identical circuit quantities, allowing omitted zero counts", () => {
    expect(sameCircuitQuantities({ lamp: 1 }, { lamp: 1, wire: 0 })).toBe(true);
    expect(sameCircuitQuantities({ lamp: 1 }, { lamp: 2 })).toBe(false);
  });
  it("preserves offscreen parts and separate colors in full inventory snapshots", () => {
    const stock = [
      { part_id: "wheel", color: 4, quantity: 2 },
      { part_id: "wheel", color: 15, quantity: 8 },
      { part_id: "axle", color: 71, quantity: 4 },
    ];
    const edited = setBrickQuantity(stock, "wheel", 4, 0);
    expect(edited).toEqual(stock.slice(1));
    expect(stock).toHaveLength(3);
    expect(setBrickQuantity(edited, "wheel", 4, 3).at(-1)?.quantity).toBe(3);
  });
  it("isolates drafts across both account and workspace", () => {
    expect(
      new Set([
        makerScope("a", "one"),
        makerScope("a", "two"),
        makerScope("b", "one"),
      ]).size,
    ).toBe(3);
  });
  it("keeps edited rows in place and leaves other rows unchanged", () => {
    const stock = [
      { part_id: "brick", color: 4, quantity: 2 },
      { part_id: "axle", color: 71, quantity: 4 },
    ];
    const edited = setBrickQuantity(stock, "brick", 4, 8);
    expect(edited.map((v) => v.part_id)).toEqual(["brick", "axle"]);
    expect(edited[1]).toBe(stock[1]);
    expect(stock[0]?.quantity).toBe(2);
  });
  it("does not remove stock when a number field is cleared for replacement", () => {
    expect(quantityInput("", 8, 999)).toBe(8);
    expect(quantityInput("0", 8, 999)).toBe(0);
    expect(quantityInput("012", 8, 999)).toBe(12);
    expect(quantityInput("1000", 8, 999)).toBe(999);
  });
  it("revisits completed builds and circuit observations without a write", () => {
    for (const last of [4, 7]) {
      for (let viewed = 0; viewed < last; viewed++)
        expect(guideNavigation(viewed, last, last, true).next).toBe("preview");
      expect(guideNavigation(null, last, last, true).next).toBe("done");
    }
  });
  it("only confirms the current unsaved step and clamps stale previews", () => {
    expect(guideNavigation(1, 3, 5).next).toBe("preview");
    expect(guideNavigation(null, 3, 5)).toEqual({
      step: 3,
      next: "save",
      nextStep: 4,
    });
    expect(guideNavigation(null, 5, 5).next).toBe("complete");
    expect(guideNavigation(4, 2, 5).step).toBe(2);
  });
  it("only abandons a session known to be missing, not an uncertain network result", () => {
    expect(missingBuildSession({ status: 404 })).toBe(true);
    for (const status of [0, 401, 403, 409, 503])
      expect(missingBuildSession({ status })).toBe(false);
  });
  it("does not describe a revision conflict as a successful save", () => {
    expect(makerError({ status: 409 })).toContain("修改仍保留");
    expect(makerError({ status: 403 })).toContain("没有操作权限");
  });
});
