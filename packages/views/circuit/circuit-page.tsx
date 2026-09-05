"use client";

import { useEffect, useRef, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  ArrowLeft,
  ArrowRight,
  Check,
  ChevronLeft,
  ChevronRight,
  CircleHelp,
  Download,
  Layers,
  Lightbulb,
  LoaderCircle,
  Radio,
  Sparkles,
  Volume2,
} from "lucide-react";
import { Button } from "@chimii/ui/components/ui/button";
import { Textarea } from "@chimii/ui/components/ui/textarea";
import { Input } from "@chimii/ui/components/ui/input";
import { useWorkspaceId } from "@chimii/core/hooks";
import { useWorkspacePaths } from "@chimii/core/paths";
import { generateUUID } from "@chimii/core/utils";
import { ApiError } from "@chimii/core/api";
import {
  circuitCatalogOptions,
  circuitListOptions,
  circuitCreationOptions,
  useCreateCircuit,
  useCircuitProgress,
  circuitMaterials,
  circuitText,
  circuitPoint,
  circuitCoordinate,
  type CircuitCatalogResponse,
  type CircuitCreation,
  type CircuitProgressInput,
} from "@chimii/core/circuit";
import { AppLink, useNavigation } from "../navigation";
import { useT } from "../i18n";
import { CircuitBoard } from "./circuit-board";
import "@chimii/ui/styles/circuit.css";

function ErrorNotice({
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

export function CircuitPage() {
  const wsId = useWorkspaceId();
  const { t } = useT("circuit");
  const query = useQuery(circuitCatalogOptions(wsId));
  return (
    <main className="circuit-studio h-full overflow-y-auto">
      {query.isPending ? (
        <div
          role="status"
          className="flex min-h-80 items-center justify-center gap-3"
        >
          <LoaderCircle className="size-5 animate-spin" />
          {t(($) => $.loading)}
        </div>
      ) : query.isError ? (
        <div className="mx-auto max-w-xl p-8">
          <ErrorNotice retry={() => void query.refetch()} />
        </div>
      ) : (
        <CircuitHome
          key={`${wsId}:${query.data.catalog.version}`}
          wsId={wsId}
          data={query.data}
        />
      )}
    </main>
  );
}

function CircuitHome({
  wsId,
  data,
}: {
  wsId: string;
  data: CircuitCatalogResponse;
}) {
  const { t, i18n } = useT("circuit");
  const { catalog } = data;
  const paths = useWorkspacePaths();
  const navigation = useNavigation();
  const create = useCreateCircuit(wsId);
  const list = useQuery(circuitListOptions(wsId));
  const [prompt, setPrompt] = useState("");
  const [selected, setSelected] = useState(catalog.projects[0]?.id ?? "");
  const [inventory, setInventory] = useState<Record<string, number>>(() =>
    Object.fromEntries(catalog.parts.map((p) => [p.id, p.quantity])),
  );
  const request = useRef({ signature: "", id: "" });
  const mounted = useRef(true);
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, []);
  const project = catalog.projects.find((p) => p.id === selected);
  const materials = project ? circuitMaterials(project, inventory) : [];
  const missing = materials.some((m) => m.missing > 0);
  const label = (v: { en: string; zh: string }) =>
    circuitText(v, i18n.language);
  const start = async (withAI: boolean) => {
    const input = {
      kit_id: catalog.kit_id,
      catalog_version: catalog.version,
      locale: i18n.language,
      inventory,
      ...(withAI ? { prompt: prompt.trim() } : { project_id: selected }),
    };
    const signature = JSON.stringify(input);
    if (signature !== request.current.signature)
      request.current = { signature, id: generateUUID() };
    try {
      const result = await create.mutateAsync({
        ...input,
        client_request_id: request.current.id,
      });
      if (mounted.current) navigation.push(paths.circuitDetail(result.id));
    } catch {
      /* The mutation owns its visible, retryable error state. */
    }
  };
  return (
    <div className="mx-auto max-w-7xl px-5 py-7 md:px-9 md:py-10">
      <header className="mb-9 flex flex-wrap items-start justify-between gap-4">
        <div>
          <p className="mb-3 flex items-center gap-2 text-xs font-bold tracking-widest text-muted-foreground">
            <span className="size-2 rounded-full bg-primary" />
            {t(($) => $.eyebrow)}
          </p>
          <h1 className="text-3xl font-black tracking-tight md:text-4xl">
            {t(($) => $.title)}
          </h1>
          <p className="mt-3 max-w-xl text-sm leading-7 text-muted-foreground">
            {t(($) => $.subtitle)}
          </p>
        </div>
        <Button
          variant="outline"
          render={<AppLink href={paths.build()} />}
          nativeButton={false}
        >
          <ArrowLeft className="size-4" />
          {t(($) => $.bricks)}
        </Button>
      </header>
      <div className="mb-7 flex flex-wrap items-center justify-between gap-3 rounded-2xl border bg-card px-5 py-4">
        <div>
          <p className="text-xs text-muted-foreground">{t(($) => $.kit)}</p>
          <p className="mt-1 font-bold">{catalog.name}</p>
        </div>
        <span className="rounded-full bg-muted px-3 py-1.5 text-xs font-medium">
          {t(($) => $.age, { age: catalog.minimum_age })}
        </span>
        <p className="max-w-lg text-xs leading-5 text-muted-foreground">
          {t(($) => $.kit_note)}
        </p>
      </div>
      <div className="grid gap-7 lg:grid-cols-[minmax(0,1fr)_320px]">
        <section className="min-w-0">
          <div className="mb-4 flex items-center gap-3">
            <span className="font-mono text-sm text-muted-foreground">
              {"01"}
            </span>
            <h2 className="font-bold">{t(($) => $.choose)}</h2>
          </div>
          <div className="grid gap-3 sm:grid-cols-3">
            {catalog.projects.map((p, index) => {
              const Icon =
                p.id === "fm-radio"
                  ? Radio
                  : p.id === "alarm-sound"
                    ? Volume2
                    : Lightbulb;
              return (
                <button
                  key={p.id}
                  type="button"
                  aria-pressed={selected === p.id}
                  onClick={() => {
                    setSelected(p.id);
                    create.reset();
                  }}
                  className={`group min-w-0 rounded-2xl border-2 p-4 text-left transition-colors focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-ring ${selected === p.id ? "border-primary bg-card" : "border-transparent bg-card/60 hover:bg-card"}`}
                >
                  <div className="mb-4 flex items-center justify-between">
                    <Icon className="size-6" />
                    <span className="font-mono text-xs text-muted-foreground">
                      {String(index + 1).padStart(2, "0")}
                    </span>
                  </div>
                  <CircuitBoard
                    miniature
                    label={label(p.title)}
                    parts={catalog.parts}
                    placements={p.placements}
                    columns={catalog.columns}
                    rows={catalog.rows}
                  />
                  <h3 className="mt-4 text-base font-bold">{label(p.title)}</h3>
                  <p className="mt-2 text-xs leading-5 text-muted-foreground">
                    {label(p.description)}
                  </p>
                  <p className="mt-4 text-xs font-medium">
                    {t(($) => $.steps_count, { count: p.steps.length })}
                  </p>
                </button>
              );
            })}
          </div>
          {project && (
            <div className="mt-5 rounded-2xl border bg-card p-5">
              <div className="flex flex-wrap items-start justify-between gap-4">
                <div>
                  <h3 className="font-bold">{t(($) => $.materials)}</h3>
                  <p className="mt-1 text-xs text-muted-foreground">
                    {t(($) => $.materials_hint)}
                  </p>
                </div>
                <span
                  className={`text-xs font-medium ${missing ? "text-destructive" : "text-muted-foreground"}`}
                >
                  {missing ? t(($) => $.missing) : t(($) => $.ready)}
                </span>
              </div>
              <div className="my-5 flex flex-wrap gap-2">
                {materials.map((m) => (
                  <span
                    key={m.partId}
                    className={`rounded-lg border px-3 py-2 font-mono text-xs ${m.missing ? "border-destructive text-destructive" : "bg-muted/40"}`}
                  >
                    {`${m.partId} × ${m.required}`}
                    {m.missing > 0 &&
                      ` · ${t(($) => $.short_count, { count: m.missing })}`}
                  </span>
                ))}
              </div>
              <Button
                onClick={() => void start(false)}
                disabled={create.isPending || missing}
                className="h-11 w-full rounded-xl font-bold sm:w-auto"
              >
                {create.isPending ? (
                  <LoaderCircle className="size-4 animate-spin" />
                ) : (
                  <ArrowRight className="size-4" />
                )}
                {t(($) => $.start)}
              </Button>
              <p className="mt-3 text-xs leading-5 text-muted-foreground">
                {t(($) => $.reference_only)}
              </p>
            </div>
          )}
          <div className="mt-7 rounded-2xl border bg-card p-5">
            <label
              htmlFor="circuit-idea"
              className="flex items-center gap-2 font-bold"
            >
              <Sparkles className="size-4" />
              {t(($) => $.idea)}
            </label>
            <p className="mb-4 mt-2 text-xs leading-5 text-muted-foreground">
              {t(($) => $.idea_hint)}
            </p>
            <Textarea
              id="circuit-idea"
              value={prompt}
              maxLength={280}
              onChange={(e) => {
                setPrompt(e.target.value);
                create.reset();
              }}
              placeholder={t(($) => $.placeholder)}
              disabled={data.ai_available !== true || create.isPending}
              className="min-h-24 bg-background"
            />
            <div className="mt-3 flex items-center justify-between gap-3">
              <p className="text-xs text-muted-foreground">
                {data.ai_available === true
                  ? t(($) => $.ai_scope)
                  : t(($) => $.ai_unavailable)}
              </p>
              <Button
                onClick={() => void start(true)}
                disabled={
                  !prompt.trim() ||
                  create.isPending ||
                  data.ai_available !== true
                }
              >
                {create.isPending ? (
                  <LoaderCircle className="size-4 animate-spin" />
                ) : (
                  <Sparkles className="size-4" />
                )}
                {t(($) => $.generate)}
              </Button>
            </div>
          </div>
          {create.isError && (
            <div className="mt-4">
              <ErrorNotice error={create.error} />
            </div>
          )}
        </section>
        <aside className="space-y-5">
          <section className="rounded-2xl border bg-card p-5">
            <h2 className="flex items-center gap-2 font-bold">
              <Layers className="size-4" />
              {t(($) => $.shelf)}
            </h2>
            <p className="mb-4 mt-2 text-xs leading-5 text-muted-foreground">
              {t(($) => $.shelf_hint)}
            </p>
            <details>
              <summary className="cursor-pointer text-sm font-medium">
                {t(($) => $.edit_materials)}
              </summary>
              <div className="mt-4 space-y-3">
                {catalog.parts.map((p) => (
                  <label
                    key={p.id}
                    className="flex items-center justify-between gap-3"
                  >
                    <span className="min-w-0 text-xs">
                      <span className="mr-2 inline-block min-w-7 font-mono font-bold">
                        {p.id}
                      </span>
                      {label(p.name)}
                    </span>
                    <Input
                      type="number"
                      min={0}
                      max={p.quantity}
                      step={1}
                      aria-label={`${p.id} ${label(p.name)}`}
                      value={inventory[p.id] ?? 0}
                      onChange={(e) => {
                        const n = Number(e.target.value);
                        setInventory((prev) => ({
                          ...prev,
                          [p.id]: Math.max(
                            0,
                            Math.min(
                              p.quantity,
                              Number.isFinite(n) ? Math.trunc(n) : 0,
                            ),
                          ),
                        }));
                        create.reset();
                      }}
                      className="h-8 w-16 shrink-0 text-center"
                    />
                  </label>
                ))}
              </div>
            </details>
          </section>
          <section className="rounded-2xl border bg-card p-5">
            <h2 className="font-bold">{t(($) => $.before)}</h2>
            <ul className="mt-3 space-y-3">
              {catalog.preparation.map((p, i) => (
                <li
                  key={i}
                  className="flex gap-2 text-xs leading-6 text-muted-foreground"
                >
                  <Check className="mt-1.5 size-3 shrink-0" />
                  {label(p)}
                </li>
              ))}
            </ul>
          </section>
          <section className="rounded-2xl border bg-card p-5">
            <h2 className="font-bold">{t(($) => $.saved)}</h2>
            <p className="mt-1 text-xs text-muted-foreground">
              {t(($) => $.saved_hint)}
            </p>
            {list.isPending ? (
              <p className="mt-4 text-sm text-muted-foreground">
                {t(($) => $.loading)}
              </p>
            ) : list.isError ? (
              <div className="mt-4">
                <ErrorNotice retry={() => void list.refetch()} />
              </div>
            ) : list.data.creations.length === 0 ? (
              <p className="mt-4 text-sm text-muted-foreground">
                {t(($) => $.empty)}
              </p>
            ) : (
              <ul className="mt-3 divide-y">
                {list.data.creations.map((v) => (
                  <li key={v.id}>
                    <AppLink
                      href={paths.circuitDetail(v.id)}
                      className="flex items-center justify-between gap-3 py-3 text-sm hover:underline"
                    >
                      <span className="truncate">{v.title}</span>
                      <ArrowRight className="size-4 shrink-0" />
                    </AppLink>
                  </li>
                ))}
              </ul>
            )}
          </section>
        </aside>
      </div>
    </div>
  );
}

export function CircuitDetailPage({ creationId }: { creationId: string }) {
  const wsId = useWorkspaceId();
  const paths = useWorkspacePaths();
  const { t } = useT("circuit");
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
          <CircuitWorkbench
            key={`${wsId}:${creationId}`}
            wsId={wsId}
            creation={query.data}
          />
        )}
      </div>
    </main>
  );
}

function CircuitWorkbench({
  wsId,
  creation,
}: {
  wsId: string;
  creation: CircuitCreation;
}) {
  const { t, i18n } = useT("circuit");
  const { document: doc } = creation;
  const progress = useCircuitProgress(wsId, creation.id);
  const [previewStep, setPreviewStep] = useState<number | null>(null);
  const [overview, setOverview] = useState(false);
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
          {canBuild ? t(($) => $.checks_passed) : t(($) => $.checks_failed)}
        </span>
        <span className="text-muted-foreground">
          {t(($) => $.reference_only)}
        </span>
      </div>
      {!canBuild ? (
        <ErrorNotice />
      ) : (
        <div className="grid items-start gap-6 lg:grid-cols-[minmax(0,1fr)_340px]">
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
                <CircuitBoard
                  label={doc.title}
                  parts={doc.parts}
                  placements={visible}
                  columns={doc.columns}
                  rows={doc.rows}
                  highlighted={overview ? [] : step?.placement_ids}
                  layer={layer}
                />
              </div>
            </div>
            {enlarged && (
              <p className="mt-2 text-xs text-muted-foreground">
                {t(($) => $.pan_diagram)}
              </p>
            )}
            <p className="mt-4 text-xs leading-6 text-muted-foreground">
              {t(($) => $.board_hint)}
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
              {t(($) => $.source)}
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
            {progress.isError && <ErrorNotice error={progress.error} />}
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
