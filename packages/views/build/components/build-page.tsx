"use client";

import { useRef, useState } from "react";
import { useInfiniteQuery, useQuery } from "@tanstack/react-query";
import {
  ArrowRight,
  Blocks,
  Lightbulb,
  LoaderCircle,
  MessageCircle,
  Plus,
  Send,
  Zap,
} from "lucide-react";
import { Button } from "@chimii/ui/components/ui/button";
import { Textarea } from "@chimii/ui/components/ui/textarea";
import { useWorkspaceId } from "@chimii/core/hooks";
import { useWorkspacePaths } from "@chimii/core/paths";
import { generateUUID } from "@chimii/core/utils";
import { useConfigStore } from "@chimii/core/config";
import {
  buildConversationOptions,
  buildCreationOptions,
  useSendBuildMessage,
  useCancelBuildSession,
  type BuildKind,
  type BuildMessageInput,
} from "@chimii/core/build";
import {
  circuitKitsOptions,
  circuitCatalogOptions,
  circuitInventoryOptions,
  circuitCreationOptions,
  circuitText,
  circuitMaterials,
} from "@chimii/core/circuit";
import { AppLink, useNavigation } from "../../navigation";
import { useT } from "../../i18n";
import { CircuitInventoryPanel } from "../../circuit/circuit-inventory";
import { CircuitWorkbench } from "../../circuit/circuit-workbench";
import { BuildResult } from "./build-result";
import { ChildModeLauncher } from "./child-mode-controls";

type ResultRef = { kind: "brick" | "circuit"; id: string };
export function BuildPage({
  initialKind = "auto",
}: {
  initialKind?: BuildKind;
}) {
  const wsId = useWorkspaceId();
  const navigation = useNavigation();
  return (
    <ConversationStudio
      key={`${wsId}:${navigation.searchParams.toString()}`}
      initialKind={initialKind}
    />
  );
}
function ConversationStudio({ initialKind }: { initialKind: BuildKind }) {
  const { t, i18n } = useT("build");
  const { t: ct } = useT("circuit");
  const wsId = useWorkspaceId();
  const paths = useWorkspacePaths();
  const navigation = useNavigation();
  const conversationId = navigation.searchParams.get("conversation") || "";
  const sourceId = navigation.searchParams.get("source_id") || "";
  const sourceKind = navigation.searchParams.get("source_kind");
  const sourceHash = navigation.searchParams.get("source_hash") || "";
  const history = useInfiniteQuery(
    buildConversationOptions(wsId, conversationId),
  );
  const session = history.data?.pages[0]?.session;
  const messages =
    history.data?.pages
      .slice()
      .reverse()
      .flatMap((page) => page.messages) || [];
  const routeMode = navigation.searchParams.get("mode");
  const [mode, setMode] = useState<BuildKind | null>(
    routeMode === "auto" || routeMode === "brick" || routeMode === "circuit"
      ? routeMode
      : sourceKind === "brick" || sourceKind === "circuit"
        ? sourceKind
        : null,
  );
  const preferredMode = mode ?? initialKind;
  const kind =
    session?.status === "clarifying" && session.kind !== "auto"
      ? (session.kind ?? preferredMode)
      : preferredMode;
  const [draft, setDraft] = useState("");
  const [selectedKit, setSelectedKit] = useState("");
  const [selectedResult, setSelectedResult] = useState<ResultRef | null>(null);
  const [pane, setPane] = useState<"chat" | "result">("chat");
  const send = useSendBuildMessage(wsId, conversationId || undefined);
  const cancel = useCancelBuildSession();
  const inFlight = useRef(false);
  const sourceCircuit = useQuery(
    circuitCreationOptions(wsId, sourceKind === "circuit" ? sourceId : ""),
  );
  const kits = useQuery(circuitKitsOptions(wsId));
  const kitId =
    selectedKit ||
    session?.circuit?.kit_id ||
    history.data?.pages[0]?.result?.circuit?.kit_id ||
    sourceCircuit.data?.document.kit_id ||
    kits.data?.kits[0]?.kit_id ||
    "";
  const catalog = useQuery({
    ...circuitCatalogOptions(wsId, kitId),
    enabled: !!wsId && !!kitId,
  });
  const inventory = useQuery(circuitInventoryOptions(wsId, kitId));
  const buildAvailable = useConfigStore((state) => state.buildAvailable);
  const configLoaded = useConfigStore((state) => state.buildConfigLoaded);
  const resultMessage =
    history.data?.pages[0]?.result ||
    [...messages]
      .reverse()
      .find(
        (message) =>
          message.metadata.creation_id || message.metadata.circuit_creation_id,
      )?.metadata;
  const newest: ResultRef | null = session?.creation_id
    ? { kind: "brick", id: session.creation_id }
    : session?.circuit_creation_id
      ? { kind: "circuit", id: session.circuit_creation_id }
      : resultMessage?.creation_id
        ? { kind: "brick", id: resultMessage.creation_id }
        : resultMessage?.circuit_creation_id
          ? { kind: "circuit", id: resultMessage.circuit_creation_id }
          : sourceId && (sourceKind === "brick" || sourceKind === "circuit")
            ? { kind: sourceKind, id: sourceId }
            : null;
  const result = selectedResult || newest;
  const brick = useQuery(
    buildCreationOptions(wsId, result?.kind === "brick" ? result.id : ""),
  );
  const circuit = useQuery(
    circuitCreationOptions(wsId, result?.kind === "circuit" ? result.id : ""),
  );
  const isWorking =
    !session?.expired &&
    (session?.status === "queued" || session?.status === "generating");
  const isQuestion =
    !session?.expired && session?.status === "clarifying" && !!session.question;
  const disabled =
    isWorking ||
    send.isPending ||
    cancel.isPending ||
    (!!conversationId && !session);
  const errorCopy = (code?: string) =>
    code === "BUILD_CANCELLED"
      ? t(($) => $.cancelled)
      : code === "CIRCUIT_INVENTORY_CHANGED"
        ? ct(($) => $.inventory_conflict)
        : code === "CIRCUIT_CATALOG_CHANGED"
          ? ct(($) => $.catalog_changed)
          : code === "CIRCUIT_VALIDATION_FAILED"
            ? ct(($) => $.validation_failed)
            : code === "BUILD_INSUFFICIENT_INVENTORY"
              ? t(($) => $.failed_inventory)
              : code === "BUILD_UNSUPPORTED"
                ? t(($) => $.failed_unsupported)
                : code === "BUILD_REQUIREMENTS_UNMET"
                  ? t(($) => $.failed_requirements)
                  : t(($) => $.failed_description);
  const submit = async (
    value: string,
    projectId?: string,
    retry?: BuildMessageInput,
  ) => {
    if (inFlight.current || (!retry && disabled) || !value.trim()) return;
    inFlight.current = true;
    const chosenKind = projectId ? "circuit" : kind;
    const input: BuildMessageInput = retry || {
      prompt: value.trim(),
      client_request_id: generateUUID(),
      kind: chosenKind,
      circuit: {
        kit_id: kitId,
        catalog_version: catalog.data?.catalog.version || "",
        inventory_revision: inventory.data?.revision || 0,
        locale: i18n.language,
        ...(projectId ? { project_id: projectId } : {}),
      },
      ...(session
        ? {
            expected_session_id: session.id,
            expected_revision: session.revision,
            ...(isQuestion ? { question_id: session.question?.id } : {}),
          }
        : {}),
      ...(!isQuestion &&
      result &&
      (chosenKind === "auto" || result.kind === chosenKind)
        ? {
            source_creation_id: result.id,
            source_kind: result.kind,
            expected_content_hash:
              result.kind === "brick"
                ? brick.data?.build_plan.content_hash || sourceHash
                : circuit.data?.document.content_hash || sourceHash,
          }
        : {}),
    };
    try {
      const next = await send.mutateAsync(input);
      setDraft("");
      setSelectedResult(null);
      const params = new URLSearchParams({
        conversation: next.conversation_id || next.id,
      });
      if (preferredMode !== initialKind) params.set("mode", preferredMode);
      if (params.toString() !== navigation.searchParams.toString())
        navigation.replace(`${navigation.pathname}?${params}`);
    } catch {
      /* Keep the exact request for safe retry after a lost response. */
    } finally {
      inFlight.current = false;
    }
  };
  const newConversation = () =>
    navigation.push(kind === "circuit" ? paths.circuit() : paths.build());
  return (
    <main className="circuit-studio h-full overflow-y-auto bg-background text-foreground">
      <div className="mx-auto max-w-[1600px] px-4 py-6 sm:px-7">
        <header className="mb-6 flex flex-wrap items-center justify-between gap-4">
          <div className="flex items-center gap-3">
            <div className="flex size-12 -rotate-3 items-center justify-center rounded-2xl border-2 border-foreground bg-primary/15">
              <Lightbulb className="size-6" />
            </div>
            <div>
              <p className="text-xs font-bold tracking-widest text-muted-foreground">
                CHIMII
              </p>
              <h1 className="text-xl font-black">
                {t(($) => $.conversation_title)}
              </h1>
            </div>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <ChildModeLauncher />
            {conversationId && (
              <Button variant="outline" onClick={newConversation}>
                <Plus className="size-4" />
                {t(($) => $.conversation_new)}
              </Button>
            )}
            <Button
              variant="outline"
              nativeButton={false}
              render={<AppLink href={paths.creations()} />}
            >
              {t(($) => $.nav_creations)}
              <ArrowRight className="size-4" />
            </Button>
          </div>
        </header>
        <div
          className="mb-4 flex gap-2 lg:hidden"
          role="group"
          aria-label={t(($) => $.conversation_view)}
        >
          <Button
            variant={pane === "chat" ? "default" : "outline"}
            onClick={() => setPane("chat")}
          >
            <MessageCircle className="size-4" />
            {t(($) => $.conversation_chat)}
          </Button>
          <Button
            variant={pane === "result" ? "default" : "outline"}
            disabled={!result}
            onClick={() => setPane("result")}
          >
            <Blocks className="size-4" />
            {t(($) => $.conversation_result)}
          </Button>
        </div>
        <div className="grid items-start gap-6 lg:grid-cols-[minmax(310px,.75fr)_minmax(0,1.5fr)]">
          <section
            aria-label={t(($) => $.conversation_chat)}
            className={`${pane === "chat" ? "" : "hidden lg:block"} min-w-0 rounded-3xl border bg-card p-5 shadow-sm`}
          >
            {!conversationId && (
              <div className="mb-6">
                <span className="inline-flex rounded-full bg-primary/10 px-3 py-1 text-xs font-bold">
                  {t(($) => $.conversation_badge)}
                </span>
                <h2 className="mt-4 text-3xl font-black leading-tight">
                  {t(($) => $.conversation_welcome)}
                </h2>
                <p className="mt-3 text-sm leading-7 text-muted-foreground">
                  {t(($) => $.conversation_hint)}
                </p>
              </div>
            )}
            {history.hasNextPage && (
              <Button
                variant="ghost"
                disabled={history.isFetchingNextPage}
                onClick={() => void history.fetchNextPage()}
              >
                {t(($) => $.conversation_older)}
              </Button>
            )}
            {history.isError && (
              <div role="alert">
                <p>{t(($) => $.session_fetch_error)}</p>
                <Button
                  variant="outline"
                  onClick={() => void history.refetch()}
                >
                  {t(($) => $.retry_fetch)}
                </Button>
              </div>
            )}
            {conversationId && history.isPending && (
              <p role="status">{t(($) => $.understanding)}</p>
            )}
            {conversationId && session && messages.length === 0 && (
              <p className="mb-4 whitespace-pre-wrap text-sm">
                {session.prompt}
              </p>
            )}
            <ol
              className="space-y-4"
              aria-label={t(($) => $.conversation_history)}
            >
              {messages.map((message) => (
                <li
                  key={message.id}
                  className={`rounded-2xl p-4 ${message.role === "user" ? "ml-5 bg-primary/10" : "mr-3 border bg-background"}`}
                >
                  <p className="mb-2 text-xs font-bold text-muted-foreground">
                    {message.role === "user"
                      ? t(($) => $.conversation_you)
                      : t(($) => $.conversation_assistant)}
                  </p>
                  <p className="whitespace-pre-wrap break-words text-sm leading-7">
                    {message.kind === "error" && !message.metadata.message
                      ? errorCopy(message.metadata.error)
                      : message.content || t(($) => $.conversation_ready)}
                  </p>
                  {(message.metadata.creation_id ||
                    message.metadata.circuit_creation_id) && (
                    <Button
                      variant="outline"
                      size="sm"
                      className="mt-3"
                      onClick={() => {
                        setSelectedResult(
                          message.metadata.creation_id
                            ? {
                                kind: "brick",
                                id: message.metadata.creation_id,
                              }
                            : {
                                kind: "circuit",
                                id: message.metadata.circuit_creation_id!,
                              },
                        );
                        setPane("result");
                      }}
                    >
                      {t(($) => $.conversation_open)}
                    </Button>
                  )}
                </li>
              ))}
            </ol>
            {send.isPending && (
              <p role="status" className="my-4 rounded-xl bg-muted p-3 text-sm">
                {send.variables?.prompt} · {t(($) => $.conversation_sending)}
              </p>
            )}
            {send.isError && (
              <div
                role="alert"
                className="my-4 rounded-xl border border-destructive/40 p-4"
              >
                <p className="break-words text-sm">{send.variables?.prompt}</p>
                <p className="mt-2 text-sm">
                  {t(($) => $.conversation_send_error)}
                </p>
                <Button
                  variant="outline"
                  className="mt-3"
                  onClick={() => {
                    if (send.variables)
                      void submit(
                        send.variables.prompt,
                        undefined,
                        send.variables,
                      );
                  }}
                >
                  {t(($) => $.conversation_retry)}
                </Button>
              </div>
            )}
            {isWorking && (
              <div
                role="status"
                className="my-5 flex items-center gap-2 text-sm"
              >
                <LoaderCircle className="size-4 animate-spin" />
                {session.phase === "planning"
                  ? t(($) => $.understanding)
                  : t(($) => $.generating)}
              </div>
            )}
            {(isWorking || isQuestion) && (
              <Button
                variant="ghost"
                disabled={cancel.isPending || send.isPending}
                className="my-3"
                onClick={() => {
                  if (session)
                    void cancel
                      .mutateAsync({
                        sessionId: session.id,
                        revision: session.revision || 1,
                      })
                      .catch(() => {});
                }}
              >
                {t(($) => $.conversation_cancel)}
              </Button>
            )}
            {cancel.isError && (
              <p role="alert" className="text-sm">
                {t(($) => $.cancel_error)}
              </p>
            )}
            {isQuestion && (
              <div className="mt-4 flex flex-wrap gap-2">
                {session.question?.choices?.map((choice) => (
                  <Button
                    key={choice.id}
                    variant="outline"
                    disabled={disabled}
                    onClick={() => void submit(choice.id)}
                  >
                    {choice.label}
                  </Button>
                ))}
              </div>
            )}
            <div className="mt-6 border-t pt-5">
              <fieldset
                disabled={disabled || (isQuestion && session?.kind !== "auto")}
                className="mb-4 flex flex-wrap gap-2"
              >
                <legend className="mb-2 text-xs font-bold text-muted-foreground">
                  {t(($) => $.conversation_mode)}
                </legend>
                {(["auto", "brick", "circuit"] as const).map((value) => (
                  <Button
                    key={value}
                    variant={kind === value ? "default" : "outline"}
                    size="sm"
                    aria-pressed={kind === value}
                    onClick={() => {
                      setMode(value);
                      send.reset();
                    }}
                  >
                    {value === "auto"
                      ? t(($) => $.conversation_auto)
                      : value === "brick"
                        ? t(($) => $.conversation_brick)
                        : t(($) => $.conversation_circuit)}
                  </Button>
                ))}
              </fieldset>
              <label htmlFor="creation-message" className="sr-only">
                {t(($) => $.conversation_message)}
              </label>
              <Textarea
                id="creation-message"
                value={draft}
                disabled={disabled}
                maxLength={isQuestion ? 120 : 280}
                onChange={(event) => {
                  setDraft(event.target.value);
                  send.reset();
                }}
                onKeyDown={(event) => {
                  if (
                    !event.nativeEvent.isComposing &&
                    (event.metaKey || event.ctrlKey) &&
                    event.key === "Enter"
                  ) {
                    event.preventDefault();
                    void submit(draft);
                  }
                }}
                placeholder={
                  isQuestion
                    ? t(($) => $.conversation_placeholder)
                    : t(($) => $.conversation_placeholder)
                }
                className="min-h-28 resize-y rounded-xl text-base"
              />
              <div className="mt-3 flex items-center justify-between gap-3">
                <span className="text-xs text-muted-foreground">
                  {t(($) => $.shortcut)}
                </span>
                <Button
                  disabled={
                    disabled ||
                    !draft.trim() ||
                    (configLoaded && !buildAvailable)
                  }
                  onClick={() => void submit(draft)}
                >
                  <Send className="size-4" />
                  {t(($) => $.conversation_send)}
                </Button>
              </div>
              {configLoaded && !buildAvailable && (
                <p className="mt-3 text-xs leading-6 text-muted-foreground">
                  {t(($) => $.conversation_no_ai)}
                </p>
              )}
            </div>
            {kind !== "brick" && (
              <div className="mt-6 space-y-4 border-t pt-5">
                <label
                  htmlFor="circuit-kit"
                  className="block text-xs font-bold"
                >
                  {ct(($) => $.kit)}
                </label>
                <select
                  id="circuit-kit"
                  value={kitId}
                  disabled={disabled}
                  onChange={(event) => {
                    setSelectedKit(event.target.value);
                    send.reset();
                  }}
                  className="w-full rounded-xl border bg-background px-3 py-3 text-sm"
                >
                  {kits.data?.kits.map((kit) => (
                    <option key={kit.kit_id} value={kit.kit_id}>
                      {kit.name}
                    </option>
                  ))}
                </select>
                {(kits.isError || catalog.isError || inventory.isError) && (
                  <div role="alert">
                    <p>{ct(($) => $.error)}</p>
                    <Button
                      variant="outline"
                      onClick={() => {
                        void kits.refetch();
                        void catalog.refetch();
                        void inventory.refetch();
                      }}
                    >
                      {ct(($) => $.retry)}
                    </Button>
                  </div>
                )}
                {catalog.data && inventory.data && (
                  <CircuitInventoryPanel
                    key={kitId}
                    wsId={wsId}
                    catalog={catalog.data.catalog}
                    saved={inventory.data}
                  />
                )}
                <p className="text-xs font-bold">
                  {t(($) => $.conversation_references)}
                </p>
                <div className="grid gap-2">
                  {catalog.data?.catalog.projects.map((project) => {
                    const confirmed =
                      inventory.data?.confirmed === true &&
                      inventory.data.catalog_version ===
                        catalog.data.catalog.version;
                    const missing = circuitMaterials(
                      project,
                      inventory.data?.quantities ?? {},
                    ).filter((part) => part.missing > 0);
                    return (
                      <div key={project.id}>
                        <Button
                          variant="outline"
                          className="h-auto w-full justify-start whitespace-normal py-3 text-left"
                          disabled={
                            disabled || !confirmed || missing.length > 0
                          }
                          onClick={() =>
                            void submit(
                              circuitText(project.title, i18n.language),
                              project.id,
                            )
                          }
                        >
                          <Zap className="size-4 shrink-0" />
                          {circuitText(project.title, i18n.language)}
                        </Button>
                        {confirmed && missing.length > 0 && (
                          <p className="mt-1 text-xs text-muted-foreground">
                            {ct(($) => $.missing)} ·{" "}
                            {missing
                              .map(
                                (part) =>
                                  `${part.partId}: ${ct(($) => $.short_count, { count: part.missing })}`,
                              )
                              .join(" · ")}
                          </p>
                        )}
                      </div>
                    );
                  })}
                </div>
              </div>
            )}
          </section>
          <section
            aria-label={t(($) => $.conversation_result)}
            className={`${pane === "result" ? "" : "hidden lg:block"} min-w-0 rounded-3xl border bg-card p-4 sm:p-6`}
          >
            {result ? (
              <>
                <div className="mb-5 flex flex-wrap items-center justify-between gap-2">
                  <span className="rounded-full bg-primary/10 px-3 py-1 text-xs font-bold">
                    {result.kind === "brick"
                      ? t(($) => $.conversation_brick)
                      : t(($) => $.conversation_circuit)}
                  </span>
                  <AppLink
                    className="text-xs underline underline-offset-4"
                    href={
                      result.kind === "brick"
                        ? paths.creationDetail(result.id)
                        : paths.circuitDetail(result.id)
                    }
                  >
                    {t(($) => $.conversation_full)}
                  </AppLink>
                </div>
                {(result.kind === "brick" ? brick.isError : circuit.isError) ? (
                  <div role="alert">
                    <p>{t(($) => $.creation_fetch_description)}</p>
                    <Button
                      onClick={() =>
                        void (result.kind === "brick"
                          ? brick.refetch()
                          : circuit.refetch())
                      }
                    >
                      {t(($) => $.retry_fetch)}
                    </Button>
                  </div>
                ) : result.kind === "brick" && brick.data ? (
                  <BuildResult
                    key={brick.data.id}
                    creation={brick.data}
                    embedded
                  />
                ) : result.kind === "circuit" && circuit.data ? (
                  <CircuitWorkbench
                    key={circuit.data.id}
                    wsId={wsId}
                    creation={circuit.data}
                    embedded
                  />
                ) : (
                  <p role="status">{t(($) => $.opening_plan)}</p>
                )}
              </>
            ) : (
              <div className="flex min-h-96 flex-col items-center justify-center px-5 text-center">
                <Blocks className="mb-5 size-16 text-primary/60" />
                <h2 className="text-2xl font-black">
                  {t(($) => $.conversation_empty)}
                </h2>
                <p className="mt-3 max-w-md text-sm leading-7 text-muted-foreground">
                  {t(($) => $.conversation_empty_hint)}
                </p>
              </div>
            )}
          </section>
        </div>
      </div>
    </main>
  );
}
