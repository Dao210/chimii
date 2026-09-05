import { describe, expect, it } from "vitest";
import { makerScope, setBrickQuantity, makerError } from "./maker";
describe("maker data boundaries", () => {
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
  it("does not describe a revision conflict as a successful save", () => {
    expect(makerError({ status: 409 })).toContain("修改仍保留");
    expect(makerError({ status: 403 })).toContain("没有操作权限");
  });
});
