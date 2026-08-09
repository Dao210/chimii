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
  resizeLDrawStudio,
} from "./ldraw-render-preset";

type ThreeModule = typeof import("three");
type CatalogModule = typeof import("../catalog/catalog.generated");
type ModelStatus = "loading" | "ready" | "failed";

interface RuntimeState {
  three: ThreeModule;
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
  contextLost: (event: Event) => void;
  resize: () => void;
  observer?: ResizeObserver;
}

const templateCache = new Map<string, Promise<Group>>();

function decodeBase64(value: string): ArrayBuffer {
  const binary = window.atob(value);
  const bytes = new Uint8Array(binary.length);
  for (let index = 0; index < binary.length; index += 1) {
    bytes[index] = binary.charCodeAt(index);
  }
  return bytes.buffer;
}

function loadTemplate(runtime: RuntimeState, ldrawID: string): Promise<Group> {
  const normalizedID = ldrawID.toLowerCase();
  const cached = templateCache.get(normalizedID);
  if (cached) return cached;
  const asset = runtime.catalog.LDRAW_CATALOG[normalizedID];
  if (!asset) return Promise.reject(new Error(`LDraw catalog does not contain ${normalizedID}`));
  const pending = runtime.loader.parseAsync(decodeBase64(asset.glbBase64), "").then((model) => model.scene);
  templateCache.set(normalizedID, pending);
  return pending;
}

export function LDrawModelCanvas({
  placements,
  parts,
  catalogVersion,
  highlightedPlacementIds,
  yaw,
  onStatus,
}: {
  placements: BuildPlacement[];
  parts: Record<string, BuildPartSpec>;
  catalogVersion?: string;
  highlightedPlacementIds: ReadonlySet<string>;
  yaw: number;
  onStatus: (status: ModelStatus) => void;
}) {
  const hostRef = useRef<HTMLDivElement>(null);
  const runtimeRef = useRef<RuntimeState | null>(null);
  const yawRef = useRef(yaw);
  const [runtimeVersion, setRuntimeVersion] = useState(0);
  yawRef.current = yaw;

  useEffect(() => {
    let cancelled = false;
    const host = hostRef.current;
    if (!host) return;
    onStatus("loading");

    void Promise.all([
      import("three"),
      import("three/examples/jsm/loaders/GLTFLoader.js"),
      import("../catalog/catalog.generated"),
    ]).then(([three, { GLTFLoader }, catalog]) => {
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
        catalog,
        loader: new GLTFLoader(),
        renderer,
        scene,
        camera,
        modelRoot,
        viewHeight: 240,
        instanceMaterials: [],
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
    if (catalogVersion && catalogVersion !== runtime.catalog.LDRAW_CATALOG_VERSION) {
      onStatus("failed");
      return;
    }
    let cancelled = false;
    onStatus("loading");

    void Promise.all(placements.map(async (placement) => {
      const spec = parts[placement.part_id];
      if (!spec) throw new Error(`BuildPlan part ${placement.part_id} is missing`);
      const template = await loadTemplate(runtime, spec.ldraw_id);
      return { placement, spec, template };
    })).then((loadedParts) => {
      if (cancelled) return;
      const { three } = runtime;
      const assembly = new three.Group();
      const nextMaterials: Material[] = [];

      for (const { placement, spec, template } of loadedParts) {
        const highlighted = highlightedPlacementIds.has(placement.id);
        const { part, materials } = instantiateLDrawPart(three, template, placement.color, highlighted);
        nextMaterials.push(...materials);

        const rotated = Math.abs(placement.rotation % 180) === 90;
        const studsX = rotated ? spec.studs_z : spec.studs_x;
        const studsZ = rotated ? spec.studs_x : spec.studs_z;
        part.position.set(
          placement.x * 20 + studsX * 10,
          placement.y * 8 - (spec.origin_y_offset_ldu ?? 0),
          placement.z * 20 + studsZ * 10 + (spec.origin_center_z_offset_ldu ?? 0),
        );
        part.rotation.y = -three.MathUtils.degToRad(placement.rotation);
        part.scale.y = -1;
        assembly.add(part);
      }

      runtime.viewHeight = fitLDrawAssembly(three, runtime.camera, assembly);

      disposeLDrawMaterials(runtime.instanceMaterials);
      runtime.instanceMaterials = nextMaterials;
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
  }, [catalogVersion, highlightedPlacementIds, onStatus, parts, placements, runtimeVersion]);

  useEffect(() => {
    const runtime = runtimeRef.current;
    if (!runtime) return;
    runtime.modelRoot.rotation.y = yaw;
    runtime.renderer.render(runtime.scene, runtime.camera);
  }, [yaw]);

  return <div ref={hostRef} className="pointer-events-none absolute inset-0 z-[2]" />;
}
