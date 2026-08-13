"use client";

import { Box, Eye, PackageCheck, WandSparkles } from "lucide-react";
import type { BuildCatalogColor, BuildPartSpec } from "@chimii/core/build";
import { cn } from "@chimii/ui/lib/utils";
import { BrickThumbnail } from "./brick-thumbnail";

type Capability = "auto" | "inventory" | "preview";

function capabilityFor(part: BuildPartSpec): Capability {
  if (part.auto_build_eligible === true) return "auto";
  if (part.inventory_eligible === true) return "inventory";
  return "preview";
}

export function BlockCatalogCard({
  part,
  color,
  colorName,
  selected,
  total,
  unlimited,
  labels,
  onSelect,
}: {
  part: BuildPartSpec;
  color: BuildCatalogColor;
  colorName: string;
  selected: boolean;
  total: number;
  unlimited: boolean;
  labels: Record<Capability, string>;
  onSelect(): void;
}) {
  const capability = capabilityFor(part);
  const CapabilityIcon = capability === "auto" ? WandSparkles : capability === "inventory" ? PackageCheck : Eye;

  return (
    <article
      className={cn(
        "[contain-intrinsic-size:0_248px] [content-visibility:auto] rounded-[1.35rem] border-2 p-3",
        selected
          ? "border-[#1d241f] bg-[#eef5ff] shadow-[3px_4px_0_#1d241f]"
          : "border-[#cfc5b5] bg-white hover:border-[#1d241f]",
      )}
    >
      <button
        type="button"
        className="w-full touch-manipulation rounded-xl text-left focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-[#3c6fc6]/30"
        aria-pressed={selected}
        onClick={onSelect}
      >
        <BrickThumbnail
          ldrawID={part.ldraw_id}
          colorCode={color.code}
          alt={`${colorName} ${part.name}`}
          studsX={part.studs_x}
          studsZ={part.studs_z}
          platesY={part.plates_y}
        />
        <div className="mt-3 flex min-w-0 items-start justify-between gap-2">
          <div className="min-w-0">
            <p className="truncate text-[10px] font-black uppercase tracking-[.14em] text-[#39715a]">
              #{part.popularity_rank ?? "–"} · {part.category}
            </p>
            <h3 className="mt-0.5 line-clamp-2 min-h-10 text-sm font-black leading-5">{part.name}</h3>
          </div>
          <span className="shrink-0 rounded-full border border-[#1d241f] bg-[#ffd85a] px-2 py-1 text-xs font-black tabular-nums">
            {unlimited ? "∞" : total}
          </span>
        </div>
        <div className="mt-3 flex items-center justify-between gap-2 border-t border-dashed border-[#d8cfbf] pt-2.5">
          <span
            className={cn(
              "inline-flex min-w-0 items-center gap-1.5 rounded-full px-2 py-1 text-[10px] font-black",
              capability === "auto" && "bg-[#d8f0e4] text-[#225f47]",
              capability === "inventory" && "bg-[#fff0b8] text-[#72520b]",
              capability === "preview" && "bg-[#ece8e0] text-[#5f625f]",
            )}
          >
            <CapabilityIcon className="size-3 shrink-0" aria-hidden="true" />
            <span className="truncate">{labels[capability]}</span>
          </span>
          <span className="inline-flex items-center gap-1 font-mono text-[10px] text-[#69736d]" translate="no">
            <Box className="size-3" aria-hidden="true" /> {part.ldraw_id}
          </span>
        </div>
      </button>
    </article>
  );
}
