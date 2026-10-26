"use client";

import { useMemo } from "react";
import { Check, Download, ShieldCheck } from "lucide-react";
import { brickInventoryOptions, buildBillOfMaterials, buildMaterialsCSV, buildOccupancySize, type BuildCreation } from "@chimii/core/build";
import { useQuery } from "@tanstack/react-query";
import { useWorkspaceId } from "@chimii/core/hooks";
import { Button } from "@chimii/ui/components/ui/button";
import { useT } from "../../i18n";
import { LDRAW_OFFICIAL_COLORS } from "./ldraw-render-preset";

export function BuildSceneDetails({ creation }: { creation: BuildCreation }) {
  const { t } = useT("build");
  const plan = creation.build_plan;
  const inventoryQuery = useQuery(brickInventoryOptions(useWorkspaceId()));
  const inventory = inventoryQuery.data;
  const compared = inventory ? inventory.configured === true : plan.inventory_compared === true;
  const scene = plan.document?.scene;
  const evidence = plan.validation.evidence;
  const revised = Number(creation.recipe.metadata?.["scene_repairs"] ?? 0) > 0;
  const rows = useMemo(() => buildBillOfMaterials(plan, inventory), [plan, inventory]);
  const size = useMemo(() => buildOccupancySize(plan), [plan]);
  const missingCount = rows.reduce((total, row) => total + (row.missing ?? 0), 0);
  const download = () => {
    const url = URL.createObjectURL(new Blob([buildMaterialsCSV(rows)], { type: "text/csv;charset=utf-8" }));
    const anchor = document.createElement("a");
    anchor.href = url;
    anchor.download = `${creation.title || "chimii"}-parts.csv`;
    anchor.click();
    URL.revokeObjectURL(url);
  };
  return <section className="col-span-full grid gap-4 text-foreground">
    {scene && <details className="rounded-2xl border border-border bg-card p-5" open>
      <summary className="cursor-pointer font-bold">{t($ => $.scene_design)}</summary>
      {size && <p className="mt-3 text-sm font-medium">{t($ => $.scene_dimensions, size)}</p>}
      <p className="mt-3 text-sm leading-6 text-muted-foreground">{revised ? t($ => $.scene_revised_description) : scene.brief}</p>
      <div className="mt-4 grid gap-3 sm:grid-cols-2">
        {scene.objects.map(object => <div key={object.id} className="rounded-xl bg-muted p-4">
          <h3 className="text-sm font-bold">{object.label}</h3>
          <p className="mt-2 text-sm leading-6 text-muted-foreground">{object.modules.map(module => module.label).join(t($ => $.result_separator))}</p>
        </div>)}
      </div>
    </details>}
    {evidence && <div className="rounded-2xl border border-border bg-card p-5">
      <h3 className="flex items-center gap-2 font-bold"><ShieldCheck className="size-4" />{t($ => $.scene_checks)}</h3>
      <p className="mt-3 text-sm leading-6">{t($ => $.scene_checks_passed)}</p>
      <p className="mt-2 text-sm text-muted-foreground">{evidence.semantic === "model_reviewed_not_proven" ? t($ => $.scene_reviewed) : t($ => $.scene_unreviewed)}</p>
      {evidence.visual && <p className="mt-2 text-sm text-muted-foreground">{t($ => $.scene_visual_reviewed)}</p>}
      {!!evidence.interfaces?.length && <p className="mt-2 text-sm text-muted-foreground">{t($ => $.scene_interfaces_checked, { count: evidence.interfaces.length })}</p>}
      {evidence.load_estimate && <p className="mt-2 text-sm text-muted-foreground">{t($ => evidence.load_estimate?.status === "feasible_in_model" ? $.scene_load_feasible : evidence.load_estimate?.status === "infeasible_in_model" ? $.scene_load_infeasible : $.scene_load_unavailable)}</p>}
      <p className="mt-3 text-sm leading-6 text-muted-foreground">{t($ => $.scene_check_limits)}</p>
      {evidence.requirements.length > 0 && <ul className="mt-4 grid gap-2 text-sm">
        {evidence.requirements.map((item, i) => <li key={i} className="flex gap-2"><Check aria-hidden="true" className="mt-0.5 size-4 shrink-0" /><span>{item.requirement}</span></li>)}
      </ul>}
    </div>}
    <details className="rounded-2xl border border-border bg-card p-5">
      <summary className="cursor-pointer font-bold">{t($ => $.scene_parts_list, { types: rows.length, count: plan.placements.length })}</summary>
      <p className="mt-3 text-sm text-muted-foreground">{compared ? t($ => missingCount > 0 ? $.scene_missing : $.scene_inventory_ready, { count: missingCount }) : t($ => $.scene_inventory_unchecked)}</p>
      {inventoryQuery.isError && <p role="status" className="mt-2 text-xs text-muted-foreground">{t($ => $.scene_inventory_refresh_error)}</p>}
      <Button variant="outline" size="sm" onClick={download} className="my-4"><Download className="size-4" />{t($ => $.scene_parts_download)}</Button>
      <div className="max-h-80 overflow-auto rounded-xl border border-border">
        <table className="w-full text-left text-sm">
          <thead className="sticky top-0 bg-muted"><tr>
            <th className="px-3 py-2">{t($ => $.scene_part)}</th><th className="px-3 py-2">{t($ => $.editor_color)}</th><th className="px-3 py-2 text-right">{t($ => $.scene_quantity)}</th>
            {compared && <th className="px-3 py-2 text-right">{t($ => $.scene_shortage)}</th>}
          </tr></thead>
          <tbody>{rows.map(row => <tr key={`${row.partId}:${row.color}`} className="border-t border-border">
            <td className="px-3 py-2"><span className="font-medium">{row.name}</span><span className="mt-1 block text-xs text-muted-foreground">{row.ldrawId}</span></td>
            <td className="px-3 py-2"><span className="inline-flex items-center gap-2"><span aria-hidden="true" className="size-3 rounded-full border border-border" style={{ backgroundColor: `#${(LDRAW_OFFICIAL_COLORS[row.color] ?? LDRAW_OFFICIAL_COLORS[71]!).toString(16).padStart(6, "0")}` }} />{row.color}</span></td>
            <td className="px-3 py-2 text-right tabular-nums">{row.quantity}</td>
            {compared && <td className="px-3 py-2 text-right tabular-nums">{row.missing ?? 0}</td>}
          </tr>)}</tbody>
        </table>
      </div>
    </details>
  </section>;
}
