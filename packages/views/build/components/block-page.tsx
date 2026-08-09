"use client";

import { useEffect, useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  ArrowRight,
  Box,
  Check,
  Infinity as InfinityIcon,
  LoaderCircle,
  Minus,
  PackageOpen,
  Plus,
  RotateCcw,
  Save,
  Sparkles,
} from "lucide-react";
import { Button } from "@chimii/ui/components/ui/button";
import { cn } from "@chimii/ui/lib/utils";
import { useWorkspaceId } from "@chimii/core/hooks";
import { useWorkspacePaths } from "@chimii/core/paths";
import {
  brickInventoryOptions,
  buildCatalogOptions,
  useResetBrickInventory,
  useSaveBrickInventory,
  type BrickInventoryItem,
} from "@chimii/core/build";
import { AppLink } from "../../navigation";
import { useT } from "../../i18n";
import { BrickThumbnail } from "./brick-thumbnail";
import { BuildModelViewer } from "./build-model-viewer";

const quantityKey = (partId: string, color: number) => `${partId}:${color}`;

export function BlockPage() {
  const { t } = useT("build");
  const workspaceId = useWorkspaceId();
  const workspacePaths = useWorkspacePaths();
  const catalogQuery = useQuery(buildCatalogOptions(workspaceId));
  const inventoryQuery = useQuery(brickInventoryOptions(workspaceId));
  const saveInventory = useSaveBrickInventory();
  const resetInventory = useResetBrickInventory();
  const [draft, setDraft] = useState<Record<string, number>>({});
  const [editing, setEditing] = useState(false);
  const [selectedPartId, setSelectedPartId] = useState("");
  const [previewColor, setPreviewColor] = useState(4);
  const [message, setMessage] = useState("");
  const [hydratedRevision, setHydratedRevision] = useState("");

  const inventory = inventoryQuery.data;
  const catalog = catalogQuery.data;
  const revisionKey = inventory ? `${inventory.configured}:${inventory.revision}` : "";

  useEffect(() => {
    if (!inventory || revisionKey === hydratedRevision) return;
    const next: Record<string, number> = {};
    for (const item of inventory.items) next[quantityKey(item.part_id, item.color)] = item.quantity;
    setDraft(next);
    setEditing(inventory.configured);
    setHydratedRevision(revisionKey);
  }, [hydratedRevision, inventory, revisionKey]);

  useEffect(() => {
    if (!selectedPartId && catalog?.parts[0]) setSelectedPartId(catalog.parts[0].id);
  }, [catalog, selectedPartId]);

  const partNames = useMemo<Record<string, string>>(() => ({
    "brick-2x4": t($ => $.block_part_brick_2x4),
    "brick-2x2": t($ => $.block_part_brick_2x2),
    "brick-1x2": t($ => $.block_part_brick_1x2),
    "brick-1x1": t($ => $.block_part_brick_1x1),
    "plate-2x4": t($ => $.block_part_plate_2x4),
    "plate-2x2": t($ => $.block_part_plate_2x2),
    "plate-1x2": t($ => $.block_part_plate_1x2),
    "slope-2x2": t($ => $.block_part_slope_2x2),
    "wheel-holder-2x2": t($ => $.block_part_wheel_holder),
    wheel: t($ => $.block_part_wheel),
  }), [t]);
  const colorNames = useMemo<Record<number, string>>(() => ({
    1: t($ => $.block_color_blue), 2: t($ => $.block_color_green), 4: t($ => $.block_color_red),
    14: t($ => $.block_color_yellow), 15: t($ => $.block_color_white), 71: t($ => $.block_color_gray),
  }), [t]);
  const categoryNames = useMemo<Record<string, string>>(() => ({
    brick: t($ => $.block_category_brick), plate: t($ => $.block_category_plate),
    slope: t($ => $.block_category_slope), wheel: t($ => $.block_category_wheel),
  }), [t]);

  const draftItems = useMemo<BrickInventoryItem[]>(() => Object.entries(draft)
    .map(([key, quantity]) => {
      const separator = key.lastIndexOf(":");
      return { part_id: key.slice(0, separator), color: Number(key.slice(separator + 1)), quantity };
    })
    .filter((item) => item.quantity > 0)
    .sort((a, b) => a.part_id.localeCompare(b.part_id) || a.color - b.color), [draft]);
  const totalPieces = draftItems.reduce((sum, item) => sum + item.quantity, 0);
  const ownedPartTypes = new Set(draftItems.map((item) => item.part_id)).size;
  const selectedPart = catalog?.parts.find((part) => part.id === selectedPartId) ?? catalog?.parts[0];
  const selectedColor = catalog?.colors.find((color) => color.code === previewColor) ?? catalog?.colors[0];

  const setQuantity = (partId: string, color: number, quantity: number) => {
    setMessage("");
    setDraft((current) => ({ ...current, [quantityKey(partId, color)]: Math.max(0, Math.min(999, quantity)) }));
  };

  const save = async () => {
    if (!inventory) return;
    setMessage("");
    try {
      await saveInventory.mutateAsync({ expectedRevision: inventory.revision, items: draftItems });
      setMessage(t($ => $.block_saved));
    } catch {
      setMessage(t($ => $.block_save_error));
    }
  };

  const reset = async () => {
    if (!inventory) return;
    setMessage("");
    try {
      await resetInventory.mutateAsync(inventory.revision);
      setMessage(t($ => $.block_reset_done));
    } catch {
      setMessage(t($ => $.block_reset_error));
    }
  };

  if (catalogQuery.isPending || inventoryQuery.isPending) {
    return <div className="flex h-full items-center justify-center bg-[#f4ead5] text-[#1d241f]"><LoaderCircle className="mr-3 size-6 animate-spin" />{t($ => $.block_loading)}</div>;
  }
  if (
    catalogQuery.isError ||
    inventoryQuery.isError ||
    !catalog ||
    !inventory ||
    catalog.catalog_version === "" ||
    catalog.parts.length === 0 ||
    catalog.colors.length === 0 ||
    inventory.catalog_version === ""
  ) {
    return (
      <div className="flex h-full items-center justify-center bg-[#f4ead5] p-6 text-[#1d241f]">
        <div className="max-w-md rounded-[2rem] border-2 border-[#1d241f] bg-[#fffdf7] p-8 text-center shadow-[6px_7px_0_#1d241f]">
          <PackageOpen className="mx-auto mb-4 size-10" /><h1 className="text-2xl font-black">{t($ => $.block_error_title)}</h1>
          <p className="mt-2 text-sm text-[#59645d]">{t($ => $.block_error_description)}</p>
          <Button className="mt-5 rounded-full border-2 border-[#1d241f] bg-[#ffd85a] font-black text-[#1d241f]" onClick={() => { void catalogQuery.refetch(); void inventoryQuery.refetch(); }}>{t($ => $.block_retry)}</Button>
        </div>
      </div>
    );
  }

  return (
    <main className="h-full overflow-y-auto bg-[#f4ead5] text-[#1d241f]">
      <div className="pointer-events-none fixed inset-0 opacity-35 [background-image:radial-gradient(#bfae8d_1px,transparent_1px)] [background-size:22px_22px]" />
      <div className="relative mx-auto max-w-[1380px] px-5 py-6 md:px-8 md:py-8">
        <header className="mb-7 flex flex-wrap items-center justify-between gap-4">
          <div className="flex items-center gap-3">
            <div className="flex size-11 rotate-[-4deg] items-center justify-center rounded-2xl border-2 border-[#1d241f] bg-[#f26b5b] shadow-[3px_4px_0_#1d241f]"><Box className="size-6 text-white" /></div>
            <div><p className="text-xs font-black uppercase tracking-[.2em] text-[#39715a]">{t($ => $.block_brand)}</p><h1 className="text-xl font-black">{t($ => $.block_title)}</h1></div>
          </div>
          <Button nativeButton={false} className="rounded-full border-2 border-[#1d241f] bg-[#ffd85a] font-black text-[#1d241f] shadow-[3px_3px_0_#1d241f] hover:bg-[#ffe27c]" render={<AppLink href={workspacePaths.build()} />}>
            {t($ => $.block_build)} <ArrowRight className="size-4" />
          </Button>
        </header>

        <section className={cn("mb-7 grid gap-5 rounded-[2rem] border-2 border-[#1d241f] p-5 shadow-[6px_7px_0_#1d241f] md:grid-cols-[1fr_auto] md:p-7", inventory.configured ? "bg-[#fffdf7]" : "bg-[#dcecff]")}>
          <div className="flex items-start gap-4">
            <div className={cn("flex size-12 shrink-0 items-center justify-center rounded-2xl border-2 border-[#1d241f]", inventory.configured ? "bg-[#ffd85a]" : "bg-[#3c6fc6] text-white")}>
              {inventory.configured ? <PackageOpen className="size-6" /> : <InfinityIcon className="size-7" />}
            </div>
            <div>
              <div className="mb-1 inline-flex rounded-full bg-[#1d241f] px-2.5 py-1 text-[10px] font-black uppercase tracking-[.15em] text-white">{inventory.configured ? t($ => $.block_custom_badge) : t($ => $.block_unlimited_badge)}</div>
              <h2 className="text-xl font-black">{inventory.configured ? t($ => $.block_custom_title) : t($ => $.block_unlimited_title)}</h2>
              <p className="mt-1 max-w-2xl text-sm font-medium leading-6 text-[#59645d]">{inventory.configured ? t($ => $.block_custom_description) : t($ => $.block_unlimited_description)}</p>
            </div>
          </div>
          <div className="flex flex-wrap items-center gap-2 md:justify-end">
            {!editing && <Button className="rounded-full border-2 border-[#1d241f] bg-[#fffdf7] font-black text-[#1d241f]" onClick={() => setEditing(true)}>{t($ => $.block_custom_start)}</Button>}
            {inventory.configured && <Button variant="outline" className="rounded-full border-2 border-[#1d241f] bg-transparent font-bold" disabled={resetInventory.isPending} onClick={() => void reset()}><RotateCcw className="size-4" />{t($ => $.block_reset)}</Button>}
          </div>
        </section>

        <div className="mb-7 grid gap-5 lg:grid-cols-[minmax(0,1fr)_360px]">
          <section className="rounded-[2rem] border-2 border-[#1d241f] bg-[#fffdf7] p-5 shadow-[5px_6px_0_#1d241f]">
            <div className="mb-5 flex flex-wrap items-end justify-between gap-3">
              <div><p className="text-xs font-black uppercase tracking-[.16em] text-[#39715a]">{t($ => $.block_catalog_label)}</p><h2 className="mt-1 text-2xl font-black">{t($ => $.block_catalog_title)}</h2><p className="mt-1 text-sm text-[#667068]">{t($ => $.block_catalog_description)}</p></div>
              {editing && <div className="rounded-2xl border-2 border-[#1d241f] bg-[#f4ead5] px-4 py-2 text-right"><div className="text-xs font-bold text-[#667068]">{t($ => $.block_summary)}</div><div className="font-black">{t($ => $.block_summary_value, { types: ownedPartTypes, pieces: totalPieces })}</div></div>}
            </div>

            <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
              {catalog.parts.map((part, index) => {
                const activeItems = catalog.colors.filter((color) => (draft[quantityKey(part.id, color.code)] ?? 0) > 0);
                const fallbackCardColor = activeItems[0] ?? catalog.colors[index % catalog.colors.length]!;
                const cardColor = selectedPart?.id === part.id
                  ? catalog.colors.find((color) => color.code === previewColor) ?? fallbackCardColor
                  : fallbackCardColor;
                const total = activeItems.reduce((sum, color) => sum + (draft[quantityKey(part.id, color.code)] ?? 0), 0);
                return (
                  <article key={part.id} className={cn("rounded-[1.5rem] border-2 p-3 transition", selectedPart?.id === part.id ? "border-[#1d241f] bg-[#eef5ff] shadow-[3px_4px_0_#1d241f]" : "border-[#cfc5b5] bg-white hover:border-[#1d241f]")}>
                    <button type="button" className="w-full rounded-xl text-left focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-[#3c6fc6]/30" onClick={() => { setSelectedPartId(part.id); setPreviewColor(cardColor.code); }}>
                      <BrickThumbnail
                        ldrawID={part.ldraw_id}
                        colorCode={cardColor.code}
                        alt={`${colorNames[cardColor.code] ?? cardColor.name} ${partNames[part.id] ?? part.name}`}
                        eager={index < 3}
                      />
                      <div className="mt-3 flex items-start justify-between gap-2"><div><p className="text-[10px] font-black uppercase tracking-[.14em] text-[#39715a]">{categoryNames[part.category] ?? part.category}</p><h3 className="font-black leading-tight">{partNames[part.id] ?? part.name}</h3></div><span className="rounded-full border border-[#1d241f] bg-[#ffd85a] px-2 py-1 text-xs font-black">{inventory.configured || editing ? total : "∞"}</span></div>
                    </button>
                    <div className="mt-3 flex flex-wrap gap-1.5" aria-label={t($ => $.block_colors)}>
                      {catalog.colors.map((color) => {
                        const quantity = draft[quantityKey(part.id, color.code)] ?? 0;
                        return <button key={color.code} type="button" title={colorNames[color.code] ?? color.name} aria-label={`${colorNames[color.code] ?? color.name}: ${quantity}`} aria-pressed={quantity > 0} disabled={!editing} className={cn("relative size-7 rounded-full border-2 border-[#1d241f] transition disabled:cursor-default", quantity > 0 ? "scale-100 shadow-[2px_2px_0_#1d241f]" : "scale-90 opacity-35")} style={{ backgroundColor: color.hex }} onClick={() => { setPreviewColor(color.code); setSelectedPartId(part.id); setQuantity(part.id, color.code, quantity > 0 ? 0 : Math.max(1, Math.min(20, part.quantity))); }}>{quantity > 0 && <Check className={cn("absolute inset-0 m-auto size-4", color.code === 15 || color.code === 14 ? "text-[#1d241f]" : "text-white")} />}</button>;
                      })}
                    </div>
                    {editing && activeItems.length > 0 && <div className="mt-3 space-y-2 border-t border-dashed border-[#cfc5b5] pt-3">
                      {activeItems.map((color) => {
                        const quantity = draft[quantityKey(part.id, color.code)] ?? 0;
                        return <div key={color.code} className="flex items-center gap-2"><span className="size-3 rounded-full border border-[#1d241f]" style={{ backgroundColor: color.hex }} /><span className="min-w-0 flex-1 truncate text-xs font-bold">{colorNames[color.code] ?? color.name}</span><button type="button" className="flex size-7 items-center justify-center rounded-full border border-[#1d241f] bg-[#f4ead5]" aria-label={t($ => $.block_decrease)} onClick={() => setQuantity(part.id, color.code, quantity - 1)}><Minus className="size-3" /></button><input type="number" min={0} max={999} value={quantity} aria-label={t($ => $.block_quantity, { part: partNames[part.id] ?? part.name, color: colorNames[color.code] ?? color.name })} className="h-8 w-14 rounded-lg border-2 border-[#1d241f] bg-white text-center text-sm font-black" onChange={(event) => setQuantity(part.id, color.code, Number(event.target.value) || 0)} /><button type="button" className="flex size-7 items-center justify-center rounded-full border border-[#1d241f] bg-[#ffd85a]" aria-label={t($ => $.block_increase)} onClick={() => setQuantity(part.id, color.code, quantity + 1)}><Plus className="size-3" /></button></div>;
                      })}
                    </div>}
                  </article>
                );
              })}
            </div>
          </section>

          <aside className="space-y-5 lg:sticky lg:top-6 lg:self-start">
            <section className="overflow-hidden rounded-[2rem] border-2 border-[#1d241f] bg-[#3c6fc6] shadow-[5px_6px_0_#1d241f]">
              <div className="flex items-center justify-between px-5 pt-4 text-white"><div><p className="text-[10px] font-black uppercase tracking-[.16em] text-white/70">{t($ => $.block_preview_label)}</p><h2 className="font-black">{selectedPart ? partNames[selectedPart.id] ?? selectedPart.name : t($ => $.block_preview)}</h2></div><Sparkles className="size-5 text-[#ffd85a]" /></div>
              {selectedPart && selectedColor && <BuildModelViewer className="min-h-[290px] border-0 shadow-none" placements={[{ id: "preview", part_id: selectedPart.id, color: selectedColor.code, x: 0, y: 0, z: 0, rotation: 0, step: 1, module: "preview" }]} parts={{ [selectedPart.id]: selectedPart }} catalogVersion={catalog.catalog_version} />}
            </section>
            {editing && <section className="rounded-[2rem] border-2 border-[#1d241f] bg-[#fffdf7] p-5 shadow-[5px_6px_0_#1d241f]">
              <h2 className="text-lg font-black">{t($ => $.block_ready_title)}</h2><p className="mt-1 text-sm leading-6 text-[#667068]">{totalPieces > 0 ? t($ => $.block_ready_description) : t($ => $.block_empty_description)}</p>
              {message && <p role="status" className="mt-3 rounded-xl bg-[#f4ead5] px-3 py-2 text-sm font-bold">{message}</p>}
              <Button className="mt-4 w-full rounded-full border-2 border-[#1d241f] bg-[#f26b5b] font-black text-white shadow-[3px_3px_0_#1d241f] hover:bg-[#e65e4f]" disabled={saveInventory.isPending} onClick={() => void save()}>{saveInventory.isPending ? <LoaderCircle className="size-4 animate-spin" /> : <Save className="size-4" />}{saveInventory.isPending ? t($ => $.block_saving) : t($ => $.block_save)}</Button>
            </section>}
          </aside>
        </div>
      </div>
    </main>
  );
}
