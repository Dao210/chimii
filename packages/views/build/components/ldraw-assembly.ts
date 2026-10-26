import type { BufferGeometry, Group, InstancedMesh, LineSegments, Material, Matrix4 } from "three";
import type { BuildPartSpec, BuildPlacement } from "@chimii/core/build";
import { instantiateLDrawPart } from "./ldraw-render-preset";

type ThreeModule = typeof import("three");
interface LoadedPart { placement: BuildPlacement; spec: BuildPartSpec; template: Group }
interface SurfaceSlot { normal: InstancedMesh; active: InstancedMesh; index: number; matrix: Matrix4 }
interface LineSlot { line: LineSegments; normal: Material | Material[]; active: Material | Material[] }
export interface BatchedLDrawAssembly {
  group: Group;
  materials: Material[];
  geometries: BufferGeometry[];
  batches: InstancedMesh[];
  instances: { placement: BuildPlacement; lines: Group; surfaces: SurfaceSlot[]; lineSlots: LineSlot[] }[];
}

export function placeLDrawPart(three: ThreeModule, part: Group, placement: BuildPlacement, spec: BuildPartSpec) {
  const rotated = Math.abs(placement.rotation % 180) === 90;
  part.position.set(placement.x * 20 + (rotated ? spec.studs_z : spec.studs_x) * 10,
    (placement.y + spec.plates_y) * 8 - (spec.origin_y_offset_ldu ?? 0),
    placement.z * 20 + (rotated ? spec.studs_x : spec.studs_z) * 10 + (spec.origin_center_z_offset_ldu ?? 0));
  part.rotation.y = -three.MathUtils.degToRad(placement.rotation);
  part.scale.y = -1;
}

// Surface geometry and both material states are shared by part/color. Original
// type-2/type-5 lines keep their per-placement matrices and conditional shader.
export function batchLDrawAssembly(three: ThreeModule, loaded: LoadedPart[]): BatchedLDrawAssembly {
  const result: BatchedLDrawAssembly = { group: new three.Group(), materials: [], geometries: [], batches: [], instances: [] };
  const groups = new Map<string, LoadedPart[]>();
  for (const part of loaded) {
    const key = `${part.template.uuid}:${part.placement.color}`;
    const items = groups.get(key) ?? [];
    items.push(part);
    groups.set(key, items);
  }
  for (const items of groups.values()) {
    const first = items[0]!;
    const normal = instantiateLDrawPart(three, first.template, first.placement.color);
    const active = instantiateLDrawPart(three, first.template, first.placement.color, true);
    result.materials.push(...normal.materials, ...active.materials);
    normal.part.updateWorldMatrix(true, true);
    active.part.updateWorldMatrix(true, true);
    const meshes: InstanceType<ThreeModule["Mesh"]>[] = [];
    const activeMeshes: InstanceType<ThreeModule["Mesh"]>[] = [];
    const lines: LineSegments[] = [], activeLines: LineSegments[] = [];
    normal.part.traverse(node => { if (node instanceof three.Mesh) meshes.push(node); else if (node instanceof three.LineSegments) lines.push(node); });
    active.part.traverse(node => { if (node instanceof three.Mesh) activeMeshes.push(node); else if (node instanceof three.LineSegments) activeLines.push(node); });
    const pairs = meshes.map((mesh, i) => {
      // InstancedMesh disallows negative scale; bake the LDraw Y reflection
      // once into geometry, retaining a positive instance transformation.
      const matrix = new three.Matrix4().makeScale(1, -1, 1).multiply(mesh.matrixWorld);
      const geometry = mesh.geometry.clone().applyMatrix4(matrix);
      if (matrix.determinant() < 0) {
        if (!geometry.index) geometry.setIndex(Array.from({ length: geometry.getAttribute("position").count }, (_, n) => n));
        const indices = geometry.index!;
        for (let triangle = 0; triangle < indices.count; triangle += 3) {
          const second = indices.getX(triangle + 1);
          indices.setX(triangle + 1, indices.getX(triangle + 2));
          indices.setX(triangle + 2, second);
        }
      }
      result.geometries.push(geometry);
      const pair = { normal: new three.InstancedMesh(geometry, mesh.material, items.length), active: new three.InstancedMesh(geometry, activeMeshes[i]!.material, items.length) };
      pair.active.userData.excludeFromFraming = true;
      pair.active.userData.excludeFromFraming = true;
      for (const batch of [pair.normal, pair.active]) {
        batch.instanceMatrix.setUsage(three.DynamicDrawUsage);
        batch.frustumCulled = false;
        result.group.add(batch);
        result.batches.push(batch);
      }
      return pair;
    });
    for (const [index, item] of items.entries()) {
      const lineGroup = new three.Group();
      placeLDrawPart(three, lineGroup, item.placement, item.spec);
      const lineSlots = lines.map((source, i) => {
        const line = source.clone(false);
        line.matrix.copy(source.matrixWorld);
        line.matrixAutoUpdate = false;
        lineGroup.add(line);
        return { line, normal: source.material, active: activeLines[i]!.material };
      });
      result.group.add(lineGroup);
      const transform = new three.Group();
      placeLDrawPart(three, transform, item.placement, item.spec);
      transform.scale.y = 1;
      transform.updateMatrix();
      const surfaces = pairs.map(pair => {
        pair.normal.setMatrixAt(index, transform.matrix);
        pair.active.setMatrixAt(index, new three.Matrix4().makeScale(0, 0, 0));
        return { ...pair, index, matrix: transform.matrix.clone() };
      });
      result.instances.push({ placement: item.placement, lines: lineGroup, surfaces, lineSlots });
    }
    for (const pair of pairs) pair.normal.computeBoundingBox();
  }
  return result;
}

export function updateBatchedLDraw(three: ThreeModule, assembly: BatchedLDrawAssembly, maxStep: number | undefined, highlighted: ReadonlySet<string>) {
  const hidden = new three.Matrix4().makeScale(0, 0, 0);
  for (const instance of assembly.instances) {
    const visible = maxStep == null || instance.placement.step <= maxStep;
    const selected = highlighted.has(instance.placement.id);
    instance.lines.visible = visible;
    for (const slot of instance.lineSlots) slot.line.material = selected ? slot.active : slot.normal;
    for (const slot of instance.surfaces) {
      slot.normal.setMatrixAt(slot.index, visible && !selected ? slot.matrix : hidden);
      slot.active.setMatrixAt(slot.index, visible && selected ? slot.matrix : hidden);
    }
  }
  for (const batch of assembly.batches) batch.instanceMatrix.needsUpdate = true;
}

export function disposeBatchedLDraw(assembly: BatchedLDrawAssembly | undefined) {
  if (!assembly) return;
  for (const batch of assembly.batches) batch.dispose();
  for (const geometry of assembly.geometries) geometry.dispose();
}
