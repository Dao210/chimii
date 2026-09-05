import { infiniteQueryOptions, keepPreviousData, queryOptions } from "@tanstack/react-query";
import { api } from "../api";
import type { BuildCatalogPartFilters } from "./types";

export const buildKeys = {
  all: (workspaceId: string) => ["build", workspaceId] as const,
  catalog: (workspaceId: string) => [...buildKeys.all(workspaceId), "catalog"] as const,
  catalogParts: (workspaceId: string, filters: Omit<BuildCatalogPartFilters, "cursor">) => [...buildKeys.catalog(workspaceId), "parts", filters] as const,
  catalogSync: (workspaceId: string) => [...buildKeys.catalog(workspaceId), "sync"] as const,
  inventory: (workspaceId: string) => [...buildKeys.all(workspaceId), "inventory"] as const,
  session: (workspaceId: string, id: string) => [...buildKeys.all(workspaceId), "session", id] as const,
  creations: (workspaceId: string) => [...buildKeys.all(workspaceId), "creations"] as const,
  creation: (workspaceId: string, id: string) => [...buildKeys.all(workspaceId), "creation", id] as const,
};

export function buildCatalogOptions(workspaceId: string) {
  return queryOptions({
    queryKey: buildKeys.catalog(workspaceId),
    queryFn: () => api.getBuildCatalog(),
    staleTime: 60 * 60 * 1000,
  });
}

export function buildCatalogPartsOptions(
  workspaceId: string,
  filters: Omit<BuildCatalogPartFilters, "cursor">,
) {
  return infiniteQueryOptions({
    queryKey: buildKeys.catalogParts(workspaceId, filters),
    queryFn: ({ pageParam }) => api.listBuildCatalogParts({ ...filters, cursor: pageParam }),
    initialPageParam: "",
    getNextPageParam: (lastPage) => lastPage.next_cursor,
    placeholderData: keepPreviousData,
    enabled: workspaceId.length > 0,
    staleTime: 60 * 60 * 1000,
  });
}

export function ldrawCatalogSyncOptions(workspaceId: string) {
  return queryOptions({
    queryKey: buildKeys.catalogSync(workspaceId),
    queryFn: () => api.getLDrawCatalogSyncStatus(),
    enabled: workspaceId.length > 0,
    refetchInterval: (query) => {
      const status = query.state.data?.status;
      return status === "queued" || status === "running" ? 2_000 : false;
    },
  });
}

export function brickInventoryOptions(workspaceId: string) {
  return queryOptions({
    queryKey: buildKeys.inventory(workspaceId),
    queryFn: () => api.getBrickInventory(),
  });
}

export function buildSessionOptions(workspaceId: string, id: string) {
  return queryOptions({
    queryKey: buildKeys.session(workspaceId, id),
    queryFn: () => api.getBuildSession(id),
    enabled: id.length > 0,
    refetchInterval: (query) => {
      const status = query.state.data?.status;
      return status === "queued" || status === "generating"
        ? Math.min(1_000 + query.state.dataUpdateCount * 400, 3_000) : false;
    },
  });
}

export function buildCreationsOptions(workspaceId: string) {
  return queryOptions({
    queryKey: buildKeys.creations(workspaceId),
    queryFn: () => api.listBuildCreations(),
    select: (data) => data.creations,
  });
}

export function buildCreationOptions(workspaceId: string, id: string) {
  return queryOptions({
    queryKey: buildKeys.creation(workspaceId, id),
    queryFn: () => api.getBuildCreation(id),
    enabled: id.length > 0,
  });
}
