"use client";

import { useCallback, useMemo, useRef, useState } from "react";
import { Boxes, Rotate3D } from "lucide-react";
import { cn } from "@chimii/ui/lib/utils";
import type { BuildPartSpec, BuildPlacement } from "@chimii/core/build";
import { useT } from "../../i18n";
import { LDrawModelCanvas } from "./ldraw-model-canvas";

const LDRAW_COLORS: Record<number, { top: string; front: string; side: string }> = {
  1: { top: "#4e83e6", front: "#2f62bf", side: "#254e99" },
  2: { top: "#43bd8a", front: "#249366", side: "#1d7251" },
  4: { top: "#f06a5e", front: "#cf4238", side: "#a72f29" },
  14: { top: "#ffd85a", front: "#e9ad25", side: "#bc8415" },
  15: { top: "#fffdf7", front: "#e9e1d2", side: "#cfc5b5" },
  71: { top: "#b9c2c8", front: "#8d999f", side: "#707c82" },
};

interface Point { x: number; y: number }

function polygon(points: Point[]) {
  return points.map((point) => `${point.x.toFixed(1)},${point.y.toFixed(1)}`).join(" ");
}

function shadeFor(color: number) {
  return LDRAW_COLORS[color] ?? LDRAW_COLORS[71]!;
}

function projectedPart(placement: BuildPlacement, part: BuildPartSpec, yaw: number, highlighted: boolean) {
  const rotated = placement.rotation % 180 !== 0;
  const width = rotated ? part.studs_z : part.studs_x;
  const depth = rotated ? part.studs_x : part.studs_z;
  const height = part.plates_y;
  const cos = Math.cos(yaw);
  const sin = Math.sin(yaw);
  const project = (x: number, y: number, z: number): Point & { depth: number } => {
    const rx = x * cos - z * sin;
    const rz = x * sin + z * cos;
    return { x: rx * 19, y: rz * 8.5 - y * 7.5, depth: rz };
  };
  const x = placement.x;
  const y = placement.y;
  const z = placement.z;
  const p000 = project(x, y, z);
  const p100 = project(x + width, y, z);
  const p110 = project(x + width, y, z + depth);
  const p001 = project(x, y + height, z);
  const p101 = project(x + width, y + height, z);
  const p111 = project(x + width, y + height, z + depth);
  const p011 = project(x, y + height, z + depth);
  return {
    id: placement.id,
    highlighted,
    depth: (p000.depth + p110.depth) / 2 + y / 20,
    top: [p001, p101, p111, p011],
    front: [p000, p100, p101, p001],
    side: [p100, p110, p111, p101],
    shade: shadeFor(placement.color),
  };
}

export function BuildModelViewer({
  placements,
  parts,
  maxStep,
  highlightPlacementIds = [],
  className,
  renderMode = "ldraw",
  catalogVersion,
}: {
  placements: BuildPlacement[];
  parts: Record<string, BuildPartSpec>;
  maxStep?: number;
  highlightPlacementIds?: readonly string[];
  className?: string;
  renderMode?: "ldraw" | "projection";
  catalogVersion?: string;
}) {
  const { t } = useT("build");
  const [yaw, setYaw] = useState(-Math.PI / 4);
  const [ldrawStatus, setLDrawStatus] = useState<"loading" | "ready" | "failed">("loading");
  const drag = useRef<{ x: number; yaw: number } | null>(null);
  const handleLDrawStatus = useCallback((status: "loading" | "ready" | "failed") => setLDrawStatus(status), []);
  const visible = useMemo(
    () => placements.filter((placement) => maxStep == null || placement.step <= maxStep),
    [maxStep, placements],
  );
  const highlighted = useMemo(() => new Set(highlightPlacementIds), [highlightPlacementIds]);
  const shapes = useMemo(
    () => visible
      .map((placement) => {
        const part = parts[placement.part_id];
        return part ? projectedPart(placement, part, yaw, highlighted.has(placement.id)) : null;
      })
      .filter((shape): shape is NonNullable<typeof shape> => shape != null)
      .sort((a, b) => a.depth - b.depth),
    [highlighted, parts, visible, yaw],
  );
  const modelLayout = useMemo(() => {
    const points = shapes.flatMap((shape) => [...shape.top, ...shape.front, ...shape.side]);
    if (points.length === 0) return { scale: 1, translateX: 320, translateY: 290 };
    const minX = Math.min(...points.map((point) => point.x));
    const maxX = Math.max(...points.map((point) => point.x));
    const minY = Math.min(...points.map((point) => point.y));
    const maxY = Math.max(...points.map((point) => point.y));
    const width = Math.max(1, maxX - minX);
    const height = Math.max(1, maxY - minY);
    const scale = Math.min(2.25, 410 / width, 215 / height);
    return {
      scale,
      translateX: 320 - ((minX + maxX) / 2) * scale,
      translateY: 285 - ((minY + maxY) / 2) * scale,
    };
  }, [shapes]);

  return (
    <div
      role="img"
      aria-label={t($ => $.viewer_aria)}
      tabIndex={0}
      data-renderer={renderMode === "ldraw" && ldrawStatus === "ready" ? "ldraw-glb" : "projection"}
      className={cn("relative isolate touch-none cursor-grab overflow-hidden rounded-[2rem] border-2 border-[#1d241f] bg-[#3c6fc6] active:cursor-grabbing", renderMode === "ldraw" && "min-h-[320px]", className)}
      onPointerDown={(event) => {
        drag.current = { x: event.clientX, yaw };
        event.currentTarget.setPointerCapture(event.pointerId);
      }}
      onPointerMove={(event) => {
        if (!drag.current) return;
        setYaw(drag.current.yaw + (event.clientX - drag.current.x) / 170);
      }}
      onPointerUp={() => { drag.current = null; }}
      onPointerCancel={() => { drag.current = null; }}
      onKeyDown={(event) => {
        if (event.key === "ArrowLeft" || event.key === "ArrowRight") {
          event.preventDefault();
          setYaw((current) => current + (event.key === "ArrowLeft" ? -0.18 : 0.18));
        }
      }}
    >
      <div className="absolute inset-0 opacity-25 [background-image:linear-gradient(rgba(255,255,255,.35)_1px,transparent_1px),linear-gradient(90deg,rgba(255,255,255,.35)_1px,transparent_1px)] [background-size:24px_24px]" />
      <div className="absolute left-4 top-4 z-10 flex items-center gap-2 rounded-full border border-white/50 bg-[#214f9d]/80 px-3 py-1.5 text-xs font-bold text-white backdrop-blur">
        <Boxes className="size-3.5" />
        {t($ => $.viewer_bricks, { count: visible.length })}
      </div>
      <div className="absolute right-4 top-4 z-10 flex items-center gap-1.5 text-xs font-semibold text-white/80">
        <Rotate3D className="size-4" /> {t($ => $.viewer_rotate)}
      </div>
      {renderMode === "ldraw" && (
        <LDrawModelCanvas
          placements={visible}
          parts={parts}
          catalogVersion={catalogVersion}
          highlightedPlacementIds={highlighted}
          yaw={yaw}
          onStatus={handleLDrawStatus}
        />
      )}
      {renderMode === "ldraw" && ldrawStatus === "ready" && (
        <div className="absolute bottom-4 right-4 z-10 rounded-full border border-white/35 bg-[#173b79]/75 px-3 py-1.5 text-[10px] font-black uppercase tracking-[.12em] text-white/85 backdrop-blur">
          {t($ => $.viewer_catalog)}
        </div>
      )}
      <svg
        viewBox="0 0 640 420"
        aria-hidden="true"
        className={cn("relative z-[1] h-full w-full transition-opacity duration-300", renderMode === "ldraw" && "min-h-[320px]", renderMode === "ldraw" && ldrawStatus === "ready" && "opacity-0")}
      >
        <ellipse cx="320" cy="335" rx="175" ry="42" fill="#173b79" opacity=".32" />
        <g transform={`translate(${modelLayout.translateX} ${modelLayout.translateY}) scale(${modelLayout.scale})`}>
          {shapes.map((shape) => (
            <g key={shape.id} stroke={shape.highlighted ? "#ffe46d" : "#18201b"} strokeWidth={shape.highlighted ? 4 : 1.8} strokeLinejoin="round">
              <polygon vectorEffect="non-scaling-stroke" points={polygon(shape.front)} fill={shape.shade.front} />
              <polygon vectorEffect="non-scaling-stroke" points={polygon(shape.side)} fill={shape.shade.side} />
              <polygon vectorEffect="non-scaling-stroke" points={polygon(shape.top)} fill={shape.shade.top} />
            </g>
          ))}
        </g>
      </svg>
    </div>
  );
}
