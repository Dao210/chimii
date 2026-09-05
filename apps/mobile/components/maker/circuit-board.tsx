import { useMemo } from "react";
import { View } from "react-native";
import Svg, { Circle, G, Line, Polygon, Text as SvgText } from "react-native-svg";
import type { CircuitDocument } from "@chimii/core/circuit/schemas";
import { circuitPoint, circuitCoordinate } from "@chimii/core/circuit/geometry";
import { Text } from "@/components/ui/text";
import { useMaker } from "./shared";
export function CircuitBoard({ document: doc, step }: { document: CircuitDocument; step?: number }) {
 const { theme } = useMaker(); const unit = 34; const margin = 32;
 const byPart = useMemo(() => new Map(doc.parts.map(v => [v.id, v])), [doc.parts]);
 const visible = useMemo(() => new Set((step === undefined ? doc.project.steps : doc.project.steps.slice(0, step + 1)).flatMap(v => v.placement_ids)), [doc.project.steps, step]);
 const hot = new Set(step === undefined ? [] : doc.project.steps[step]?.placement_ids);
 const placements = doc.project.placements.filter(v => visible.has(v.id)).sort((a, b) => a.layer - b.layer);
 const endpoint = (key: string) => { const [id, port] = key.split(":"); const p = placements.find(v => v.id === id); const pin = p && byPart.get(p.part_id)?.ports.find(v => v.id === port); return p && pin ? circuitPoint(p, pin.x, pin.y) : null; };
 const projectPoint = (v: { x: number; y: number }) => ({ x: margin + v.x * unit, y: margin + v.y * unit });
 return <View className="rounded-xl bg-secondary p-2 gap-2"><Svg width="100%" height={290} viewBox={`0 0 ${(doc.columns - 1) * unit + margin * 2} ${(doc.rows - 1) * unit + margin * 2}`} accessibilityLabel="当前步骤电路连接图">
  {Array.from({ length: doc.rows }, (_, y) => Array.from({ length: doc.columns }, (_, x) => <G key={`${x}:${y}`}><Circle cx={margin + x * unit} cy={margin + y * unit} r={2} fill={theme.border} />{x === 0 && <SvgText x={10} y={margin + y * unit + 4} fontSize={10} fill={theme.mutedForeground}>{String.fromCharCode(65 + y)}</SvgText>}{y === 0 && <SvgText x={margin + x * unit} y={14} fontSize={10} fill={theme.mutedForeground} textAnchor="middle">{x + 1}</SvgText>}</G>))}
  {placements.map(p => { const part = byPart.get(p.part_id); if (!part || part.kind === "cable") return null; const center = projectPoint(p); const points = part.body.map(v => projectPoint(circuitPoint(p, v.x, v.y))).map(v => `${v.x},${v.y}`).join(" "); return <G key={p.id}><Polygon points={points} fill={hot.has(p.id) ? theme.warning : theme.card} stroke={theme.foreground} strokeWidth={hot.has(p.id) ? 2.5 : 1.2} />{part.ports.map(port => { const point = projectPoint(circuitPoint(p, port.x, port.y)); return <G key={port.id}><Circle cx={point.x} cy={point.y} r={5} fill={theme.card} stroke={theme.foreground} strokeWidth={1.5} /><SvgText x={point.x + 7} y={point.y - 6} fontSize={8} fill={theme.foreground}>{port.id}</SvgText></G>; })}<SvgText x={center.x} y={center.y + 4} fontSize={10} fontWeight="bold" textAnchor="middle" fill={theme.foreground}>{part.manufacturer_id || part.id}</SvgText></G>; })}
  {doc.project.connections?.map(w => { const a = endpoint(w.from), b = endpoint(w.to); if (!a || !b || w.cable_id && !visible.has(w.cable_id)) return null; const from = projectPoint(a), to = projectPoint(b); return <Line key={w.id} x1={from.x} y1={from.y} x2={to.x} y2={to.y} stroke={w.cable_id && hot.has(w.cable_id) ? theme.warning : theme.primary} strokeWidth={4} />; })}
 </Svg><Text className="text-xs text-muted-foreground text-center">{doc.connection_system === "boson" ? "模块连接示意，不代表实际安装尺寸" : "按坐标摆放，检查层数和端口"} · 黄色为本步新增</Text>
 {step !== undefined && placements.filter(p => hot.has(p.id)).map(p => <Text key={p.id} className="text-sm">{byPart.get(p.part_id)?.name.zh} · {circuitCoordinate(p.x, p.y)} · 第 {p.layer} 层 · {p.rotation}°</Text>)}
 </View>;
}
