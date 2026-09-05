import {
  queryOptions,
  useMutation,
  useQueryClient,
} from "@tanstack/react-query";
import { api } from "../api";
import type {
  CreateCircuitInput,
  CircuitProgressInput,
  SaveCircuitInventoryInput,
  CircuitTrialInput,
} from "./schemas";

export const circuitKeys = {
  all: (wsId: string) => ["circuit", wsId] as const,
  catalog: (wsId: string, kitId?: string) =>
    [...circuitKeys.all(wsId), "catalog", kitId ?? "default"] as const,
  kits: (wsId: string) => [...circuitKeys.all(wsId), "kits"] as const,
  inventory: (wsId: string, kitId: string) =>
    [...circuitKeys.all(wsId), "inventory", kitId] as const,
  trials: (wsId: string, id: string) =>
    [...circuitKeys.all(wsId), "trials", id] as const,
  creations: (wsId: string) => [...circuitKeys.all(wsId), "creations"] as const,
  creation: (wsId: string, id: string) =>
    [...circuitKeys.creations(wsId), id] as const,
};
export const circuitCatalogOptions = (wsId: string, kitId?: string) =>
  queryOptions({
    queryKey: circuitKeys.catalog(wsId, kitId),
    queryFn: () => api.getCircuitCatalog(kitId),
    enabled: wsId.length > 0,
    staleTime: 60_000,
  });

export const circuitKitsOptions = (wsId: string) =>
  queryOptions({
    queryKey: circuitKeys.kits(wsId),
    queryFn: () => api.listCircuitKits(),
    enabled: !!wsId,
    staleTime: 60_000,
  });
export const circuitInventoryOptions = (wsId: string, kitId: string) =>
  queryOptions({
    queryKey: circuitKeys.inventory(wsId, kitId),
    queryFn: () => api.getCircuitInventory(kitId),
    enabled: !!wsId && !!kitId,
  });
export const circuitTrialsOptions = (wsId: string, id: string) =>
  queryOptions({
    queryKey: circuitKeys.trials(wsId, id),
    queryFn: () => api.listCircuitTrials(id),
    enabled: !!wsId && !!id,
  });
export function useCircuitInventory(wsId: string, kitId: string) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (input: SaveCircuitInventoryInput) =>
      api.saveCircuitInventory(kitId, input),
    onSuccess: (v) =>
      client.setQueryData(circuitKeys.inventory(wsId, kitId), v),
    onError: () => {
      void client.invalidateQueries({
        queryKey: circuitKeys.inventory(wsId, kitId),
      });
    },
  });
}
export function useCircuitTrial(wsId: string, id: string) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (input: CircuitTrialInput) => api.createCircuitTrial(id, input),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: circuitKeys.trials(wsId, id) });
    },
  });
}
export const circuitListOptions = (wsId: string) =>
  queryOptions({
    queryKey: circuitKeys.creations(wsId),
    queryFn: () => api.listCircuitCreations(),
    enabled: wsId.length > 0,
  });
export const circuitCreationOptions = (wsId: string, id: string) =>
  queryOptions({
    queryKey: circuitKeys.creation(wsId, id),
    queryFn: () => api.getCircuitCreation(id),
    enabled: wsId.length > 0 && id.length > 0,
  });

export function useCreateCircuit(wsId: string) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (input: CreateCircuitInput) => api.createCircuit(input),
    onSuccess: (v) => {
      client.setQueryData(circuitKeys.creation(wsId, v.id), v);
      void client.invalidateQueries({
        queryKey: circuitKeys.creations(wsId),
        exact: true,
      });
    },
    onError: (_error, input) => {
      void client.invalidateQueries({
        queryKey: circuitKeys.inventory(wsId, input.kit_id),
      });
    },
  });
}
export function useCircuitProgress(wsId: string, id: string) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (input: CircuitProgressInput) =>
      api.updateCircuitProgress(id, input),
    onSuccess: async (v) => {
      await Promise.all([
        client.cancelQueries({ queryKey: circuitKeys.creation(wsId, id), exact: true }),
        client.cancelQueries({ queryKey: circuitKeys.creations(wsId), exact: true }),
      ]);
      client.setQueryData(circuitKeys.creation(wsId, id), v);
      client.setQueryData<import("./schemas").CircuitList>(circuitKeys.creations(wsId), old => old && ({
        ...old, creations: old.creations.map(item => item.id === id ? {
          ...item, current_step: v.current_step, observation: v.observation,
        } : item),
      }));
    },
    onError: () => {
      void client.invalidateQueries({
        queryKey: circuitKeys.creation(wsId, id),
      });
    },
  });
}
