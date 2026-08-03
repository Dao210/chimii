"use client";

import { useEffect, useRef, useState } from "react";
import type {
  BufferGeometry,
  Group,
  Material,
  OrthographicCamera,
  Scene,
  WebGLRenderer,
} from "three";
import type { BuildPartSpec, BuildPlacement } from "@chimii/core/build";

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

const OFFICIAL_COLORS: Record<number, number> = {
  1: 0x1e5aa8,
  2: 0x00852b,
  4: 0xb40000,
  14: 0xfac80a,
  15: 0xf4f4f4,
  71: 0x969696,
};

const templateCache = new Map<string, Promise<Group>>();
const edgeGeometryCache = new WeakMap<BufferGeometry, BufferGeometry>();

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

function disposeInstanceMaterials(materials: Material[]) {
  for (const material of materials) material.dispose();
}

function colorMaterial(three: ThreeModule, source: Material, colorCode: number, highlighted: boolean): Material {
  const sourceName = source.name.toLowerCase();
  const material = sourceName === "current"
    ? new three.MeshStandardMaterial({
        color: OFFICIAL_COLORS[colorCode] ?? OFFICIAL_COLORS[71],
        roughness: 0.32,
        metalness: 0,
        side: three.DoubleSide,
      })
    : source.clone();
  if (material instanceof three.MeshStandardMaterial) {
    material.side = three.DoubleSide;
    material.roughness = 0.32;
    if (highlighted) {
      material.emissive.set(0xffd85a);
      material.emissiveIntensity = 0.35;
    }
  }
  return material;
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
      renderer.setClearColor(0x000000, 0);
      renderer.outputColorSpace = three.SRGBColorSpace;
      renderer.toneMapping = three.ACESFilmicToneMapping;
      renderer.toneMappingExposure = 1.15;
      renderer.domElement.className = "h-full w-full";
      renderer.domElement.setAttribute("aria-hidden", "true");
      const contextLost = (event: Event) => {
        event.preventDefault();
        if (!cancelled) onStatus("failed");
      };
      renderer.domElement.addEventListener("webglcontextlost", contextLost);
      host.replaceChildren(renderer.domElement);

      const scene = new three.Scene();
      const camera = new three.OrthographicCamera(-100, 100, 100, -100, 0.1, 4000);
      camera.position.set(0, 150, 420);
      camera.lookAt(0, 0, 0);
      scene.add(new three.HemisphereLight(0xfff3d2, 0x173866, 2.5));
      const keyLight = new three.DirectionalLight(0xffffff, 3.1);
      keyLight.position.set(-180, 260, 220);
      scene.add(keyLight);
      const fillLight = new three.DirectionalLight(0x9bc8ff, 1.2);
      fillLight.position.set(220, 80, -160);
      scene.add(fillLight);
      const modelRoot = new three.Group();
      modelRoot.rotation.y = yawRef.current;
      scene.add(modelRoot);

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
        const width = Math.max(1, host.clientWidth);
        const height = Math.max(1, host.clientHeight);
        const aspect = width / height;
        camera.left = -(runtime.viewHeight * aspect) / 2;
        camera.right = (runtime.viewHeight * aspect) / 2;
        camera.top = runtime.viewHeight / 2;
        camera.bottom = -runtime.viewHeight / 2;
        camera.updateProjectionMatrix();
        renderer.setSize(width, height, false);
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
      disposeInstanceMaterials(runtime.instanceMaterials);
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
        const part = template.clone(true);
        const highlighted = highlightedPlacementIds.has(placement.id);
        part.traverse((object) => {
          if (!(object instanceof three.Mesh)) return;
          const sourceMaterials = Array.isArray(object.material) ? object.material : [object.material];
          const materials = sourceMaterials.map((source) => {
            const material = colorMaterial(three, source, placement.color, highlighted);
            nextMaterials.push(material);
            return material;
          });
          object.material = Array.isArray(object.material) ? materials : materials[0]!;
          const baseGeometry = object.geometry as BufferGeometry;
          let edgeGeometry = edgeGeometryCache.get(baseGeometry);
          if (!edgeGeometry) {
            edgeGeometry = new three.EdgesGeometry(baseGeometry, 28);
            edgeGeometryCache.set(baseGeometry, edgeGeometry);
          }
          const edgeMaterial = new three.LineBasicMaterial({
            color: highlighted ? 0xffeb78 : 0x15213a,
            transparent: true,
            opacity: highlighted ? 0.9 : 0.3,
          });
          nextMaterials.push(edgeMaterial);
          object.add(new three.LineSegments(edgeGeometry, edgeMaterial));
        });

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

      const box = new three.Box3().setFromObject(assembly);
      if (!box.isEmpty()) {
        const center = box.getCenter(new three.Vector3());
        const size = box.getSize(new three.Vector3());
        assembly.position.sub(center);
        const horizontal = Math.max(size.x, size.z, 80);
        runtime.viewHeight = Math.max(size.y * 1.5, horizontal * 0.95, 130);
        runtime.camera.position.set(0, Math.max(120, size.y * 1.15), Math.max(360, horizontal * 2.25));
        runtime.camera.lookAt(0, 0, 0);
      }

      disposeInstanceMaterials(runtime.instanceMaterials);
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
