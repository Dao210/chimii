import { queryOptions } from "@tanstack/react-query";
import { api } from "@/data/api";

export const creationKeys = {
  list: (wsId: string | null, kind: "build" | "circuit") => ["creations", wsId, kind] as const,
};
export const brickCreationsOptions = (wsId: string | null, enabled = true) => queryOptions({
  queryKey: creationKeys.list(wsId, "build"),
  queryFn: ({ signal }) => api.listBuildSummaries({ signal }),
  enabled: !!wsId && enabled,
});
export const circuitCreationsOptions = (wsId: string | null, enabled = true) => queryOptions({
  queryKey: creationKeys.list(wsId, "circuit"),
  queryFn: ({ signal }) => api.listCircuitCreations({ signal }),
  enabled: !!wsId && enabled,
});
