import { render as renderView, waitFor } from "@testing-library/react";
import { describe, expect, it, vi, afterEach } from "vitest";
import * as three from "three";
import { LDrawConditionalLineMaterial } from "three/addons/materials/LDrawConditionalLineMaterial.js";
import type { BuildPartSpec, BuildPlacement } from "@chimii/core/build";
import { LDrawModelCanvas, loadLDrawTemplate } from "./ldraw-model-canvas";

const capture = vi.hoisted(() => ({ draw: vi.fn(), parse: vi.fn() }));
vi.mock("three", async (importOriginal) => {
  const actual = await importOriginal<typeof import("three")>();
  return { ...actual, WebGLRenderer: class {
    domElement = document.createElement("canvas");
    setPixelRatio() {} setClearColor() {} setSize() {} dispose() {}
    render = capture.draw;
  } };
});
vi.mock("three/examples/jsm/loaders/GLTFLoader.js", () => ({ GLTFLoader: class { parseAsync = capture.parse; } }));
vi.mock("../catalog/catalog.generated", () => ({ LDRAW_CATALOG_VERSION: "test-bundled", LDRAW_CATALOG: {
  "test.dat": { hash: "test", glbBase64: "AA==", assetName: "test.glb", triangleCount: 12, dependencies: [], bounds: [0, 0, 0, 20, 8, 20] },
} }));
afterEach(() => vi.unstubAllGlobals());
function model() {
  const scene = new three.Group();
  scene.add(new three.Mesh(new three.BoxGeometry(20, 8, 20), new three.MeshStandardMaterial({ name: "current" })));
  return { scene };
}

it("loads the requested catalog version and retries failed fetches", async () => {
  const fetcher = vi.fn().mockRejectedValueOnce(new Error("offline")).mockResolvedValue(new Response(new Uint8Array([0])));
  vi.stubGlobal("fetch", fetcher);
  const runtime = { three, ConditionalLineMaterial: LDrawConditionalLineMaterial, loader: { parseAsync: vi.fn(async () => model()) }, catalog: {
    LDRAW_CATALOG_VERSION: "bundled", LDRAW_SOURCE: { release: "", archiveSha256: "", url: "" },
    LDRAW_CATALOG: { "asset.dat": { hash: "old", assetName: "old.glb", glbBase64: "AA==", triangleCount: 1, dependencies: [], bounds: [0, 0, 0, 1, 1, 1] as const } },
  } };
  await expect(loadLDrawTemplate(runtime, "new-catalog", "asset.dat")).rejects.toThrow("offline");
  await loadLDrawTemplate(runtime, "new-catalog", "asset.dat");
  expect(fetcher).toHaveBeenCalledTimes(2);
  expect(fetcher).toHaveBeenLastCalledWith("/api/build/catalog/new-catalog/parts/asset.dat");
  await loadLDrawTemplate(runtime, "new-catalog", "asset.dat");
  expect(fetcher).toHaveBeenCalledTimes(2);
});

it("refetches an asset after parsing fails instead of keeping the failed payload", async () => {
  const fetcher = vi.fn().mockImplementation(async () => new Response(new Uint8Array([0])));
  vi.stubGlobal("fetch", fetcher);
  const runtime = {
    three,
    ConditionalLineMaterial: LDrawConditionalLineMaterial,
    loader: { parseAsync: vi.fn().mockRejectedValueOnce(new Error("invalid GLB")).mockImplementation(async () => model()) },
    catalog: { LDRAW_CATALOG_VERSION: "bundled", LDRAW_CATALOG: {} },
  };
  await expect(loadLDrawTemplate(runtime, "retry-parse", "asset.dat")).rejects.toThrow("invalid GLB");
  await loadLDrawTemplate(runtime, "retry-parse", "asset.dat");
  expect(fetcher).toHaveBeenCalledTimes(2);
  expect(runtime.loader.parseAsync).toHaveBeenCalledTimes(2);
});

describe("step navigation", () => {
  it("reuses instances, geometry, materials and camera framing across steps", async () => {
    capture.parse.mockImplementation(async () => model());
    const parts: Record<string, BuildPartSpec> = { brick: { id: "brick", name: "Brick", category: "brick", quantity: 2, ldraw_id: "test.dat", studs_x: 1, studs_z: 1, plates_y: 1 } };
    const placements = [
      { id: "first", part_id: "brick", color: 4, x: 0, y: 0, z: 0, rotation: 0, step: 1 },
      { id: "second", part_id: "brick", color: 14, x: 0, y: 1, z: 0, rotation: 0, step: 2 },
    ] as BuildPlacement[];
    const onStatus = vi.fn();
    const props = { parts, placements, catalogVersion: "test-bundled", onStatus, yaw: 0 };
    const { rerender, unmount } = renderView(<LDrawModelCanvas {...props} maxStep={1} highlightedPlacementIds={new Set(["first"])} />);
    await waitFor(() => expect(onStatus).toHaveBeenLastCalledWith("ready"));
    const [scene, camera] = capture.draw.mock.calls.at(-1)! as [three.Scene, three.OrthographicCamera];
    const modelRoot = scene.children.find((child) => child instanceof three.Group)!;
    const assembly = modelRoot.children[0]!;
    const [first, second] = assembly.children;
    const mesh = first!.children[0] as three.Mesh<three.BufferGeometry, three.MeshStandardMaterial>;
    const geometry = mesh.geometry, material = mesh.material;
    const cameraPosition = camera.position.toArray();
    const readyCalls = onStatus.mock.calls.length;
    expect(second!.visible).toBe(false);
    rerender(<LDrawModelCanvas {...props} maxStep={2} highlightedPlacementIds={new Set(["second"])} />);
    expect(modelRoot.children[0]).toBe(assembly);
    expect(assembly.children[0]).toBe(first);
    expect(mesh.geometry).toBe(geometry);
    expect(mesh.material).toBe(material);
    expect(material.emissive.getHex()).toBe(0);
    expect(second!.visible).toBe(true);
    expect(camera.position.toArray()).toEqual(cameraPosition);
    expect(onStatus.mock.calls.length).toBe(readyCalls);
    unmount();
  });
});
