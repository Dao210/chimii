"use client";

import { useEffect, useMemo, useState } from "react";
import type { FormEvent } from "react";
import { useInfiniteQuery, useQuery } from "@tanstack/react-query";
import {
  ArrowRight,
  Box,
  Check,
  Eye,
  Infinity as InfinityIcon,
  LoaderCircle,
  Minus,
  PackageCheck,
  PackageOpen,
  Plus,
  RotateCcw,
  Save,
  Search,
  Sparkles,
  WandSparkles,
  X,
} from "lucide-react";
import { Button } from "@chimii/ui/components/ui/button";
import { Input } from "@chimii/ui/components/ui/input";
import { cn } from "@chimii/ui/lib/utils";
import { useWorkspaceId } from "@chimii/core/hooks";
import { useWorkspacePaths } from "@chimii/core/paths";
import {
  brickInventoryOptions,
  buildCatalogOptions,
  buildCatalogPartsOptions,
  useResetBrickInventory,
  useSaveBrickInventory,
  type BrickInventoryItem,
  type BuildCatalogCapability,
} from "@chimii/core/build";
import { AppLink, useNavigation } from "../../navigation";
import { useT } from "../../i18n";
import { BlockCatalogCard } from "./block-catalog-card";
import { BuildModelViewer } from "./build-model-viewer";

const quantityKey = (partId: string, color: number) => `${partId}:${color}`;
const validCapabilities = new Set<BuildCatalogCapability>(["all", "auto_build", "inventory", "preview"]);

export function BlockPage() {
  const { t } = useT("build");
  const workspaceId = useWorkspaceId();
  const workspacePaths = useWorkspacePaths();
  const navigation = useNavigation();
  const urlQuery = navigation.searchParams.get("q")?.trim() ?? "";
  const urlCategory = navigation.searchParams.get("category")?.trim() ?? "";
  const requestedCapability = navigation.searchParams.get("capability") ?? "all";
  const urlCapability: BuildCatalogCapability = validCapabilities.has(requestedCapability as BuildCatalogCapability)
    ? requestedCapability as BuildCatalogCapability
    : "all";
  const [searchInput, setSearchInput] = useState(urlQuery);

  const catalogQuery = useQuery(buildCatalogOptions(workspaceId));
  const partsQuery = useInfiniteQuery(buildCatalogPartsOptions(workspaceId, {
    query: urlQuery,
    category: urlCategory,
    capability: urlCapability,
    limit: 24,
  }));
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
  const pageInfo = partsQuery.data?.pages[0];
  const parts = useMemo(
    () => partsQuery.data?.pages.flatMap((page) => page.parts) ?? [],
    [partsQuery.data?.pages],
  );
  const revisionKey = inventory ? `${inventory.configured}:${inventory.revision}` : "";

  useEffect(() => setSearchInput(urlQuery), [urlQuery]);
  useEffect(() => {
    if (!inventory || revisionKey === hydratedRevision) return;
    const next: Record<string, number> = {};
    for (const item of inventory.items) next[quantityKey(item.part_id, item.color)] = item.quantity;
    setDraft(next);
    setEditing(inventory.configured);
    setHydratedRevision(revisionKey);
  }, [hydratedRevision, inventory, revisionKey]);
  useEffect(() => {
    if (parts.length === 0) {
      setSelectedPartId("");
      return;
    }
    if (!parts.some((part) => part.id === selectedPartId)) setSelectedPartId(parts[0]!.id);
  }, [parts, selectedPartId]);

  const colorNames = useMemo<Record<number, string>>(() => ({
    1: t($ => $.block_color_blue), 2: t($ => $.block_color_green), 4: t($ => $.block_color_red),
    14: t($ => $.block_color_yellow), 15: t($ => $.block_color_white), 71: t($ => $.block_color_gray),
  }), [t]);
  const capabilityLabels = useMemo(() => ({
    auto: t($ => $.block_capability_auto_badge),
    inventory: t($ => $.block_capability_inventory_badge),
    preview: t($ => $.block_capability_preview_badge),
  }), [t]);

  const draftItems = useMemo<BrickInventoryItem[]>(() => Object.entries(draft)
    .map(([key, quantity]) => {
      const separator = key.lastIndexOf(":");
      return { part_id: key.slice(0, separator), color: Number(key.slice(separator + 1)), quantity };
    })
    .filter((item) => item.quantity > 0)
    .sort((a, b) => a.part_id.localeCompare(b.part_id) || a.color - b.color), [draft]);
  const savedItems = useMemo(() => [...(inventory?.items ?? [])]
    .sort((a, b) => a.part_id.localeCompare(b.part_id) || a.color - b.color), [inventory?.items]);
  const isDirty = editing && JSON.stringify(draftItems) !== JSON.stringify(savedItems);
  useEffect(() => {
    if (!isDirty) return;
    const warn = (event: BeforeUnloadEvent) => event.preventDefault();
    window.addEventListener("beforeunload", warn);
    return () => window.removeEventListener("beforeunload", warn);
  }, [isDirty]);

  const totalPieces = draftItems.reduce((sum, item) => sum + item.quantity, 0);
  const ownedPartTypes = new Set(draftItems.map((item) => item.part_id)).size;
  const selectedPart = parts.find((part) => part.id === selectedPartId) ?? parts[0];
  const selectedColor = catalog?.colors.find((color) => color.code === previewColor) ?? catalog?.colors[0];
  const selectedActiveColors = catalog?.colors.filter((color) => (draft[quantityKey(selectedPart?.id ?? "", color.code)] ?? 0) > 0) ?? [];

  const replaceFilters = (next: { query?: string; category?: string; capability?: BuildCatalogCapability }) => {
    const params = new URLSearchParams(navigation.searchParams);
    const query = next.query ?? urlQuery;
    const category = next.category ?? urlCategory;
    const capability = next.capability ?? urlCapability;
    if (query) params.set("q", query); else params.delete("q");
    if (category) params.set("category", category); else params.delete("category");
    if (capability !== "all") params.set("capability", capability); else params.delete("capability");
    const serialized = params.toString();
    navigation.replace(`${navigation.pathname}${serialized ? `?${serialized}` : ""}`);
  };
  const submitSearch = (event: FormEvent) => {
    event.preventDefault();
    replaceFilters({ query: searchInput.trim() });
  };
  const clearFilters = () => {
    setSearchInput("");
    replaceFilters({ query: "", category: "", capability: "all" });
  };
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
    if (!inventory || !window.confirm(t($ => $.block_reset_confirm))) return;
    setMessage("");
    try {
      await resetInventory.mutateAsync(inventory.revision);
      setMessage(t($ => $.block_reset_done));
    } catch {
      setMessage(t($ => $.block_reset_error));
    }
  };

  if (catalogQuery.isPending || inventoryQuery.isPending || partsQuery.isPending) {
    return <div className="flex h-full items-center justify-center bg-[#f4ead5] text-[#1d241f]"><LoaderCircle className="mr-3 size-6 animate-spin motion-reduce:animate-none" aria-hidden="true" />{t($ => $.block_loading)}</div>;
  }
  if (
    catalogQuery.isError || inventoryQuery.isError || partsQuery.isError || !catalog || !inventory || !pageInfo ||
    catalog.catalog_version === "" || catalog.colors.length === 0 || inventory.catalog_version === "" || pageInfo.kit_id === ""
  ) {
    return (
      <div className="flex h-full items-center justify-center bg-[#f4ead5] p-6 text-[#1d241f]">
        <div className="max-w-md rounded-[2rem] border-2 border-[#1d241f] bg-[#fffdf7] p-8 text-center shadow-[6px_7px_0_#1d241f]">
          <PackageOpen className="mx-auto mb-4 size-10" aria-hidden="true" /><h1 className="text-pretty text-2xl font-black">{t($ => $.block_error_title)}</h1>
          <p className="mt-2 text-sm text-[#59645d]">{t($ => $.block_error_description)}</p>
          <Button className="mt-5 rounded-full border-2 border-[#1d241f] bg-[#ffd85a] font-black text-[#1d241f]" onClick={() => { void catalogQuery.refetch(); void inventoryQuery.refetch(); void partsQuery.refetch(); }}>{t($ => $.block_retry)}</Button>
        </div>
      </div>
    );
  }

  return (
    <main className="h-full overflow-y-auto overflow-x-hidden bg-[#f4ead5] text-[#1d241f]">
      <div className="pointer-events-none fixed inset-0 opacity-35 [background-image:radial-gradient(#bfae8d_1px,transparent_1px)] [background-size:22px_22px]" />
      <div className="relative mx-auto max-w-[1440px] px-5 py-6 md:px-8 md:py-8">
        <header className="mb-7 flex flex-wrap items-center justify-between gap-4">
          <div className="flex items-center gap-3">
            <div className="flex size-11 rotate-[-4deg] items-center justify-center rounded-2xl border-2 border-[#1d241f] bg-[#f26b5b] shadow-[3px_4px_0_#1d241f]"><Box className="size-6 text-white" aria-hidden="true" /></div>
            <div><p className="text-xs font-black uppercase tracking-[.2em] text-[#39715a]">{t($ => $.block_brand)}</p><h1 className="text-pretty text-xl font-black">{t($ => $.block_title)}</h1></div>
          </div>
          <Button nativeButton={false} className="rounded-full border-2 border-[#1d241f] bg-[#ffd85a] font-black text-[#1d241f] shadow-[3px_3px_0_#1d241f] hover:bg-[#ffe27c]" render={<AppLink href={workspacePaths.build()} onClick={(event) => { if (isDirty && !window.confirm(t($ => $.block_unsaved_confirm))) event.preventDefault(); }} />}>
            {t($ => $.block_build)} <ArrowRight className="size-4" aria-hidden="true" />
          </Button>
        </header>

        <section className={cn("mb-7 grid gap-5 rounded-[2rem] border-2 border-[#1d241f] p-5 shadow-[6px_7px_0_#1d241f] md:grid-cols-[1fr_auto] md:p-7", inventory.configured ? "bg-[#fffdf7]" : "bg-[#dcecff]")}>
          <div className="flex items-start gap-4">
            <div className={cn("flex size-12 shrink-0 items-center justify-center rounded-2xl border-2 border-[#1d241f]", inventory.configured ? "bg-[#ffd85a]" : "bg-[#3c6fc6] text-white")}>
              {inventory.configured ? <PackageOpen className="size-6" aria-hidden="true" /> : <InfinityIcon className="size-7" aria-hidden="true" />}
            </div>
            <div>
              <div className="mb-1 inline-flex rounded-full bg-[#1d241f] px-2.5 py-1 text-[10px] font-black uppercase tracking-[.15em] text-white">{inventory.configured ? t($ => $.block_custom_badge) : t($ => $.block_unlimited_badge)}</div>
              <h2 className="text-pretty text-xl font-black">{inventory.configured ? t($ => $.block_custom_title) : t($ => $.block_unlimited_title)}</h2>
              <p className="mt-1 max-w-2xl text-sm font-medium leading-6 text-[#59645d]">{inventory.configured ? t($ => $.block_custom_description) : t($ => $.block_unlimited_description)}</p>
            </div>
          </div>
          <div className="flex flex-wrap items-center gap-2 md:justify-end">
            {!editing && <Button className="rounded-full border-2 border-[#1d241f] bg-[#fffdf7] font-black text-[#1d241f]" onClick={() => setEditing(true)}>{t($ => $.block_custom_start)}</Button>}
            {inventory.configured && <Button variant="outline" className="rounded-full border-2 border-[#1d241f] bg-transparent font-bold" disabled={resetInventory.isPending} onClick={() => void reset()}><RotateCcw className="size-4" aria-hidden="true" />{t($ => $.block_reset)}</Button>}
          </div>
        </section>

        <div className="mb-7 grid gap-5 lg:grid-cols-[minmax(0,1fr)_380px]">
          <section className="min-w-0 rounded-[2rem] border-2 border-[#1d241f] bg-[#fffdf7] p-5 shadow-[5px_6px_0_#1d241f]">
            <div className="mb-5 flex flex-wrap items-end justify-between gap-3">
              <div className="min-w-0"><p className="text-xs font-black uppercase tracking-[.16em] text-[#39715a]">{t($ => $.block_catalog_label)}</p><h2 className="mt-1 text-pretty text-2xl font-black">{pageInfo.kit_name}</h2><p className="mt-1 text-sm text-[#667068]">{t($ => $.block_kit_description, { count: pageInfo.profile_total })}</p></div>
              {editing && <div className="rounded-2xl border-2 border-[#1d241f] bg-[#f4ead5] px-4 py-2 text-right"><div className="text-xs font-bold text-[#667068]">{t($ => $.block_summary)}</div><div className="font-black tabular-nums">{t($ => $.block_summary_value, { types: ownedPartTypes, pieces: totalPieces })}</div></div>}
            </div>

            <form className="mb-5 grid gap-2 rounded-[1.35rem] border border-[#cfc5b5] bg-[#f8f4e9] p-3 md:grid-cols-[minmax(180px,1fr)_auto_auto_auto]" role="search" onSubmit={submitSearch}>
              <label className="relative block min-w-0">
                <span className="sr-only">{t($ => $.block_search_label)}</span>
                <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-[#69736d]" aria-hidden="true" />
                <Input name="catalog-search" autoComplete="off" value={searchInput} onChange={(event) => setSearchInput(event.target.value)} placeholder={t($ => $.block_search_placeholder)} className="h-11 rounded-xl border-2 border-[#9c9079] bg-white pl-9 pr-10" />
                {searchInput && <button type="button" aria-label={t($ => $.block_clear_search)} className="absolute right-2 top-1/2 flex size-7 -translate-y-1/2 items-center justify-center rounded-full hover:bg-[#f4ead5] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[#3c6fc6]" onClick={() => setSearchInput("")}><X className="size-4" aria-hidden="true" /></button>}
              </label>
              <label className="min-w-0">
                <span className="sr-only">{t($ => $.block_category_label)}</span>
                <select name="category" aria-label={t($ => $.block_category_label)} value={urlCategory} className="h-11 min-w-40 rounded-xl border-2 border-[#9c9079] bg-white px-3 text-sm font-bold text-[#1d241f] focus-visible:outline-none focus-visible:ring-3 focus-visible:ring-[#3c6fc6]/30" onChange={(event) => replaceFilters({ category: event.target.value })}>
                  <option value="">{t($ => $.block_category_all)}</option>
                  {pageInfo.categories.map((category) => <option key={category} value={category}>{category}</option>)}
                </select>
              </label>
              <label className="min-w-0">
                <span className="sr-only">{t($ => $.block_capability_label)}</span>
                <select name="capability" aria-label={t($ => $.block_capability_label)} value={urlCapability} className="h-11 min-w-40 rounded-xl border-2 border-[#9c9079] bg-white px-3 text-sm font-bold text-[#1d241f] focus-visible:outline-none focus-visible:ring-3 focus-visible:ring-[#3c6fc6]/30" onChange={(event) => replaceFilters({ capability: event.target.value as BuildCatalogCapability })}>
                  <option value="all">{t($ => $.block_capability_all)}</option>
                  <option value="auto_build">{t($ => $.block_capability_auto_badge)}</option>
                  <option value="inventory">{t($ => $.block_capability_inventory_badge)}</option>
                  <option value="preview">{t($ => $.block_capability_preview_badge)}</option>
                </select>
              </label>
              <Button type="submit" className="h-11 rounded-xl border-2 border-[#1d241f] bg-[#ffd85a] font-black text-[#1d241f]">{t($ => $.block_search_action)}</Button>
            </form>

            <div className="mb-3 flex items-center justify-between gap-3 text-sm text-[#667068]" aria-live="polite">
              <p className="font-bold tabular-nums">{t($ => $.block_results, { shown: parts.length, total: pageInfo.filtered_total })}</p>
              {(urlQuery || urlCategory || urlCapability !== "all") && <button type="button" className="rounded-full px-3 py-1 font-bold text-[#39715a] hover:bg-[#d8f0e4] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[#3c6fc6]" onClick={clearFilters}>{t($ => $.block_clear_filters)}</button>}
            </div>

            {parts.length > 0 ? (
              <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
                {parts.map((part, index) => {
                  const activeItems = catalog.colors.filter((color) => (draft[quantityKey(part.id, color.code)] ?? 0) > 0);
                  const fallbackColor = activeItems[0] ?? catalog.colors[(part.popularity_rank ?? index) % catalog.colors.length]!;
                  const cardColor = selectedPart?.id === part.id ? selectedColor ?? fallbackColor : fallbackColor;
                  const total = activeItems.reduce((sum, color) => sum + (draft[quantityKey(part.id, color.code)] ?? 0), 0);
                  return <BlockCatalogCard key={part.id} part={part} color={cardColor} colorName={colorNames[cardColor.code] ?? cardColor.name} selected={selectedPart?.id === part.id} total={total} unlimited={!inventory.configured && !editing} labels={capabilityLabels} onSelect={() => { setSelectedPartId(part.id); setPreviewColor(cardColor.code); }} />;
                })}
              </div>
            ) : (
              <div className="rounded-[1.5rem] border-2 border-dashed border-[#cfc5b5] px-6 py-14 text-center">
                <Search className="mx-auto size-9 text-[#69736d]" aria-hidden="true" /><h3 className="mt-3 text-lg font-black">{t($ => $.block_no_results_title)}</h3><p className="mt-1 text-sm text-[#667068]">{t($ => $.block_no_results_description)}</p><Button variant="outline" className="mt-4 rounded-full" onClick={clearFilters}>{t($ => $.block_clear_filters)}</Button>
              </div>
            )}
            {partsQuery.hasNextPage && <div className="mt-6 flex justify-center"><Button variant="outline" className="rounded-full border-2 border-[#1d241f] bg-white font-black" disabled={partsQuery.isFetchingNextPage} onClick={() => void partsQuery.fetchNextPage()}>{partsQuery.isFetchingNextPage ? <LoaderCircle className="size-4 animate-spin motion-reduce:animate-none" aria-hidden="true" /> : null}{partsQuery.isFetchingNextPage ? t($ => $.block_loading_more) : t($ => $.block_load_more)}</Button></div>}
          </section>

          <aside className="space-y-5 lg:sticky lg:top-6 lg:self-start">
            <section className="overflow-hidden rounded-[2rem] border-2 border-[#1d241f] bg-[#3c6fc6] shadow-[5px_6px_0_#1d241f]">
              <div className="flex min-w-0 items-center justify-between gap-3 px-5 pt-4 text-white"><div className="min-w-0"><p className="text-[10px] font-black uppercase tracking-[.16em] text-white/70">{t($ => $.block_preview_label)}</p><h2 className="truncate font-black">{selectedPart?.name ?? t($ => $.block_preview)}</h2></div><Sparkles className="size-5 shrink-0 text-[#ffd85a]" aria-hidden="true" /></div>
              {selectedPart && selectedColor && <BuildModelViewer className="min-h-[300px] border-0 shadow-none" placements={[{ id: "preview", part_id: selectedPart.id, color: selectedColor.code, x: 0, y: 0, z: 0, rotation: 0, step: 1, module: "preview" }]} parts={{ [selectedPart.id]: selectedPart }} catalogVersion={pageInfo.catalog_version} />}
              {selectedPart && <div className="border-t border-white/20 bg-[#214f9d] px-5 py-4 text-white"><CapabilityDescription part={selectedPart} /></div>}
            </section>

            {editing && selectedPart && <section className="rounded-[2rem] border-2 border-[#1d241f] bg-[#fffdf7] p-5 shadow-[5px_6px_0_#1d241f]">
              <h2 className="text-pretty text-lg font-black">{t($ => $.block_selected_colors)}</h2>
              {selectedPart.inventory_eligible === true ? <>
                <div className="mt-3 flex flex-wrap gap-2" aria-label={t($ => $.block_colors)}>
                  {catalog.colors.map((color) => {
                    const quantity = draft[quantityKey(selectedPart.id, color.code)] ?? 0;
                    return <button key={color.code} type="button" title={colorNames[color.code] ?? color.name} aria-label={`${colorNames[color.code] ?? color.name}: ${quantity}`} aria-pressed={quantity > 0} className={cn("relative size-10 touch-manipulation rounded-full border-2 border-[#1d241f] focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-[#3c6fc6]/30", quantity > 0 ? "shadow-[2px_2px_0_#1d241f]" : "scale-90 opacity-40")} style={{ backgroundColor: color.hex }} onClick={() => { setPreviewColor(color.code); setQuantity(selectedPart.id, color.code, quantity > 0 ? 0 : Math.max(1, Math.min(20, selectedPart.quantity))); }}>{quantity > 0 && <Check className={cn("absolute inset-0 m-auto size-5", color.code === 15 || color.code === 14 ? "text-[#1d241f]" : "text-white")} aria-hidden="true" />}</button>;
                  })}
                </div>
                {selectedActiveColors.length > 0 && <div className="mt-4 space-y-2 border-t border-dashed border-[#cfc5b5] pt-4">
                  {selectedActiveColors.map((color) => {
                    const quantity = draft[quantityKey(selectedPart.id, color.code)] ?? 0;
                    const inputID = `quantity-${selectedPart.id}-${color.code}`;
                    return <div key={color.code} className="flex items-center gap-2"><span className="size-3 shrink-0 rounded-full border border-[#1d241f]" style={{ backgroundColor: color.hex }} /><label htmlFor={inputID} className="min-w-0 flex-1 truncate text-xs font-bold">{colorNames[color.code] ?? color.name}</label><button type="button" className="flex size-8 touch-manipulation items-center justify-center rounded-full border border-[#1d241f] bg-[#f4ead5] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[#3c6fc6]" aria-label={t($ => $.block_decrease)} onClick={() => setQuantity(selectedPart.id, color.code, quantity - 1)}><Minus className="size-3" aria-hidden="true" /></button><Input id={inputID} name={inputID} type="number" inputMode="numeric" autoComplete="off" min={0} max={999} value={quantity} className="h-9 w-16 border-2 border-[#1d241f] bg-white text-center font-black tabular-nums" onChange={(event) => setQuantity(selectedPart.id, color.code, Number(event.target.value) || 0)} /><button type="button" className="flex size-8 touch-manipulation items-center justify-center rounded-full border border-[#1d241f] bg-[#ffd85a] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[#3c6fc6]" aria-label={t($ => $.block_increase)} onClick={() => setQuantity(selectedPart.id, color.code, quantity + 1)}><Plus className="size-3" aria-hidden="true" /></button></div>;
                  })}
                </div>}
              </> : <p className="mt-2 rounded-xl bg-[#ece8e0] px-3 py-3 text-sm leading-6 text-[#5f625f]">{t($ => $.block_inventory_unavailable)}</p>}
            </section>}

            {editing && <section className="rounded-[2rem] border-2 border-[#1d241f] bg-[#fffdf7] p-5 shadow-[5px_6px_0_#1d241f]">
              <h2 className="text-pretty text-lg font-black">{t($ => $.block_ready_title)}</h2><p className="mt-1 text-sm leading-6 text-[#667068]">{totalPieces > 0 ? t($ => $.block_ready_description) : t($ => $.block_empty_description)}</p>
              {message && <p role="status" aria-live="polite" className="mt-3 rounded-xl bg-[#f4ead5] px-3 py-2 text-sm font-bold">{message}</p>}
              <Button className="mt-4 w-full rounded-full border-2 border-[#1d241f] bg-[#f26b5b] font-black text-white shadow-[3px_3px_0_#1d241f] hover:bg-[#e65e4f]" disabled={saveInventory.isPending || !isDirty} onClick={() => void save()}>{saveInventory.isPending ? <LoaderCircle className="size-4 animate-spin motion-reduce:animate-none" aria-hidden="true" /> : <Save className="size-4" aria-hidden="true" />}{saveInventory.isPending ? t($ => $.block_saving) : t($ => $.block_save)}</Button>
            </section>}
          </aside>
        </div>
      </div>
    </main>
  );
}

function CapabilityDescription({ part }: { part: { auto_build_eligible?: boolean; inventory_eligible?: boolean } }) {
  const { t } = useT("build");
  if (part.auto_build_eligible === true) return <p className="flex items-start gap-2 text-sm leading-5"><WandSparkles className="mt-0.5 size-4 shrink-0 text-[#ffd85a]" aria-hidden="true" />{t($ => $.block_capability_auto_description)}</p>;
  if (part.inventory_eligible === true) return <p className="flex items-start gap-2 text-sm leading-5"><PackageCheck className="mt-0.5 size-4 shrink-0 text-[#ffd85a]" aria-hidden="true" />{t($ => $.block_capability_inventory_description)}</p>;
  return <p className="flex items-start gap-2 text-sm leading-5"><Eye className="mt-0.5 size-4 shrink-0 text-[#ffd85a]" aria-hidden="true" />{t($ => $.block_capability_preview_description)}</p>;
}
