"use client";

import { useReducer, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Plus, Redo2, Undo2, X } from "lucide-react";
import { Button } from "@chimii/ui/components/ui/button";
import { Input } from "@chimii/ui/components/ui/input";
import { Textarea } from "@chimii/ui/components/ui/textarea";
import { buildSessionOptions, reduceBuildDesignHistory, useCreateBuildSession, useSubmitBuildAnswers, useCancelBuildSession, type BuildCreation, type BuildDesignHistory, type BuildShapeNode } from "@chimii/core/build";
import { useWorkspaceId } from "@chimii/core/hooks";
import { generateUUID } from "@chimii/core/utils";
import { useWorkspacePaths } from "@chimii/core/paths";
import { AppLink } from "../../navigation";
import { useT } from "../../i18n";

export function BuildDesignEditor({ creation, onClose, onDiscuss, discussDisabled = false }: { creation: BuildCreation; onClose: () => void; onDiscuss?: (prompt: string) => Promise<boolean>; discussDisabled?: boolean }) {
  const { t } = useT("build");
  const wsId = useWorkspaceId();
  const workspacePaths = useWorkspacePaths();
  const design = creation.build_plan.document!.design;
  const [history, dispatch] = useReducer(reduceBuildDesignHistory, { present: design, past: [], future: [] } satisfies BuildDesignHistory);
  const [selectedId, setSelectedId] = useState(design.shapes[0]?.id ?? "");
  const [prompt, setPrompt] = useState("");
  const [error, setError] = useState("");
  const [sessionId, setSessionId] = useState("");
  const [requestId, setRequestId] = useState(generateUUID);
  const [answer, setAnswer] = useState("");
  const create = useCreateBuildSession();
  const submitAnswer = useSubmitBuildAnswers();
  const cancel = useCancelBuildSession();
  const { data: session, isError: sessionError, refetch } = useQuery(buildSessionOptions(wsId, sessionId));
  const selected = history.present.shapes.find(shape => shape.id === selectedId) ?? history.present.shapes[0];
  const working = create.isPending || submitAnswer.isPending || cancel.isPending || (!!sessionId && !sessionError && (!session || session.status === "queued" || session.status === "generating"));
  const locked = working || session?.status === "clarifying";
  const dirty = JSON.stringify(design) !== JSON.stringify(history.present);
  const colors = [1, 2, 4, 14, 15, 71];
  const colorLabels = [t($ => $.editor_blue), t($ => $.editor_green), t($ => $.editor_red), t($ => $.editor_yellow), t($ => $.editor_white), t($ => $.editor_gray)];
  const changed = () => { setRequestId(generateUUID()); setError(""); setSessionId(""); };
  const replace = (shape: BuildShapeNode) => {
    try { dispatch({ type: "replace_shape", shape }); changed(); }
    catch { setError(t($ => $.editor_invalid)); }
  };
  const submit = async (naturalLanguage: boolean) => {
    if (locked || (naturalLanguage ? !prompt.trim() : !dirty)) return;
    setError("");
    if (naturalLanguage && onDiscuss) {
      if (!discussDisabled && await onDiscuss(prompt.trim())) setPrompt("");
      return;
    }
    try {
      const nextRequestId = session?.status === "failed" || session?.status === "completed" ? generateUUID() : requestId;
      setRequestId(nextRequestId);
      const next = await create.mutateAsync({
        prompt: naturalLanguage ? prompt.trim() : t($ => $.editor_revision_prompt, { title: creation.title }),
        clientRequestId: nextRequestId,
        designInput: { source_creation_id: creation.id, expected_content_hash: creation.build_plan.content_hash, ...(naturalLanguage ? {} : { design: history.present }) },
      });
      if (!next.id) throw new Error("Invalid session");
      setSessionId(next.id);
    } catch { setError(t($ => $.editor_submit_error)); }
  };

  const close = async () => {
    try {
      if (session && session.status !== "completed" && session.status !== "failed") {
        await cancel.mutateAsync({ sessionId: session.id, revision: session.revision ?? 1 });
      }
      onClose();
    } catch { setError(t($ => $.cancel_error)); }
  };
  const respond = async (value: string) => {
    if (!session?.question || !value.trim()) return;
    try {
      await submitAnswer.mutateAsync({ sessionId: session.id, revision: session.revision, answers: { [session.question.id]: value.trim() } });
      setAnswer("");setError("");
    } catch { setError(t($ => $.answer_error)); }
  };

  return <section className="col-span-full rounded-2xl border-2 border-border bg-background p-5 text-foreground shadow-sm" aria-label={t($ => $.editor_title)}>
    <div className="mb-4 flex items-start justify-between gap-4">
      <div><h3 className="text-lg font-bold">{t($ => $.editor_title)}</h3><p className="mt-1 max-w-2xl text-sm text-muted-foreground">{t($ => $.editor_hint)}</p></div>
      <Button variant="ghost" size="icon" disabled={create.isPending || cancel.isPending} onClick={() => void close()} aria-label={t($ => $.editor_close)}><X className="size-4" /></Button>
    </div>
    <div className="grid gap-5 md:grid-cols-[minmax(0,1.7fr)_minmax(0,1fr)]">
      <fieldset disabled={locked} className="min-w-0 space-y-4">
        <div className="flex flex-wrap items-center gap-2">
          <label className="flex min-w-0 flex-1 flex-col gap-1 text-xs font-semibold">{t($ => $.editor_shape)}
            <select aria-label={t($ => $.editor_shape)} className="h-9 min-w-0 rounded-md border border-input bg-background px-2 text-sm" value={selected?.id} onChange={e => setSelectedId(e.target.value)}>
              {history.present.shapes.map(shape => <option key={shape.id} value={shape.id}>{shape.label || shape.id}</option>)}
            </select>
          </label>
          <Button variant="outline" size="icon" disabled={working || !history.past.length} aria-label={t($ => $.editor_undo)} onClick={() => { dispatch({ type: "undo" }); changed(); }}><Undo2 className="size-4" /></Button>
          <Button variant="outline" size="icon" disabled={working || !history.future.length} aria-label={t($ => $.editor_redo)} onClick={() => { dispatch({ type: "redo" }); changed(); }}><Redo2 className="size-4" /></Button>
        </div>
        {selected && <>
          <div className="grid grid-cols-3 gap-3">
            {(["x", "y", "z"] as const).map((axis, i) => <label key={axis} className="space-y-1 text-xs font-semibold">
              <span>{[t($ => $.editor_width), t($ => $.editor_height), t($ => $.editor_depth)][i]}</span>
              <Input type="number" min={1} max={axis === "y" ? 48 : 32} value={selected.size[axis]} onChange={e => {
                const value = e.target.valueAsNumber;
                if (Number.isInteger(value) && value >= 1 && value <= (axis === "y" ? 48 : 32)) replace({ ...selected, size: { ...selected.size, [axis]: value } });
              }} />
            </label>)}
          </div>
          <div className="grid grid-cols-3 gap-3">
            {(["x", "y", "z"] as const).map((axis, i) => <label key={axis} className="space-y-1 text-xs font-semibold">
              <span>{[t($ => $.editor_left_right), t($ => $.editor_up_down), t($ => $.editor_front_back)][i]}</span>
              <Input type="number" min={axis === "y" ? 0 : -16} max={axis === "y" ? 47 : 16} value={selected.position[axis]} onChange={e => {
                const value = e.target.valueAsNumber;
                if (Number.isInteger(value) && value >= (axis === "y" ? 0 : -16) && value <= (axis === "y" ? 47 : 16)) replace({ ...selected, position: { ...selected.position, [axis]: value } });
              }} />
            </label>)}
          </div>
          <div className="grid grid-cols-2 gap-3">
            <label className="flex flex-col gap-1 text-xs font-semibold">{t($ => $.editor_color)}
              <select aria-label={t($ => $.editor_color)} className="h-9 rounded-md border border-input bg-background px-2 text-sm" value={selected.color} onChange={e => replace({ ...selected, color: Number(e.target.value) })}>
                {colors.map((color, i) => <option key={color} value={color}>{colorLabels[i]}</option>)}
              </select>
            </label>
            <label className="flex flex-col gap-1 text-xs font-semibold">{t($ => $.editor_kind)}
              <select className="h-9 rounded-md border border-input bg-background px-2 text-sm" value={selected.kind} disabled={selected.kind === "polygon"} onChange={e => replace({ ...selected, kind: e.target.value === "ellipse" ? "ellipse" : "box" })}>
                <option value="box">{t($ => $.editor_box)}</option><option value="ellipse">{t($ => $.editor_ellipse)}</option>
                {selected.kind === "polygon" && <option value="polygon">{t($ => $.editor_polygon)}</option>}
              </select>
            </label>
          </div>
        </>}
        <div className="flex flex-wrap gap-2">
          <Button variant="outline" size="sm" disabled={working || history.present.shapes.length >= 48} onClick={() => {
            const id = generateUUID();
            dispatch({ type: "add_shape", shape: { id, label: t($ => $.editor_new_shape), kind: "box", operation: "add", position: { x: 0, y: 0, z: 0 }, size: { x: 2, y: 3, z: 2 }, color: 1 } });
            setSelectedId(id); changed();
          }}><Plus className="size-4" />{t($ => $.editor_add)}</Button>
          <Button variant="outline" size="sm" disabled={working || history.present.shapes.length < 2} onClick={() => { if (selected) { dispatch({ type: "remove_shape", id: selected.id }); changed(); } }}>{t($ => $.editor_remove)}</Button>
        </div>
        <Button disabled={working || !dirty} onClick={() => void submit(false)}>{t($ => $.editor_compile)}</Button>
      </fieldset>
      <fieldset disabled={locked} className="min-w-0 space-y-3 rounded-xl bg-muted/40 p-4">
        <label className="block text-sm font-bold" htmlFor={`design-prompt-${creation.id}`}>{t($ => $.editor_describe)}</label>
        <Textarea id={`design-prompt-${creation.id}`} value={prompt} maxLength={280} rows={4} placeholder={t($ => $.editor_prompt_placeholder)} onChange={e => { setPrompt(e.target.value); changed(); }} />
        <p className="text-xs text-muted-foreground">{t($ => $.editor_describe_hint)}</p>
        <Button variant="outline" disabled={working || discussDisabled || !prompt.trim() || dirty} onClick={() => void submit(true)}>{t($ => $.editor_generate)}</Button>
      </fieldset>
    </div>
    {(error || sessionError || session?.status === "failed") && <div role="alert" className="mt-4 space-y-2 text-sm text-destructive">
      <p>{error || (sessionError ? t($ => $.session_fetch_error) : session?.error === "BUILD_SEARCH_LIMIT" ? t($ => $.failed_search_limit) : session?.error === "BUILD_INSUFFICIENT_INVENTORY" ? t($ => $.failed_inventory) : session?.message || t($ => $.editor_compile_error))}</p>
      {sessionError && <Button size="sm" variant="outline" onClick={() => void refetch()}>{t($ => $.retry_fetch)}</Button>}
    </div>}
    {working && <p role="status" className="mt-4 text-sm text-muted-foreground">{t($ => $.editor_working)}</p>}
    {session?.status === "clarifying" && session.question && <div className="mt-4 space-y-3 rounded-xl border border-border p-4">
      <p className="font-semibold">{session.question.prompt}</p>
      <div className="flex flex-wrap gap-2">{session.question.choices?.map(choice => <Button key={choice.id} disabled={working} variant="outline" onClick={() => void respond(choice.id)}>{choice.label}</Button>)}</div>
      <Textarea aria-label={t($ => $.editor_describe)} maxLength={120} value={answer} onChange={e => setAnswer(e.target.value)} />
      <Button disabled={working || !answer.trim()} onClick={() => void respond(answer)}>{t($ => $.editor_generate)}</Button>
    </div>}
    {session?.creation_id && session.status === "completed" && <div role="status" className="mt-4 flex flex-wrap items-center gap-3 rounded-xl border border-border p-3">
      <p className="text-sm font-semibold">{t($ => $.editor_ready)}</p>
      <AppLink className="text-sm font-bold underline underline-offset-4" href={workspacePaths.creationDetail(session.creation_id)}>{t($ => $.editor_open)}</AppLink>
    </div>}
  </section>;
}
