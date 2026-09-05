"use client";

import { useRef, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { ClipboardCheck, Download } from "lucide-react";
import { Button } from "@chimii/ui/components/ui/button";
import { Input } from "@chimii/ui/components/ui/input";
import { Textarea } from "@chimii/ui/components/ui/textarea";
import {
  circuitTrialsOptions,
  useCircuitTrial,
  type CircuitCreation,
} from "@chimii/core/circuit";
import { generateUUID } from "@chimii/core/utils";
import { useT } from "../i18n";

export function CircuitTrialPanel({
  wsId,
  creation,
}: {
  wsId: string;
  creation: CircuitCreation;
}) {
  const { t } = useT("circuit");
  const query = useQuery(circuitTrialsOptions(wsId, creation.id));
  const save = useCircuitTrial(wsId, creation.id);
  const [hardware, setHardware] = useState("");
  const [notes, setNotes] = useState("");
  const [result, setResult] = useState<"worked" | "needs_help">("worked");
  const [checked, setChecked] = useState(false);
  const request = useRef({ signature: "", id: "" });
  const ready =
    creation.current_step === creation.document.project.steps.length - 1;
  const submit = async () => {
    const input = {
      hardware_label: hardware.trim(),
      notes: notes.trim(),
      result,
      adult_checked: checked,
    };
    const signature = JSON.stringify(input);
    if (request.current.signature !== signature)
      request.current = { signature, id: generateUUID() };
    try {
      await save.mutateAsync({
        ...input,
        client_request_id: request.current.id,
      });
      setNotes("");
      setChecked(false);
      request.current = { signature: "", id: "" };
    } catch {
      /* Preserve the attempted report and retry ID. */
    }
  };
  const download = () => {
    if (!query.data) return;
    const data = {
      creation_id: creation.id,
      document: creation.document,
      evidence_kind: "family_report",
      trials: query.data.trials,
    };
    const url = URL.createObjectURL(
      new Blob([JSON.stringify(data, null, 2)], { type: "application/json" }),
    );
    const a = document.createElement("a");
    a.href = url;
    a.download = `circuit-${creation.id}-trials.json`;
    a.click();
    setTimeout(() => URL.revokeObjectURL(url), 1000);
  };
  return (
    <details className="rounded-2xl border bg-card p-5">
      <summary className="cursor-pointer text-sm font-bold">
        <ClipboardCheck className="mr-2 inline size-4" />
        {t(($) => $.trial_title)}
      </summary>
      <p className="my-3 text-xs leading-6 text-muted-foreground">
        {t(($) => $.trial_note)}
      </p>
      {ready ? (
        <div className="space-y-3">
          <label className="block text-xs">
            {t(($) => $.trial_hardware)}
            <Input
              value={hardware}
              maxLength={120}
              onChange={(e) => {
                setHardware(e.target.value);
                save.reset();
              }}
              disabled={save.isPending}
              className="mt-2"
            />
          </label>
          <label className="block text-xs">
            {t(($) => $.trial_result)}
            <select
              value={result}
              onChange={(e) => {
                setResult(
                  e.target.value === "worked" ? "worked" : "needs_help",
                );
                save.reset();
              }}
              disabled={save.isPending}
              className="mt-2 block w-full rounded-lg border bg-background p-2"
            >
              <option value="worked">{t(($) => $.worked)}</option>
              <option value="needs_help">{t(($) => $.needs_help)}</option>
            </select>
          </label>
          <label className="block text-xs">
            {t(($) => $.trial_observed)}
            <Textarea
              value={notes}
              maxLength={1000}
              onChange={(e) => {
                setNotes(e.target.value);
                save.reset();
              }}
              disabled={save.isPending}
              className="mt-2"
            />
          </label>
          <label className="flex gap-2 text-xs leading-6">
            <input
              type="checkbox"
              className="mt-1.5 size-4 shrink-0 accent-current"
              checked={checked}
              onChange={(e) => setChecked(e.target.checked)}
              disabled={save.isPending}
            />
            {t(($) => $.trial_check)}
          </label>
          <Button
            size="sm"
            onClick={() => void submit()}
            disabled={!hardware.trim() || !checked || save.isPending}
          >
            {t(($) => $.trial_save)}
          </Button>
          {save.isError && (
            <p role="alert" className="text-xs text-destructive">
              {t(($) => $.error)}
            </p>
          )}
          {save.isSuccess && (
            <p role="status" className="text-xs">
              {t(($) => $.trial_saved)}
            </p>
          )}
        </div>
      ) : (
        <p className="text-xs leading-6 text-muted-foreground">
          {t(($) => $.trial_finish)}
        </p>
      )}
      <div className="mt-4 border-t pt-4">
        {query.isPending ? (
          <p className="text-xs">{t(($) => $.loading)}</p>
        ) : query.isError ? (
          <Button
            size="sm"
            variant="outline"
            onClick={() => void query.refetch()}
          >
            {t(($) => $.retry)}
          </Button>
        ) : (
          <>
            <p className="text-xs text-muted-foreground">
              {t(($) => $.trial_count, { count: query.data.trials.length })}
            </p>
            <ul className="mt-3 space-y-3">
              {query.data.trials.map((v) => (
                <li key={v.id} className="rounded-lg bg-muted/40 p-3 text-xs">
                  <p className="font-bold">
                    {v.result === "worked"
                      ? t(($) => $.worked)
                      : v.result === "needs_help"
                        ? t(($) => $.needs_help)
                        : t(($) => $.observation_unknown)}
                  </p>
                  <p className="mt-1 break-words">{v.hardware_label}</p>
                  <p className="mt-1 text-muted-foreground">{v.created_at}</p>
                  <p className="mt-2 whitespace-pre-wrap break-words leading-6">
                    {v.notes}
                  </p>
                </li>
              ))}
            </ul>
            <Button
              className="mt-3"
              variant="outline"
              size="sm"
              onClick={download}
            >
              <Download className="size-3" />
              {t(($) => $.trial_download)}
            </Button>
          </>
        )}
      </div>
    </details>
  );
}
