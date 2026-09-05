import {
  infiniteQueryOptions,
  queryOptions,
  useMutation,
  useQueryClient,
} from "@tanstack/react-query";
import { api } from "@/data/api";
import { creationKeys } from "./creations";
import type { BrickInventoryItem } from "@chimii/core/build/types";
import type {
  CircuitProgressInput,
  CreateCircuitInput,
  SaveCircuitInventoryInput,
} from "@chimii/core/circuit/schemas";

export const makerKeys = {
  root: (ws: string | null) => ["maker", ws] as const,
  item: (ws: string | null, resource: string, id = "") =>
    ["maker", ws, resource, id] as const,
};
export const catalogOptions = (ws: string | null, enabled = true) =>
  queryOptions({
    queryKey: makerKeys.item(ws, "catalog"),
    queryFn: ({ signal }) => api.buildCatalog({ signal }),
    enabled: !!ws && enabled,
    staleTime: 300_000,
  });
export const partsOptions = (
  ws: string | null,
  search: string,
  enabled = true,
) =>
  infiniteQueryOptions({
    queryKey: makerKeys.item(ws, "parts", search),
    initialPageParam: "",
    queryFn: ({ signal, pageParam }) =>
      api.buildParts(search, pageParam, { signal }),
    getNextPageParam: (last) => last.next_cursor || undefined,
    enabled: !!ws && enabled,
    staleTime: 300_000,
  });
export const inventoryOptions = (ws: string | null, enabled = true) =>
  queryOptions({
    queryKey: makerKeys.item(ws, "inventory"),
    queryFn: ({ signal }) => api.brickInventory({ signal }),
    enabled: !!ws && enabled,
  });
export const buildSessionOptions = (
  ws: string | null,
  id: string,
  enabled = true,
) =>
  queryOptions({
    queryKey: makerKeys.item(ws, "session", id),
    queryFn: ({ signal }) => api.buildSession(id, { signal }),
    enabled: !!ws && !!id && enabled,
    refetchInterval: (q) =>
      ["queued", "generating"].includes(q.state.data?.status ?? "")
        ? 2000
        : false,
  });
export const buildDetailOptions = (ws: string | null, id: string) =>
  queryOptions({
    queryKey: makerKeys.item(ws, "build", id),
    queryFn: ({ signal }) => api.buildCreation(id, { signal }),
    enabled: !!ws && !!id,
    staleTime: 300_000,
  });
export const buildProgressOptions = (ws: string | null, id: string) =>
  queryOptions({
    queryKey: makerKeys.item(ws, "progress", id),
    queryFn: ({ signal }) => api.buildProgress(id, { signal }),
    enabled: !!ws && !!id,
  });
export const kitsOptions = (ws: string | null, enabled = true) =>
  queryOptions({
    queryKey: makerKeys.item(ws, "kits"),
    queryFn: ({ signal }) => api.circuitKits({ signal }),
    enabled: !!ws && enabled,
    staleTime: 300_000,
  });
export const circuitCatalogOptions = (
  ws: string | null,
  kit: string,
  enabled = true,
) =>
  queryOptions({
    queryKey: makerKeys.item(ws, "circuit-catalog", kit),
    queryFn: ({ signal }) => api.circuitCatalog(kit, { signal }),
    enabled: !!ws && !!kit && enabled,
    staleTime: 300_000,
  });
export const circuitInventoryOptions = (
  ws: string | null,
  kit: string,
  enabled = true,
) =>
  queryOptions({
    queryKey: makerKeys.item(ws, "circuit-inventory", kit),
    queryFn: ({ signal }) => api.circuitInventory(kit, { signal }),
    enabled: !!ws && !!kit && enabled,
  });
export const circuitDetailOptions = (ws: string | null, id: string) =>
  queryOptions({
    queryKey: makerKeys.item(ws, "circuit", id),
    queryFn: ({ signal }) => api.circuitCreation(id, { signal }),
    enabled: !!ws && !!id,
  });
export const makerConfigOptions = (ws: string | null) =>
  queryOptions({
    queryKey: makerKeys.item(ws, "config"),
    queryFn: ({ signal }) => api.getConfig({ signal }),
    enabled: !!ws,
    staleTime: 60_000,
  });
// Cancel reads before writes; only a canonical server response advances progress.
export function useBuildActions(ws: string | null, id = "") {
  const qc = useQueryClient();
  const sessionKey = makerKeys.item(ws, "session", id);
  const session = useMutation({
    mutationFn: (input: { prompt: string; client_request_id: string }) =>
      api.createBuild(input),
    onSuccess: (data) => {
      qc.setQueryData(makerKeys.item(ws, "session", data.id), data);
    },
  });
  const answer = useMutation({
    mutationFn: (input: {
      answers: Record<string, string>;
      revision: number;
    }) => api.answerBuild(id, input),
    onMutate: () => qc.cancelQueries({ queryKey: sessionKey }),
    onSuccess: (data) => {
      qc.setQueryData(sessionKey, data);
    },
    onError: () => {
      void qc.invalidateQueries({ queryKey: sessionKey });
    },
  });
  const cancel = useMutation({
    mutationFn: (revision: number) => api.cancelBuild(id, revision),
    onSuccess: (data) => {
      qc.setQueryData(sessionKey, data);
    },
  });
  return { session, answer, cancel };
}
export function useSaveInventory(ws: string | null) {
  const qc = useQueryClient();
  const key = makerKeys.item(ws, "inventory");
  return useMutation({
    mutationFn: (input: {
      expected_revision: number;
      items: BrickInventoryItem[];
    }) => api.saveBrickInventory(input),
    onMutate: () => qc.cancelQueries({ queryKey: key }),
    onSuccess: (data) => {
      qc.setQueryData(key, data);
    },
    onError: () => {
      void qc.invalidateQueries({ queryKey: key });
    },
  });
}
export function useSaveCircuitInventory(ws: string | null, kit: string) {
  const qc = useQueryClient();
  const key = makerKeys.item(ws, "circuit-inventory", kit);
  return useMutation({
    mutationFn: (input: SaveCircuitInventoryInput) =>
      api.saveCircuitInventory(kit, input),
    onMutate: () => qc.cancelQueries({ queryKey: key }),
    onSuccess: (data) => {
      qc.setQueryData(key, data);
    },
    onError: () => {
      void qc.invalidateQueries({ queryKey: key });
    },
  });
}
export function useCreateCircuit(ws: string | null) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: CreateCircuitInput) => api.createCircuit(input),
    onSuccess: (data) => {
      qc.setQueryData(makerKeys.item(ws, "circuit", data.id), data);
      void qc.invalidateQueries({ queryKey: creationKeys.list(ws, "circuit") });
    },
  });
}
export function useBuildProgress(ws: string | null, id: string) {
  const qc = useQueryClient();
  const key = makerKeys.item(ws, "progress", id);
  return useMutation({
    mutationFn: (input: {
      current_step: number;
      expected_revision: number;
      completed?: boolean;
    }) => api.saveBuildProgress(id, input),
    onMutate: () => qc.cancelQueries({ queryKey: key }),
    onSuccess: (data) => {
      qc.setQueryData(key, data);
      void qc.invalidateQueries({ queryKey: creationKeys.list(ws, "build") });
    },
    onError: () => {
      void qc.invalidateQueries({ queryKey: key });
    },
  });
}
export function useCircuitProgress(ws: string | null, id: string) {
  const qc = useQueryClient();
  const key = makerKeys.item(ws, "circuit", id);
  return useMutation({
    mutationFn: (input: CircuitProgressInput) =>
      api.saveCircuitProgress(id, input),
    onMutate: () => qc.cancelQueries({ queryKey: key }),
    onSuccess: (data) => {
      qc.setQueryData(key, data);
      void qc.invalidateQueries({ queryKey: creationKeys.list(ws, "circuit") });
    },
    onError: () => {
      void qc.invalidateQueries({ queryKey: key });
    },
  });
}
