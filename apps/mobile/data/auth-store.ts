import { resetSessionState } from "./session-state";
/**
 * Mobile auth store — Zustand. Logic mirrors packages/core/auth/store.ts:
 *   - Token written ONLY on successful verifyCode
 *   - 401 → clear token; non-401 (5xx / network blip) → preserve token so
 *     the next launch can retry
 *   - logout = clear token + clear in-memory user + setToken(null)
 *
 * NOT shared with web/desktop (per Sharing Principles in root CLAUDE.md).
 * Storage backend is expo-secure-store (mobile only); web uses HttpOnly
 * cookies, desktop uses localStorage via StorageAdapter.
 */
import { queryClient } from "./query-client";
import { create } from "zustand";
import type { User } from "@chimii/core/types";

import { api, ApiError } from "./api";
import { clearToken, getToken, setToken } from "./secure-storage";
import { useWorkspaceStore } from "./workspace-store";

interface AuthState {
  user: User | null;
  isLoading: boolean;
  initializationError: string | null;
  initialize: () => Promise<void>;
  sendCode: (email: string) => Promise<void>;
  verifyCode: (email: string, code: string) => Promise<User>;
  logout: () => Promise<void>;
  /** Overwrite the in-memory user — call after PATCH /api/me so name/avatar
   *  edits land without a refetch. Server response is the source of truth. */
  setUser: (user: User) => void;
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  isLoading: true,
  initializationError: null,

  initialize: async () => {
    set({ isLoading: true, initializationError: null });
    try {
      await useWorkspaceStore.getState().restoreSlug();
      const token = await getToken();
      api.setToken(token);
      if (!token) { set({ user: null }); return; }
      const user = await api.getMe();
      if (!user.id) throw new Error("Account response invalid");
      set({ user });
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        await useAuthStore.getState().logout();
      } else {
        // Preserve the saved session on a network/storage failure. A retry
        // screen keeps deep links in place and avoids an incorrect login redirect.
        set({ initializationError: "Unable to restore your session. Check your connection and retry." });
      }
    } finally {
      set({ isLoading: false });
    }
  },

  sendCode: async (email) => {
    await api.sendCode(email);
  },

  verifyCode: async (email, code) => {
    const { token, user } = await api.verifyCode(email, code);
    await setToken(token);
    api.setToken(token);
    set({ user });
    return user;
  },

  logout: async () => {
    api.setToken(null);
    set({ user: null, initializationError: null, isLoading: false });
    await queryClient.cancelQueries();
    queryClient.clear();
    resetSessionState();
    await Promise.all([clearToken(), useWorkspaceStore.getState().clear()]);
  },

  setUser: (user) => set({ user }),
}));
