"use client";

import { useQuery } from "@tanstack/react-query";
import { AlertCircle, ArrowLeft, Blocks, Plus, Sparkles } from "lucide-react";
import { Button } from "@chimii/ui/components/ui/button";
import { useWorkspaceId } from "@chimii/core/hooks";
import { useWorkspacePaths } from "@chimii/core/paths";
import { buildCreationsOptions } from "@chimii/core/build";
import { AppLink } from "../../navigation";
import { BuildModelViewer } from "./build-model-viewer";
import { useT } from "../../i18n";

export function CreationsPage() {
  const { t } = useT("build");
  const workspaceId = useWorkspaceId();
  const workspacePaths = useWorkspacePaths();
  const { data: creations = [], isLoading, isError, refetch } = useQuery(buildCreationsOptions(workspaceId));

  return (
    <main className="h-full overflow-y-auto bg-[#f4ead5] text-[#1d241f]">
      <div className="mx-auto max-w-[1320px] px-5 py-7 md:px-8">
        <header className="mb-8 flex flex-wrap items-end justify-between gap-4">
          <div>
            <AppLink href={workspacePaths.build()} className="mb-4 inline-flex items-center gap-2 text-sm font-black text-[#39715a] hover:underline"><ArrowLeft className="size-4" /> {t($ => $.creations_back)}</AppLink>
            <h1 className="text-4xl font-black tracking-tight md:text-5xl">{t($ => $.creations_title)}</h1>
            <p className="mt-2 font-medium text-[#687068]">{t($ => $.creations_description)}</p>
          </div>
          <Button nativeButton={false} className="h-12 rounded-xl border-2 border-[#1d241f] bg-[#ef5c4f] font-black text-white shadow-[4px_5px_0_#1d241f]" render={<AppLink href={workspacePaths.build()} />}>
            <Plus className="size-4" /> {t($ => $.creations_new)}
          </Button>
        </header>

        {isLoading ? (
          <div className="grid gap-5 sm:grid-cols-2 xl:grid-cols-3">{[1, 2, 3].map((item) => <div key={item} className="h-80 animate-pulse rounded-[2rem] bg-[#ded2bb]" />)}</div>
        ) : isError ? (
          <div className="flex min-h-[480px] flex-col items-center justify-center rounded-[2rem] border-2 border-dashed border-[#9c9079] bg-[#fff8e9] text-center">
            <div className="flex size-20 items-center justify-center rounded-[1.7rem] border-2 border-[#1d241f] bg-[#ffd85a] shadow-[5px_6px_0_#1d241f]"><AlertCircle className="size-9" /></div>
            <h2 className="mt-6 text-2xl font-black">{t($ => $.creations_error_title)}</h2>
            <p className="mt-2 max-w-sm text-[#687068]">{t($ => $.creations_error_description)}</p>
            <Button onClick={() => void refetch()} className="mt-6 rounded-xl bg-[#1d241f] font-black">{t($ => $.creations_retry)}</Button>
          </div>
        ) : creations.length === 0 ? (
          <div className="flex min-h-[480px] flex-col items-center justify-center rounded-[2rem] border-2 border-dashed border-[#9c9079] bg-[#fff8e9] text-center">
            <div className="flex size-20 items-center justify-center rounded-[1.7rem] border-2 border-[#1d241f] bg-[#ffd85a] shadow-[5px_6px_0_#1d241f]"><Blocks className="size-9" /></div>
            <h2 className="mt-6 text-2xl font-black">{t($ => $.creations_empty_title)}</h2>
            <p className="mt-2 max-w-sm text-[#687068]">{t($ => $.creations_empty_description)}</p>
            <Button nativeButton={false} className="mt-6 rounded-xl bg-[#1d241f] font-black" render={<AppLink href={workspacePaths.build()} />}><Sparkles className="size-4" /> {t($ => $.creations_empty_action)}</Button>
          </div>
        ) : (
          <div className="grid gap-6 sm:grid-cols-2 xl:grid-cols-3">
            {creations.map((creation) => (
              <AppLink
                key={creation.id}
                href={workspacePaths.creationDetail(creation.id)}
                className="group overflow-hidden rounded-[2rem] border-2 border-[#1d241f] bg-[#fffdf7] shadow-[5px_6px_0_#1d241f] transition hover:-translate-y-1 hover:shadow-[8px_10px_0_#1d241f]"
              >
                <BuildModelViewer placements={creation.build_plan.placements} parts={creation.build_plan.parts} renderMode="projection" className="m-3 h-56 rounded-[1.3rem] border" />
                <div className="px-5 pb-5 pt-2">
                  <p className="text-xs font-black uppercase tracking-[.14em] text-[#39715a]">{t($ => $.creations_stats, { parts: creation.validation.part_count, steps: creation.validation.step_count })}</p>
                  <h2 className="mt-1 text-xl font-black group-hover:text-[#3767bb]">{creation.title}</h2>
                  <p className="mt-2 line-clamp-2 text-sm font-medium text-[#74736d]">{creation.prompt}</p>
                </div>
              </AppLink>
            ))}
          </div>
        )}
      </div>
    </main>
  );
}
