import { queryOptions } from "@tanstack/react-query";
import type { BuildPlan } from "@chimii/core/build/types";
import { api } from "@/data/api";
import { makerKeys } from "./maker";
import { LDRAW_CATALOG, LDRAW_CATALOG_VERSION } from "@/lib/maker-catalog.generated";
// Imported only by the lazy model viewer; the startup graph excludes GLB data.
export const modelAssetsOptions = (ws: string | null, plan: BuildPlan) => queryOptions({
    queryKey: makerKeys.item(ws, "model-assets", plan.content_hash),
    queryFn: async ({ signal }) => {
      const ids = [
        ...new Set(plan.placements.map((p) => plan.parts[p.part_id]!.ldraw_id)),
      ];
      const result: Record<string, string> = {};
      // Four bounded requests, rather than one per placement. Cancellation covers body reads.
      let cursor = 0;
      await Promise.all(
        Array.from({ length: Math.min(4, ids.length) }, async () => {
          while (cursor < ids.length) {
            const id = ids[cursor++]!;
            const bundled =
              plan.catalog_version === LDRAW_CATALOG_VERSION
                ? LDRAW_CATALOG[id.toLowerCase()]
                : undefined;
            result[id] =
              bundled?.glbBase64 ??
              (await api.buildPartAsset(plan.catalog_version, id, { signal }));
          }
        }),
      );
      return result;
    },
    enabled: !!ws,
    staleTime: Infinity,
    gcTime: 60_000,
    retry: 1,
  });
