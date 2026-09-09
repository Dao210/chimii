import * as three from "three";
import { GLTFLoader } from "three/examples/jsm/loaders/GLTFLoader.js";
import { LDrawConditionalLineMaterial } from "three/addons/materials/LDrawConditionalLineMaterial.js";
import type { Group, Material } from "three";
import {
  LDRAW_CATALOG,
  LDRAW_CATALOG_VERSION,
} from "../../packages/views/build/catalog/catalog.generated";
import {
  configureLDrawRenderer,
  createLDrawStudio,
  disposeLDrawMaterials,
  fitLDrawAssembly,
  instantiateLDrawPart,
  prepareLDrawTemplate,
  LDRAW_DEFAULT_YAW,
  LDRAW_OFFICIAL_COLORS,
  LDRAW_RENDER_PRESET_VERSION,
  resizeLDrawStudio,
} from "../../packages/views/build/components/ldraw-render-preset";

const THUMBNAIL_WIDTH = 640;
const THUMBNAIL_HEIGHT = 400;
const canvas = document.querySelector<HTMLCanvasElement>("#thumbnail");
if (!canvas) throw new Error("Thumbnail canvas is missing");

const renderer = new three.WebGLRenderer({
  canvas,
  antialias: true,
  alpha: true,
  powerPreference: "high-performance",
  preserveDrawingBuffer: true,
});
renderer.setPixelRatio(1);
configureLDrawRenderer(three, renderer);
const { scene, camera, modelRoot } = createLDrawStudio(three, LDRAW_DEFAULT_YAW);
const loader = new GLTFLoader();
const templateCache = new Map<string, Promise<Group>>();
let activeMaterials: Material[] = [];

function decodeBase64(value: string): ArrayBuffer {
  const binary = window.atob(value);
  const bytes = new Uint8Array(binary.length);
  for (let index = 0; index < binary.length; index += 1) {
    bytes[index] = binary.charCodeAt(index);
  }
  return bytes.buffer;
}

function loadTemplate(ldrawID: string): Promise<Group> {
  const normalizedID = ldrawID.toLowerCase();
  const cached = templateCache.get(normalizedID);
  if (cached) return cached;
  const asset = LDRAW_CATALOG[normalizedID];
  if (!asset) return Promise.reject(new Error(`LDraw catalog does not contain ${normalizedID}`));
  const pending = loader.parseAsync(decodeBase64(asset.glbBase64), "").then((model) => prepareLDrawTemplate(three, model.scene, LDrawConditionalLineMaterial));
  templateCache.set(normalizedID, pending);
  return pending;
}

async function renderThumbnail(ldrawID: string, colorCode: number) {
  if (!(colorCode in LDRAW_OFFICIAL_COLORS)) throw new Error(`Unsupported LDraw color ${colorCode}`);
  const template = await loadTemplate(ldrawID);
  const assembly = new three.Group();
  const { part, materials } = instantiateLDrawPart(three, template, colorCode);
  part.scale.y = -1;
  assembly.add(part);
  const viewHeight = fitLDrawAssembly(three, camera, assembly, {
    minimumHorizontal: 0,
    minimumViewHeight: 34,
    verticalPadding: 1.5,
    horizontalPadding: 0.95,
  });

  disposeLDrawMaterials(activeMaterials);
  activeMaterials = materials;
  modelRoot.clear();
  modelRoot.add(assembly);
  modelRoot.rotation.y = LDRAW_DEFAULT_YAW;
  resizeLDrawStudio(renderer, camera, viewHeight, THUMBNAIL_WIDTH, THUMBNAIL_HEIGHT);
  renderer.clear();
  renderer.render(scene, camera);

  return canvas.toDataURL("image/webp", 0.92);
}

window.ldrawThumbnailRenderer = {
  metadata: {
    catalogVersion: LDRAW_CATALOG_VERSION,
    renderPresetVersion: LDRAW_RENDER_PRESET_VERSION,
    width: THUMBNAIL_WIDTH,
    height: THUMBNAIL_HEIGHT,
    colors: Object.keys(LDRAW_OFFICIAL_COLORS).map(Number).sort((a, b) => a - b),
    parts: Object.entries(LDRAW_CATALOG)
      .map(([ldrawID, asset]) => ({ ldrawID, assetHash: asset.hash }))
      .sort((a, b) => a.ldrawID.localeCompare(b.ldrawID)),
  },
  renderThumbnail,
};

declare global {
  interface Window {
    ldrawThumbnailRenderer: {
      metadata: {
        catalogVersion: string;
        renderPresetVersion: string;
        width: number;
        height: number;
        colors: number[];
        parts: Array<{ ldrawID: string; assetHash: string }>;
      };
      renderThumbnail: (ldrawID: string, colorCode: number) => Promise<string>;
    };
  }
}
