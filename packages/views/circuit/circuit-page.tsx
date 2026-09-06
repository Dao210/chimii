"use client";
import { useQuery } from "@tanstack/react-query";
import { ArrowLeft } from "lucide-react";
import { useWorkspaceId } from "@chimii/core/hooks";
import { useWorkspacePaths } from "@chimii/core/paths";
import { circuitCreationOptions } from "@chimii/core/circuit";
import { AppLink } from "../navigation";
import { useT } from "../i18n";
import { BuildPage } from "../build/components/build-page";
import {
  CircuitWorkbench,
  CircuitErrorNotice as ErrorNotice,
} from "./circuit-workbench";
export function CircuitPage() {
  return <BuildPage initialKind="circuit" />;
}
export function CircuitDetailPage({ creationId }: { creationId: string }) {
  const wsId = useWorkspaceId();
  const paths = useWorkspacePaths();
  const { t } = useT("circuit");
  const { t: bt } = useT("build");
  const query = useQuery(circuitCreationOptions(wsId, creationId));
  return (
    <main className="circuit-studio h-full overflow-y-auto">
      <div className="mx-auto max-w-7xl px-5 py-7 md:px-9">
        <AppLink
          href={paths.circuit()}
          className="mb-6 inline-flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground"
        >
          <ArrowLeft className="size-4" />
          {t(($) => $.back)}
        </AppLink>
        {query.isPending ? (
          <p role="status">{t(($) => $.loading)}</p>
        ) : query.isError ? (
          <ErrorNotice retry={() => void query.refetch()} />
        ) : (
          <>
            <AppLink
              className="mb-5 inline-flex rounded-xl border bg-card px-4 py-3 text-sm font-bold"
              href={`${paths.circuit()}?${new URLSearchParams({ source_kind: "circuit", source_id: query.data.id, source_hash: query.data.document.content_hash })}`}
            >
              {bt(($) => $.conversation_continue)}
            </AppLink>
            <CircuitWorkbench
              key={`${wsId}:${creationId}`}
              wsId={wsId}
              creation={query.data}
            />
          </>
        )}
      </div>
    </main>
  );
}
