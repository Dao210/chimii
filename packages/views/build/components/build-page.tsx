"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  ArrowRight,
  Blocks,
  Lightbulb,
  LoaderCircle,
  Sparkles,
  Wrench,
} from "lucide-react";
import { motion } from "motion/react";
import { Button } from "@chimii/ui/components/ui/button";
import { Textarea } from "@chimii/ui/components/ui/textarea";
import { useWorkspaceId } from "@chimii/core/hooks";
import { useWorkspacePaths } from "@chimii/core/paths";
import {
  buildCreationOptions,
  useBuildConversation,
  type BuildSession,
} from "@chimii/core/build";
import { AppLink, useNavigation } from "../../navigation";
import { BuildResult } from "./build-result";
import { ChildModeLauncher } from "./child-mode-controls";
import { useConfigStore } from "@chimii/core/config";
import {
  CreationHistory,
  CreationRequestNotice,
} from "../../common/creation-history";
import "@chimii/ui/styles/brick-studio.css";
import { useT } from "../../i18n";

export function BuildPage() {
  const wsId = useWorkspaceId();
  const navigation = useNavigation();
  return <BrickStudio key={`${wsId}:${navigation.searchParams.toString()}`} />;
}

function BrickStudio() {
  const { t } = useT("build");
  const workspaceId = useWorkspaceId();
  const workspacePaths = useWorkspacePaths();
  const navigation = useNavigation();
  const params = navigation.searchParams;
  const conversationId = params.get("conversation") || "";
  const sourceId =
    params.get("source_kind") === "brick" ? params.get("source_id") || "" : "";
  const foreignEntry =
    params.get("source_kind") === "circuit" || params.get("mode") === "circuit";
  const conversation = useBuildConversation(
    workspaceId,
    "brick",
    foreignEntry ? "" : conversationId,
  );
  const { session, isWorking, isQuestion } = conversation;
  const [answer, setAnswer] = useState("");
  const [prompt, setPrompt] = useState(params.get("idea") || "");
  const buildAvailable = useConfigStore((state) => state.buildAvailable);
  const buildConfigLoaded = useConfigStore((state) => state.buildConfigLoaded);
  const creationId = conversation.resultId || sourceId;
  const creationQuery = useQuery(buildCreationOptions(workspaceId, creationId));
  const creationIsUnavailable = creationQuery.isError;
  const ideaStarters = [
    t(($) => $.starter_car),
    t(($) => $.starter_dragon),
    t(($) => $.starter_robot),
  ];
  const statusCopy =
    session?.status === "generating"
      ? session.phase === "planning"
        ? t(($) => $.understanding)
        : t(($) => $.generating)
      : t(($) => $.queued);
  const failedDescription =
    session?.error === "BUILD_PLANNER_TIMEOUT"
      ? t(($) => $.failed_timeout)
      : session?.error === "BUILD_SEARCH_LIMIT"
        ? t(($) => $.failed_search_limit)
        : session?.error === "BUILD_UNSUPPORTED"
          ? t(($) => $.failed_unsupported)
          : session?.error === "BUILD_REQUIREMENTS_UNMET"
            ? t(($) => $.failed_requirements)
            : session?.error === "BUILD_CANCELLED"
              ? t(($) => $.cancelled)
              : session?.error === "BUILD_INSUFFICIENT_INVENTORY"
                ? t(($) => $.failed_inventory)
                : session?.error === "BUILD_COUNT_UNSUPPORTED"
                  ? t(($) => $.failed_count)
                  : session?.error === "BUILD_STRUCTURE_INVALID"
                    ? t(($) => $.failed_structure)
                    : t(($) => $.failed_description);
  const onSent = (next: BuildSession) => {
    setAnswer("");
    const query = new URLSearchParams({
      conversation: next.conversation_id || next.id,
    });
    if (query.toString() !== params.toString())
      navigation.replace(`${workspacePaths.build()}?${query}`);
  };
  const discuss = async (value: string) => {
    const creation = creationQuery.data;
    const next = await conversation.sendMessage(value, {
      ...(creation
        ? {
            source: {
              kind: "brick",
              id: creation.id,
              hash: sourceId
                ? params.get("source_hash") || creation.build_plan.content_hash
                : creation.build_plan.content_hash,
            },
          }
        : {}),
    });
    if (next) onSent(next);
    return !!next;
  };
  const start = () => discuss(prompt);
  const sendAnswer = async (value: string) => {
    const next = await conversation.sendMessage(value);
    if (next) onSent(next);
  };
  const reset = () => {
    setPrompt("");
    navigation.push(workspacePaths.build());
  };
  const editIdea = async () => {
    if ((isWorking || isQuestion) && !(await conversation.cancelTurn())) return;
    const query = new URLSearchParams({ idea: session?.prompt || prompt });
    if (creationQuery.data) {
      query.set("source_kind", "brick");
      query.set("source_id", creationQuery.data.id);
      query.set("source_hash", creationQuery.data.build_plan.content_hash);
    }
    navigation.push(`${workspacePaths.build()}?${query}`);
  };
  const otherParams = new URLSearchParams(params);
  otherParams.delete("mode");
  const circuitHref =
    foreignEntry || session?.kind === "circuit"
      ? `${workspacePaths.circuit()}?${otherParams}`
      : `${workspacePaths.circuit()}?${new URLSearchParams({ idea: session?.prompt || prompt })}`;

  return (
    <main className="brick-studio relative isolate h-full overflow-y-auto bg-[var(--brick-paper)] text-[var(--brick-ink)]">
      <div className="pointer-events-none absolute inset-0 opacity-35 [background-image:radial-gradient(var(--brick-grid)_1px,transparent_1px)] [background-size:22px_22px]" />
      <div className="relative mx-auto max-w-[1320px] px-5 py-6 md:px-8 md:py-8">
        <header className="mb-7 flex flex-wrap items-center justify-between gap-4">
          <div className="flex items-center gap-3">
            <div className="flex size-11 rotate-[-4deg] items-center justify-center rounded-2xl border-2 border-[var(--brick-ink)] bg-[var(--brick-yellow)] shadow-[3px_4px_0_var(--brick-ink)]">
              <Blocks className="size-6" />
            </div>
            <div>
              <p className="text-xs font-black uppercase tracking-[.2em] text-[var(--brick-label)]">
                {t(($) => $.brand)}
              </p>
              <h1 className="whitespace-nowrap text-xl font-black">
                {t(($) => $.title)}
              </h1>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <ChildModeLauncher />
            <AppLink
              className="text-sm font-bold underline underline-offset-4"
              href={workspacePaths.circuit()}
            >
              {t(($) => $.open_circuit)}
            </AppLink>
            <Button
              nativeButton={false}
              variant="outline"
              className="rounded-full border-2 border-[var(--brick-ink)] bg-[var(--brick-card)] font-bold"
              render={<AppLink href={workspacePaths.creations()} />}
            >
              {t(($) => $.nav_creations)} <ArrowRight className="size-4" />
            </Button>
          </div>
        </header>

        {session?.status === "completed" && session.message && !creationId && (
          <p
            role="status"
            className="mb-5 rounded-2xl border border-border bg-card p-4 text-foreground"
          >
            {session.message}
          </p>
        )}
        {foreignEntry || conversation.readOnly ? (
          <section className="mx-auto max-w-2xl rounded-3xl border-2 border-border bg-card p-8 text-foreground">
            <h2 className="text-2xl font-black">
              {t(($) => $.domain_history_title)}
            </h2>
            <p className="mt-3 leading-7">{t(($) => $.domain_history_hint)}</p>
            <div className="mt-5 flex flex-wrap gap-3">
              <Button onClick={reset}>{t(($) => $.start_brick)}</Button>
              <AppLink
                className="inline-flex items-center font-bold underline"
                href={circuitHref}
              >
                {t(($) => $.open_circuit)}
              </AppLink>
            </div>
          </section>
        ) : !buildConfigLoaded && !creationId && !conversationId ? (
          <section
            className="mx-auto flex min-h-[560px] max-w-2xl flex-col items-center justify-center text-center"
            aria-live="polite"
          >
            <div className="flex size-20 items-center justify-center rounded-[1.8rem] border-2 border-[var(--brick-ink)] bg-[var(--brick-mint)] shadow-[6px_7px_0_var(--brick-ink)]">
              <LoaderCircle className="size-9 animate-spin" />
            </div>
            <h2 className="mt-7 text-3xl font-black">
              {t(($) => $.loading_studio)}
            </h2>
          </section>
        ) : !buildAvailable && !creationId && !conversationId ? (
          <section className="mx-auto flex min-h-[560px] max-w-2xl flex-col items-center justify-center text-center">
            <div className="flex size-20 items-center justify-center rounded-[1.8rem] border-2 border-[var(--brick-ink)] bg-[var(--brick-yellow)] shadow-[6px_7px_0_var(--brick-ink)]">
              <Wrench className="size-9" />
            </div>
            <p className="mt-7 text-xs font-black uppercase tracking-[.2em] text-[var(--brick-label)]">
              {t(($) => $.config_label)}
            </p>
            <h2 className="mt-3 text-3xl font-black md:text-5xl">
              {t(($) => $.config_title)}
            </h2>
            <p className="mt-4 max-w-xl font-medium leading-7 text-[var(--brick-muted)]">
              {t(($) => $.config_description)}
            </p>
          </section>
        ) : creationQuery.data?.id && !isWorking && !isQuestion ? (
          <>
            {session?.message && (
              <p
                role="status"
                className="mb-5 rounded-2xl border border-border bg-card p-4 text-foreground"
              >
                {session.message}
              </p>
            )}
            {session?.status === "failed" && (
              <p
                role="alert"
                className="mb-5 rounded-2xl border border-destructive/40 bg-card p-4 text-foreground"
              >
                {session.message || failedDescription}
              </p>
            )}
            {session?.status === "failed" && (
              <AppLink
                className="mt-4 inline-flex text-sm font-bold underline"
                href={circuitHref}
              >
                {t(($) => $.open_circuit)}
              </AppLink>
            )}
            <BuildResult
              key={creationQuery.data.id}
              creation={creationQuery.data}
              onAgain={reset}
              onDiscuss={discuss}
              discussDisabled={conversation.disabled || !buildAvailable}
              initialEditing={!!sourceId}
            />
          </>
        ) : creationId && !isWorking && !isQuestion ? (
          <section
            className="mx-auto flex min-h-[560px] max-w-2xl flex-col items-center justify-center text-center"
            aria-live="polite"
          >
            {creationIsUnavailable ? (
              <>
                <div className="flex size-20 items-center justify-center rounded-[1.8rem] border-2 border-[var(--brick-ink)] bg-[var(--brick-yellow)] shadow-[6px_7px_0_var(--brick-ink)]">
                  <Wrench className="size-9" />
                </div>
                <h2 className="mt-7 text-3xl font-black">
                  {t(($) => $.creation_fetch_title)}
                </h2>
                <p className="mt-3 max-w-lg font-medium text-[var(--brick-muted)]">
                  {t(($) => $.creation_fetch_description)}
                </p>
                <Button
                  onClick={() => void creationQuery.refetch()}
                  className="mt-6 rounded-xl bg-[var(--brick-ink)] font-black"
                >
                  {t(($) => $.retry_fetch)}
                </Button>
              </>
            ) : (
              <>
                <div className="flex size-20 items-center justify-center rounded-[1.8rem] border-2 border-[var(--brick-ink)] bg-[var(--brick-mint)] shadow-[6px_7px_0_var(--brick-ink)]">
                  <LoaderCircle className="size-9 animate-spin" />
                </div>
                <h2 className="mt-7 text-3xl font-black">
                  {t(($) => $.opening_plan)}
                </h2>
              </>
            )}
          </section>
        ) : conversationId && !session ? (
          <section
            className="mx-auto max-w-xl py-20 text-center"
            aria-live="polite"
          >
            <p>
              {conversation.history.isError
                ? t(($) => $.session_fetch_error)
                : t(($) => $.understanding)}
            </p>
            {conversation.history.isError && (
              <Button onClick={() => void conversation.history.refetch()}>
                {t(($) => $.retry_fetch)}
              </Button>
            )}
          </section>
        ) : isQuestion && session?.question ? (
          <motion.section
            initial={{ opacity: 0, y: 16 }}
            animate={{ opacity: 1, y: 0 }}
            className="mx-auto max-w-2xl pt-6 md:pt-10"
          >
            <div className="rounded-[2rem] border-2 border-[var(--brick-ink)] bg-[var(--brick-card)] p-7 shadow-[8px_9px_0_var(--brick-ink)] md:p-9">
              <div className="mb-5 flex size-12 items-center justify-center rounded-2xl border-2 border-[var(--brick-ink)] bg-[var(--brick-mint)]">
                <Lightbulb className="size-6" />
              </div>
              <p className="text-sm font-black text-[var(--brick-label)]">
                {t(($) => $.clarify_label)}
              </p>
              <p className="mt-3 break-words text-sm text-muted-foreground">
                {session.prompt}
              </p>
              {session.summary && session.summary !== session.prompt && (
                <p className="mt-2 break-words text-sm font-medium">
                  {session.summary}
                </p>
              )}
              <h2 className="mt-2 text-3xl font-black tracking-tight">
                {session.question.prompt}
              </h2>
              <div className="mt-7 grid gap-3">
                {(session.question.choices?.length
                  ? session.question.choices
                  : session.question.options.map((label) => ({
                      id: label,
                      label,
                    }))
                ).map((choice) => (
                  <button
                    key={choice.id}
                    type="button"
                    disabled={conversation.disabled}
                    onClick={() => void sendAnswer(choice.id)}
                    className="group flex min-h-14 items-center justify-between gap-3 rounded-2xl border-2 border-current bg-background px-5 py-3 text-left font-black transition hover:bg-accent disabled:opacity-50"
                  >
                    <span className="break-words">{choice.label}</span>
                    <ArrowRight className="size-5 shrink-0" />
                  </button>
                ))}
              </div>
              {session.question.allow_free_text !== false && (
                <form
                  key={session.question.id}
                  onSubmit={(event) => {
                    event.preventDefault();
                    void sendAnswer(answer);
                  }}
                  className="mt-5 space-y-3"
                >
                  <label htmlFor="build-answer" className="text-sm font-bold">
                    {t(($) => $.answer_label)}
                  </label>
                  <Textarea
                    id="build-answer"
                    value={answer}
                    maxLength={120}
                    onChange={(event) => setAnswer(event.target.value)}
                    disabled={conversation.disabled}
                    className="min-h-24 bg-background"
                  />
                  <Button
                    type="submit"
                    disabled={!answer.trim() || conversation.disabled}
                  >
                    {conversation.send.isPending && (
                      <LoaderCircle className="size-4 animate-spin" />
                    )}
                    {t(($) => $.answer_submit)}
                  </Button>
                </form>
              )}
              <Button
                variant="ghost"
                onClick={() => void editIdea()}
                disabled={conversation.disabled}
                className="mt-3"
              >
                {t(($) => $.edit_idea)}
              </Button>
            </div>
          </motion.section>
        ) : isWorking ? (
          <section className="mx-auto flex min-h-[560px] max-w-3xl flex-col items-center justify-center text-center">
            <motion.div
              animate={{ rotate: [0, -6, 5, 0], y: [0, -8, 0] }}
              transition={{
                duration: 1.8,
                repeat: Infinity,
                ease: "easeInOut",
              }}
              className="mb-7 flex size-24 items-center justify-center rounded-[2rem] border-2 border-[var(--brick-ink)] bg-[var(--brick-yellow)] shadow-[7px_8px_0_var(--brick-ink)]"
            >
              <LoaderCircle className="size-11 animate-spin" />
            </motion.div>
            <p className="text-xs font-black uppercase tracking-[.2em] text-[var(--brick-label)]">
              {t(($) => $.checking_label)}
            </p>
            <h2 className="mt-3 text-3xl font-black md:text-5xl">
              {statusCopy}
            </h2>
            <p className="mt-4 max-w-lg font-medium text-[var(--brick-muted)]">
              {session?.summary ||
                (session?.phase === "planning"
                  ? t(($) => $.planning_hint)
                  : t(($) => $.checking_hint))}
            </p>
            <Button
              variant="ghost"
              className="mt-5"
              disabled={conversation.cancel.isPending}
              onClick={() => void editIdea()}
            >
              {t(($) => $.edit_idea)}
            </Button>
          </section>
        ) : session?.status === "failed" ? (
          <section className="mx-auto max-w-xl pt-20 text-center">
            <h2 className="text-3xl font-black">
              {session.error === "BUILD_PLANNER_TIMEOUT"
                ? t(($) => $.failed_timeout_title)
                : t(($) => $.failed_title)}
            </h2>
            <p className="mt-3 text-[var(--brick-muted)]">
              {session.message || failedDescription}
            </p>
            <AppLink
              className="mt-4 inline-flex text-sm font-bold underline"
              href={circuitHref}
            >
              {t(($) => $.open_circuit)}
            </AppLink>
            <Button
              onClick={() => void editIdea()}
              className="mt-6 rounded-xl bg-[var(--brick-ink)]"
            >
              {session.error === "BUILD_PLANNER_TIMEOUT"
                ? t(($) => $.back_to_idea)
                : t(($) => $.edit_idea)}
            </Button>
          </section>
        ) : (
          <section className="grid items-center gap-8 pt-5 lg:grid-cols-[minmax(0,1fr)_430px] lg:pt-14">
            <div>
              <motion.div
                initial={{ opacity: 0, y: 12 }}
                animate={{ opacity: 1, y: 0 }}
              >
                <span className="inline-flex items-center gap-2 rounded-full border-2 border-[var(--brick-ink)] bg-[var(--brick-mint)] px-4 py-2 text-xs font-black uppercase tracking-[.12em] shadow-[3px_4px_0_var(--brick-ink)]">
                  <Sparkles className="size-4" /> {t(($) => $.hero_badge)}
                </span>
                <h2 className="mt-7 max-w-3xl text-5xl font-black leading-[.98] tracking-[-.045em] sm:text-6xl xl:text-7xl">
                  {t(($) => $.hero_line_start)}
                  <br />
                  <span className="text-[var(--brick-accent)]">
                    {t(($) => $.hero_accent)}
                  </span>{" "}
                  {t(($) => $.hero_line_end)}
                </h2>
                <p className="mt-6 max-w-xl text-lg font-medium leading-8 text-[var(--brick-body)]">
                  {t(($) => $.hero_description)}
                </p>
              </motion.div>
              <div className="mt-8 flex flex-wrap gap-2">
                {ideaStarters.map((idea) => (
                  <button
                    key={idea}
                    type="button"
                    onClick={() => {
                      setPrompt(idea);
                      conversation.send.reset();
                    }}
                    className="rounded-full border border-[var(--brick-outline)] bg-[var(--brick-chip)] px-4 py-2 text-sm font-bold transition hover:-translate-y-0.5 hover:border-[var(--brick-ink)] hover:bg-[var(--brick-yellow)]"
                  >
                    {idea}
                  </button>
                ))}
              </div>
            </div>

            <motion.div
              initial={{ opacity: 0, rotate: 2, y: 20 }}
              animate={{ opacity: 1, rotate: -1, y: 0 }}
              className="rounded-[2rem] border-2 border-[var(--brick-ink)] bg-[var(--brick-card)] p-5 shadow-[10px_12px_0_var(--brick-ink)] md:p-6"
            >
              <div className="mb-4 flex items-center gap-2 text-sm font-black text-[var(--brick-label)]">
                <Lightbulb className="size-4" /> {t(($) => $.idea_label)}
              </div>
              <Textarea
                aria-label={t(($) => $.idea_label)}
                value={prompt}
                onChange={(event) => {
                  setPrompt(event.target.value.slice(0, 280));
                  conversation.send.reset();
                }}
                onKeyDown={(event) => {
                  if (
                    !event.nativeEvent.isComposing &&
                    (event.metaKey || event.ctrlKey) &&
                    event.key === "Enter"
                  ) {
                    event.preventDefault();
                    void start();
                  }
                }}
                placeholder={t(($) => $.idea_placeholder)}
                className="min-h-44 resize-none rounded-2xl border-2 border-[var(--brick-ink)] bg-[var(--brick-input)] p-4 text-base font-semibold leading-7 placeholder:text-[var(--brick-placeholder)] focus-visible:ring-[var(--brick-accent)]"
              />
              <div className="mt-3 flex items-center justify-between text-xs font-semibold text-[var(--brick-caption)]">
                <span>{t(($) => $.shortcut)}</span>
                <span>{prompt.length}/280</span>
              </div>
              <Button
                onClick={() => void start()}
                disabled={!prompt.trim() || conversation.disabled}
                className="mt-5 h-14 w-full rounded-2xl border-2 border-[var(--brick-ink)] bg-[var(--brick-action)] text-base font-black text-white shadow-[4px_5px_0_var(--brick-ink)] transition hover:-translate-y-0.5 hover:bg-[var(--brick-action-hover)] hover:shadow-[6px_7px_0_var(--brick-ink)] active:translate-y-1 active:shadow-none"
              >
                {conversation.send.isPending ? (
                  <LoaderCircle className="size-5 animate-spin" />
                ) : (
                  <Sparkles className="size-5" />
                )}
                {t(($) => $.start)}
              </Button>
            </motion.div>
          </section>
        )}
        <CreationRequestNotice conversation={conversation} onSent={onSent} />
        <CreationHistory conversation={conversation} />
      </div>
    </main>
  );
}
