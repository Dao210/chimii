import { createHash } from "node:crypto";
import { describe, expect, it } from "vitest";
import { LDRAW_CATALOG, LDRAW_CATALOG_VERSION, LDRAW_SOURCE } from "./catalog.generated";

const STARTER_PARTS = [
  "3001.dat",
  "3003.dat",
  "3004.dat",
  "3005.dat",
  "3020.dat",
  "3022.dat",
  "3023.dat",
  "3039.dat",
  "4600.dat",
  "4624c04.dat",
];

describe("generated LDraw catalog", () => {
  it("covers the complete Starter Kit allowlist from one pinned official release", () => {
    expect(Object.keys(LDRAW_CATALOG).sort()).toEqual(STARTER_PARTS);
    expect(LDRAW_CATALOG_VERSION).toContain(LDRAW_SOURCE.release);
    expect(LDRAW_CATALOG_VERSION).toContain(LDRAW_SOURCE.archiveSha256.slice(0, 12));
    expect(LDRAW_SOURCE.archiveSha256).toHaveLength(64);
  });

  it.each(STARTER_PARTS)("stores a valid, content-addressed GLB for %s", (partID) => {
    const asset = LDRAW_CATALOG[partID];
    expect(asset).toBeDefined();
    const glb = Buffer.from(asset!.glbBase64, "base64");
    expect(glb.subarray(0, 4).toString("ascii")).toBe("glTF");
    expect(glb.readUInt32LE(4)).toBe(2);
    expect(glb.readUInt32LE(8)).toBe(glb.byteLength);
    expect(createHash("sha256").update(glb).digest("hex")).toBe(asset!.hash);
    expect(asset!.assetName).toBe(`${asset!.hash}.glb`);
    expect(asset!.triangleCount).toBeGreaterThan(0);
    expect([...asset!.dependencies].sort()).toEqual(asset!.dependencies);
  });
});
