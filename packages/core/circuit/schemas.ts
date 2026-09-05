import { z } from "zod";
import { parseWithFallback } from "../api/schema";

const text = z.looseObject({ en: z.string(), zh: z.string() });
const point = z.looseObject({ x: z.number().int(), y: z.number().int() });
export const CircuitPartSchema = z.looseObject({
  id: z.string().min(1),
  manufacturer_id: z.string(),
  name: text,
  purpose: text,
  kind: z.string(),
  ports: z.array(point.extend({ id: z.string() })).min(1),
  body: z.array(point).min(1),
  quantity: z.number().int().nonnegative(),
  conductive: z.array(z.array(z.string()).min(1)),
});
export const CircuitPlacementSchema = z.looseObject({
  id: z.string().min(1),
  part_id: z.string(),
  x: z.number().int(),
  y: z.number().int(),
  rotation: z.union([
    z.literal(0),
    z.literal(90),
    z.literal(180),
    z.literal(270),
  ]),
  layer: z.number().int().min(1).max(3),
});
export const CircuitProjectSchema = z
  .looseObject({
    id: z.string().min(1),
    title: text,
    description: text,
    explanation: text,
    test_instruction: text,
    troubleshooting: z.array(text),
    source: z.looseObject({
      url: z.url().startsWith("https://"),
      title: z.string(),
      page: z.number().int().positive(),
      sha256: z.string().length(64),
    }),
    placements: z.array(CircuitPlacementSchema).min(1).max(128),
    steps: z
      .array(
        z.looseObject({
          id: z.string(),
          title: text,
          instruction: text,
          placement_ids: z.array(z.string()).min(1),
        }),
      )
      .min(1)
      .max(128),
    expected_nets: z.array(z.array(z.string())),
  })
  .refine((p) => {
    const placements = new Set(p.placements.map((v) => v.id));
    const steps = new Set(p.steps.map((v) => v.id));
    const used = p.steps.flatMap((v) => v.placement_ids);
    return (
      placements.size === p.placements.length &&
      steps.size === p.steps.length &&
      new Set(used).size === used.length &&
      used.length === placements.size &&
      used.every((id) => placements.has(id))
    );
  }, "Steps must reference every placed component exactly once");
export const CircuitCatalogSchema = z
  .looseObject({
    catalog: z.looseObject({
      version: z.string(),
      kit_id: z.string(),
      name: z.string(),
      columns: z.number().int().min(1).max(30),
      rows: z.number().int().min(1).max(30),
      minimum_age: z.number().int(),
      parts: z.array(CircuitPartSchema).min(1),
      projects: z.array(CircuitProjectSchema).min(1),
      preparation: z.array(text),
    }),
    ai_available: z.boolean().catch(false),
  })
  .refine(
    (v) =>
      v.catalog.projects.every((p) =>
        p.placements.every((placement) =>
          v.catalog.parts.some((part) => part.id === placement.part_id),
        ),
      ),
    "Unknown catalogue component",
  );
export const CircuitDocumentSchema = z
  .looseObject({
    version: z.literal(1),
    catalog_version: z.string(),
    kit_id: z.string(),
    prompt: z.string(),
    title: z.string(),
    planner: z.string(),
    project: CircuitProjectSchema,
    parts: z.array(CircuitPartSchema).min(1),
    preparation: z.array(text),
    columns: z.number().int().min(1).max(30),
    rows: z.number().int().min(1).max(30),
    inventory: z.record(z.string(), z.number().int().nonnegative()),
    content_hash: z.string().length(64),
    validation: z.looseObject({
      passed: z.boolean(),
      physical_verification: z.string(),
      used_parts: z.record(z.string(), z.number().int().positive()),
      issues: z.array(
        z.looseObject({
          code: z.string(),
          placement_id: z.string().optional(),
          part_id: z.string().optional(),
          required: z.number().optional(),
          available: z.number().optional(),
        }),
      ),
      nets: z.array(z.array(z.string())),
    }),
  })
  .superRefine((doc, ctx) => {
    const parts = new Map(doc.parts.map((p) => [p.id, p]));
    const counts: Record<string, number> = {};
    let valid = parts.size === doc.parts.length;
    for (const p of doc.project.placements) {
      valid &&= parts.has(p.part_id);
      counts[p.part_id] = (counts[p.part_id] ?? 0) + 1;
    }
    if (!valid)
      ctx.addIssue({
        code: "custom",
        message: "Unknown or duplicated component",
      });
    if (
      doc.validation.passed === true &&
      (doc.validation.issues.length !== 0 ||
        Object.keys(counts).length !==
          Object.keys(doc.validation.used_parts).length ||
        Object.entries(counts).some(
          ([id, count]) =>
            doc.validation.used_parts[id] !== count ||
            (doc.inventory[id] ?? 0) < count,
        ))
    ) {
      ctx.addIssue({
        code: "custom",
        message: "Inconsistent verification evidence",
      });
    }
  });
export const CircuitCreationSchema = z
  .looseObject({
    id: z.string().min(1),
    document: CircuitDocumentSchema,
    current_step: z.number().int().nonnegative(),
    observation: z.string(),
    progress_revision: z.number().int().nonnegative(),
    created_at: z.string(),
  })
  .refine(
    (v) => v.current_step < v.document.project.steps.length,
    "Invalid saved step",
  );
export const CircuitListSchema = z.looseObject({
  creations: z.array(
    z.looseObject({
      id: z.string().min(1),
      title: z.string(),
      project_id: z.string(),
      observation: z.string(),
      current_step: z.number().int(),
      created_at: z.string(),
    }),
  ),
});

// A malformed circuit must show a recoverable error, never a successful empty
// design. Unknown non-critical fields remain compatible with newer servers.
export function parseCircuit<T>(
  raw: unknown,
  schema: z.ZodType<T>,
  endpoint: string,
): T {
  const parsed = parseWithFallback<T | null>(raw, schema, null, { endpoint });
  if (parsed === null) throw new Error("Invalid circuit response");
  return parsed;
}

export type CircuitCatalogResponse = z.infer<typeof CircuitCatalogSchema>;
export type CircuitCatalog = CircuitCatalogResponse["catalog"];
export type CircuitPart = z.infer<typeof CircuitPartSchema>;
export type CircuitPlacement = z.infer<typeof CircuitPlacementSchema>;
export type CircuitProject = z.infer<typeof CircuitProjectSchema>;
export type CircuitDocument = z.infer<typeof CircuitDocumentSchema>;
export type CircuitCreation = z.infer<typeof CircuitCreationSchema>;
export type CircuitList = z.infer<typeof CircuitListSchema>;

export interface CreateCircuitInput {
  client_request_id: string;
  kit_id: string;
  catalog_version: string;
  project_id?: string;
  prompt?: string;
  locale: string;
  inventory: Record<string, number>;
}
export interface CircuitProgressInput {
  current_step: number;
  observation: "not_tried" | "worked" | "needs_help";
  expected_revision: number;
}
