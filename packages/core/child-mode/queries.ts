import { queryOptions } from "@tanstack/react-query";
import { api } from "../api";

export const childModeKeys = {
  mode: () => ["child-mode"] as const,
  profiles: (workspaceId: string) => ["child-profiles", workspaceId] as const,
};

export function childModeOptions() {
  return queryOptions({ queryKey: childModeKeys.mode(), queryFn: () => api.getChildMode(), staleTime: 30_000 });
}

export function childProfilesOptions(workspaceId: string) {
  return queryOptions({ queryKey: childModeKeys.profiles(workspaceId), queryFn: () => api.listChildProfiles() });
}
