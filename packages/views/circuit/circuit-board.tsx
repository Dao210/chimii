"use client";

import { useId } from "react";
import {
  circuitCoordinate,
  circuitPoint,
  circuitText,
  type CircuitPart,
  type CircuitPlacement,
} from "@chimii/core/circuit";
import { useT } from "../i18n";

interface Props {
  parts: CircuitPart[];
  placements: CircuitPlacement[];
  columns: number;
  rows: number;
  highlighted?: string[];
  layer?: number;
  label: string;
  miniature?: boolean;
}

export function CircuitBoard({
  parts,
  placements,
  columns,
  rows,
  highlighted = [],
  layer = 0,
  label,
  miniature = false,
}: Props) {
  const { i18n } = useT("circuit");
  const titleId = useId();
  const scale = 58,
    margin = 48;
  const px = (x: number) => margin + x * scale;
  const width = (columns - 1) * scale + margin * 2,
    height = (rows - 1) * scale + margin * 2;
  const portLabels = new Map<
    string,
    { x: number; y: number; labels: Set<string>; layers: Set<number> }
  >();
  for (const p of placements) {
    const part = parts.find((v) => v.id === p.part_id);
    if (!part || part.kind === "wire") continue;
    for (const port of part.ports) {
      if (["a", "b"].includes(port.id)) continue;
      const q = circuitPoint(p, port.x, port.y);
      const key = `${q.x}:${q.y}`;
      const entry = portLabels.get(key) ?? {
        ...q,
        labels: new Set<string>(),
        layers: new Set<number>(),
      };
      entry.labels.add(port.id);
      entry.layers.add(p.layer);
      portLabels.set(key, entry);
    }
  }
  return (
    <svg
      role="img"
      aria-labelledby={titleId}
      viewBox={`0 0 ${width} ${height}`}
      className="circuit-board"
      data-testid="circuit-board"
    >
      <title id={titleId}>{label}</title>
      <rect
        x="1"
        y="1"
        width={width - 2}
        height={height - 2}
        rx="20"
        className="circuit-board-base"
      />
      {Array.from({ length: rows }, (_, y) => (
        <g key={`r${y}`}>
          <line
            x1={margin}
            y1={px(y)}
            x2={width - margin}
            y2={px(y)}
            className="circuit-grid-line"
          />
          {!miniature && (
            <text x="18" y={px(y) + 4} className="circuit-grid-label">
              {String.fromCharCode(65 + y)}
            </text>
          )}
          {Array.from({ length: columns }, (_, x) => (
            <circle
              key={x}
              cx={px(x)}
              cy={px(y)}
              r="3"
              className="circuit-grid-dot"
            />
          ))}
        </g>
      ))}
      {Array.from({ length: columns }, (_, x) => (
        <g key={`c${x}`}>
          <line
            x1={px(x)}
            y1={margin}
            x2={px(x)}
            y2={height - margin}
            className="circuit-grid-line"
          />
          {!miniature && (
            <text
              x={px(x)}
              y="23"
              textAnchor="middle"
              className="circuit-grid-label"
            >
              {x + 1}
            </text>
          )}
        </g>
      ))}
      {[...placements]
        .sort((a, b) => a.layer - b.layer)
        .map((p) => {
          const part = parts.find((v) => v.id === p.part_id);
          if (!part) return null;
          const vertices = part.body.map((v) => circuitPoint(p, v.x, v.y));
          const ports = part.ports.map((v) => ({
            ...circuitPoint(p, v.x, v.y),
            id: v.id,
          }));
          const cx =
            (Math.min(...vertices.map((v) => v.x)) +
              Math.max(...vertices.map((v) => v.x))) /
            2;
          const cy =
            (Math.min(...vertices.map((v) => v.y)) +
              Math.max(...vertices.map((v) => v.y))) /
            2;
          const active = highlighted.includes(p.id);
          const dimmed = layer > 0 && p.layer !== layer;
          const points = vertices.map((v) => `${px(v.x)},${px(v.y)}`).join(" ");
          return (
            <g
              key={p.id}
              data-placement={p.id}
              data-layer={p.layer}
              data-highlighted={active}
              className={`circuit-piece circuit-kind-${part.kind}${active ? " circuit-piece-active" : ""}`}
              opacity={dimmed ? 0.18 : 1}
            >
              <title>{`${part.id} · ${circuitText(part.name, i18n.language)} · ${ports.map((v) => `${v.id}: ${circuitCoordinate(v.x, v.y)}`).join(", ")}`}</title>
              {vertices.length === 1 ? (
                <circle
                  cx={px(cx)}
                  cy={px(cy)}
                  r="13"
                  className="circuit-piece-body"
                />
              ) : vertices.length >= 3 ? (
                <polygon
                  points={points}
                  className="circuit-piece-body"
                  strokeLinejoin="round"
                />
              ) : (
                <polyline
                  points={points}
                  className="circuit-piece-body circuit-piece-bar"
                  strokeLinecap="round"
                />
              )}
              {active &&
                (vertices.length === 1 ? (
                  <circle
                    cx={px(cx)}
                    cy={px(cy)}
                    r="16"
                    className="circuit-piece-halo"
                  />
                ) : vertices.length >= 3 ? (
                  <polygon points={points} className="circuit-piece-halo" />
                ) : (
                  <polyline
                    points={points}
                    className="circuit-piece-halo circuit-piece-bar"
                    strokeLinecap="round"
                  />
                ))}
              {ports.map((v) => (
                <g key={v.id}>
                  <circle
                    cx={px(v.x)}
                    cy={px(v.y)}
                    r="9"
                    className="circuit-snap-outer"
                  />
                  <circle
                    cx={px(v.x)}
                    cy={px(v.y)}
                    r="3.5"
                    className="circuit-snap-inner"
                  />
                </g>
              ))}
              <g transform={`translate(${px(cx)},${px(cy)})`}>
                {part.ports.length > 1 && (
                  <>
                    <rect
                      x="-15"
                      y="-11"
                      width="30"
                      height="22"
                      rx="6"
                      className="circuit-part-label-bg"
                    />
                    <text
                      textAnchor="middle"
                      y="5"
                      className="circuit-part-label"
                    >
                      {part.id}
                    </text>
                  </>
                )}
              </g>
            </g>
          );
        })}
      {!miniature &&
        [...portLabels].map(([key, v]) => (
          <text
            key={key}
            x={px(v.x) + 12}
            y={px(v.y) - 12}
            opacity={layer > 0 && !v.layers.has(layer) ? 0.18 : 1}
            className="circuit-port-label"
          >
            {[...v.labels].join("/")}
          </text>
        ))}
    </svg>
  );
}
