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
    mutationFn: ({ prompt, clientRequestId = generateUUID(), designInput }: { prompt: string; clientRequestId?: string; designInput?: import("./types").BuildDesignInput }) =>
      api.createBuildSession(prompt, clientRequestId, designInput),
    onSuccess: (session) => queryClient.setQueryData(buildKeys.session(workspaceId, session.id), session),
  });
}

export function useSubmitBuildAnswers() {
  const workspaceId = useWorkspaceId();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ sessionId, answers, revision }: { sessionId: string; answers: Record<string, string>; revision?: number }) =>
      api.submitBuildAnswers(sessionId, answers, revision),
    onSuccess: (session) => queryClient.setQueryData(buildKeys.session(workspaceId, session.id), session),
    onError: (_error, variables) => queryClient.invalidateQueries({ queryKey: buildKeys.session(workspaceId, variables.sessionId) }),
  });
}

export function useCancelBuildSession() {
  const workspaceId = useWorkspaceId();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ sessionId, revision }: { sessionId: string; revision: number }) => api.cancelBuildSession(sessionId, revision),
    onSuccess: (session) => queryClient.setQueryData(buildKeys.session(workspaceId, session.id), session),
    onError: (_error, variables) => queryClient.invalidateQueries({ queryKey: buildKeys.session(workspaceId, variables.sessionId) }),
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

export function useSaveBuildProgress(wsId: string, id: string) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (input: import("./schemas").BuildProgressInput) => api.updateBuildProgress(id, input),
    onSuccess: async (progress) => {
      await Promise.all([
        client.cancelQueries({ queryKey: buildKeys.progress(wsId, id), exact: true }),
        client.cancelQueries({ queryKey: buildKeys.summaries(wsId), exact: true }),
      ]);
      client.setQueryData(buildKeys.progress(wsId, id), progress);
      client.setQueryData<{ creations: import("./schemas").BuildSummary[] }>(buildKeys.summaries(wsId), old => old && ({
        ...old, creations: old.creations.map(item => item.id === id ? { ...item, progress } : item),
      }));
    },
    onError: () => { void client.invalidateQueries({ queryKey: buildKeys.progress(wsId, id) }); },
  });
}
