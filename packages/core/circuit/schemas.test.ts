import { afterEach, describe, expect, it, vi } from "vitest";
import { ApiClient } from "../api/client";
import { circuitFixture } from "./test-fixture";
import { circuitCoordinate, circuitMaterials, circuitPoint } from "./geometry";

afterEach(() => vi.unstubAllGlobals());
const api = new ApiClient("https://api.example.test");
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

describe("circuit response boundary", () => {
  it("accepts the server catalogue and future non-critical fields", async () => {
    const { catalog, creation } = circuitFixture();
    respond({ ...catalog, ai_available: "unknown", future_field: true });
    expect((await api.getCircuitCatalog()).ai_available).toBe(false);
    respond({ ...creation, observation: "future_result", future_field: true });
    expect((await api.getCircuitCreation("circuit-1")).observation).toBe(
      "future_result",
    );
  });

  it.each([
    "null steps",
    "missing component",
    "dangling step",
    "empty inventory",
    "contradictory evidence",
    "invalid progress",
  ])("rejects %s rather than displaying a buildable design", async (defect) => {
    const { creation } = circuitFixture();
    const raw = JSON.parse(JSON.stringify(creation));
    if (defect === "null steps") raw.document.project.steps = null;
    if (defect === "missing component") raw.document.parts = [];
    if (defect === "dangling step")
      raw.document.project.steps[0].placement_ids = ["imaginary"];
    if (defect === "empty inventory") raw.document.inventory = {};
    if (defect === "contradictory evidence")
      raw.document.validation.issues = [{ code: "short_circuit" }];
    if (defect === "invalid progress") raw.current_step = 999;
    respond(raw);
    await expect(api.getCircuitCreation("circuit-1")).rejects.toThrow(
      "Invalid circuit response",
    );
    respond(raw);
    await expect(
      api.updateCircuitProgress("circuit-1", {
        current_step: 1,
        observation: "not_tried",
        expected_revision: 0,
      }),
    ).rejects.toThrow("Invalid circuit response");
  });

  it("rejects malformed create and list responses", async () => {
    respond({ id: "missing-document" });
    await expect(
      api.createCircuit({
        client_request_id: "request",
        kit_id: "kit",
        catalog_version: "v1",
        project_id: "switch-light",
        locale: "en",
        inventory: {},
      }),
    ).rejects.toThrow("Invalid circuit response");
    respond({ creations: null });
    await expect(api.listCircuitCreations()).rejects.toThrow(
      "Invalid circuit response",
    );
  });
});

it("maps the radio amplifier pins to the manufacturer's board coordinates", () => {
  const { creation } = circuitFixture("fm-radio");
  const amplifier = creation.document.project.placements.find(
    (p) => p.part_id === "U4",
  )!;
  const pins = creation.document.parts.find((p) => p.id === "U4")!.ports;
  const result = Object.fromEntries(
    pins.map((p) => {
      const point = circuitPoint(amplifier, p.x, p.y);
      return [p.id, circuitCoordinate(point.x, point.y)];
    }),
  );
  expect(result).toEqual({
    "-": "D6",
    INP: "C6",
    FIL: "B6",
    OUT: "D7",
    "+": "B7",
  });
  expect(
    circuitMaterials(creation.document.project, {}).every(
      (p) => p.missing === p.required,
    ),
  ).toBe(true);
});
