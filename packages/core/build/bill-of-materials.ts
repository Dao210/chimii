import type { BuildPlan, BrickInventory } from "./types";

export interface BuildMaterialRow {
  partId: string;
  ldrawId: string;
  name: string;
  color: number;
  quantity: number;
  missing?: number;
}

// The rendered placements and exported model are the same BOM source.
export function buildBillOfMaterials(plan: BuildPlan, inventory?: BrickInventory): BuildMaterialRow[] {
  const counts = new Map<string, BuildMaterialRow>();
  const missing = new Map((plan.missing_parts ?? []).map(item => [`${item.part_id}:${item.color}`, item.quantity]));
  const available = new Map(inventory?.items.map(item => [`${item.part_id}:${item.color}`, item.quantity]) ?? []);
  const compared = inventory ? inventory.configured === true : plan.inventory_compared === true;
  for (const placement of plan.placements) {
    const key = `${placement.part_id}:${placement.color}`;
    const row = counts.get(key);
    if (row) row.quantity += 1;
    else {
      const part = plan.parts[placement.part_id];
      counts.set(key, { partId: placement.part_id, ldrawId: part?.ldraw_id ?? "", name: part?.name || placement.part_id, color: placement.color, quantity: 1, missing: compared ? missing.get(key) ?? 0 : undefined });
    }
  }
  const rows = [...counts.values()];
  if (inventory && compared) for (const row of rows) row.missing = Math.max(0, row.quantity - (available.get(`${row.partId}:${row.color}`) ?? 0));
  return rows.sort((a, b) => a.partId.localeCompare(b.partId) || a.color - b.color);
}

export function buildMaterialsCSV(rows: BuildMaterialRow[]): string {
  const cell = (value: string | number | undefined) => {
    const raw = String(value ?? "");
    const safe = /^[=+@\-\t\r]/.test(raw) ? `'${raw}` : raw;
    return `"${safe.replaceAll('"', '""')}"`;
  };
  return "\uFEFF" + [["Part ID", "LDraw ID", "Name", "LDraw color", "Quantity", "Missing"], ...rows.map(row => [row.partId, row.ldrawId, row.name, row.color, row.quantity, row.missing])].map(row => row.map(cell).join(",")).join("\r\n");
}
