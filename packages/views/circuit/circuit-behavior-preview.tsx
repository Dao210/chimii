"use client";

import { useState } from "react";
import { Lightbulb, Fan, Power } from "lucide-react";
import { Button } from "@chimii/ui/components/ui/button";
import { circuitText, type CircuitDocument } from "@chimii/core/circuit";
import { useT } from "../i18n";

export function CircuitBehaviorPreview({
  document: doc,
}: {
  document: CircuitDocument;
}) {
  const { t, i18n } = useT("circuit");
  const [signals, setSignals] = useState<Record<string, boolean>>({});
  const [powered, setPowered] = useState(true);
  const spec = doc.composition,
    report = doc.behavior;
  if (!spec || !report?.passed) return null;
  const row = report.cases.find(
    (c) =>
      c.powered === powered &&
      (!powered || spec.inputs.every((id) => c.inputs[id] === !!signals[id])),
  );
  const active = row?.actual === true;
  const Output = spec.output === "BOS0021" ? Fan : Lightbulb;
  const label = (id: string) => {
    const part = doc.parts.find((p) => p.id === id);
    return part
      ? `${part.marking} · ${circuitText(part.name, i18n.language)}`
      : id;
  };
  return (
    <section
      className="circuit-behavior mb-5 rounded-2xl border bg-card p-5"
      aria-label={t(($) => $.behavior_title)}
    >
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 className="font-bold">{t(($) => $.behavior_title)}</h2>
          <p className="mt-1 text-xs text-muted-foreground">
            {t(($) => $.behavior_passed, { count: report.cases.length })}
          </p>
        </div>
        <span className="rounded-full bg-muted px-3 py-1.5 text-xs">
          {t(($) => $.logic_preview)}
        </span>
      </div>
      <div className="mt-5 grid gap-5 sm:grid-cols-[minmax(0,1fr)_auto]">
        <div className="flex flex-wrap items-center gap-2">
          <Button
            variant={powered ? "default" : "outline"}
            aria-pressed={powered}
            onClick={() => setPowered(!powered)}
          >
            <Power className="size-4" />
            {t(($) => $.preview_power)} ·{" "}
            {powered ? t(($) => $.output_on) : t(($) => $.output_off)}
          </Button>
          {spec.inputs.map((id) => (
            <Button
              key={id}
              variant={signals[id] ? "default" : "outline"}
              aria-pressed={!!signals[id]}
              onClick={() => setSignals((v) => ({ ...v, [id]: !v[id] }))}
            >
              {label(id)} ·{" "}
              {signals[id] ? t(($) => $.signal_high) : t(($) => $.signal_low)}
            </Button>
          ))}
        </div>
        <div
          role="status"
          className={`flex items-center gap-3 rounded-xl border px-4 py-3 ${active ? "bg-primary/10" : "bg-muted/40"}`}
        >
          <Output className="size-7" />
          <div>
            <p className="text-xs text-muted-foreground">
              {label(spec.output)}
            </p>
            <p className="font-bold">
              {active ? t(($) => $.output_on) : t(($) => $.output_off)}
            </p>
          </div>
        </div>
      </div>
      <p className="mt-4 text-xs leading-6 text-muted-foreground">
        {t(($) => $.behavior_boundary)}
      </p>
      <details className="mt-3">
        <summary className="cursor-pointer text-xs font-semibold">
          {t(($) => $.behavior_cases)}
        </summary>
        <div className="mt-3 overflow-x-auto">
          <table className="w-full text-left text-xs">
            <thead>
              <tr className="border-b">
                <th className="p-2">{t(($) => $.preview_power)}</th>
                {spec.inputs.map((id) => (
                  <th key={id} className="p-2">
                    {label(id)}
                  </th>
                ))}
                <th className="p-2">{t(($) => $.expected_output)}</th>
                <th className="p-2">{t(($) => $.checked_output)}</th>
              </tr>
            </thead>
            <tbody>
              {report.cases.map((c, i) => (
                <tr key={i} className="border-b last:border-0">
                  <td className="p-2">
                    {c.powered ? t(($) => $.output_on) : t(($) => $.output_off)}
                  </td>
                  {spec.inputs.map((id) => (
                    <td key={id} className="p-2">
                      {c.inputs[id]
                        ? t(($) => $.signal_high)
                        : t(($) => $.signal_low)}
                    </td>
                  ))}
                  <td className="p-2">
                    {c.expected
                      ? t(($) => $.output_on)
                      : t(($) => $.output_off)}
                  </td>
                  <td className="p-2">
                    {c.actual ? t(($) => $.output_on) : t(($) => $.output_off)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </details>
    </section>
  );
}
