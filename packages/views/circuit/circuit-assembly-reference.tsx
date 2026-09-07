"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { ArrowLeft, ArrowRight, ExternalLink, LoaderCircle } from "lucide-react";
import { Button } from "@chimii/ui/components/ui/button";
import { circuitKitsOptions, circuitText, type AssemblyReference } from "@chimii/core/circuit";
import { useWorkspacePaths } from "@chimii/core/paths";
import { AppLink } from "../navigation";
import { useT } from "../i18n";

export function CircuitAssemblyReferenceEntry({ references }: { references: AssemblyReference[] }) {
  const paths = useWorkspacePaths();
  const { t, i18n } = useT("circuit");
  if (!references.length) return null;
  return (
    <section className="mb-7 rounded-2xl border border-primary/20 bg-primary/5 p-5" aria-label={t(($) => $.reference_title)}>
      <h2 className="font-bold">{t(($) => $.reference_title)}</h2>
      <p className="mt-2 max-w-3xl text-sm leading-6 text-muted-foreground">{t(($) => $.reference_hint)}</p>
      <div className="mt-4 flex flex-wrap gap-3">
        {references.map((ref) => (
          <AppLink className="inline-flex items-center gap-3 rounded-xl border bg-card px-4 py-3 text-sm font-semibold hover:bg-accent focus-visible:outline-2 focus-visible:outline-ring" key={ref.id} href={`${paths.circuit()}?${new URLSearchParams({ reference: ref.id })}`}>
            {circuitText(ref.title, i18n.language)}
            <span className="text-xs font-normal text-muted-foreground">{t(($) => $.reference_entry)}</span>
            <ArrowRight className="size-4" />
          </AppLink>
        ))}
      </div>
    </section>
  );
}

export function CircuitAssemblyReferencePage({ wsId, referenceId }: { wsId: string; referenceId: string }) {
  const paths = useWorkspacePaths();
  const { t } = useT("circuit");
  const query = useQuery(circuitKitsOptions(wsId));
  const reference = query.data?.assembly_references?.find((r) => r.id === referenceId);
  return (
    <main className="circuit-studio h-full overflow-y-auto">
      <div className="mx-auto max-w-6xl px-5 py-7 md:px-9 md:py-9">
        <AppLink className="mb-7 inline-flex items-center gap-2 text-sm font-semibold underline underline-offset-4" href={paths.circuit()}>
          <ArrowLeft className="size-4" />{t(($) => $.reference_back)}
        </AppLink>
        {query.isPending ? (
          <p role="status" className="flex items-center gap-2"><LoaderCircle className="size-4 animate-spin" />{t(($) => $.loading)}</p>
        ) : reference ? (
          <CircuitAssemblyReferenceWorkbench key={`${reference.id}:${reference.content_hash}`} reference={reference} />
        ) : (
          <div role="alert" className="rounded-2xl border bg-card p-6">
            <p>{t(($) => $.reference_not_found)}</p>
            <Button className="mt-4" variant="outline" onClick={() => void query.refetch()}>{t(($) => $.retry)}</Button>
          </div>
        )}
      </div>
    </main>
  );
}

// This illustration explains the source product's composition. It deliberately
// has no physical coordinates, animated motion or inferred mounting geometry.
function GateOverview({ reference }: { reference: AssemblyReference }) {
  const { t } = useT("circuit");
  const sensorPort = reference.connections.find((c) => c.part_id === "ultrasonic")?.port;
  const servoPort = reference.connections.find((c) => c.part_id === "servo")?.port;
  return (
    <figure className="rounded-2xl border bg-card p-4 md:p-6">
      <svg viewBox="0 0 640 360" role="img" aria-label={t(($) => $.reference_diagram)} className="w-full text-foreground">
        <path d="M65 307H575" stroke="currentColor" opacity=".15" strokeWidth="2" />
        <g fill="var(--muted)" stroke="currentColor" strokeWidth="2">
          <rect x="250" y="211" width="72" height="90" rx="10" />
          <rect x="253" y="137" width="290" height="16" rx="8" />
          <rect x="253" y="228" width="290" height="16" rx="8" />
          {[260, 350, 440, 530].map((x) => <rect key={x} x={x} y="153" width="10" height="75" rx="4" />)}
          <path d="M235 138V92H183V185" fill="none" />
        </g>
        <rect x="126" y="183" width="122" height="117" rx="12" fill="var(--background)" stroke="currentColor" strokeWidth="2" />
        <rect x="139" y="158" width="98" height="52" rx="6" fill="var(--muted)" stroke="currentColor" strokeWidth="2" />
        <g fill="var(--primary)">
          {[0, 1, 2, 3, 4].flatMap((row) => [0, 1, 2, 3, 4].map((col) => <circle key={`${row}-${col}`} cx={176 + col * 6} cy={171 + row * 6} r="1.5" />))}
        </g>
        <rect x="142" y="223" width="42" height="23" rx="4" fill="var(--primary)" opacity=".15" />
        <rect x="191" y="257" width="42" height="23" rx="4" fill="var(--primary)" opacity=".15" />
        <text x="163" y="239" textAnchor="middle" fill="currentColor" fontSize="12">{sensorPort}</text>
        <text x="212" y="273" textAnchor="middle" fill="currentColor" fontSize="12">{servoPort}</text>
        <g stroke="var(--primary)" strokeWidth="2" fill="var(--background)">
          <rect x="146" y="68" width="120" height="54" rx="19" />
          <circle cx="177" cy="95" r="18" /><circle cx="234" cy="95" r="18" />
          <circle cx="274" cy="238" r="10" />
        </g>
        <g stroke="currentColor" strokeWidth="1" opacity=".45" fill="none">
          <path d="M279 96H345" /><path d="M187 305V328H83" />
          <path d="M330 276H379" /><path d="M456 124V85H520" />
        </g>
        <g fill="currentColor" fontSize="14">
          <text x="351" y="101">{t(($) => $.reference_sensor)}</text>
          <text x="70" y="349">{t(($) => $.reference_controller)}</text>
          <text x="387" y="281">{t(($) => $.reference_drive)}</text>
          <text x="527" y="90">{t(($) => $.reference_gate)}</text>
        </g>
      </svg>
      <figcaption className="mt-3 text-center text-xs leading-5 text-muted-foreground">{t(($) => $.reference_diagram)}</figcaption>
    </figure>
  );
}

function SourceLink({ href, children }: { href: string; children: React.ReactNode }) {
  return <a className="inline-flex items-center gap-2 text-sm font-semibold underline underline-offset-4 focus-visible:outline-2 focus-visible:outline-ring" href={href} target="_blank" rel="noopener noreferrer">{children}<ExternalLink className="size-3.5 shrink-0" /></a>;
}

export function CircuitAssemblyReferenceWorkbench({ reference: ref }: { reference: AssemblyReference }) {
  const { t, i18n } = useT("circuit");
  const [current, setCurrent] = useState(0);
  const label = (value: { en: string; zh: string }) => circuitText(value, i18n.language);
  const step = ref.steps[current] ?? ref.steps[0];
  const total = ref.parts.reduce((n, p) => n + p.pictured_quantity, 0);
  const parts = new Map(ref.parts.map((p) => [p.id, p]));
  return (
    <article className="space-y-8">
      <header>
        <p className="mb-3 text-xs font-bold tracking-widest text-muted-foreground">{t(($) => $.reference_eyebrow)}</p>
        <div className="flex flex-wrap items-center gap-3">
          <h1 className="text-3xl font-black tracking-tight md:text-4xl">{label(ref.title)}</h1>
          <span className="rounded-full border px-3 py-1 text-xs text-muted-foreground">{t(($) => $.reference_status)}</span>
        </div>
        <p className="mt-3 max-w-3xl text-sm leading-7 text-muted-foreground">{label(ref.description)}</p>
        <p className="mt-3 text-sm font-semibold">{ref.kit_name}</p>
      </header>
      <section aria-label={t(($) => $.reference_overview)} className="grid items-start gap-5 lg:grid-cols-[1.4fr_1fr]">
        {ref.id === "nezha-v2-ultrasonic-gate" && <GateOverview reference={ref} />}
        <div className="space-y-5 rounded-2xl border bg-card p-5 md:p-6">
          <p className="text-sm leading-7">{t(($) => $.reference_blocked)}</p>
          <p className="text-sm font-semibold">{t(($) => $.reference_counts, { types: ref.parts.length, count: total, steps: ref.steps.length })}</p>
          <div className="flex flex-col items-start gap-4">
            <SourceLink href={ref.source_url}>{t(($) => $.reference_source)}</SourceLink>
            <SourceLink href={ref.product_url}>{t(($) => $.reference_product)}</SourceLink>
          </div>
        </div>
      </section>
      <section aria-label={t(($) => $.reference_steps)} className="rounded-2xl border bg-card p-5 md:p-7">
        <h2 className="text-xl font-bold">{t(($) => $.reference_steps)}</h2>
        <p className="mt-2 max-w-3xl text-sm leading-6 text-muted-foreground">{t(($) => $.reference_step_hint)}</p>
        <div className="my-5 flex flex-wrap gap-2">
          {ref.steps.map((s, index) => (
            <button key={s.number} type="button" aria-label={`${s.number}. ${label(s.title)}`} aria-pressed={index === current} onClick={() => setCurrent(index)} className={`flex size-10 items-center justify-center rounded-xl border font-mono text-sm focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring ${index === current ? "border-primary bg-primary text-primary-foreground" : "bg-background hover:bg-accent"}`}>
              {s.number}
            </button>
          ))}
        </div>
        {step && (
          <div aria-live="polite" aria-atomic="true" className="rounded-xl bg-muted/50 p-5">
            <p className="mb-2 text-xs font-semibold text-muted-foreground">{t(($) => $.reference_step, { step: step.number, total: ref.steps.length })}</p>
            <h3 className="text-lg font-bold">{label(step.title)}</h3>
            <ul className="my-4 flex flex-wrap gap-2">
              {Object.entries(step.introduced_parts).map(([id, n]) => <li key={id} className="rounded-lg border bg-background px-3 py-2 text-sm">{parts.get(id) ? label(parts.get(id)!.name) : id} <span className="ml-2 font-mono font-bold">×{n}</span></li>)}
            </ul>
            <SourceLink href={step.source_url}>{t(($) => $.reference_step_source)}</SourceLink>
          </div>
        )}
        <div className="mt-5 flex flex-wrap justify-between gap-3">
          <Button variant="outline" disabled={current === 0} onClick={() => setCurrent((n) => Math.max(0, n - 1))}><ArrowLeft className="size-4" />{t(($) => $.reference_prev)}</Button>
          <Button variant="outline" disabled={current >= ref.steps.length - 1} onClick={() => setCurrent((n) => Math.min(ref.steps.length - 1, n + 1))}>{t(($) => $.reference_next)}<ArrowRight className="size-4" /></Button>
        </div>
      </section>
      <section aria-label={t(($) => $.reference_materials)} className="rounded-2xl border bg-card p-5 md:p-7">
        <h2 className="text-xl font-bold">{t(($) => $.reference_materials)}</h2>
        <p className="mb-5 mt-2 max-w-3xl text-sm leading-6 text-muted-foreground">{t(($) => $.reference_counts_hint)}</p>
        <SourceLink href={ref.parts_source_url}>{t(($) => $.reference_parts_source)}</SourceLink>
        <table className="mt-5 w-full text-left text-sm">
          <thead><tr className="border-b"><th scope="col" className="pb-3 pr-4">{t(($) => $.reference_part_name)}</th><th scope="col" className="pb-3 text-right">{t(($) => $.reference_quantity)}</th></tr></thead>
          <tbody>{ref.parts.map((p) => <tr key={p.id} className="border-b last:border-b-0"><th scope="row" className="py-3 pr-4 font-normal">{label(p.name)}</th><td className="py-3 text-right font-mono">{p.pictured_quantity}</td></tr>)}</tbody>
        </table>
      </section>
      <section className="grid gap-5 md:grid-cols-2" aria-label={t(($) => $.reference_wiring)}>
        <div className="rounded-2xl border bg-card p-5">
          <h2 className="font-bold">{t(($) => $.reference_wiring)}</h2>
          <ul className="my-4 space-y-3">{ref.connections.map((c) => <li key={`${c.controller_id}:${c.port}`} className="flex flex-wrap items-center gap-2 text-sm"><span>{parts.get(c.part_id) ? label(parts.get(c.part_id)!.name) : c.part_id}</span><ArrowRight className="size-4 shrink-0" /><span>{parts.get(c.controller_id) ? label(parts.get(c.controller_id)!.name) : c.controller_id}</span><code className="rounded-md border bg-background px-2 py-1">{c.port}</code></li>)}</ul>
          <p className="text-xs leading-6 text-muted-foreground">{t(($) => $.reference_wiring_hint)}</p>
        </div>
        <div className="rounded-2xl border bg-card p-5">
          <SourceLink href={ref.program_url}>{t(($) => $.reference_program)}</SourceLink>
          <p className="mt-4 text-sm leading-7 text-muted-foreground">{t(($) => $.reference_program_hint)}</p>
        </div>
      </section>
      <details className="rounded-2xl border p-5">
        <summary className="cursor-pointer font-bold">{t(($) => $.reference_gaps)}</summary>
        <ul className="mt-4 list-disc space-y-2 pl-5 text-sm leading-7 text-muted-foreground">{ref.unresolved.map((gap) => <li key={gap.id}>{label(gap.detail)}</li>)}</ul>
      </details>
      <p className="text-xs leading-6 text-muted-foreground">{t(($) => $.reference_checked, { date: ref.checked_on, version: ref.version })}</p>
    </article>
  );
}
