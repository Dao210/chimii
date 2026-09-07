"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  ArrowLeft,
  ArrowRight,
  Lightbulb,
  LoaderCircle,
  Plus,
  Radio,
  Sparkles,
  Volume2,
} from "lucide-react";
import { Button } from "@chimii/ui/components/ui/button";
import { Textarea } from "@chimii/ui/components/ui/textarea";
import { useWorkspaceId } from "@chimii/core/hooks";
import { useWorkspacePaths } from "@chimii/core/paths";
import { useBuildConversation, type BuildSession } from "@chimii/core/build";
import {
  circuitKitsOptions,
  circuitCatalogOptions,
  circuitInventoryOptions,
  circuitCreationOptions,
  circuitMaterials,
  circuitText,
} from "@chimii/core/circuit";
import { AppLink, useNavigation } from "../navigation";
import { useT } from "../i18n";
import {
  CreationHistory,
  CreationRequestNotice,
} from "../common/creation-history";
import { ChildModeLauncher } from "../build/components/child-mode-controls";
import { CircuitInventoryPanel } from "./circuit-inventory";
import { CircuitBoard } from "./circuit-board";
import { CircuitModuleBoard } from "./circuit-module-board";
import {
  CircuitWorkbench,
  CircuitErrorNotice as ErrorNotice,
} from "./circuit-workbench";

export function CircuitPage() {
  const wsId = useWorkspaceId();
  const navigation = useNavigation();
  return (
    <CircuitStudio key={`${wsId}:${navigation.searchParams.toString()}`} />
  );
}

function CircuitStudio() {
  const wsId = useWorkspaceId();
  const paths = useWorkspacePaths();
  const navigation = useNavigation();
  const { t, i18n } = useT("circuit");
  const { t: bt } = useT("build");
  const params = navigation.searchParams;
  const conversationId = params.get("conversation") || "";
  const sourceId =
    params.get("source_kind") === "circuit"
      ? params.get("source_id") || ""
      : "";
  const foreignEntry =
    params.get("source_kind") === "brick" || params.get("mode") === "brick";
  const conversation = useBuildConversation(
    wsId,
    "circuit",
    foreignEntry ? "" : conversationId,
  );
  const { session, isWorking, isQuestion } = conversation;
  const creationId = conversation.resultId || sourceId;
  const creation = useQuery(circuitCreationOptions(wsId, creationId));
  const kits = useQuery(circuitKitsOptions(wsId));
  const [selectedKit, setSelectedKit] = useState("");
  const [selectedProject, setSelectedProject] = useState("");
  const [prompt, setPrompt] = useState(params.get("idea") || "");
  const kitId =
    selectedKit ||
    session?.circuit?.kit_id ||
    conversation.history.data?.pages[0]?.result?.circuit?.kit_id ||
    creation.data?.document.kit_id ||
    (!conversationId && !sourceId ? kits.data?.kits[0]?.kit_id : "") ||
    "";
  const catalog = useQuery({
    ...circuitCatalogOptions(wsId, kitId),
    enabled: !!wsId && !!kitId && !foreignEntry,
  });
  const inventory = useQuery(circuitInventoryOptions(wsId, kitId));
  const data = catalog.data?.catalog;
  const project =
    data?.projects.find((p) => p.id === selectedProject) || data?.projects[0];
  const materials = project
    ? circuitMaterials(project, inventory.data?.quantities ?? {})
    : [];
  const missing = materials.some((m) => m.missing > 0);
  const boxReady =
    inventory.data?.confirmed === true &&
    inventory.data.catalog_version === data?.version;
  const aiAvailable = catalog.data?.ai_available === true;
  const disabled =
    conversation.disabled ||
    !data ||
    !inventory.data ||
    (!!creationId && !creation.data);
  const label = (value: { en: string; zh: string }) =>
    circuitText(value, i18n.language);
  const onSent = (next: BuildSession) => {
    setPrompt("");
    const query = new URLSearchParams({
      conversation: next.conversation_id || next.id,
    });
    if (query.toString() !== params.toString())
      navigation.replace(`${paths.circuit()}?${query}`);
  };
  const submit = async (value: string, projectId?: string) => {
    if (disabled || !data || !inventory.data) return;
    const next = await conversation.sendMessage(value, {
      circuit: {
        kit_id: kitId,
        catalog_version: data.version,
        inventory_revision: inventory.data.revision,
        locale: i18n.language,
        ...(projectId ? { project_id: projectId } : {}),
      },
      ...(creation.data
        ? {
            source: {
              kind: "circuit",
              id: creation.data.id,
              hash: sourceId
                ? params.get("source_hash") ||
                  creation.data.document.content_hash
                : creation.data.document.content_hash,
            },
          }
        : {}),
    });
    if (next) onSent(next);
  };
  const otherParams = new URLSearchParams(params);
  otherParams.delete("mode");
  const brickHref =
    foreignEntry || session?.kind === "brick"
      ? `${paths.build()}?${otherParams}`
      : `${paths.build()}?${new URLSearchParams({ idea: session?.prompt || prompt })}`;
  const newCircuit = () => navigation.push(paths.circuit());
  const references = data && (
    <div className="space-y-5">
      <div
        className={
          creationId
            ? "grid gap-3 sm:grid-cols-2 xl:grid-cols-3"
            : "grid gap-3 sm:grid-cols-2"
        }
      >
        {data.projects.map((p, index) => {
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
              aria-label={label(p.title)}
              aria-pressed={project?.id === p.id}
              disabled={disabled || isQuestion}
              onClick={() => {
                setSelectedProject(p.id);
                conversation.send.reset();
              }}
              className={`min-w-0 rounded-2xl border-2 bg-card p-4 text-left transition-colors focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-ring disabled:opacity-60 ${project?.id === p.id ? "border-primary" : "border-transparent hover:border-border"}`}
            >
              <div className="mb-3 flex items-center justify-between">
                <Icon className="size-5" />
                <span className="font-mono text-xs text-muted-foreground">
                  {String(index + 1).padStart(2, "0")}
                </span>
              </div>
              {data.connection_system === "boson" ? (
                <CircuitModuleBoard
                  miniature
                  label={label(p.title)}
                  parts={data.parts}
                  placements={p.placements}
                  connections={p.connections ?? []}
                  columns={data.columns}
                  rows={data.rows}
                />
              ) : (
                <CircuitBoard
                  miniature
                  label={label(p.title)}
                  parts={data.parts}
                  placements={p.placements}
                  columns={data.columns}
                  rows={data.rows}
                />
              )}
              <h3 className="mt-4 font-bold">{label(p.title)}</h3>
              <p className="mt-2 text-xs leading-5 text-muted-foreground">
                {label(p.description)}
              </p>
              <p className="mt-3 text-xs font-medium">
                {t(($) => $.steps_count, { count: p.steps.length })}
              </p>
            </button>
          );
        })}
      </div>
      {project && (
        <div className="rounded-2xl border bg-card p-5">
          <div className="flex flex-wrap justify-between gap-3">
            <h3 className="font-bold">{t(($) => $.materials)}</h3>
            <p
              className={
                missing
                  ? "text-sm text-destructive"
                  : "text-sm text-muted-foreground"
              }
            >
              {missing ? t(($) => $.missing) : t(($) => $.ready)}
            </p>
          </div>
          <p className="mt-2 text-xs text-muted-foreground">
            {t(($) => $.materials_hint)}
          </p>
          <div className="my-5 flex flex-wrap gap-2">
            {materials.map((m) => (
              <span
                key={m.partId}
                className={`rounded-lg border px-3 py-2 font-mono text-xs ${m.missing ? "border-destructive text-destructive" : "bg-muted/40"}`}
              >
                {m.partId} × {m.required}
                {m.missing > 0 &&
                  ` · ${t(($) => $.short_count, { count: m.missing })}`}
              </span>
            ))}
          </div>
          <Button
            disabled={disabled || isQuestion || !boxReady || missing}
            onClick={() => void submit(label(project.title), project.id)}
          >
            <ArrowRight className="size-4" />
            {t(($) => $.start)}
          </Button>
          <p className="mt-3 text-xs leading-5 text-muted-foreground">
            {t(($) => $.reference_only)}
          </p>
        </div>
      )}
    </div>
  );

  const composer = (
    <section
      className="mx-auto max-w-3xl rounded-2xl border bg-card p-5 sm:p-6"
      aria-label={t(($) => $.discuss)}
    >
      <label
        htmlFor="circuit-idea"
        className="flex items-center gap-2 font-bold"
      >
        <Sparkles className="size-4" />
        {creationId ? t(($) => $.discuss) : t(($) => $.idea)}
      </label>
      <p className="mb-4 mt-2 text-sm leading-6 text-muted-foreground">
        {catalog.data?.composition_available
          ? t(($) => $.composition_hint)
          : t(($) => $.idea_hint)}
      </p>
      {isWorking && (
        <p role="status" className="my-4 flex items-center gap-2">
          <LoaderCircle className="size-4 animate-spin" />
          {bt(($) =>
            session?.phase === "planning" ? $.understanding : $.generating,
          )}
        </p>
      )}
      {(isWorking || isQuestion) && (
        <Button
          className="mb-4"
          variant="ghost"
          disabled={
            conversation.cancel.isPending || conversation.send.isPending
          }
          onClick={() => void conversation.cancelTurn()}
        >
          {bt(($) => $.conversation_cancel)}
        </Button>
      )}
      {isQuestion && (
        <div className="mb-5">
          <h2 className="mb-3 text-xl font-bold">
            {session?.question?.prompt}
          </h2>
          <div className="flex flex-wrap gap-2">
            {session?.question?.choices?.map((choice) => (
              <Button
                variant="outline"
                key={choice.id}
                disabled={disabled}
                onClick={() => void submit(choice.id)}
              >
                {choice.label}
              </Button>
            ))}
          </div>
        </div>
      )}
      {!isQuestion && session?.message && (
        <p role="status" className="mb-4 whitespace-pre-wrap text-sm leading-7">
          {session.message}
        </p>
      )}
      {session?.status === "failed" && (
        <div role="alert" className="mb-4">
          <p className="text-sm">
            {session.message ||
              (session.error === "BUILD_CANCELLED"
                ? bt(($) => $.cancelled)
                : t(($) => $.error))}
          </p>
          <AppLink
            className="mt-3 inline-flex text-sm underline"
            href={brickHref}
          >
            {t(($) => $.bricks)}
          </AppLink>
        </div>
      )}
      {!isQuestion &&
        catalog.data?.composition_available === true &&
        aiAvailable &&
        !creationId && (
          <div className="mb-3 flex flex-wrap gap-2">
            {[
              t(($) => $.starter_button_fan),
              t(($) => $.starter_motion_light),
              t(($) => $.starter_both_fan),
            ].map((idea) => (
              <Button
                key={idea}
                size="sm"
                variant="outline"
                disabled={disabled}
                onClick={() => setPrompt(idea)}
              >
                {idea}
              </Button>
            ))}
          </div>
        )}
      <Textarea
        id="circuit-idea"
        value={prompt}
        maxLength={isQuestion ? 120 : 280}
        disabled={disabled || !aiAvailable}
        onChange={(e) => {
          setPrompt(e.target.value);
          conversation.send.reset();
        }}
        onKeyDown={(e) => {
          if (
            !e.nativeEvent.isComposing &&
            (e.metaKey || e.ctrlKey) &&
            e.key === "Enter"
          ) {
            e.preventDefault();
            void submit(prompt);
          }
        }}
        placeholder={
          isQuestion ? bt(($) => $.answer_label) : t(($) => $.placeholder)
        }
        className="min-h-28 bg-background"
      />
      <div className="mt-4 flex flex-wrap items-center justify-between gap-3">
        <p className="max-w-md text-xs leading-6 text-muted-foreground">
          {aiAvailable
            ? catalog.data?.composition_available
              ? t(($) => $.composition_scope)
              : t(($) => $.ai_scope)
            : t(($) => $.ai_unavailable)}
        </p>
        <Button
          disabled={disabled || !aiAvailable || !prompt.trim()}
          onClick={() => void submit(prompt)}
        >
          {bt(($) => $.conversation_send)}
          <ArrowRight className="size-4" />
        </Button>
      </div>
      <CreationRequestNotice conversation={conversation} onSent={onSent} />
    </section>
  );

  return (
    <main className="circuit-studio h-full overflow-y-auto">
      <div className="mx-auto max-w-[1400px] px-5 py-7 md:px-9 md:py-9">
        <header className="mb-8 flex flex-wrap items-start justify-between gap-4">
          <div>
            <p className="mb-3 text-xs font-bold tracking-widest text-muted-foreground">
              {t(($) => $.eyebrow)}
            </p>
            <h1 className="text-3xl font-black tracking-tight md:text-4xl">
              {t(($) => $.title)}
            </h1>
            <p className="mt-3 max-w-xl text-sm leading-7 text-muted-foreground">
              {t(($) => $.subtitle)}
            </p>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <ChildModeLauncher />
            <Button
              variant="outline"
              nativeButton={false}
              render={<AppLink href={paths.build()} />}
            >
              <ArrowLeft className="size-4" />
              {t(($) => $.bricks)}
            </Button>
            <Button
              variant="outline"
              nativeButton={false}
              render={<AppLink href={paths.creations()} />}
            >
              {bt(($) => $.nav_creations)}
            </Button>
            {(conversationId || sourceId) && (
              <Button variant="outline" onClick={newCircuit}>
                <Plus className="size-4" />
                {t(($) => $.new_circuit)}
              </Button>
            )}
          </div>
        </header>
        {foreignEntry || conversation.readOnly ? (
          <section className="mb-6 rounded-2xl border bg-card p-6">
            <h2 className="text-xl font-bold">
              {bt(($) => $.domain_history_title)}
            </h2>
            <p className="mt-3 leading-7">{bt(($) => $.domain_history_hint)}</p>
            <div className="mt-4 flex flex-wrap gap-4">
              <Button onClick={newCircuit}>{t(($) => $.new_circuit)}</Button>
              <AppLink
                className="inline-flex items-center font-bold underline"
                href={brickHref}
              >
                {t(($) => $.bricks)}
              </AppLink>
            </div>
          </section>
        ) : (
          <>
            <section className="mb-7 flex flex-wrap items-center gap-4 rounded-2xl border bg-card px-5 py-4">
              <label htmlFor="circuit-kit" className="text-xs font-bold">
                {t(($) => $.kit)}
              </label>
              <select
                id="circuit-kit"
                value={kitId}
                disabled={
                  (disabled && !!data) || !!conversationId || !!sourceId
                }
                onChange={(e) => {
                  setSelectedKit(e.target.value);
                  setSelectedProject("");
                  conversation.send.reset();
                }}
                className="min-w-0 max-w-full flex-1 rounded-xl border bg-background px-3 py-3 text-sm sm:flex-none"
              >
                {kits.data?.kits.map((kit) => (
                  <option key={kit.kit_id} value={kit.kit_id}>
                    {kit.name}
                  </option>
                ))}
              </select>
              {data && (
                <span className="text-xs text-muted-foreground">
                  {t(($) => $.age, { age: data.minimum_age })}
                </span>
              )}
            </section>
            {(kits.isError || catalog.isError || inventory.isError) && (
              <ErrorNotice
                retry={() => {
                  void kits.refetch();
                  void catalog.refetch();
                  void inventory.refetch();
                }}
              />
            )}
            {conversation.history.isError && (
              <ErrorNotice retry={() => void conversation.history.refetch()} />
            )}
            {!data && !catalog.isError && !conversation.history.isError && (
              <p role="status" className="my-8 flex items-center gap-2">
                <LoaderCircle className="size-4 animate-spin" />
                {t(($) => $.loading)}
              </p>
            )}
            {data && inventory.data && (
              <details
                className="mb-7 rounded-2xl border bg-card p-4"
                open={!boxReady}
                key={`${kitId}:${boxReady}`}
              >
                <summary className="cursor-pointer text-sm font-bold">
                  {t(($) => $.shelf)} ·{" "}
                  {boxReady
                    ? t(($) => $.box_saved, {
                        revision: inventory.data.revision,
                      })
                    : t(($) => $.box_before_start)}
                </summary>
                <p className="my-4 max-w-3xl text-xs leading-6 text-muted-foreground">
                  {t(($) => $.kit_note)}
                </p>
                <CircuitInventoryPanel
                  key={kitId}
                  wsId={wsId}
                  catalog={data}
                  saved={inventory.data}
                />
              </details>
            )}
            {creationId && (
              <section
                className="mb-7"
                aria-label={bt(($) => $.conversation_result)}
              >
                <AppLink
                  className="mb-4 inline-flex text-sm font-semibold underline underline-offset-4"
                  href={paths.circuitDetail(creationId)}
                >
                  {bt(($) => $.conversation_full)}
                </AppLink>
                {creation.isError ? (
                  <ErrorNotice retry={() => void creation.refetch()} />
                ) : creation.data ? (
                  <CircuitWorkbench
                    key={creation.data.id}
                    wsId={wsId}
                    creation={creation.data}
                  />
                ) : (
                  <p role="status">{bt(($) => $.opening_plan)}</p>
                )}
              </section>
            )}
            {creationId ? (
              <details className="mb-7 rounded-2xl border bg-card p-5">
                <summary className="cursor-pointer font-bold">
                  {bt(($) => $.conversation_references)}
                </summary>
                <div className="mt-5">{references}</div>
              </details>
            ) : (
              <div className="mb-7 grid items-start gap-6 lg:grid-cols-[minmax(0,1fr)_340px]">
                <section className="min-w-0">
                  <h2 className="mb-4 font-bold">{t(($) => $.choose)}</h2>
                  {references}
                </section>
                <aside className="min-w-0 lg:sticky lg:top-6">{composer}</aside>
              </div>
            )}
            {creationId && composer}
          </>
        )}
        <CreationHistory conversation={conversation} />
      </div>
    </main>
  );
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
