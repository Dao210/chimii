import * as THREE from "three";
import { LDrawConditionalLineMaterial } from "three/addons/materials/LDrawConditionalLineMaterial.js";

// Mirrors prepareLDrawTemplate in the web/desktop ldraw-render-preset.
// Mobile owns its renderer; both consume the same GLB line data and transforms.
export function prepareLDrawTemplate(template) {
  const nodes = [];
  template.traverse((node) => {
    if (node.userData.ldrawLines != null) nodes.push(node);
  });
  for (const node of nodes) {
    const data = node.userData.ldrawLines;
    if (data.version !== 1 || !Array.isArray(data.groups)) throw new Error("Unsupported LDraw line data");
    for (const group of data.groups) {
      const stride = group.conditional === true ? 12 : 6;
      const values = group.vertices;
      if (typeof group.color !== "string" || typeof group.conditional !== "boolean" || !Array.isArray(values) || values.length % stride !== 0 || !values.every((v) => typeof v === "number" && Number.isFinite(v))) throw new Error("Invalid LDraw line data");
      if (values.length === 0) continue;
      const positions = [], control0 = [], control1 = [], directions = [];
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
      const geometry = new THREE.BufferGeometry();
      geometry.setAttribute("position", new THREE.Float32BufferAttribute(positions, 3));
      if (group.conditional) {
        geometry.setAttribute("control0", new THREE.Float32BufferAttribute(control0, 3));
        geometry.setAttribute("control1", new THREE.Float32BufferAttribute(control1, 3));
        geometry.setAttribute("direction", new THREE.Float32BufferAttribute(directions, 3));
      }
      const options = { name: group.color, color: group.color.startsWith("#") ? group.color : "#333333", transparent: true, opacity: 0.55, depthWrite: false };
      const material = group.conditional ? new LDrawConditionalLineMaterial(options) : new THREE.LineBasicMaterial(options);
      node.add(new THREE.LineSegments(geometry, material));
    }
    delete node.userData.ldrawLines;
  }
  return template;
}

export function ldrawMeshBounds(assembly) {
  assembly.updateWorldMatrix(true, true);
  const box = new THREE.Box3();
  assembly.traverse((object) => {
    if (!object.isMesh) return;
    if (!object.geometry.boundingBox) object.geometry.computeBoundingBox();
    if (object.geometry.boundingBox) box.union(object.geometry.boundingBox.clone().applyMatrix4(object.matrixWorld));
  });
  return box;
}
