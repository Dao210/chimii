import { queryOptions } from "@tanstack/react-query";
import { api } from "../api";

export const buildKeys = {
  all: (workspaceId: string) => ["build", workspaceId] as const,
  session: (workspaceId: string, id: string) => [...buildKeys.all(workspaceId), "session", id] as const,
  creations: (workspaceId: string) => [...buildKeys.all(workspaceId), "creations"] as const,
  creation: (workspaceId: string, id: string) => [...buildKeys.all(workspaceId), "creation", id] as const,
};

export function buildSessionOptions(workspaceId: string, id: string) {
  return queryOptions({
    queryKey: buildKeys.session(workspaceId, id),
    queryFn: () => api.getBuildSession(id),
    enabled: id.length > 0,
    refetchInterval: (query) => {
      const status = query.state.data?.status;
      return status === "queued" || status === "generating" ? 900 : false;
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
