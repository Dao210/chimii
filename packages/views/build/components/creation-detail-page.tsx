"use client";

import { useQuery } from "@tanstack/react-query";
import { ArrowLeft } from "lucide-react";
import { Button } from "@chimii/ui/components/ui/button";
import { useWorkspaceId } from "@chimii/core/hooks";
import { useWorkspacePaths } from "@chimii/core/paths";
import { buildCreationOptions } from "@chimii/core/build";
import { AppLink } from "../../navigation";
import { BuildResult } from "./build-result";
import { useT } from "../../i18n";

export function CreationDetailPage({ creationId }: { creationId: string }) {
  const { t } = useT("build");
  const workspaceId = useWorkspaceId();
  const workspacePaths = useWorkspacePaths();
  const { data, isLoading, isError, refetch } = useQuery(buildCreationOptions(workspaceId, creationId));
  return (
    <main className="h-full overflow-y-auto bg-[#f4ead5] text-[#1d241f]">
      <div className="mx-auto max-w-[1320px] px-5 py-7 md:px-8">
        <AppLink href={workspacePaths.creations()} className="mb-5 inline-flex items-center gap-2 text-sm font-black text-[#39715a] hover:underline"><ArrowLeft className="size-4" /> {t($ => $.detail_back)}</AppLink>
        {isLoading ? (
          <div className="h-[560px] animate-pulse rounded-[2rem] bg-[#ded2bb]" />
        ) : isError || !data?.id ? (
          <div className="flex min-h-[560px] flex-col items-center justify-center rounded-[2rem] border-2 border-dashed border-[#9c9079] bg-[#fff8e9] text-center">
            <h1 className="text-2xl font-black">{t($ => $.detail_error_title)}</h1>
            <p className="mt-2 text-[#687068]">{t($ => $.detail_error_description)}</p>
            <Button onClick={() => void refetch()} className="mt-6 rounded-xl bg-[#1d241f] font-black">{t($ => $.retry_fetch)}</Button>
          </div>
        ) : <BuildResult creation={data} />}
      </div>
    </main>
  );
}
