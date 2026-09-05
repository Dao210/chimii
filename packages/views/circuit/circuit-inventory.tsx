"use client";

import { useState } from "react";
import { Check, PackageOpen, Save } from "lucide-react";
import { Button } from "@chimii/ui/components/ui/button";
import { Input } from "@chimii/ui/components/ui/input";
import {
  circuitText,
  useCircuitInventory,
  type CircuitCatalog,
  type CircuitInventory,
} from "@chimii/core/circuit";
import { useT } from "../i18n";

export function CircuitInventoryPanel({
  wsId,
  catalog,
  saved,
}: {
  wsId: string;
  catalog: CircuitCatalog;
  saved: CircuitInventory;
}) {
  const { t, i18n } = useT("circuit");
  const save = useCircuitInventory(wsId, catalog.kit_id);
  const [draft, setDraft] = useState<{
    quantities: Record<string, number>;
    revision: number;
  } | null>(null);
  const [checked, setChecked] = useState(false);
  const label = (v: { en: string; zh: string }) =>
    circuitText(v, i18n.language);
  const current = saved.confirmed && saved.catalog_version === catalog.version;
  const begin = (importKit: boolean) => {
    setDraft({
      quantities: Object.fromEntries(
        catalog.parts.map((p) => [
          p.id,
          importKit
            ? p.quantity
            : Math.min(p.quantity, saved.quantities[p.id] ?? 0),
        ]),
      ),
      revision: saved.revision,
    });
    setChecked(false);
    save.reset();
  };
  const submit = async () => {
    if (!draft || !checked) return;
    try {
      await save.mutateAsync({
        catalog_version: catalog.version,
        quantities: draft.quantities,
        expected_revision: draft.revision,
      });
      setDraft(null);
    } catch {
      /* Keep the draft available for comparison and retry. */
    }
  };
  return (
    <section className="rounded-2xl border bg-card p-5">
      <h2 className="flex items-center gap-2 font-bold">
        <PackageOpen className="size-4" />
        {t(($) => $.shelf)}
      </h2>
      <p className="mt-2 text-xs leading-6 text-muted-foreground">
        {t(($) => $.shelf_hint)}
      </p>
      <p className="my-3 text-xs font-medium" role="status">
        {current
          ? t(($) => $.box_saved, { revision: saved.revision })
          : t(($) => $.box_unconfirmed)}
      </p>
      {saved.can_edit ? (
        <>
          {!draft ? (
            <Button
              size="sm"
              variant="outline"
              className="w-full"
              onClick={() => begin(!current)}
            >
              {current ? t(($) => $.edit_materials) : t(($) => $.box_import)}
            </Button>
          ) : (
            <div className="space-y-4">
              <div className="max-h-96 space-y-3 overflow-y-auto pr-1">
                {catalog.parts.map((p) => (
                  <label
                    key={p.id}
                    className="flex items-center justify-between gap-2"
                  >
                    <span className="min-w-0 text-xs">
                      <span className="block break-all font-mono font-bold">
                        {p.marking ?? p.id}
                      </span>
                      {label(p.name)}
                    </span>
                    <Input
                      aria-label={`${p.id} ${label(p.name)}`}
                      type="number"
                      min={0}
                      max={p.quantity}
                      step={1}
                      value={draft.quantities[p.id] ?? 0}
                      disabled={save.isPending}
                      className="h-8 w-16 shrink-0 text-center"
                      onChange={(e) => {
                        const n = Number(e.target.value);
                        setDraft({
                          ...draft,
                          quantities: {
                            ...draft.quantities,
                            [p.id]: Math.max(
                              0,
                              Math.min(
                                p.quantity,
                                Number.isFinite(n) ? Math.trunc(n) : 0,
                              ),
                            ),
                          },
                        });
                        setChecked(false);
                        save.reset();
                      }}
                    />
                  </label>
                ))}
              </div>
              <label className="flex items-start gap-2 text-xs leading-6">
                <input
                  type="checkbox"
                  className="mt-1.5 size-4 shrink-0 accent-current"
                  checked={checked}
                  onChange={(e) => setChecked(e.target.checked)}
                  disabled={save.isPending}
                />
                {t(($) => $.box_check)}
              </label>
              {draft.revision !== saved.revision && (
                <p role="alert" className="text-xs leading-6 text-destructive">
                  {t(($) => $.inventory_conflict)}
                </p>
              )}
              {save.isError && (
                <p role="alert" className="text-xs leading-6 text-destructive">
                  {t(($) => $.box_save_error)}
                </p>
              )}
              <div className="flex flex-wrap gap-2">
                <Button
                  size="sm"
                  disabled={
                    !checked ||
                    save.isPending ||
                    draft.revision !== saved.revision
                  }
                  onClick={() => void submit()}
                >
                  <Save className="size-3" />
                  {t(($) => $.box_save)}
                </Button>
                <Button
                  size="sm"
                  variant="ghost"
                  disabled={save.isPending}
                  onClick={() => {
                    setDraft(null);
                    save.reset();
                  }}
                >
                  {t(($) => $.box_cancel)}
                </Button>
              </div>
            </div>
          )}
        </>
      ) : (
        <p className="text-xs leading-6 text-muted-foreground">
          {t(($) => $.box_parent)}
        </p>
      )}
      {!draft && current && (
        <details className="mt-4">
          <summary className="cursor-pointer text-xs">
            {t(($) => $.box_contents)}
          </summary>
          <ul className="mt-3 space-y-2">
            {catalog.parts.map((p) => (
              <li key={p.id} className="flex justify-between gap-3 text-xs">
                <span>
                  {p.marking ?? p.id} · {label(p.name)}
                </span>
                <span className="font-mono">
                  ×{saved.quantities[p.id] ?? 0}
                </span>
              </li>
            ))}
          </ul>
        </details>
      )}
      {save.isSuccess && !draft && (
        <p role="status" className="mt-3 flex items-center gap-1 text-xs">
          <Check className="size-3" />
          {t(($) => $.box_save_done)}
        </p>
      )}
      {catalog.hardware && (
        <details className="mt-5 border-t pt-4">
          <summary className="cursor-pointer text-xs font-medium">
            {t(($) => $.purchase)}
          </summary>
          <p className="my-3 text-xs leading-6 text-muted-foreground">
            {label(catalog.hardware.notes)}
          </p>
          <ul className="space-y-2">
            {catalog.hardware.purchase_links.map((link) => (
              <li key={link.url}>
                <a
                  href={link.url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-xs underline underline-offset-4"
                >
                  {label(link.label)}
                </a>
              </li>
            ))}
          </ul>
          <p className="mt-3 text-xs leading-6 text-muted-foreground">
            {t(($) => $.purchase_note)}
          </p>
        </details>
      )}
    </section>
  );
}
