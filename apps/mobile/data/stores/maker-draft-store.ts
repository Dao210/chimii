import { create } from "zustand";
import type { BrickInventoryItem } from "@chimii/core/build/types";

export interface MakerDraft {
  prompt?: string;
  buildRequest?: string;
  sessionId?: string;
  kit?: string;
  circuitPrompt?: string;
  circuitRequest?: { signature: string; id: string };
  bricks?: { revision: number; items: BrickInventoryItem[] };
  circuits?: Record<string, { revision: number; quantities: Record<string, number> }>;
}
const EMPTY: MakerDraft = {};
// Only client input lives here. Account + workspace scope prevents draft leakage.
// Mounted tabs retain scroll position; server records stay in React Query.
export const useMakerDraftStore = create<{
  drafts: Record<string, MakerDraft>;
  patch: (scope: string, patch: Partial<MakerDraft>) => void;
  reset: () => void;
}>(set => ({
  drafts: {},
  patch: (scope, patch) => set(state => ({ drafts: { ...state.drafts, [scope]: { ...state.drafts[scope], ...patch } } })),
  reset: () => set({ drafts: {} }),
}));
export const selectMakerDraft = (scope: string) => (state: ReturnType<typeof useMakerDraftStore.getState>) => state.drafts[scope] ?? EMPTY;
