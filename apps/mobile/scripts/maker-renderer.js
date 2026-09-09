/* global window, document, atob */
import * as THREE from "three";
import { GLTFLoader } from "three/addons/loaders/GLTFLoader.js";
import { OrbitControls } from "three/addons/controls/OrbitControls.js";
import { prepareLDrawTemplate, ldrawMeshBounds } from "./ldraw-lines.js";
const post = (type) =>
  window.ReactNativeWebView?.postMessage(JSON.stringify({ type }));
const colors = {
  1: 0x1e5aa8,
  2: 0x00852b,
  4: 0xb40000,
  14: 0xfac80a,
  15: 0xf4f4f4,
  71: 0x969696,
};
let scene,
  camera,
  renderer,
  controls,
  assembly,
  objects = [],
  initialized = false;
const render = () => renderer && renderer.render(scene, camera);
function resize() {
  if (!renderer) return;
  const w = window.innerWidth,
    h = window.innerHeight;
  renderer.setSize(w, h, false);
  camera.aspect = w / h;
  camera.updateProjectionMatrix();
  render();
}
function step(number, ids) {
  const hot = new Set(ids || []);
  for (const entry of objects) {
    entry.part.visible = number === undefined || entry.placement.step <= number;
    entry.part.traverse((obj) => {
      if (obj.isLineSegments) {
        for (const material of [].concat(obj.material)) {
          material.color.setHex(hot.has(entry.placement.id) ? 0xffeb78 : material.userData.baseColor);
          material.opacity = hot.has(entry.placement.id) ? 0.9 : 0.55;
        }
      }
      if (obj.isMesh)
        for (const material of [].concat(obj.material)) {
          if (material.emissive) {
            material.emissive.set(hot.has(entry.placement.id) ? 0xffc531 : 0);
            material.emissiveIntensity = 0.4;
          }
        }
    });
  }
  render();
}
async function load(value) {
  if (initialized) return;
  initialized = true;
  try {
    renderer = new THREE.WebGLRenderer({
      antialias: true,
      alpha: true,
      powerPreference: "low-power",
    });
    renderer.setPixelRatio(Math.min(window.devicePixelRatio, 1.5));
    renderer.outputColorSpace = THREE.SRGBColorSpace;
    renderer.toneMapping = THREE.ACESFilmicToneMapping;
    renderer.toneMappingExposure = 1.15;
    document.body.appendChild(renderer.domElement);
    renderer.domElement.addEventListener("webglcontextlost", (event) => {
      event.preventDefault();
      post("error");
    });
    scene = new THREE.Scene();
    camera = new THREE.PerspectiveCamera(35, 1, 0.1, 10000);
    scene.add(new THREE.HemisphereLight(0xfff3d2, 0x173866, 2.5));
    const key = new THREE.DirectionalLight(0xffffff, 3.1);
    key.position.set(-180, 260, 220);
    scene.add(key);
    const fill = new THREE.DirectionalLight(0x9bc8ff, 1.2);
    fill.position.set(220, 80, -160);
    scene.add(fill);
    assembly = new THREE.Group();
    scene.add(assembly);
    const loader = new GLTFLoader(),
      templates = {};
    for (const [id, b64] of Object.entries(value.assets)) {
      const raw = atob(b64),
        bytes = new Uint8Array(raw.length);
      for (let i = 0; i < raw.length; i++) bytes[i] = raw.charCodeAt(i);
      // GLBs contain embedded assets; CSP blocks every network request.
      templates[id] = prepareLDrawTemplate((await loader.parseAsync(bytes.buffer, "")).scene);
    }
    for (const placement of value.plan.placements) {
      const spec = value.plan.parts[placement.part_id];
      const part = templates[spec.ldraw_id].clone(true);
      part.traverse((obj) => {
        if (obj.isLineSegments) {
          const materials = [].concat(obj.material).map((source) => {
            const material = source.clone();
            if (source.name === "current") material.color.setHex(colors[placement.color] ?? colors[71]);
            if (source.name === "edge-current") material.color.setHex(0x333333);
            material.userData.baseColor = material.color.getHex();
            return material;
          });
          obj.material = Array.isArray(obj.material) ? materials : materials[0];
          return;
        }
        if (!obj.isMesh) return;
        const materials = [].concat(obj.material).map((source) => {
          const material =
            source.name.toLowerCase() === "current"
              ? new THREE.MeshStandardMaterial({
                  color: colors[placement.color] ?? colors[71],
                  roughness: 0.32,
                  side: THREE.DoubleSide,
                })
              : source.clone();
          return material;
        });
        obj.material = Array.isArray(obj.material) ? materials : materials[0];
      });
      const rotated = Math.abs(placement.rotation % 180) === 90;
      part.position.set(
        placement.x * 20 + (rotated ? spec.studs_z : spec.studs_x) * 10,
        (placement.y + spec.plates_y) * 8 - (spec.origin_y_offset_ldu ?? 0),
        placement.z * 20 +
          (rotated ? spec.studs_x : spec.studs_z) * 10 +
          (spec.origin_center_z_offset_ldu ?? 0),
      );
      part.rotation.y = -THREE.MathUtils.degToRad(placement.rotation);
      part.scale.y = -1;
      assembly.add(part);
      objects.push({ part, placement });
    }
    const box = ldrawMeshBounds(assembly),
      center = box.getCenter(new THREE.Vector3()),
      size = box.getSize(new THREE.Vector3());
    assembly.position.sub(center);
    const distance = Math.max(80, size.length()) * 1.7;
    camera.position.set(distance * 0.7, distance * 0.65, distance);
    camera.lookAt(0, 0, 0);
    controls = new OrbitControls(camera, renderer.domElement);
    controls.enableDamping = false;
    controls.minDistance = distance * 0.25;
    controls.maxDistance = distance * 4;
    controls.enablePan = false;
    controls.addEventListener("change", render);
    controls.saveState();
    resize();
    step(value.step, value.highlight);
    post("loaded");
  } catch {
    post("error");
  }
}
function receive(event) {
  try {
    const value = JSON.parse(event.data);
    if (value.type === "model") void load(value);
    if (value.type === "step") step(value.step, value.highlight);
    if (value.type === "reset") {
      controls?.reset();
      render();
    }
    if (value.type === "zoom" && controls) {
      camera.position.multiplyScalar(value.factor);
      controls.update();
      render();
    }
  } catch {
    post("error");
  }
}
window.addEventListener("message", receive);
document.addEventListener("message", receive);
window.addEventListener("resize", resize);
post("ready");
