import type { CircuitPlacement, CircuitProject } from "./schemas";

export function circuitPoint(p: CircuitPlacement, x: number, y: number) {
  switch (p.rotation) {
    case 90:
      [x, y] = [-y, x];
      break;
    case 180:
      [x, y] = [-x, -y];
      break;
    case 270:
      [x, y] = [y, -x];
      break;
    default:
      break;
  }
  return { x: p.x + x, y: p.y + y };
}
export function circuitMaterials(
  project: CircuitProject,
  inventory: Record<string, number>,
) {
  const counts: Record<string, number> = {};
  for (const p of project.placements)
    counts[p.part_id] = (counts[p.part_id] ?? 0) + 1;
  return Object.entries(counts).map(([partId, required]) => ({
    partId,
    required,
    available: inventory[partId] ?? 0,
    missing: Math.max(0, required - (inventory[partId] ?? 0)),
  }));
}
export function circuitText(
  text: { en: string; zh: string },
  language: string,
) {
  return language.startsWith("zh") ? text.zh : text.en;
}
export function circuitCoordinate(x: number, y: number) {
  return `${String.fromCharCode(65 + y)}${x + 1}`;
}
