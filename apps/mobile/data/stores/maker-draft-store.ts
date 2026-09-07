import { create } from "zustand";
import type { BrickInventoryItem } from "@chimii/core/build/types";
import * as SecureStore from "expo-secure-store";
import {
  readInventoryDraft,
  writeInventoryDraft,
  deleteInventoryDraft,
} from "./maker-inventory-storage";

export interface MakerDraft {
  prompt?: string;
  buildRequest?: string;
  sessionId?: string;
  kit?: string;
  circuitPrompt?: string;
  circuitCreationId?: string;
  circuitRequest?: { signature: string; id: string };
  bricks?: { revision: number; items: BrickInventoryItem[] };
  circuits?: Record<
    string,
    { revision: number; quantities: Record<string, number> }
  >;
}
const EMPTY: MakerDraft = {};
const restoring = new Set<string>();
const restorationTasks = new Map<string, Promise<void>>();
const pending = new Map<string, ReturnType<typeof setTimeout>>();
let storageQueue = Promise.resolve();
let epoch = 0;
const storageKey = (scope: string) =>
  `chimii_maker_${Array.from(scope)
    .map((c) => c.charCodeAt(0).toString(16))
    .join("")}`;
function persist(scope: string, value: MakerDraft) {
  const old = pending.get(scope);
  if (old) clearTimeout(old);
  const generation = epoch;
  // Keep small session pointers in SecureStore and larger unsaved edits in a file.
  pending.set(
    scope,
    setTimeout(() => {
      pending.delete(scope);
      storageQueue = storageQueue
        .then(async () => {
          await restorationTasks.get(scope);
          if (generation !== epoch) return;
          const latest = useMakerDraftStore.getState().drafts[scope] ?? value;
          const snapshot = JSON.stringify({
            prompt: latest.prompt,
            buildRequest: latest.buildRequest,
            sessionId: latest.sessionId,
            circuitPrompt: latest.circuitPrompt,
            kit: latest.kit,
            circuitCreationId: latest.circuitCreationId,
          });
          const results = await Promise.allSettled([
            SecureStore.setItemAsync(storageKey(scope), snapshot),
            writeInventoryDraft(storageKey(scope), {
              bricks: latest.bricks,
              circuits: latest.circuits,
            }),
          ]);
          if (generation === epoch)
            useMakerDraftStore.setState((s) => ({
              storageErrors: {
                ...s.storageErrors,
                [scope]: results.some((r) => r.status === "rejected"),
              },
            }));
        })
        .catch(() => {
          /* In-memory drafts remain available. */
        });
    }, 350),
  );
}
export function restoreMakerDraft(scope: string) {
  const existing = restorationTasks.get(scope);
  if (existing) return existing;
  const task = hydrateMakerDraft(scope);
  restorationTasks.set(scope, task);
  return task;
}
async function hydrateMakerDraft(scope: string) {
  if (restoring.has(scope)) return;
  restoring.add(scope);
  const generation = epoch;
  try {
    const [stored, inventory] = await Promise.allSettled([
      SecureStore.getItemAsync(storageKey(scope)),
      readInventoryDraft(storageKey(scope)),
    ]);
    if (generation !== epoch) return;
    const clean: MakerDraft = {};
    let value: Record<string, unknown> = {};
    if (stored.status === "fulfilled" && stored.value) {
      try {
        const parsed: unknown = JSON.parse(stored.value);
        if (parsed && typeof parsed === "object")
          value = parsed as Record<string, unknown>;
      } catch {
        /* Invalid short pointers do not discard a valid inventory draft. */
      }
    }
    for (const key of [
      "prompt",
      "buildRequest",
      "sessionId",
      "circuitPrompt",
      "kit",
      "circuitCreationId",
    ] as const) {
      if (typeof value[key] === "string") clean[key] = value[key].slice(0, 280);
    }
    if (inventory.status === "fulfilled") Object.assign(clean, inventory.value);
    // In-flight user edits win per field, so typing an idea does not discard
    // an unrelated inventory draft restored at the same time.
    useMakerDraftStore.setState((s) => ({
      drafts: { ...s.drafts, [scope]: { ...clean, ...s.drafts[scope] } },
    }));
  } catch {
    /* A damaged local draft does not block the account. */
  }
}
// Only client input lives here. Account + workspace scope prevents draft leakage.
// Mounted tabs retain scroll position; server records stay in React Query.
export const useMakerDraftStore = create<{
  drafts: Record<string, MakerDraft>;
  storageErrors: Record<string, boolean>;
  patch: (scope: string, patch: Partial<MakerDraft>) => void;
  reset: () => void;
}>((set) => ({
  drafts: {},
  storageErrors: {},
  patch: (scope, patch) =>
    set((state) => {
      const value = { ...state.drafts[scope], ...patch };
      persist(scope, value);
      return { drafts: { ...state.drafts, [scope]: value } };
    }),
  reset: () => {
    epoch++;
    for (const timer of pending.values()) clearTimeout(timer);
    pending.clear();
    for (const scope of new Set([
      ...restoring,
      ...Object.keys(useMakerDraftStore.getState().drafts),
    ]))
      storageQueue = storageQueue
        .then(async () => {
          await Promise.allSettled([
            SecureStore.deleteItemAsync(storageKey(scope)),
            deleteInventoryDraft(storageKey(scope)),
          ]);
        })
        .catch(() => {});
    restoring.clear();
    restorationTasks.clear();
    set({ drafts: {}, storageErrors: {} });
  },
}));
export const selectMakerDraft =
  (scope: string) => (state: ReturnType<typeof useMakerDraftStore.getState>) =>
    state.drafts[scope] ?? EMPTY;
