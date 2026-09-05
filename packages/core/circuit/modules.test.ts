import { afterEach, describe, expect, it, vi } from "vitest";
import { ApiClient } from "../api/client";
import { circuitFixture } from "./test-fixture";

const api = new ApiClient("https://api.example.test");
afterEach(() => vi.unstubAllGlobals());
function respond(value: unknown) {
  vi.stubGlobal(
    "fetch",
    vi
      .fn()
      .mockResolvedValue(
        new Response(JSON.stringify(value), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      ),
  );
}

describe("module and inventory response boundaries", () => {
  it("reads the five BOSON reference projects and version two document", async () => {
    const { catalog, creation } = circuitFixture("boson-motion-sound-fan");
    respond(catalog);
    expect(
      (await api.getCircuitCatalog(catalog.catalog.kit_id)).catalog.projects,
    ).toHaveLength(5);
    expect(vi.mocked(fetch).mock.calls[0]?.[0]).toContain(
      "kit_id=dfrobot-edu0080-en",
    );
    respond(creation);
    expect((await api.getCircuitCreation("circuit-1")).document.version).toBe(
      2,
    );
  });
  it.each([
    "unknown port",
    "reversed",
    "missing cable",
    "reused cable",
    "wrong reference",
    "unknown system",
  ])("rejects %s", async (defect) => {
    const { creation } = circuitFixture("boson-motion-sound-fan");
    const raw = JSON.parse(JSON.stringify(creation));
    const wires = raw.document.project.connections;
    if (defect === "unknown port") wires[0].to = "madeup:PIN";
    if (defect === "reversed")
      [wires[0].from, wires[0].to] = [wires[0].to, wires[0].from];
    if (defect === "missing cable") delete wires[0].cable_id;
    if (defect === "reused cable") wires[1].cable_id = wires[0].cable_id;
    if (defect === "wrong reference") raw.document.project.expected_nets = [];
    if (defect === "unknown system") raw.document.connection_system = "unknown";
    respond(raw);
    await expect(api.getCircuitCreation("circuit-1")).rejects.toThrow(
      "Invalid circuit response",
    );
  });
  it("rejects malformed kit, inventory, save and trial results", async () => {
    respond({ kits: [] });
    await expect(api.listCircuitKits()).rejects.toThrow(
      "Invalid circuit response",
    );
    const inventory = {
      kit_id: "kit",
      catalog_version: "v1",
      quantities: {},
      revision: 0,
      confirmed: true,
      can_edit: true,
      updated_at: "",
    };
    respond(inventory);
    await expect(api.getCircuitInventory("kit")).rejects.toThrow(
      "Invalid circuit response",
    );
    respond(inventory);
    await expect(
      api.saveCircuitInventory("kit", {
        catalog_version: "v1",
        expected_revision: 0,
        quantities: {},
      }),
    ).rejects.toThrow("Invalid circuit response");
    respond({ trials: null });
    await expect(api.listCircuitTrials("id")).rejects.toThrow(
      "Invalid circuit response",
    );
    respond({ id: "fake", evidence_kind: "physically_verified" });
    await expect(
      api.createCircuitTrial("id", {
        client_request_id: "request",
        hardware_label: "kit",
        notes: "",
        result: "worked",
        adult_checked: true,
      }),
    ).rejects.toThrow("Invalid circuit response");
  });
  it("defaults an unknown edit capability to read only", async () => {
    respond({
      kit_id: "kit",
      catalog_version: "v1",
      quantities: { A: 0 },
      revision: 0,
      confirmed: false,
      can_edit: "unknown",
      updated_at: "",
    });
    expect((await api.getCircuitInventory("kit")).can_edit).toBe(false);
  });
});
