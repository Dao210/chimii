import type {
  BufferGeometry,
  Group,
  Material,
  OrthographicCamera,
  Scene,
  WebGLRenderer,
} from "three";

type ThreeModule = typeof import("three");

export const LDRAW_RENDER_PRESET_VERSION = "ldraw-studio-v1";
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

function colorMaterial(
  three: ThreeModule,
  source: Material,
  colorCode: number,
  highlighted: boolean,
): Material {
  const sourceName = source.name.toLowerCase();
  const material = sourceName === "current"
    ? new three.MeshStandardMaterial({
        color: LDRAW_OFFICIAL_COLORS[colorCode] ?? LDRAW_OFFICIAL_COLORS[71],
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

export function instantiateLDrawPart(
  three: ThreeModule,
  template: Group,
  colorCode: number,
  highlighted = false,
): { part: Group; materials: Material[] } {
  const part = template.clone(true);
  const materials: Material[] = [];

  part.traverse((object) => {
    if (!(object instanceof three.Mesh)) return;
    const sourceMaterials = Array.isArray(object.material) ? object.material : [object.material];
    const nextMaterials = sourceMaterials.map((source) => {
      const material = colorMaterial(three, source, colorCode, highlighted);
      materials.push(material);
      return material;
    });
    object.material = Array.isArray(object.material) ? nextMaterials : nextMaterials[0]!;

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
    materials.push(edgeMaterial);
    object.add(new three.LineSegments(edgeGeometry, edgeMaterial));
  });

  return { part, materials };
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
  const box = new three.Box3().setFromObject(assembly);
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
