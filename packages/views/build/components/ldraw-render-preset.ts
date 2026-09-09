import type {
  BufferGeometry,
  Color,
  Group,
  Material,
  Object3D,
  OrthographicCamera,
  Scene,
  WebGLRenderer,
} from "three";

type ThreeModule = typeof import("three");

export const LDRAW_RENDER_PRESET_VERSION = "ldraw-studio-v2";
export const LDRAW_DEFAULT_YAW = -Math.PI / 4;

export const LDRAW_OFFICIAL_COLORS: Readonly<Record<number, number>> = {
  1: 0x1e5aa8,
  2: 0x00852b,
  4: 0xb40000,
  14: 0xfac80a,
  15: 0xf4f4f4,
  71: 0x969696,
};

const edgeGeometryCache = new WeakMap<BufferGeometry, BufferGeometry>();

export interface LDrawStudio {
  scene: Scene;
  camera: OrthographicCamera;
  modelRoot: Group;
}

export function configureLDrawRenderer(three: ThreeModule, renderer: WebGLRenderer) {
  renderer.setClearColor(0x000000, 0);
  renderer.outputColorSpace = three.SRGBColorSpace;
  renderer.toneMapping = three.ACESFilmicToneMapping;
  renderer.toneMappingExposure = 1.15;
}

export function createLDrawStudio(three: ThreeModule, yaw = LDRAW_DEFAULT_YAW): LDrawStudio {
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
  modelRoot.rotation.y = yaw;
  scene.add(modelRoot);

  return { scene, camera, modelRoot };
}

type ConditionalMaterial = typeof import("three/addons/materials/LDrawConditionalLineMaterial.js").LDrawConditionalLineMaterial;

// Prepare shared geometry once, before cloning any placement. Discard the raw
// extras afterwards so Object3D.clone does not copy all control points per brick.
export function prepareLDrawTemplate(three: ThreeModule, template: Group, ConditionalLineMaterial: ConditionalMaterial): Group {
  if (template.userData.hasLDrawLines === true) return template;
  const nodes: Object3D[] = [];
  template.traverse((node) => {
    if (node.userData.ldrawLines != null) nodes.push(node);
  });
  for (const node of nodes) {
    const data = node.userData.ldrawLines;
    if (data.version !== 1 || !Array.isArray(data.groups)) throw new Error("Unsupported LDraw line data");
    for (const group of data.groups) {
      const stride = group.conditional === true ? 12 : 6;
      const values: unknown = group.vertices;
      if (typeof group.color !== "string" || typeof group.conditional !== "boolean" || !Array.isArray(values) || values.length % stride !== 0 || !values.every((v) => typeof v === "number" && Number.isFinite(v))) {
        throw new Error("Invalid LDraw line data");
      }
      if (values.length === 0) continue;
      const positions: number[] = [], control0: number[] = [], control1: number[] = [], directions: number[] = [];
      for (let i = 0; i < values.length; i += stride) {
        positions.push(...values.slice(i, i + 6));
        if (group.conditional) {
          for (let endpoint = 0; endpoint < 2; endpoint++) {
            control0.push(...values.slice(i + 6, i + 9));
            control1.push(...values.slice(i + 9, i + 12));
            directions.push(values[i + 3] - values[i], values[i + 4] - values[i + 1], values[i + 5] - values[i + 2]);
          }
        }
      }
      const geometry = new three.BufferGeometry();
      geometry.setAttribute("position", new three.Float32BufferAttribute(positions, 3));
      if (group.conditional) {
        geometry.setAttribute("control0", new three.Float32BufferAttribute(control0, 3));
        geometry.setAttribute("control1", new three.Float32BufferAttribute(control1, 3));
        geometry.setAttribute("direction", new three.Float32BufferAttribute(directions, 3));
      }
      const options = { name: group.color, color: group.color.startsWith("#") ? group.color : "#333333", transparent: true, opacity: 0.55, depthWrite: false };
      const material = group.conditional ? new ConditionalLineMaterial(options) : new three.LineBasicMaterial(options);
      node.add(new three.LineSegments(geometry, material));
    }
    delete node.userData.ldrawLines;
  }
  template.userData.hasLDrawLines = nodes.length > 0;
  return template;
}

export function setLDrawPartHighlighted(three: ThreeModule, part: Group, highlighted: boolean) {
  part.traverse((object) => {
    if (object instanceof three.Mesh) {
      for (const material of Array.isArray(object.material) ? object.material : [object.material]) {
        if (material instanceof three.MeshStandardMaterial) {
          material.emissive.setHex(highlighted ? 0xffd85a : 0);
          material.emissiveIntensity = highlighted ? 0.35 : 0;
        }
      }
    } else if (object instanceof three.LineSegments) {
      for (const material of Array.isArray(object.material) ? object.material : [object.material]) {
        const line = material as Material & { color: Color };
        line.color.setHex(highlighted ? 0xffeb78 : line.userData.baseColor);
        line.opacity = highlighted ? 0.9 : line.userData.baseOpacity;
      }
    }
  });
}

export function instantiateLDrawPart(
  three: ThreeModule,
  template: Group,
  colorCode: number,
  highlighted = false,
): { part: Group; materials: Material[] } {
  const part = template.clone(true);
  const materials: Material[] = [];
  const meshes: InstanceType<ThreeModule["Mesh"]>[] = [];
  part.traverse((object) => {
    if (!(object instanceof three.Mesh) && !(object instanceof three.LineSegments)) return;
    const nextMaterials = (Array.isArray(object.material) ? object.material : [object.material]).map((source: Material) => {
      const material = source.clone();
      if (material instanceof three.MeshStandardMaterial) {
        if (source.name === "current") material.color.setHex(LDRAW_OFFICIAL_COLORS[colorCode] ?? LDRAW_OFFICIAL_COLORS[71]!);
        material.side = three.DoubleSide;
        material.roughness = 0.32;
      } else if (object instanceof three.LineSegments) {
        const line = material as Material & { color: Color };
        if (source.name === "current") line.color.setHex(LDRAW_OFFICIAL_COLORS[colorCode] ?? LDRAW_OFFICIAL_COLORS[71]!);
        // All six supported placement colors use #333333 edges in the pinned LDConfig.
        if (source.name === "edge-current") line.color.setHex(0x333333);
        line.userData.baseColor = line.color.getHex();
        line.userData.baseOpacity = line.opacity;
      }
      materials.push(material);
      return material;
    });
    object.material = Array.isArray(object.material) ? nextMaterials : nextMaterials[0]!;
    if (object instanceof three.Mesh) meshes.push(object);
  });
  // Historical catalog assets have no original lines. Keep their existing outlines.
  if (template.userData.hasLDrawLines !== true) {
    for (const mesh of meshes) {
      let geometry = edgeGeometryCache.get(mesh.geometry);
      if (!geometry) {
        geometry = new three.EdgesGeometry(mesh.geometry, 28);
        edgeGeometryCache.set(mesh.geometry, geometry);
      }
      const material = new three.LineBasicMaterial({ color: 0x15213a, transparent: true, opacity: 0.3, depthWrite: false });
      material.userData.baseColor = 0x15213a;
      material.userData.baseOpacity = 0.3;
      materials.push(material);
      mesh.add(new three.LineSegments(geometry, material));
    }
  }
  setLDrawPartHighlighted(three, part, highlighted);
  return { part, materials };
}

// Camera framing depends only on surfaces, never on line endpoints or controls.
export function ldrawMeshBounds(three: ThreeModule, assembly: Group) {
  assembly.updateWorldMatrix(true, true);
  const box = new three.Box3();
  assembly.traverse((object) => {
    if (!(object instanceof three.Mesh)) return;
    if (!object.geometry.boundingBox) object.geometry.computeBoundingBox();
    if (object.geometry.boundingBox) box.union(object.geometry.boundingBox.clone().applyMatrix4(object.matrixWorld));
  });
  return box;
}

export function fitLDrawAssembly(
  three: ThreeModule,
  camera: OrthographicCamera,
  assembly: Group,
  options: {
    minimumHorizontal?: number;
    minimumViewHeight?: number;
    verticalPadding?: number;
    horizontalPadding?: number;
  } = {},
): number {
  const box = ldrawMeshBounds(three, assembly);
  if (box.isEmpty()) return 240;

  const center = box.getCenter(new three.Vector3());
  const size = box.getSize(new three.Vector3());
  assembly.position.sub(center);
  const horizontal = Math.max(size.x, size.z, options.minimumHorizontal ?? 80);
  camera.position.set(0, Math.max(120, size.y * 1.15), Math.max(360, horizontal * 2.25));
  camera.lookAt(0, 0, 0);
  return Math.max(
    size.y * (options.verticalPadding ?? 1.5),
    horizontal * (options.horizontalPadding ?? 0.95),
    options.minimumViewHeight ?? 130,
  );
}

export function resizeLDrawStudio(
  renderer: WebGLRenderer,
  camera: OrthographicCamera,
  viewHeight: number,
  width: number,
  height: number,
) {
  const safeWidth = Math.max(1, width);
  const safeHeight = Math.max(1, height);
  const aspect = safeWidth / safeHeight;
  camera.left = -(viewHeight * aspect) / 2;
  camera.right = (viewHeight * aspect) / 2;
  camera.top = viewHeight / 2;
  camera.bottom = -viewHeight / 2;
  camera.updateProjectionMatrix();
  renderer.setSize(safeWidth, safeHeight, false);
}

export function disposeLDrawMaterials(materials: readonly Material[]) {
  for (const material of materials) material.dispose();
}
