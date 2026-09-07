import { afterEach, describe, expect, it, vi } from "vitest";
import referenceData from "../../../server/internal/circuit/references/nezha-v2-gate.json";
import { ApiClient } from "../api/client";
import { AssemblyReferenceSchema } from "./assembly-reference";
import { CircuitDocumentSchema } from "./schemas";
import { circuitFixture } from "./test-fixture";

const reference = () => structuredClone({ ...referenceData, content_hash: "a".repeat(64) });
const { catalog } = circuitFixture();
const kits = [{
  kit_id: catalog.catalog.kit_id, name: catalog.catalog.name,
  version: catalog.catalog.version, connection_system: "snap", hardware: {
    manufacturer: "Elenco", model: "SC-500", notes: { en: "Reference", zh: "参考" },
    checked_on: "2026-09-07", purchase_links: [],
  },
}];
afterEach(() => vi.unstubAllGlobals());

describe("assembly research API boundary", () => {
  it("reads the server-owned source transcription without treating it as a design", () => {
    const ref = AssemblyReferenceSchema.parse(reference());
    expect(ref.parts.reduce((n, p) => n + p.pictured_quantity, 0)).toBe(36);
    expect(ref.steps).toHaveLength(15);
    expect(CircuitDocumentSchema.safeParse(ref).success).toBe(false);
  });

  it.each(["count", "duplicate", "missing step", "wire", "unsafe URL", "invalid URL", "ready"])("rejects malformed %s evidence", (defect) => {
    const ref = reference();
    if (defect === "count") ref.parts[0]!.pictured_quantity++;
    if (defect === "duplicate") ref.parts[1]!.id = ref.parts[0]!.id;
    if (defect === "missing step") ref.steps.shift();
    if (defect === "wire") ref.connections[0]!.part_id = "axle";
    if (defect === "unsafe URL") ref.source_url = "javascript:alert(1)";
    if (defect === "invalid URL") ref.source_url = "not a URL";
    if (defect === "ready") ref.status = "ready";
    expect(AssemblyReferenceSchema.safeParse(ref).success).toBe(false);
  });

  it.each([undefined, null, [{ status: "future" }], [reference(), reference()]])("preserves existing kits when optional research is absent or malformed", async (assembly_references) => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({ kits, assembly_references }))));
    const result = await new ApiClient("https://example.test").listCircuitKits();
    expect(result.kits[0]?.kit_id).toBe(kits[0]!.kit_id);
    expect(result.assembly_references).toEqual([]);
  });

  it("returns validated references through the existing kits API", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({ kits, assembly_references: [reference()] }))));
    const result = await new ApiClient("https://example.test").listCircuitKits();
    expect(result.assembly_references[0]?.status).toBe("research");
    expect(result.kits.map((kit) => kit.kit_id)).not.toContain(referenceData.kit_id);
  });
});
