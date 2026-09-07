"use client";
import { useState } from "react";
import {
  Check,
  ChevronLeft,
  ChevronRight,
  CircleHelp,
  Download,
  Layers,
  Lightbulb,
  LoaderCircle,
} from "lucide-react";
import { Button } from "@chimii/ui/components/ui/button";
import { ApiError } from "@chimii/core/api";
import {
  useCircuitProgress,
  circuitText,
  circuitPoint,
  circuitCoordinate,
  type CircuitCreation,
  type CircuitProgressInput,
} from "@chimii/core/circuit";
import { useT } from "../i18n";
import { CircuitBoard } from "./circuit-board";
import { CircuitBehaviorPreview } from "./circuit-behavior-preview";
import { CircuitModuleBoard } from "./circuit-module-board";
import { CircuitTrialPanel } from "./circuit-trials";
import "@chimii/ui/styles/circuit.css";
export function CircuitErrorNotice({
  error,
  retry,
}: {
  error?: unknown;
  retry?: () => void;
}) {
  const { t } = useT("circuit");
  let message = t(($) => $.error);
  if (
    error instanceof ApiError &&
    error.body &&
    typeof error.body === "object" &&
    "code" in error.body
  ) {
    switch (error.body.code) {
      case "circuit_inventory_conflict":
        message = t(($) => $.inventory_conflict);
        break;
      case "circuit_unsupported":
        message = t(($) => $.unsupported);
        break;
      case "circuit_validation_failed":
        message = t(($) => $.validation_failed);
        break;
      case "circuit_progress_conflict":
        message = t(($) => $.progress_conflict);
        break;
      case "circuit_catalog_changed":
        message = t(($) => $.catalog_changed);
        break;
      case "circuit_ai_unavailable":
        message = t(($) => $.ai_unavailable);
        break;
      default:
        break;
    }
  }
  return (
    <div
      role="alert"
      className="rounded-xl border border-destructive/30 bg-card p-4 text-sm"
    >
      <p>{message}</p>
      {retry && (
        <Button variant="outline" size="sm" className="mt-3" onClick={retry}>
          {t(($) => $.retry)}
        </Button>
      )}
    </div>
  );
}

export function CircuitWorkbench({
  wsId,
  creation,
  embedded = false,
}: {
  wsId: string;
  creation: CircuitCreation;
  embedded?: boolean;
}) {
  const { t, i18n } = useT("circuit");
  const { document: doc } = creation;
  const isModule = doc.connection_system === "boson";
  const progress = useCircuitProgress(wsId, creation.id);
  const [previewStep, setPreviewStep] = useState<number | null>(null);
  const [overview, setOverview] = useState(false);
  const [signalView, setSignalView] = useState(false);
  const [enlarged, setEnlarged] = useState(false);
  const [layer, setLayer] = useState(0);
  const [checked, setChecked] = useState(false);
  const current = previewStep ?? creation.current_step;
  const step = doc.project.steps[current];
  const canBuild =
    doc.validation.passed === true && doc.validation.issues.length === 0;
  const label = (v: { en: string; zh: string }) =>
    circuitText(v, i18n.language);
  const visibleIds = new Set(
    doc.project.steps.slice(0, current + 1).flatMap((s) => s.placement_ids),
  );
  const visible = overview
    ? doc.project.placements
    : doc.project.placements.filter((p) => visibleIds.has(p.id));
  const focusedWire = !overview
    ? doc.project.connections?.find((w) =>
        step?.placement_ids.includes(w.cable_id ?? w.from.split(":")[0] ?? ""),
      )
    : undefined;
  const focusPlacements = focusedWire
    ? [focusedWire.from, focusedWire.to]
        .map((key, i) => {
          const p = doc.project.placements.find(
            (v) => v.id === key.split(":")[0],
          );
          return p ? { ...p, x: 1 + i * 9, y: 1 } : undefined;
        })
        .filter((p) => p !== undefined)
    : [];
  if (focusedWire?.cable_id) {
    const cable = doc.project.placements.find(
      (p) => p.id === focusedWire.cable_id,
    );
    if (cable) focusPlacements.push(cable);
  }
  const last = current === doc.project.steps.length - 1;
  const save = async (
    stepIndex: number,
    observation: CircuitProgressInput["observation"] = "not_tried",
  ) => {
    try {
      await progress.mutateAsync({
        current_step: stepIndex,
        observation,
        expected_revision: creation.progress_revision,
      });
      setPreviewStep(null);
      setChecked(false);
    } catch {
      /* Display the mutation error without advancing. */
    }
  };
  const download = () => {
    const blob = new Blob([JSON.stringify(doc, null, 2)], {
      type: "application/json",
    });
    const url = URL.createObjectURL(blob);
    const a = window.document.createElement("a");
    a.href = url;
    a.download = `circuit-${creation.id}.json`;
    a.click();
    setTimeout(() => URL.revokeObjectURL(url), 1000);
  };
  return (
    <>
      <header className="mb-7 flex flex-wrap items-start justify-between gap-4">
        <div>
          <p className="mb-2 text-xs font-medium text-muted-foreground">
            {t(($) => $.workbench)}
          </p>
          <h1 className="text-3xl font-black tracking-tight">{doc.title}</h1>
          <p className="mt-2 text-sm text-muted-foreground">
            {label(doc.project.description)}
          </p>
        </div>
        <Button variant="outline" onClick={download}>
          <Download className="size-4" />
          {t(($) => $.download)}
        </Button>
      </header>
      <div className="mb-5 flex flex-wrap items-center gap-3 text-xs">
        <span className="flex items-center gap-1.5 rounded-full border bg-card px-3 py-2">
          <Check className="size-3" />
          {canBuild
            ? doc.composition
              ? t(($) => $.composition_checks)
              : t(($) => $.checks_passed)
            : t(($) => $.checks_failed)}
        </span>
        <span className="text-muted-foreground">
          {doc.composition
            ? t(($) => $.composition_evidence)
            : t(($) => $.reference_only)}
        </span>
      </div>
      {canBuild && (
        <CircuitBehaviorPreview key={doc.content_hash} document={doc} />
      )}
      {!canBuild ? (
        <CircuitErrorNotice />
      ) : (
        <div
          className={
            embedded
              ? "grid items-start gap-6"
              : "grid items-start gap-6 lg:grid-cols-[minmax(0,1fr)_340px]"
          }
        >
          <section className="min-w-0 rounded-2xl border bg-card p-4 md:p-5">
            <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
              <div className="flex gap-2">
                <Button
                  size="sm"
                  variant={overview ? "outline" : "default"}
                  onClick={() => setOverview(false)}
                >
                  {t(($) => $.step_view)}
                </Button>
                <Button
                  size="sm"
                  variant={overview ? "default" : "outline"}
                  onClick={() => setOverview(true)}
                >
                  {t(($) => $.overview)}
                </Button>
              </div>
              {isModule && (
                <Button
                  size="sm"
                  variant="outline"
                  aria-pressed={signalView}
                  onClick={() => setSignalView(!signalView)}
                >
                  {signalView
                    ? t(($) => $.show_modules)
                    : t(($) => $.show_signals)}
                </Button>
              )}
              {!isModule && (
                <label className="flex items-center gap-2 text-xs text-muted-foreground">
                  <Layers className="size-4" />
                  <span className="sr-only">{t(($) => $.layers)}</span>
                  <select
                    aria-label={t(($) => $.layers)}
                    value={layer}
                    onChange={(e) => setLayer(Number(e.target.value))}
                    className="rounded-md border bg-background px-2 py-2 text-foreground"
                  >
                    <option value={0}>{t(($) => $.all_layers)}</option>
                    {[1, 2, 3].map((n) => (
                      <option key={n} value={n}>
                        {t(($) => $.layer, { n })}
                      </option>
                    ))}
                  </select>
                </label>
              )}
            </div>
            <Button
              variant="outline"
              size="sm"
              aria-pressed={enlarged}
              onClick={() => setEnlarged(!enlarged)}
              className="mb-3"
            >
              {enlarged ? t(($) => $.fit_diagram) : t(($) => $.enlarge_diagram)}
            </Button>
            <div
              className="overflow-auto rounded-xl"
              role="region"
              aria-label={doc.title}
              tabIndex={enlarged ? 0 : undefined}
            >
              <div style={enlarged ? { minWidth: 900 } : undefined}>
                {isModule ? (
                  <CircuitModuleBoard
                    label={doc.title}
                    schematic={signalView}
                    parts={doc.parts}
                    placements={visible}
                    connections={doc.project.connections ?? []}
                    columns={doc.columns}
                    rows={doc.rows}
                    highlighted={overview ? [] : step?.placement_ids}
                  />
                ) : (
                  <CircuitBoard
                    label={doc.title}
                    parts={doc.parts}
                    placements={visible}
                    columns={doc.columns}
                    rows={doc.rows}
                    highlighted={overview ? [] : step?.placement_ids}
                    layer={layer}
                  />
                )}
              </div>
            </div>
            {enlarged && (
              <p className="mt-2 text-xs text-muted-foreground">
                {t(($) => $.pan_diagram)}
              </p>
            )}
            <p className="mt-4 text-xs leading-6 text-muted-foreground">
              {isModule
                ? t(($) => $.module_board_hint)
                : t(($) => $.board_hint)}
            </p>
            <div className="mt-5 border-t pt-4">
              <h2 className="mb-2 text-sm font-bold">{t(($) => $.how)}</h2>
              <p className="text-sm leading-7 text-muted-foreground">
                {label(doc.project.explanation)}
              </p>
            </div>
            <a
              href={`${doc.project.source.url}#page=${doc.project.source.page}`}
              target="_blank"
              rel="noopener noreferrer"
              className="mt-4 inline-block text-xs underline underline-offset-4"
            >
              {doc.composition ? t(($) => $.module_source) : t(($) => $.source)}
            </a>
          </section>
          <aside className="space-y-4">
            <details className="rounded-2xl border bg-card p-5">
              <summary className="cursor-pointer text-sm font-bold">
                {t(($) => $.before)}
              </summary>
              <ul className="mt-3 space-y-3">
                {doc.preparation.map((v, i) => (
                  <li
                    key={i}
                    className="text-xs leading-6 text-muted-foreground"
                  >
                    {label(v)}
                  </li>
                ))}
              </ul>
            </details>
            <section
              className="rounded-2xl border bg-card p-5"
              aria-live="polite"
            >
              <p className="font-mono text-xs text-muted-foreground">
                {t(($) => $.step_of, {
                  current: current + 1,
                  total: doc.project.steps.length,
                })}
              </p>
              <div className="mb-5 mt-3 h-1.5 overflow-hidden rounded-full bg-muted">
                <div
                  className="h-full rounded-full bg-primary"
                  style={{
                    width: `${((current + 1) / doc.project.steps.length) * 100}%`,
                  }}
                />
              </div>
              {step && (
                <>
                  <h2 className="text-xl font-bold">{label(step.title)}</h2>
                  <p className="mt-3 text-sm leading-7 text-muted-foreground">
                    {label(step.instruction)}
                  </p>
                  {focusedWire && (
                    <div
                      className="mt-4 rounded-xl border p-3"
                      data-testid="connection-focus"
                    >
                      <p className="mb-2 text-xs font-bold">
                        {t(($) => $.connect_now)}
                      </p>
                      <CircuitModuleBoard
                        label={t(($) => $.connect_now)}
                        parts={doc.parts}
                        placements={focusPlacements}
                        connections={[focusedWire]}
                        columns={16}
                        rows={5}
                        highlighted={step.placement_ids}
                      />
                    </div>
                  )}
                  <div className="my-4 space-y-3">
                    {step.placement_ids.map((id) => {
                      const p = doc.project.placements.find((v) => v.id === id);
                      const part = doc.parts.find((v) => v.id === p?.part_id);
                      if (!p || !part) return null;
                      return (
                        <div key={id} className="rounded-xl bg-muted/50 p-3">
                          <p className="text-sm font-bold">{`${part.id} · ${label(part.name)}`}</p>
                          <p className="mt-1 text-xs text-muted-foreground">
                            {label(part.purpose)}
                          </p>
                          {isModule ? (
                            <div className="mt-2 space-y-2 text-xs">
                              {part.marking && (
                                <p className="font-mono font-bold">
                                  {part.marking}
                                </p>
                              )}
                              {(doc.project.connections ?? [])
                                .filter(
                                  (w) =>
                                    w.cable_id === id ||
                                    w.from.startsWith(id + ":") ||
                                    w.to.startsWith(id + ":"),
                                )
                                .map((w) => {
                                  const name = (key: string) => {
                                    const [instance, pin] = key.split(":");
                                    const v = doc.project.placements.find(
                                      (v) => v.id === instance,
                                    );
                                    return `${doc.parts.find((part) => part.id === v?.part_id)?.marking ?? instance} ${pin}`;
                                  };
                                  return (
                                    <p
                                      key={w.id}
                                      className="rounded border bg-card p-2 font-mono"
                                    >
                                      {name(w.from)} → {name(w.to)}
                                    </p>
                                  );
                                })}
                            </div>
                          ) : (
                            <>
                              <p className="my-2 text-xs font-bold">
                                {t(($) => $.layer, { n: p.layer })}
                              </p>
                              <div className="flex flex-wrap gap-1.5">
                                {part.ports.map((port) => {
                                  const q = circuitPoint(p, port.x, port.y);
                                  return (
                                    <span
                                      key={port.id}
                                      className="rounded border bg-card px-2 py-1 font-mono text-xs"
                                    >{`${port.id} → ${circuitCoordinate(q.x, q.y)}`}</span>
                                  );
                                })}
                              </div>
                            </>
                          )}
                        </div>
                      );
                    })}
                  </div>
                </>
              )}
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  disabled={current === 0 || progress.isPending}
                  onClick={() => {
                    setPreviewStep(Math.max(0, current - 1));
                    setOverview(false);
                    setChecked(false);
                  }}
                  aria-label={t(($) => $.previous)}
                >
                  <ChevronLeft className="size-4" />
                </Button>
                <Button
                  className="flex-1"
                  disabled={progress.isPending || last}
                  onClick={() => {
                    setOverview(false);
                    void save(current + 1);
                  }}
                >
                  {progress.isPending ? (
                    <LoaderCircle className="size-4 animate-spin" />
                  ) : (
                    <ChevronRight className="size-4" />
                  )}
                  {t(($) => $.next)}
                </Button>
              </div>
              <p className="mt-3 text-xs text-muted-foreground">
                {t(($) => $.saved_step, { n: creation.current_step + 1 })}
              </p>
            </section>
            {last && (
              <section className="rounded-2xl border bg-card p-5">
                <h2 className="flex items-center gap-2 font-bold">
                  <Lightbulb className="size-4" />
                  {t(($) => $.try_it)}
                </h2>
                <p className="mt-3 text-sm leading-7">
                  {label(doc.project.test_instruction)}
                </p>
                <label className="mt-4 flex cursor-pointer items-start gap-3 text-xs leading-6">
                  <input
                    type="checkbox"
                    checked={checked}
                    onChange={(e) => setChecked(e.target.checked)}
                    className="mt-1 size-4 shrink-0 accent-primary"
                  />
                  {t(($) => $.checked)}
                </label>
                <div className="mt-4 flex gap-2">
                  <Button
                    size="sm"
                    disabled={!checked || progress.isPending}
                    onClick={() => void save(current, "worked")}
                  >
                    <Check className="size-4" />
                    {t(($) => $.worked)}
                  </Button>
                  <Button
                    size="sm"
                    variant="outline"
                    disabled={!checked || progress.isPending}
                    onClick={() => void save(current, "needs_help")}
                  >
                    <CircleHelp className="size-4" />
                    {t(($) => $.needs_help)}
                  </Button>
                </div>
              </section>
            )}
            {creation.observation !== "not_tried" && (
              <p
                role="status"
                className="rounded-xl border bg-card p-4 text-xs leading-6"
              >
                {creation.observation === "worked"
                  ? t(($) => $.reported_worked)
                  : creation.observation === "needs_help"
                    ? t(($) => $.reported_help)
                    : t(($) => $.observation_unknown)}
              </p>
            )}
            {progress.isError && <CircuitErrorNotice error={progress.error} />}
            <CircuitTrialPanel wsId={wsId} creation={creation} />
            <details className="rounded-2xl border bg-card p-5">
              <summary className="cursor-pointer text-sm font-bold">
                {t(($) => $.all_steps)}
              </summary>
              <ol className="circuit-step-list mt-3 max-h-64 space-y-1 overflow-y-auto">
                {doc.project.steps.map((s, i) => (
                  <li key={s.id}>
                    <button
                      type="button"
                      aria-current={i === current ? "step" : undefined}
                      onClick={() => {
                        setPreviewStep(i);
                        setOverview(false);
                        setChecked(false);
                      }}
                      className={`w-full rounded-lg px-3 py-2 text-left text-xs ${i === current ? "bg-muted font-bold" : "hover:bg-muted/50"}`}
                    >{`${i + 1}. ${label(s.title)}`}</button>
                  </li>
                ))}
              </ol>
            </details>
            <details
              className="rounded-2xl border bg-card p-5"
              open={creation.observation === "needs_help"}
            >
              <summary className="cursor-pointer text-sm font-bold">
                {t(($) => $.help)}
              </summary>
              <ul className="mt-3 space-y-3">
                {doc.project.troubleshooting.map((v, i) => (
                  <li
                    key={i}
                    className="text-xs leading-6 text-muted-foreground"
                  >
                    {label(v)}
                  </li>
                ))}
              </ul>
            </details>
          </aside>
        </div>
      )}
    </>
  );
}
