"use client";

import { useMemo, useState } from "react";
import { CheckCircle2, ChevronLeft, ChevronRight, Download, PackageCheck, RotateCcw } from "lucide-react";
import { Button } from "@chimii/ui/components/ui/button";
import { Slider } from "@chimii/ui/components/ui/slider";
import { toast } from "sonner";
import { api } from "@chimii/core/api";
import { buildProgressOptions, useSaveBuildProgress, type BuildCreation } from "@chimii/core/build";
import { useWorkspaceId } from "@chimii/core/hooks";
import { useQuery } from "@tanstack/react-query";
import { BuildModelViewer } from "./build-model-viewer";
import { useT } from "../../i18n";
import { BuildDesignEditor } from "./build-design-editor";

export function BuildResult({ creation, onAgain, embedded = false }: { creation: BuildCreation; onAgain?: () => void; embedded?: boolean }) {
  const { t } = useT("build");
  const wsId = useWorkspaceId();
  const progressQuery = useQuery(buildProgressOptions(wsId, creation.id));
  const save = useSaveBuildProgress(wsId, creation.id);
  const progress = progressQuery.data;
  const [building, setBuilding] = useState(false);
  const [editing, setEditing] = useState(false);
  const [previewStep, setPreviewStep] = useState(creation.validation.step_count);
  const step = building ? Math.max(1, progress?.current_step ?? 1) : previewStep;
  const busy = save.isPending || progressQuery.isFetching;
  const persist = async (currentStep: number, completed = false) => {
    if (!progress || busy) return;
    try {
      await save.mutateAsync({ current_step: currentStep, completed, expected_revision: progress.revision });
      setBuilding(true);
    } catch { /* The inline error remains visible until the user retries. */ }
  };
  const lastStep = Math.max(1, creation.validation.step_count);
  const currentStep = useMemo(
    () => creation.build_plan.steps.find((item) => item.number === step),
    [creation.build_plan.steps, step],
  );
  const currentParts = useMemo(() => {
    const ids = new Set(currentStep?.added_placement_ids ?? []);
    return creation.build_plan.placements
      .filter((placement) => ids.has(placement.id))
      .map((placement) => creation.build_plan.parts[placement.part_id]?.name ?? placement.part_id);
  }, [creation.build_plan.parts, creation.build_plan.placements, currentStep]);

  const downloadMPD = async () => {
    try {
      const blob = await api.downloadBuildMPD(creation.id);
      const url = URL.createObjectURL(blob);
      const anchor = document.createElement("a");
      anchor.href = url;
      anchor.download = `${creation.title || "chimii-creation"}.mpd`;
      anchor.click();
      URL.revokeObjectURL(url);
    } catch {
      toast.error(t($ => $.result_download_error));
    }
  };

  const goToPrevStep = () => {
    if (building) void persist(Math.max(1, step - 1));
    else setPreviewStep(Math.max(1, step - 1));
  };

  const goToNextStep = () => {
    if (building) void persist(Math.min(lastStep, step + 1));
    else setPreviewStep(Math.min(lastStep, step + 1));
  };

  return (
    <div className={embedded ? "grid gap-5" : "grid gap-5 xl:grid-cols-[minmax(0,1.25fr)_minmax(320px,.75fr)]"}>
      <div className="space-y-4">
        <BuildModelViewer
          placements={creation.build_plan.placements}
          parts={creation.build_plan.parts}
          catalogVersion={creation.build_plan.catalog_version}
          maxStep={step}
          highlightPlacementIds={currentStep?.added_placement_ids}
          className="min-h-[420px] shadow-[8px_9px_0_#1d241f]"
        />
        <div className="rounded-[1.4rem] border-2 border-[#1d241f] bg-[#fffdf7] p-4 shadow-[4px_5px_0_#1d241f]">
          <div className="mb-3 flex items-center justify-between text-sm font-black text-[#1d241f]">
            <span>{t($ => $.result_step, { step, total: creation.validation.step_count })}</span>
            <span className="text-[#39715a]">{t($ => $.result_placed, { count: creation.build_plan.placements.filter((part) => part.step <= step).length })}</span>
          </div>
          <div className="mt-3 flex items-center gap-3">
            <button
              type="button"
              onClick={goToPrevStep}
              disabled={step <= 1 || (building && busy)}
              aria-label={t($ => $.result_prev_step)}
              className="inline-flex h-10 min-w-10 items-center justify-center rounded-xl border-2 border-[#1d241f] bg-white px-2 text-sm font-black text-[#1d241f] transition hover:bg-[#1d241f] hover:text-white disabled:cursor-not-allowed disabled:opacity-45 disabled:hover:bg-white disabled:hover:text-[#1d241f]"
            >
              <ChevronLeft className="size-4" />
            </button>
            <div className="flex-1">
              <Slider
                disabled={building}
                min={1}
                max={lastStep}
                step={1}
                value={[step]}
                onValueChange={(value) => setPreviewStep((Array.isArray(value) ? value[0] : value) ?? 1)}
                aria-label={t($ => $.result_step, { step, total: creation.validation.step_count })}
              />
            </div>
            <button
              type="button"
              onClick={goToNextStep}
              disabled={step >= lastStep || (building && busy)}
              aria-label={t($ => $.result_next_step)}
              className="inline-flex h-10 min-w-10 items-center justify-center rounded-xl border-2 border-[#1d241f] bg-white px-2 text-sm font-black text-[#1d241f] transition hover:bg-[#1d241f] hover:text-white disabled:cursor-not-allowed disabled:opacity-45 disabled:hover:bg-white disabled:hover:text-[#1d241f]"
            >
              <ChevronRight className="size-4" />
            </button>
          </div>
          <div className="mt-4 rounded-xl bg-[#edf3ff] px-4 py-3 text-sm font-bold text-[#294d8c]">
            {t($ => $.result_add, { parts: currentParts.length > 0 ? currentParts.join(t($ => $.result_separator)) : t($ => $.result_check_existing) })}
          </div>
        </div>
      </div>

      <aside className="flex flex-col rounded-[2rem] border-2 border-[#1d241f] bg-[#fffdf7] p-6 shadow-[7px_8px_0_#1d241f]">
        <div className="mb-5 inline-flex size-12 items-center justify-center rounded-2xl border-2 border-[#1d241f] bg-[#ffd85a]">
          <PackageCheck className="size-6" />
        </div>
        <p className="text-xs font-black uppercase tracking-[.18em] text-[#39715a]">{t($ => $.result_validated)}</p>
        <h2 className="mt-2 text-3xl font-black tracking-tight text-[#1d241f]">{creation.title}</h2>
        <p className="mt-3 text-sm leading-6 text-[#59615b]">“{creation.prompt}”</p>

        <div className="my-6 grid grid-cols-2 gap-3">
          <div className="rounded-2xl bg-[#edf5df] p-4">
            <p className="text-2xl font-black text-[#1d241f]">{creation.validation.part_count}</p>
            <p className="text-xs font-bold text-[#65705f]">{t($ => $.result_parts)}</p>
          </div>
          <div className="rounded-2xl bg-[#e5edff] p-4">
            <p className="text-2xl font-black text-[#1d241f]">{creation.validation.step_count}</p>
            <p className="text-xs font-bold text-[#65705f]">{t($ => $.result_steps)}</p>
          </div>
        </div>

        <div className="mb-6 flex items-start gap-2 rounded-2xl border border-[#9bc7ae] bg-[#e9f7ed] p-3 text-sm font-semibold text-[#285a43]">
          <CheckCircle2 className="mt-0.5 size-4 shrink-0" />
          {t($ => $.result_validation_hint)}
        </div>

        <div className="mb-5 grid gap-2">
          {progress?.completed_at && <p role="status" className="text-sm font-semibold text-foreground">{t($ => $.progress_completed)}</p>}
          {progress && <p className="text-xs text-muted-foreground">{progress.current_step > 0 ? t($ => $.progress_saved, { step: progress.current_step }) : t($ => $.progress_not_started)}</p>}
          {!building ? (
            <Button disabled={!progress || busy} onClick={() => { if (progress?.current_step) setBuilding(true); else void persist(1); }}>
              {t($ => progress?.current_step ? $.progress_continue : $.progress_start)}
            </Button>
          ) : (
            <>
              <Button variant="outline" disabled={busy} onClick={() => { setPreviewStep(lastStep); setBuilding(false); }}>{t($ => $.progress_preview)}</Button>
              {step === lastStep && !progress?.completed_at && <Button disabled={busy} onClick={() => void persist(lastStep, true)}>{t($ => $.progress_complete)}</Button>}
            </>
          )}
          {busy && <p role="status" className="text-xs text-muted-foreground">{t($ => $.progress_loading)}</p>}
          {(save.isError || progressQuery.isError) && <div role="alert" className="text-sm text-destructive">
            <p>{t($ => $.progress_error)}</p>
            <Button variant="outline" size="sm" onClick={() => { save.reset(); void progressQuery.refetch(); }}>{t($ => $.retry_fetch)}</Button>
          </div>}
        </div>

        <div className="mt-auto grid gap-2">
          {creation.build_plan.document?.design?.shapes?.length ? <Button variant="outline" disabled={building} onClick={() => setEditing(true)}>{t($ => $.editor_title)}</Button> : null}
          <Button onClick={() => void downloadMPD()} className="h-12 rounded-xl bg-[#1d241f] font-bold text-white hover:bg-[#333d36]">
            <Download className="size-4" /> {t($ => $.result_download)}
          </Button>
          {onAgain && (
            <Button variant="outline" onClick={onAgain} className="h-12 rounded-xl border-2 border-[#1d241f] bg-transparent font-bold">
              <RotateCcw className="size-4" /> {t($ => $.result_another)}
            </Button>
          )}
          <p className="pt-2 text-center text-[11px] font-medium leading-4 text-[#77766f]">
            {t($ => $.result_attribution)}
          </p>
        </div>
      </aside>
      {editing && creation.build_plan.document?.design?.shapes?.length ? <BuildDesignEditor key={creation.id} creation={creation} onClose={() => setEditing(false)} /> : null}
    </div>
  );
}
