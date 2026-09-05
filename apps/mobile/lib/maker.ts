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
  const next = items.filter((v) => v.part_id !== part || v.color !== color);
  const count = Math.max(0, Math.min(999, Math.trunc(quantity)));
  return count > 0
    ? [...next, { part_id: part, color, quantity: count }]
    : next;
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
