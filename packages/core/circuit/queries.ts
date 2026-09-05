import {
  queryOptions,
  useMutation,
  useQueryClient,
} from "@tanstack/react-query";
import { api } from "../api";
import type { CreateCircuitInput, CircuitProgressInput } from "./schemas";

export const circuitKeys = {
  all: (wsId: string) => ["circuit", wsId] as const,
  catalog: (wsId: string) => [...circuitKeys.all(wsId), "catalog"] as const,
  creations: (wsId: string) => [...circuitKeys.all(wsId), "creations"] as const,
  creation: (wsId: string, id: string) =>
    [...circuitKeys.creations(wsId), id] as const,
};
export const circuitCatalogOptions = (wsId: string) =>
  queryOptions({
    queryKey: circuitKeys.catalog(wsId),
    queryFn: () => api.getCircuitCatalog(),
    enabled: wsId.length > 0,
    staleTime: 60_000,
  });
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
  });
}
export function useCircuitProgress(wsId: string, id: string) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (input: CircuitProgressInput) =>
      api.updateCircuitProgress(id, input),
    onSuccess: (v) => {
      client.setQueryData(circuitKeys.creation(wsId, id), v);
      void client.invalidateQueries({
        queryKey: circuitKeys.creations(wsId),
        exact: true,
      });
    },
    onError: () => {
      void client.invalidateQueries({
        queryKey: circuitKeys.creation(wsId, id),
      });
    },
  });
}
