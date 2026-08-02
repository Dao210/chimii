"use client";

import { useLocale } from "../i18n";

/**
 * ComparisonSection — "同样一件事·两种活法" two-column layout inspired by
 * meridianos.ai. Past column uses muted red accents (slow/risky), Present
 * column uses emerald accents (fast/safe).
 */
export function ComparisonSection() {
  const { t } = useLocale();
  const cmp = t.comparison;

  return (
    <section
      id="comparison"
      className="bg-white py-24 text-[#0a0d12] sm:py-32 lg:py-40"
    >
      <div className="mx-auto max-w-[1320px] px-4 sm:px-6 lg:px-8">
        <div className="max-w-[860px]">
          <p className="text-[11px] font-semibold uppercase tracking-[0.22em] text-[#0a0d12]/40">
            {cmp.label}
          </p>
          <h2 className="mt-4 font-semibold text-[2.6rem] leading-[1.05] tracking-[-0.03em] sm:text-[3.4rem] lg:text-[4.2rem]">
            {cmp.headlineLine1}
            <br />
            <span className="text-[#0a0d12]/48">{cmp.headlineLine2}</span>
          </h2>
        </div>

        <div className="mt-16 grid gap-6 lg:grid-cols-2 lg:gap-8">
          {/* Past column */}
          <ComparisonColumn
            label={cmp.pastLabel}
            tone="past"
            items={cmp.past}
          />
          {/* Present column */}
          <ComparisonColumn
            label={cmp.presentLabel}
            tone="present"
            items={cmp.present}
          />
        </div>
      </div>
    </section>
  );
}

function ComparisonColumn({
  label,
  tone,
  items,
}: {
  label: string;
  tone: "past" | "present";
  items: { question: string; answer: string }[];
}) {
  const isPast = tone === "past";
  return (
    <div
      className={
        isPast
          ? "rounded-2xl border border-[#0a0d12]/8 bg-[#f7f5f3] p-6 sm:p-8 lg:p-10"
          : "rounded-2xl border border-emerald-500/20 bg-emerald-50/40 p-6 sm:p-8 lg:p-10"
      }
    >
      <div className="flex items-center gap-2">
        <span
          className={
            isPast
              ? "size-2 rounded-full bg-[#0a0d12]/30"
              : "size-2 rounded-full bg-emerald-500"
          }
        />
        <h3
          className={
            isPast
              ? "text-[12px] font-semibold uppercase tracking-[0.14em] text-[#0a0d12]/48"
              : "text-[12px] font-semibold uppercase tracking-[0.14em] text-emerald-700"
          }
        >
          {label}
        </h3>
      </div>

      <dl className="mt-6 divide-y divide-[#0a0d12]/8">
        {items.map((item, i) => (
          <div
            key={i}
            className="grid grid-cols-[100px_1fr] gap-x-4 py-4 sm:grid-cols-[140px_1fr] sm:gap-x-6"
          >
            <dt
              className={
                isPast
                  ? "text-[12px] font-semibold uppercase tracking-wider text-[#0a0d12]/40 sm:text-[13px]"
                  : "text-[12px] font-semibold uppercase tracking-wider text-emerald-700/80 sm:text-[13px]"
              }
            >
              {item.question}
            </dt>
            <dd
              className={
                isPast
                  ? "text-[14px] leading-[1.6] text-[#0a0d12]/70 sm:text-[15px]"
                  : "text-[14px] font-medium leading-[1.6] text-[#0a0d12] sm:text-[15px]"
              }
            >
              {item.answer}
            </dd>
          </div>
        ))}
      </dl>
    </div>
  );
}
