import { z } from "zod";

const text = z.looseObject({ en: z.string().min(1), zh: z.string().min(1) });
const sourceURL = z.url().refine((value) => {
  try {
    const url = new URL(value);
    return url.protocol === "https:" && !url.username && !url.password;
  } catch {
    return false;
  }
});
const quantity = z.number().int().min(1).max(128);

// Research records deliberately cannot parse as a CircuitDocument or kit.
// Counts describe the source illustrations, never verified family inventory.
export const AssemblyReferenceSchema = z
  .looseObject({
    id: z.string().min(1),
    version: z.string().min(1),
    status: z.literal("research"),
    kit_id: z.string().min(1),
    kit_name: z.string().min(1),
    title: text,
    description: text,
    checked_on: z.iso.date(),
    source_url: sourceURL,
    product_url: sourceURL,
    parts_source_url: sourceURL,
    program_url: sourceURL,
    content_hash: z.string().regex(/^[a-f0-9]{64}$/),
    parts: z.array(z.looseObject({
      id: z.string().min(1),
      name: text,
      pictured_quantity: quantity,
      kind: z.enum(["structure", "electronics"]),
    })).min(1).max(128),
    steps: z.array(z.looseObject({
      number: quantity,
      title: text,
      source_url: sourceURL,
      introduced_parts: z.record(z.string(), quantity),
    })).min(1).max(128),
    connections: z.array(z.looseObject({
      part_id: z.string().min(1),
      controller_id: z.string().min(1),
      port: z.string().min(1),
    })).max(128),
    unresolved: z.array(z.looseObject({ id: z.string().min(1), detail: text })).min(1),
  })
  .refine((ref) => {
    const parts = new Map(ref.parts.map((p) => [p.id, p]));
    if (parts.size !== ref.parts.length) return false;
    const totals = new Map<string, number>();
    for (const [index, step] of ref.steps.entries()) {
      if (step.number !== index + 1 || !Object.keys(step.introduced_parts).length)
        return false;
      for (const [id, count] of Object.entries(step.introduced_parts)) {
        if (!parts.has(id)) return false;
        totals.set(id, (totals.get(id) ?? 0) + count);
      }
    }
    if (ref.parts.some((p) => totals.get(p.id) !== p.pictured_quantity)) return false;
    const ports = new Set<string>();
    for (const wire of ref.connections) {
      const port = `${wire.controller_id}:${wire.port}`;
      if (
        parts.get(wire.part_id)?.kind !== "electronics" ||
        parts.get(wire.controller_id)?.kind !== "electronics" ||
        wire.part_id === wire.controller_id || ports.has(port)
      ) return false;
      ports.add(port);
    }
    return new Set(ref.unresolved.map((gap) => gap.id)).size === ref.unresolved.length;
  }, "Inconsistent assembly reference");

export type AssemblyReference = z.infer<typeof AssemblyReferenceSchema>;

// Optional reference data must not break supported Snap/BOSON kits when an
// older server omits it or a newer server sends an unsupported reference shape.
export const AssemblyReferencesSchema = z.array(AssemblyReferenceSchema)
  .refine((refs) => new Set(refs.map((r) => r.id)).size === refs.length)
  .catch([]);
