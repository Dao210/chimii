import { create } from "zustand";
import type { BrickInventoryItem } from "@chimii/core/build/types";
import * as SecureStore from "expo-secure-store";

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
const pending = new Map<string, ReturnType<typeof setTimeout>>();
let storageQueue = Promise.resolve();
const storageKey = (scope: string) =>
  `chimii_maker_${Array.from(scope)
    .map((c) => c.charCodeAt(0).toString(16))
    .join("")}`;
function persist(scope: string, value: MakerDraft) {
  const old = pending.get(scope);
  if (old) clearTimeout(old);
  // Only the short idea and idempotency/session pointers survive a cold start.
  // Full inventory drafts remain in memory; server records are always re-fetched.
  const snapshot = JSON.stringify({
    prompt: value.prompt,
    buildRequest: value.buildRequest,
    sessionId: value.sessionId,
    circuitPrompt: value.circuitPrompt,
    kit: value.kit,
    circuitCreationId: value.circuitCreationId,
  });
  pending.set(
    scope,
    setTimeout(() => {
      pending.delete(scope);
      storageQueue = storageQueue
        .then(() => SecureStore.setItemAsync(storageKey(scope), snapshot))
        .catch(() => {
          /* In-memory drafts remain available. */
        });
    }, 350),
  );
}
export async function restoreMakerDraft(scope: string) {
  if (restoring.has(scope)) return;
  restoring.add(scope);
  try {
    const raw = await SecureStore.getItemAsync(storageKey(scope));
    if (
      !restoring.has(scope) ||
      !raw ||
      useMakerDraftStore.getState().drafts[scope]
    )
      return;
    const parsed: unknown = JSON.parse(raw);
    if (!parsed || typeof parsed !== "object") return;
    const value = parsed as Record<string, unknown>;
    const clean: MakerDraft = {};
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
    // Request payload signatures can be larger than secure storage's portable
    // limit, so retain circuit retries in memory rather than persist inventories.
    useMakerDraftStore.setState((s) => ({
      drafts: { ...s.drafts, [scope]: clean },
    }));
  } catch {
    /* A damaged local draft does not block the account. */
  }
}
// Only client input lives here. Account + workspace scope prevents draft leakage.
// Mounted tabs retain scroll position; server records stay in React Query.
export const useMakerDraftStore = create<{
  drafts: Record<string, MakerDraft>;
  patch: (scope: string, patch: Partial<MakerDraft>) => void;
  reset: () => void;
}>((set) => ({
  drafts: {},
  patch: (scope, patch) =>
    set((state) => {
      const value = { ...state.drafts[scope], ...patch };
      if (
        [
          "prompt",
          "buildRequest",
          "sessionId",
          "circuitPrompt",
          "kit",
          "circuitCreationId",
        ].some((key) => key in patch)
      )
        persist(scope, value);
      return { drafts: { ...state.drafts, [scope]: value } };
    }),
  reset: () => {
    for (const timer of pending.values()) clearTimeout(timer);
    pending.clear();
    for (const scope of restoring)
      storageQueue = storageQueue
        .then(() => SecureStore.deleteItemAsync(storageKey(scope)))
        .catch(() => {});
    restoring.clear();
    set({ drafts: {} });
  },
}));
export const selectMakerDraft =
  (scope: string) => (state: ReturnType<typeof useMakerDraftStore.getState>) =>
    state.drafts[scope] ?? EMPTY;
