import type { BrickInventoryItem } from "@chimii/core/build/types";
export const makerScope = (
  user: string | null | undefined,
  workspace: string | null,
) => JSON.stringify([user, workspace]);
export function setBrickQuantity(
  items: BrickInventoryItem[],
  part: string,
  color: number,
  quantity: number,
): BrickInventoryItem[] {
  const count = Math.max(0, Math.min(999, Math.trunc(quantity)));
  const matches = (v: BrickInventoryItem) =>
    v.part_id === part && v.color === color;
  if (count === 0) return items.filter((v) => !matches(v));
  if (items.some(matches))
    return items.map((v) => (matches(v) ? { ...v, quantity: count } : v));
  return [...items, { part_id: part, color, quantity: count }];
}
// Empty text is an unfinished edit, not a request to remove an inventory row.
export function quantityInput(text: string, previous: number, max: number) {
  const digits = text.replace(/[^0-9]/g, "");
  return digits ? Math.min(max, Number(digits)) : previous;
}
export function sameBrickQuantities(
  a: BrickInventoryItem[],
  b: BrickInventoryItem[],
) {
  if (a.length !== b.length) return false;
  const quantities = new Map(
    a.map((v) => [`${v.part_id}:${v.color}`, v.quantity]),
  );
  return b.every(
    (v) => quantities.get(`${v.part_id}:${v.color}`) === v.quantity,
  );
}
export function sameCircuitQuantities(
  a: Record<string, number>,
  b: Record<string, number>,
) {
  return [...new Set([...Object.keys(a), ...Object.keys(b)])].every(
    (id) => (a[id] ?? 0) === (b[id] ?? 0),
  );
}
// Preview can revisit saved steps but never move or clear canonical progress.
// Mirrors the preview/progress separation in the web Build and Circuit workbenches.
export function guideNavigation(
  preview: number | null,
  saved: number,
  last: number,
  completed = false,
) {
  const step = Math.max(0, Math.min(preview ?? saved, saved, last));
  const next =
    step < saved || completed
      ? step < last
        ? "preview"
        : "done"
      : step < last
        ? "save"
        : "complete";
  return { step, next, nextStep: Math.min(last, step + 1) } as const;
}
export function missingBuildSession(error: unknown) {
  return (error as { status?: number } | null)?.status === 404;
}
export function makerError(error: unknown) {
  const e = error as { status?: number; message?: string } | null;
  if (e?.status === 409)
    return "其他设备已更新这份数据。请刷新后确认最新内容，再保存。你的修改仍保留在本机。";
  if (e?.status === 403)
    return "当前账号没有操作权限，请让家长或空间管理员处理。";
  if (e?.status === 404) return "找不到这件作品，或它已被移除。";
  if (e?.status === 503) return "生成服务暂时不可用，请稍后再试。";
  return "暂时没有完成，请检查网络后重试。已保存的作品和进度仍然保留。";
}

const COLORS: Record<number, string> = {
  1: "蓝色",
  2: "绿色",
  4: "红色",
  14: "黄色",
  15: "白色",
  71: "浅灰色",
};
export const brickColorName = (color: number) =>
  COLORS[color] ?? `颜色 ${color}`;
