"use client";

import { useMemo, useState } from "react";
import { CheckCircle2, Download, PackageCheck, RotateCcw } from "lucide-react";
import { Button } from "@chimii/ui/components/ui/button";
import { Slider } from "@chimii/ui/components/ui/slider";
import { toast } from "sonner";
import { api } from "@chimii/core/api";
import type { BuildCreation } from "@chimii/core/build";
import { BuildModelViewer } from "./build-model-viewer";
import { useT } from "../../i18n";

export function BuildResult({ creation, onAgain }: { creation: BuildCreation; onAgain?: () => void }) {
  const { t } = useT("build");
  const [step, setStep] = useState(creation.validation.step_count);
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

  return (
    <div className="grid gap-5 xl:grid-cols-[minmax(0,1.25fr)_minmax(320px,.75fr)]">
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
          <Slider
            min={1}
            max={Math.max(1, creation.validation.step_count)}
            step={1}
            value={[step]}
            onValueChange={(value) => setStep((Array.isArray(value) ? value[0] : value) ?? 1)}
            aria-label={t($ => $.result_step, { step, total: creation.validation.step_count })}
          />
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

        <div className="mt-auto grid gap-2">
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
    </div>
  );
}
