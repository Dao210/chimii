import type { BuildSummary } from "@chimii/core/build/schemas";
import type { CircuitList } from "@chimii/core/circuit/schemas";

export type CreationKind = "build" | "circuit";
export interface CreationSummary {
  id: string;
  title: string;
  description: string;
  progress: string;
  createdAt: string;
}
// Preserve server ordering and visibility; Circuit stores a zero-based step.
export function creationSummaries(builds: BuildSummary[] = [], circuits: CircuitList["creations"] = []): CreationSummary[] {
  return [
    ...builds.map(v => ({ id: v.id, title: v.title, description: `${v.part_count} parts · ${v.step_count} steps`,
      progress: v.progress.completed_at ? "Completed" : v.progress.current_step > 0 ? `Saved at step ${v.progress.current_step}` : "Not started", createdAt: v.created_at })),
    ...circuits.map(v => ({ id: v.id, title: v.title, description: "Circuit creation",
      progress: `Saved at step ${v.current_step + 1} · ${v.observation === "worked" ? "Worked" : v.observation === "needs_help" ? "Needs help" : v.observation === "not_tried" ? "Not tested" : "Unknown result"}`, createdAt: v.created_at })),
  ];
}
export function creationWebURL(origin: string | undefined, slug: string, kind: CreationKind, id: string): string | null {
  if (!origin || !slug || !id) return null;
  try {
    const url = new URL(origin);
    if (!["https:", "http:"].includes(url.protocol) || url.username || url.password) return null;
    url.pathname = `/${encodeURIComponent(slug)}/${kind === "build" ? "creations" : "circuit"}/${encodeURIComponent(id)}`;
    url.search = ""; url.hash = "";
    return url.toString();
  } catch { return null; }
}
