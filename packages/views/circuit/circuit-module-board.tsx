"use client";

import { useId } from "react";
import { CircuitModuleFace } from "./circuit-module-face";
import type {
  CircuitPart,
  CircuitPlacement,
  CircuitConnection,
} from "@chimii/core/circuit";

// Coordinates lay out an instructional graph, never physical mounting geometry.
export function CircuitModuleBoard({
  label,
  parts,
  placements,
  connections,
  columns,
  rows,
  highlighted = [],
  miniature = false,
  schematic = false,
}: {
  label: string;
  parts: CircuitPart[];
  placements: CircuitPlacement[];
  connections: CircuitConnection[];
  columns: number;
  rows: number;
  highlighted?: string[];
  miniature?: boolean;
  schematic?: boolean;
}) {
  const id = useId().replace(/:/g, "");
  const unit = 26,
    margin = 28;
  const byPart = new Map(parts.map((p) => [p.id, p]));
  const byPlacement = new Map(placements.map((p) => [p.id, p]));
  const endpoint = (key: string) => {
    const [instance, pin] = key.split(":");
    const placement = byPlacement.get(instance ?? "");
    const port = byPart
      .get(placement?.part_id ?? "")
      ?.ports.find((p) => p.id === pin);
    return placement && port
      ? {
          x: (placement.x + port.x) * unit + margin,
          y: (placement.y + port.y) * unit + margin,
        }
      : undefined;
  };
  const active = new Set(highlighted);
  const hotPorts = new Set(
    connections
      .filter(
        (w) =>
          active.has(w.cable_id ?? "") ||
          (!w.cable_id && active.has(w.from.split(":")[0] ?? "")),
      )
      .flatMap((w) => [w.from, w.to]),
  );
  return (
    <svg
      role="img"
      data-testid="circuit-module-board"
      aria-label={label}
      viewBox={`0 0 ${columns * unit + margin * 2} ${rows * unit + margin * 2}`}
      className="circuit-module-board block h-auto w-full rounded-xl bg-muted/30"
    >
      <title>{label}</title>
      <defs>
        <marker
          id={`${id}-arrow`}
          viewBox="0 0 10 10"
          refX="9"
          refY="5"
          markerWidth="5"
          markerHeight="5"
          orient="auto"
        >
          <path d="M 0 0 L 10 5 L 0 10 z" fill="currentColor" />
        </marker>
      </defs>
      {connections.map((w) => {
        const a = endpoint(w.from),
          b = endpoint(w.to);
        if (!a || !b || (w.cable_id && !byPlacement.has(w.cable_id)))
          return null;
        const hot =
          active.has(w.cable_id ?? "") ||
          (!w.cable_id && active.has(w.from.split(":")[0] ?? ""));
        const middle = (a.x + b.x) / 2;
        const path = w.cable_id
          ? `M${a.x},${a.y} C${middle},${a.y} ${middle},${b.y} ${b.x},${b.y}`
          : `M${a.x},${a.y} L${b.x},${b.y}`;
        return (
          <g
            key={w.id}
            data-connection-id={w.id}
            className={hot ? "text-primary" : "text-muted-foreground"}
          >
            {hot && (
              <path
                d={path}
                fill="none"
                stroke="currentColor"
                strokeWidth={12}
                opacity={0.12}
              />
            )}
            <path
              d={path}
              fill="none"
              stroke="currentColor"
              strokeWidth={hot ? 3 : 2}
              strokeDasharray={hot ? "6 3" : undefined}
              markerEnd={`url(#${id}-arrow)`}
            />
            {!miniature && w.cable_id && (
              <text
                x={middle}
                y={(a.y + b.y) / 2 - 9}
                textAnchor="middle"
                className="fill-foreground text-[10px] font-mono"
              >
                {w.cable_id.replace("cable", "#")}
              </text>
            )}
          </g>
        );
      })}
      {placements
        .filter((p) => byPart.get(p.part_id)?.kind !== "cable")
        .map((p) => {
          const part = byPart.get(p.part_id);
          if (!part) return null;
          const x = p.x * unit + margin,
            y = p.y * unit + margin;
          const hot = active.has(p.id);
          const bottomCenterPort = part.ports.some(
            (port) => port.x === 2 && port.y === 3,
          );
          return (
            <g
              key={p.id}
              data-module-id={p.id}
              transform={`translate(${x},${y})`}
            >
              {hot && (
                <rect
                  x={-8}
                  y={-8}
                  width={unit * 4 + 16}
                  height={unit * 3 + 16}
                  rx={16}
                  fill="none"
                  stroke="currentColor"
                  strokeWidth={2}
                  strokeDasharray="5 4"
                  className="text-primary"
                />
              )}
              <rect
                x={0}
                y={0}
                width={unit * 4}
                height={unit * 3}
                rx={10}
                className={`circuit-module-${part.kind}`}
                stroke="currentColor"
                strokeWidth={1.4}
              />
              <rect
                x={8}
                y={8}
                width={unit * 4 - 16}
                height={unit * 3 - 16}
                rx={5}
                className="fill-card"
              />
              {!schematic && <CircuitModuleFace part={part} />}
              <text
                x={unit * 2}
                y={schematic ? unit * 1.2 : 65}
                textAnchor="middle"
                className="fill-foreground text-[17px] font-black"
              >
                {part.marking ?? part.id}
              </text>
              <text
                x={unit * 2 + (!schematic && bottomCenterPort ? 10 : 0)}
                y={schematic ? unit * 2.05 : 112}
                textAnchor={!schematic && bottomCenterPort ? "start" : "middle"}
                className="fill-muted-foreground text-[11px] font-mono"
              >
                {part.id}
              </text>
              {part.ports.map((port) => (
                <g
                  key={port.id}
                  transform={`translate(${port.x * unit},${port.y * unit})`}
                >
                  {hotPorts.has(`${p.id}:${port.id}`) && (
                    <circle
                      r={12}
                      className="fill-primary/15 stroke-primary"
                      strokeWidth={2}
                    />
                  )}
                  <rect
                    x={-4}
                    y={-6}
                    width={8}
                    height={12}
                    rx={2}
                    className="fill-card stroke-foreground"
                  />
                  {!miniature && (
                    <text
                      x={port.x === 0 ? 8 : port.x === 4 ? -8 : 8}
                      y={port.y === 3 ? 18 : -10}
                      textAnchor={
                        port.x === 0 ? "start" : port.x === 4 ? "end" : "start"
                      }
                      className="fill-foreground text-[10px] font-mono"
                    >
                      {port.id}
                    </text>
                  )}
                </g>
              ))}
            </g>
          );
        })}
    </svg>
  );
}
