import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../api";
import { useWorkspaceId } from "../hooks";
import { buildKeys } from "./queries";
import { generateUUID } from "../utils";
import type { BrickInventoryItem } from "./types";

export function useCreateBuildSession() {
  const workspaceId = useWorkspaceId();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ prompt, clientRequestId = generateUUID() }: { prompt: string; clientRequestId?: string }) =>
      api.createBuildSession(prompt, clientRequestId),
    onSuccess: (session) => queryClient.setQueryData(buildKeys.session(workspaceId, session.id), session),
  });
}

export function useSubmitBuildAnswers() {
  const workspaceId = useWorkspaceId();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ sessionId, answers }: { sessionId: string; answers: Record<string, string> }) =>
      api.submitBuildAnswers(sessionId, answers),
    onSuccess: (session) => queryClient.setQueryData(buildKeys.session(workspaceId, session.id), session),
  });
}

export function useSaveBrickInventory() {
  const workspaceId = useWorkspaceId();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ expectedRevision, items }: { expectedRevision: number; items: BrickInventoryItem[] }) =>
      api.saveBrickInventory(expectedRevision, items),
    onSuccess: (inventory) => queryClient.setQueryData(buildKeys.inventory(workspaceId), inventory),
    onError: () => queryClient.invalidateQueries({ queryKey: buildKeys.inventory(workspaceId) }),
  });
}

export function useResetBrickInventory() {
  const workspaceId = useWorkspaceId();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (expectedRevision: number) => api.resetBrickInventory(expectedRevision),
    onSuccess: (inventory) => queryClient.setQueryData(buildKeys.inventory(workspaceId), inventory),
    onError: () => queryClient.invalidateQueries({ queryKey: buildKeys.inventory(workspaceId) }),
  });
}

export function useStartLDrawCatalogSync() {
  const workspaceId = useWorkspaceId();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => api.startLDrawCatalogSync(),
    onSuccess: (status) => queryClient.setQueryData(buildKeys.catalogSync(workspaceId), status),
    onError: () => queryClient.invalidateQueries({ queryKey: buildKeys.catalogSync(workspaceId) }),
  });
}
