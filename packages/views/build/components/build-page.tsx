"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { ArrowRight, Blocks, Lightbulb, LoaderCircle, Sparkles, Wrench } from "lucide-react";
import { motion } from "motion/react";
import { Button } from "@chimii/ui/components/ui/button";
import { Textarea } from "@chimii/ui/components/ui/textarea";
import { useWorkspaceId } from "@chimii/core/hooks";
import { useWorkspacePaths } from "@chimii/core/paths";
import {
  buildCreationOptions,
  buildSessionOptions,
  useCreateBuildSession,
  useSubmitBuildAnswers,
  useCancelBuildSession,
} from "@chimii/core/build";
import { AppLink } from "../../navigation";
import { BuildResult } from "./build-result";
import { ChildModeLauncher } from "./child-mode-controls";
import { useConfigStore } from "@chimii/core/config";
import { generateUUID } from "@chimii/core/utils";
import { useT } from "../../i18n";

export function BuildPage() {
  const { t } = useT("build");
  const workspaceId = useWorkspaceId();
  const workspacePaths = useWorkspacePaths();
  const createSession = useCreateBuildSession();
  const submitAnswers = useSubmitBuildAnswers();
  const cancelSession = useCancelBuildSession();
  const [answer, setAnswer] = useState("");
  const [prompt, setPrompt] = useState("");
  const [clientRequestId, setClientRequestId] = useState(() => generateUUID());
  const [sessionId, setSessionId] = useState("");
  const [friendlyError, setFriendlyError] = useState("");
  const buildAvailable = useConfigStore((state) => state.buildAvailable);
  const buildConfigLoaded = useConfigStore((state) => state.buildConfigLoaded);
  const sessionQuery = useQuery(buildSessionOptions(workspaceId, sessionId));
  const session = sessionQuery.data;
  const creationId = session?.creation_id ?? "";
  const creationQuery = useQuery(buildCreationOptions(workspaceId, creationId));

  const isWorking = session?.status === "queued" || session?.status === "generating";
  const creationIsPending = creationQuery.isPending || creationQuery.isFetching;
  const creationIsUnavailable = creationQuery.isError || (!creationIsPending && !creationQuery.data?.id);
  const ideaStarters = [t($ => $.starter_car), t($ => $.starter_dragon), t($ => $.starter_robot)];
  const statusCopy = session?.status === "generating"
    ? (session.phase === "planning" ? t($ => $.understanding) : t($ => $.generating)) : t($ => $.queued);
	const failedDescription = session?.error === "BUILD_SEARCH_LIMIT" ? t($ => $.failed_search_limit)
    : session?.error === "BUILD_UNSUPPORTED" ? t($ => $.failed_unsupported)
    : session?.error === "BUILD_REQUIREMENTS_UNMET" ? t($ => $.failed_requirements)
    : session?.error === "BUILD_CANCELLED" ? t($ => $.cancelled)
    : session?.error === "BUILD_INSUFFICIENT_INVENTORY"
		? t($ => $.failed_inventory)
		: session?.error === "BUILD_COUNT_UNSUPPORTED"
			? t($ => $.failed_count)
			: session?.error === "BUILD_STRUCTURE_INVALID"
				? t($ => $.failed_structure)
				: t($ => $.failed_description);

  const start = async () => {
    if (!prompt.trim()) return;
    setFriendlyError("");
    try {
      const next = await createSession.mutateAsync({ prompt: prompt.trim(), clientRequestId });
      if (!next.id) throw new Error("Invalid session response");
      setSessionId(next.id);
    } catch {
      setFriendlyError(t($ => $.start_error));
    }
  };

  const reset = () => {
    setPrompt("");
    setClientRequestId(generateUUID());
    setSessionId("");
    setAnswer("");
    setFriendlyError("");
  };

  const sendAnswer = async (value: string) => {
    if (!session?.question || !value.trim()) return;
    setFriendlyError("");
    try {
      const next = await submitAnswers.mutateAsync({ sessionId: session.id, revision: session.revision, answers: { [session.question.id]: value.trim() } });
      if (!next.id) throw new Error("Invalid answer response");
      setAnswer("");
    } catch { setFriendlyError(t($ => $.answer_error)); }
  };

  const editIdea = async () => {
    setFriendlyError("");
    try {
      if (session && session.status !== "failed" && session.status !== "completed") {
        const next = await cancelSession.mutateAsync({ sessionId: session.id, revision: session.revision ?? 1 });
        if (!next.id) throw new Error("Invalid cancellation response");
      }
      setSessionId(""); setAnswer(""); setClientRequestId(generateUUID());
      setPrompt(session?.prompt ?? prompt);
    } catch { setFriendlyError(t($ => $.cancel_error)); }
  };

  return (
    <main className="h-full overflow-y-auto bg-[#f4ead5] text-[#1d241f]">
      <div className="pointer-events-none fixed inset-0 opacity-35 [background-image:radial-gradient(#bfae8d_1px,transparent_1px)] [background-size:22px_22px]" />
      <div className="relative mx-auto max-w-[1320px] px-5 py-6 md:px-8 md:py-8">
        <header className="mb-7 flex flex-wrap items-center justify-between gap-4">
          <div className="flex items-center gap-3">
            <div className="flex size-11 rotate-[-4deg] items-center justify-center rounded-2xl border-2 border-[#1d241f] bg-[#ffd85a] shadow-[3px_4px_0_#1d241f]">
              <Blocks className="size-6" />
            </div>
            <div>
              <p className="text-xs font-black uppercase tracking-[.2em] text-[#39715a]">{t($ => $.brand)}</p>
              <h1 className="whitespace-nowrap text-xl font-black">{t($ => $.title)}</h1>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <ChildModeLauncher />
            <Button nativeButton={false} variant="outline" className="rounded-full border-2 border-[#1d241f] bg-[#fffdf7] font-bold" render={<AppLink href={workspacePaths.creations()} />}>
              {t($ => $.nav_creations)} <ArrowRight className="size-4" />
            </Button>
          </div>
        </header>

        {!buildConfigLoaded ? (
          <section className="mx-auto flex min-h-[560px] max-w-2xl flex-col items-center justify-center text-center" aria-live="polite">
            <div className="flex size-20 items-center justify-center rounded-[1.8rem] border-2 border-[#1d241f] bg-[#a8dfc2] shadow-[6px_7px_0_#1d241f]"><LoaderCircle className="size-9 animate-spin" /></div>
            <h2 className="mt-7 text-3xl font-black">{t($ => $.loading_studio)}</h2>
          </section>
        ) : !buildAvailable ? (
          <section className="mx-auto flex min-h-[560px] max-w-2xl flex-col items-center justify-center text-center">
            <div className="flex size-20 items-center justify-center rounded-[1.8rem] border-2 border-[#1d241f] bg-[#ffd85a] shadow-[6px_7px_0_#1d241f]"><Wrench className="size-9" /></div>
            <p className="mt-7 text-xs font-black uppercase tracking-[.2em] text-[#39715a]">{t($ => $.config_label)}</p>
            <h2 className="mt-3 text-3xl font-black md:text-5xl">{t($ => $.config_title)}</h2>
            <p className="mt-4 max-w-xl font-medium leading-7 text-[#687068]">{t($ => $.config_description)}</p>
          </section>
        ) : creationQuery.data?.id ? (
          <BuildResult key={creationQuery.data.id} creation={creationQuery.data} onAgain={reset} />
        ) : session?.status === "completed" && creationId ? (
          <section className="mx-auto flex min-h-[560px] max-w-2xl flex-col items-center justify-center text-center" aria-live="polite">
            {creationIsUnavailable ? (
              <>
                <div className="flex size-20 items-center justify-center rounded-[1.8rem] border-2 border-[#1d241f] bg-[#ffd85a] shadow-[6px_7px_0_#1d241f]"><Wrench className="size-9" /></div>
                <h2 className="mt-7 text-3xl font-black">{t($ => $.creation_fetch_title)}</h2>
                <p className="mt-3 max-w-lg font-medium text-[#687068]">{t($ => $.creation_fetch_description)}</p>
                <Button onClick={() => void creationQuery.refetch()} className="mt-6 rounded-xl bg-[#1d241f] font-black">{t($ => $.retry_fetch)}</Button>
              </>
            ) : (
              <>
                <div className="flex size-20 items-center justify-center rounded-[1.8rem] border-2 border-[#1d241f] bg-[#a8dfc2] shadow-[6px_7px_0_#1d241f]"><LoaderCircle className="size-9 animate-spin" /></div>
                <h2 className="mt-7 text-3xl font-black">{t($ => $.opening_plan)}</h2>
              </>
            )}
          </section>
        ) : sessionId && !session ? (
          <section className="mx-auto max-w-xl py-20 text-center" aria-live="polite">
            <p>{sessionQuery.isError ? t($ => $.session_fetch_error) : t($ => $.understanding)}</p>
            {sessionQuery.isError && <Button onClick={() => void sessionQuery.refetch()}>{t($ => $.retry_fetch)}</Button>}
          </section>
        ) : session?.status === "clarifying" && session.question ? (
          <motion.section initial={{ opacity: 0, y: 16 }} animate={{ opacity: 1, y: 0 }} className="mx-auto max-w-2xl pt-6 md:pt-10">
            <div className="rounded-[2rem] border-2 border-[#1d241f] bg-[#fffdf7] p-7 shadow-[8px_9px_0_#1d241f] md:p-9">
              <div className="mb-5 flex size-12 items-center justify-center rounded-2xl border-2 border-[#1d241f] bg-[#a8dfc2]">
                <Lightbulb className="size-6" />
              </div>
              <p className="text-sm font-black text-[#39715a]">{t($ => $.clarify_label)}</p>
              <p className="mt-3 break-words text-sm text-muted-foreground">{session.prompt}</p>
              {session.summary && session.summary !== session.prompt && <p className="mt-2 break-words text-sm font-medium">{session.summary}</p>}
              <h2 className="mt-2 text-3xl font-black tracking-tight">{session.question.prompt}</h2>
              <div className="mt-7 grid gap-3">
                {(session.question.choices?.length ? session.question.choices : session.question.options.map((label) => ({ id: label, label }))).map((choice) => (
                  <button key={choice.id} type="button" disabled={submitAnswers.isPending || cancelSession.isPending}
                    onClick={() => void sendAnswer(choice.id)}
                    className="group flex min-h-14 items-center justify-between gap-3 rounded-2xl border-2 border-current bg-background px-5 py-3 text-left font-black transition hover:bg-accent disabled:opacity-50"
                  >
                    <span className="break-words">{choice.label}</span><ArrowRight className="size-5 shrink-0" />
                  </button>
                ))}
              </div>
              {session.question.allow_free_text !== false && (
                <form key={session.question.id} onSubmit={(event) => { event.preventDefault(); void sendAnswer(answer); }} className="mt-5 space-y-3">
                  <label htmlFor="build-answer" className="text-sm font-bold">{t($ => $.answer_label)}</label>
                  <Textarea id="build-answer" value={answer} maxLength={120} onChange={(event) => setAnswer(event.target.value)} disabled={submitAnswers.isPending || cancelSession.isPending} className="min-h-24 bg-background" />
                  <Button type="submit" disabled={!answer.trim() || submitAnswers.isPending || cancelSession.isPending}>{submitAnswers.isPending && <LoaderCircle className="size-4 animate-spin" />}{t($ => $.answer_submit)}</Button>
                </form>
              )}
              <Button variant="ghost" onClick={() => void editIdea()} disabled={submitAnswers.isPending || cancelSession.isPending} className="mt-3">{t($ => $.edit_idea)}</Button>
              {friendlyError && <p role="alert" className="mt-4 text-sm font-bold text-[#a33b32]">{friendlyError}</p>}
            </div>
          </motion.section>
        ) : isWorking ? (
          <section className="mx-auto flex min-h-[560px] max-w-3xl flex-col items-center justify-center text-center">
            <motion.div
              animate={{ rotate: [0, -6, 5, 0], y: [0, -8, 0] }}
              transition={{ duration: 1.8, repeat: Infinity, ease: "easeInOut" }}
              className="mb-7 flex size-24 items-center justify-center rounded-[2rem] border-2 border-[#1d241f] bg-[#ffd85a] shadow-[7px_8px_0_#1d241f]"
            >
              <LoaderCircle className="size-11 animate-spin" />
            </motion.div>
            <p className="text-xs font-black uppercase tracking-[.2em] text-[#39715a]">{t($ => $.checking_label)}</p>
            <h2 className="mt-3 text-3xl font-black md:text-5xl">{statusCopy}</h2>
            <p className="mt-4 max-w-lg font-medium text-[#687068]">{session.summary || t($ => $.checking_hint)}</p>
            <Button variant="ghost" className="mt-5" disabled={cancelSession.isPending} onClick={() => void editIdea()}>{t($ => $.edit_idea)}</Button>
            {friendlyError && <p role="alert">{friendlyError}</p>}
          </section>
        ) : session?.status === "failed" ? (
          <section className="mx-auto max-w-xl pt-20 text-center">
            <h2 className="text-3xl font-black">{t($ => $.failed_title)}</h2>
			<p className="mt-3 text-[#687068]">{session.message || failedDescription}</p>
            <Button onClick={() => void editIdea()} className="mt-6 rounded-xl bg-[#1d241f]">{t($ => $.edit_idea)}</Button>
          </section>
        ) : (
          <section className="grid items-center gap-8 pt-5 lg:grid-cols-[minmax(0,1fr)_430px] lg:pt-14">
            <div>
              <motion.div initial={{ opacity: 0, y: 12 }} animate={{ opacity: 1, y: 0 }}>
                <span className="inline-flex items-center gap-2 rounded-full border-2 border-[#1d241f] bg-[#a8dfc2] px-4 py-2 text-xs font-black uppercase tracking-[.12em] shadow-[3px_4px_0_#1d241f]">
                  <Sparkles className="size-4" /> {t($ => $.hero_badge)}
                </span>
                <h2 className="mt-7 max-w-3xl text-5xl font-black leading-[.98] tracking-[-.045em] sm:text-6xl xl:text-7xl">
                  {t($ => $.hero_line_start)}<br /><span className="text-[#3767bb]">{t($ => $.hero_accent)}</span> {t($ => $.hero_line_end)}
                </h2>
                <p className="mt-6 max-w-xl text-lg font-medium leading-8 text-[#59615b]">{t($ => $.hero_description)}</p>
              </motion.div>
              <div className="mt-8 flex flex-wrap gap-2">
                {ideaStarters.map((idea) => (
                  <button key={idea} type="button" onClick={() => { setPrompt(idea); setClientRequestId(generateUUID()); }} className="rounded-full border border-[#9c9079] bg-[#fffaf0] px-4 py-2 text-sm font-bold transition hover:-translate-y-0.5 hover:border-[#1d241f] hover:bg-[#ffd85a]">{idea}</button>
                ))}
              </div>
            </div>

            <motion.div initial={{ opacity: 0, rotate: 2, y: 20 }} animate={{ opacity: 1, rotate: -1, y: 0 }} className="rounded-[2rem] border-2 border-[#1d241f] bg-[#fffdf7] p-5 shadow-[10px_12px_0_#1d241f] md:p-6">
              <div className="mb-4 flex items-center gap-2 text-sm font-black text-[#39715a]"><Lightbulb className="size-4" /> {t($ => $.idea_label)}</div>
              <Textarea
                value={prompt}
                onChange={(event) => {
                  setPrompt(event.target.value.slice(0, 280));
                  setClientRequestId(generateUUID());
                }}
                onKeyDown={(event) => {
                  if ((event.metaKey || event.ctrlKey) && event.key === "Enter") void start();
                }}
                placeholder={t($ => $.idea_placeholder)}
                className="min-h-44 resize-none rounded-2xl border-2 border-[#1d241f] bg-[#f7efde] p-4 text-base font-semibold leading-7 placeholder:text-[#8f887b] focus-visible:ring-[#3767bb]"
                autoFocus
              />
              <div className="mt-3 flex items-center justify-between text-xs font-semibold text-[#77766f]"><span>{t($ => $.shortcut)}</span><span>{prompt.length}/280</span></div>
              <Button
                onClick={() => void start()}
                disabled={!prompt.trim() || createSession.isPending}
                className="mt-5 h-14 w-full rounded-2xl border-2 border-[#1d241f] bg-[#ef5c4f] text-base font-black text-white shadow-[4px_5px_0_#1d241f] transition hover:-translate-y-0.5 hover:bg-[#df483c] hover:shadow-[6px_7px_0_#1d241f] active:translate-y-1 active:shadow-none"
              >
                {createSession.isPending ? <LoaderCircle className="size-5 animate-spin" /> : <Sparkles className="size-5" />}
                {t($ => $.start)}
              </Button>
              {friendlyError && <p role="alert" className="mt-3 text-sm font-bold text-[#a33b32]">{friendlyError}</p>}
            </motion.div>
          </section>
        )}
      </div>
    </main>
  );
}
