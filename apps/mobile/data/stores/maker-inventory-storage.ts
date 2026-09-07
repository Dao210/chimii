import * as FileSystem from "expo-file-system/legacy";
import { z } from "zod";

const revision = z.number().int().nonnegative();
const quantity = z.number().int().min(0).max(999);
const inventoryDraftSchema = z.object({
  version: z.literal(1),
  bricks: z
    .object({
      revision,
      items: z
        .array(
          z.object({
            part_id: z.string().min(1),
            color: z.number().int(),
            quantity,
          }),
        )
        .max(10000),
    })
    .optional(),
  circuits: z
    .record(
      z.string(),
      z.object({
        revision,
        quantities: z.record(z.string(), quantity),
      }),
    )
    .optional(),
});

// Reuse the app's async file API. These are unsubmitted edits, never a server cache.
const directory = () => {
  if (!FileSystem.documentDirectory)
    throw new Error("Draft storage unavailable");
  return `${FileSystem.documentDirectory}maker-drafts/`;
};
export async function readInventoryDraft(key: string) {
  const path = `${directory()}${key}.json`;
  if (!(await FileSystem.getInfoAsync(path)).exists) return {};
  const parsed = inventoryDraftSchema.safeParse(
    JSON.parse(await FileSystem.readAsStringAsync(path)),
  );
  if (!parsed.success) return {};
  const { version: _, ...draft } = parsed.data;
  return draft;
}
export async function writeInventoryDraft(
  key: string,
  draft: { bricks?: unknown; circuits?: unknown },
) {
  const path = `${directory()}${key}.json`;
  if (!draft.bricks && !Object.keys(draft.circuits ?? {}).length) {
    await FileSystem.deleteAsync(path, { idempotent: true });
    return;
  }
  const value = inventoryDraftSchema.parse({ version: 1, ...draft });
  await FileSystem.makeDirectoryAsync(directory(), { intermediates: true });
  await FileSystem.writeAsStringAsync(`${path}.tmp`, JSON.stringify(value));
  await FileSystem.moveAsync({ from: `${path}.tmp`, to: path });
}
export async function deleteInventoryDraft(key: string) {
  const path = `${directory()}${key}.json`;
  await FileSystem.deleteAsync(path, { idempotent: true });
  await FileSystem.deleteAsync(`${path}.tmp`, { idempotent: true });
}
