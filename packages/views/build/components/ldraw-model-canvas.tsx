"use client";

import { useEffect, useRef, useState } from "react";
import type {
  Group,
  Material,
  OrthographicCamera,
  Scene,
  WebGLRenderer,
} from "three";
import type { BuildPartSpec, BuildPlacement } from "@chimii/core/build";
import {
  configureLDrawRenderer,
  createLDrawStudio,
  disposeLDrawMaterials,
  fitLDrawAssembly,
  instantiateLDrawPart,
  prepareLDrawTemplate,
  setLDrawPartHighlighted,
  resizeLDrawStudio,
} from "./ldraw-render-preset";

type ThreeModule = typeof import("three");
type CatalogModule = {
  LDRAW_CATALOG_VERSION: string;
  LDRAW_CATALOG: typeof import("../catalog/catalog.generated").LDRAW_CATALOG;
};
type ModelStatus = "loading" | "ready" | "failed";

interface RuntimeState {
  three: ThreeModule;
  ConditionalLineMaterial: typeof import("three/addons/materials/LDrawConditionalLineMaterial.js").LDrawConditionalLineMaterial;
  catalog: CatalogModule;
  loader: {
    parseAsync(data: ArrayBuffer, path: string): Promise<{ scene: Group }>;
  };
  renderer: WebGLRenderer;
  scene: Scene;
  camera: OrthographicCamera;
  modelRoot: Group;
  viewHeight: number;
  instanceMaterials: Material[];
  instances: { placement: BuildPlacement; part: Group; highlighted: boolean }[];
  contextLost: (event: Event) => void;
  resize: () => void;
  observer?: ResizeObserver;
}

const templateCache = new Map<string, Promise<Group>>();
const catalogPartPayloadCache = new Map<string, Promise<ArrayBuffer>>();

function partPayloadCacheKey(catalogVersion: string, ldrawID: string) {
  return `${catalogVersion}:${ldrawID}`;
}

function decodeBase64(value: string): ArrayBuffer {
  const binary = window.atob(value);
  const bytes = new Uint8Array(binary.length);
  for (let index = 0; index < binary.length; index += 1) {
    bytes[index] = binary.charCodeAt(index);
  }
  return bytes.buffer;
}

function fetchCatalogPartBinary(catalogVersion: string, ldrawID: string): Promise<ArrayBuffer> {
  const key = partPayloadCacheKey(catalogVersion, ldrawID);
  const cached = catalogPartPayloadCache.get(key);
  if (cached) return cached;

  const promise = fetch(`/api/build/catalog/${encodeURIComponent(catalogVersion)}/parts/${encodeURIComponent(ldrawID)}`)
    .then((response) => {
      if (!response.ok) {
        throw new Error(`failed to load catalog part ${ldrawID}: HTTP ${response.status}`);
      }
      return response.arrayBuffer();
    }).catch((error) => {
      catalogPartPayloadCache.delete(key);
      throw error;
    });
  catalogPartPayloadCache.set(key, promise);
  return promise;
}

export function loadLDrawTemplate(runtime: Pick<RuntimeState, "three" | "catalog" | "loader" | "ConditionalLineMaterial">, catalogVersion: string | undefined, ldrawID: string): Promise<Group> {
  const normalizedID = ldrawID.toLowerCase();
  const effectiveVersion = catalogVersion || runtime.catalog.LDRAW_CATALOG_VERSION;
  const asset = effectiveVersion === runtime.catalog.LDRAW_CATALOG_VERSION ? runtime.catalog.LDRAW_CATALOG[normalizedID] : undefined;
  const cacheKey = asset == null
    ? `${effectiveVersion}:remote:${normalizedID}`
    : `${effectiveVersion}:${asset.hash}:${normalizedID}`;
  const cached = templateCache.get(cacheKey);
  if (cached) return cached;

  const source = asset == null
    ? fetchCatalogPartBinary(effectiveVersion, normalizedID)
    : Promise.resolve(decodeBase64(asset.glbBase64));
  const pending = source
    .then((data) => runtime.loader.parseAsync(data, ""))
    .then((model) => prepareLDrawTemplate(runtime.three, model.scene, runtime.ConditionalLineMaterial))
    .catch((error) => {
      templateCache.delete(cacheKey);
      catalogPartPayloadCache.delete(partPayloadCacheKey(effectiveVersion, normalizedID));
      throw error;
    });
  templateCache.set(cacheKey, pending);
  return pending;
}

function updateStep(runtime: RuntimeState, maxStep: number | undefined, highlightedIds: ReadonlySet<string>) {
  for (const instance of runtime.instances) {
    instance.part.visible = maxStep == null || instance.placement.step <= maxStep;
    const highlighted = highlightedIds.has(instance.placement.id);
    if (highlighted !== instance.highlighted) {
      setLDrawPartHighlighted(runtime.three, instance.part, highlighted);
      instance.highlighted = highlighted;
    }
  }
}

export function LDrawModelCanvas({
  placements,
  parts,
  catalogVersion,
  highlightedPlacementIds,
  maxStep,
  yaw,
  onStatus,
}: {
  placements: BuildPlacement[];
  parts: Record<string, BuildPartSpec>;
  catalogVersion?: string;
  highlightedPlacementIds: ReadonlySet<string>;
  maxStep?: number;
  yaw: number;
  onStatus: (status: ModelStatus) => void;
}) {
  const hostRef = useRef<HTMLDivElement>(null);
  const runtimeRef = useRef<RuntimeState | null>(null);
  const yawRef = useRef(yaw);
  const [runtimeVersion, setRuntimeVersion] = useState(0);
  yawRef.current = yaw;
  const stepRef = useRef({ maxStep, highlightedPlacementIds });
  stepRef.current = { maxStep, highlightedPlacementIds };

  useEffect(() => {
    let cancelled = false;
    const host = hostRef.current;
    if (!host) return;
    onStatus("loading");

    void Promise.all([
      import("three"),
      import("three/examples/jsm/loaders/GLTFLoader.js"),
      import("../catalog/catalog.generated"),
      import("three/addons/materials/LDrawConditionalLineMaterial.js"),
    ]).then(([three, { GLTFLoader }, catalog, { LDrawConditionalLineMaterial }]) => {
      if (cancelled) return;
      const renderer = new three.WebGLRenderer({ antialias: true, alpha: true, powerPreference: "high-performance" });
      renderer.setPixelRatio(Math.min(window.devicePixelRatio || 1, 2));
      configureLDrawRenderer(three, renderer);
      renderer.domElement.className = "h-full w-full";
      renderer.domElement.setAttribute("aria-hidden", "true");
      const contextLost = (event: Event) => {
        event.preventDefault();
        if (!cancelled) onStatus("failed");
      };
      renderer.domElement.addEventListener("webglcontextlost", contextLost);
      host.replaceChildren(renderer.domElement);

      const { scene, camera, modelRoot } = createLDrawStudio(three, yawRef.current);

      const runtime: RuntimeState = {
        three,
        ConditionalLineMaterial: LDrawConditionalLineMaterial,
        catalog,
        loader: new GLTFLoader(),
        renderer,
        scene,
        camera,
        modelRoot,
        viewHeight: 240,
        instanceMaterials: [],
        instances: [],
        contextLost,
        resize: () => {},
      };
      runtime.resize = () => {
        resizeLDrawStudio(renderer, camera, runtime.viewHeight, host.clientWidth, host.clientHeight);
        renderer.render(scene, camera);
      };
      runtime.resize();
      if (typeof ResizeObserver !== "undefined") {
        runtime.observer = new ResizeObserver(runtime.resize);
        runtime.observer.observe(host);
      }
      runtimeRef.current = runtime;
      setRuntimeVersion((current) => current + 1);
    }).catch(() => {
      if (!cancelled) onStatus("failed");
    });

    return () => {
      cancelled = true;
      const runtime = runtimeRef.current;
      runtimeRef.current = null;
      if (!runtime) return;
      runtime.observer?.disconnect();
      disposeLDrawMaterials(runtime.instanceMaterials);
      runtime.renderer.domElement.removeEventListener("webglcontextlost", runtime.contextLost);
      runtime.renderer.dispose();
      runtime.renderer.domElement.remove();
    };
  }, [onStatus]);

  useEffect(() => {
    const runtime = runtimeRef.current;
    if (!runtime || runtimeVersion === 0) return;
    let cancelled = false;
    onStatus("loading");

    void Promise.all(placements.map(async (placement) => {
      const spec = parts[placement.part_id];
      if (!spec) throw new Error(`BuildPlan part ${placement.part_id} is missing`);
      const template = await loadLDrawTemplate(runtime, catalogVersion, spec.ldraw_id);
      return { placement, spec, template };
    })).then((loadedParts) => {
      if (cancelled) return;
      const { three } = runtime;
      const assembly = new three.Group();
      const nextMaterials: Material[] = [];
      const instances: RuntimeState["instances"] = [];

      for (const { placement, spec, template } of loadedParts) {
        const highlighted = stepRef.current.highlightedPlacementIds.has(placement.id);
        const { part, materials } = instantiateLDrawPart(three, template, placement.color, highlighted);
        nextMaterials.push(...materials);

        const rotated = Math.abs(placement.rotation % 180) === 90;
        const studsX = rotated ? spec.studs_z : spec.studs_x;
        const studsZ = rotated ? spec.studs_x : spec.studs_z;
        part.position.set(
          placement.x * 20 + studsX * 10,
          (placement.y + spec.plates_y) * 8 - (spec.origin_y_offset_ldu ?? 0),
          placement.z * 20 + studsZ * 10 + (spec.origin_center_z_offset_ldu ?? 0),
        );
        part.rotation.y = -three.MathUtils.degToRad(placement.rotation);
        part.scale.y = -1;
        assembly.add(part);
        instances.push({ placement, part, highlighted });
      }

      runtime.viewHeight = fitLDrawAssembly(three, runtime.camera, assembly);

      disposeLDrawMaterials(runtime.instanceMaterials);
      runtime.instanceMaterials = nextMaterials;
      runtime.instances = instances;
      updateStep(runtime, stepRef.current.maxStep, stepRef.current.highlightedPlacementIds);
      runtime.modelRoot.clear();
      runtime.modelRoot.add(assembly);
      runtime.modelRoot.rotation.y = yawRef.current;
      runtime.resize();
      onStatus("ready");
    }).catch(() => {
      if (!cancelled) onStatus("failed");
    });

    return () => {
      cancelled = true;
    };
  }, [catalogVersion, onStatus, parts, placements, runtimeVersion]);

  useEffect(() => {
    const runtime = runtimeRef.current;
    if (!runtime) return;
    updateStep(runtime, maxStep, highlightedPlacementIds);
    runtime.renderer.render(runtime.scene, runtime.camera);
  }, [maxStep, highlightedPlacementIds]);

  useEffect(() => {
    const runtime = runtimeRef.current;
    if (!runtime) return;
    runtime.modelRoot.rotation.y = yaw;
    runtime.renderer.render(runtime.scene, runtime.camera);
  }, [yaw]);

  return <div ref={hostRef} className="pointer-events-none absolute inset-0 z-[2]" />;
}
