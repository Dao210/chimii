// Test-only adapter for the server-owned catalogue; never bundled by the UI.
import catalogData from "../../../server/internal/circuit/catalog.json";
import {
  CircuitCatalogSchema,
  CircuitCreationSchema,
} from "@chimii/core/circuit";

export function circuitFixture(projectId = "switch-light") {
  const catalog = CircuitCatalogSchema.parse({
    catalog: catalogData,
    ai_available: false,
  });
  const project = catalog.catalog.projects.find((p) => p.id === projectId)!;
  const used = project.placements.reduce<Record<string, number>>(
    (counts, p) => {
      counts[p.part_id] = (counts[p.part_id] ?? 0) + 1;
      return counts;
    },
    {},
  );
  const creation = CircuitCreationSchema.parse({
    id: "circuit-1",
    current_step: 0,
    observation: "not_tried",
    progress_revision: 0,
    created_at: "2026-09-05T00:00:00Z",
    document: {
      version: 1,
      catalog_version: catalog.catalog.version,
      kit_id: catalog.catalog.kit_id,
      prompt: "",
      title: project.title.en,
      planner: "project",
      project,
      parts: catalog.catalog.parts,
      preparation: catalog.catalog.preparation,
      columns: catalog.catalog.columns,
      rows: catalog.catalog.rows,
      inventory: Object.fromEntries(
        catalog.catalog.parts.map((p) => [p.id, p.quantity]),
      ),
      content_hash: "a".repeat(64),
      validation: {
        passed: true,
        physical_verification: "not_tested",
        used_parts: used,
        issues: [],
        nets: project.expected_nets,
      },
    },
  });
  return { catalog, creation };
}
