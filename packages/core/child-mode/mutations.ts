import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../api";
import { useAuthStore } from "../auth";
import { useWorkspaceId } from "../hooks";
import { childModeKeys } from "./queries";

export function useCreateChildProfile() {
  const workspaceId = useWorkspaceId();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: { display_name: string; pin: string; avatar_seed?: string }) => api.createChildProfile(data),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: childModeKeys.profiles(workspaceId) }),
  });
}

export function useEnterChildMode() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (profileId: string) => api.enterChildMode(profileId),
    onSuccess: async (response) => {
      if (!response.token) throw new Error("Child session token is missing");
      await useAuthStore.getState().loginWithToken(response.token);
      queryClient.clear();
    },
  });
}

export function useExitChildMode() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (pin: string) => api.exitChildMode(pin),
    onSuccess: async (response) => {
      if (!response.token) throw new Error("Parent session token is missing");
      await useAuthStore.getState().loginWithToken(response.token);
      queryClient.clear();
    },
  });
}
