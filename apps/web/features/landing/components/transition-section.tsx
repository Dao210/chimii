"use client";

import { useLocale } from "../i18n";

/**
 * TransitionSection — single-line bridge between the Product 01 features
 * showcase and the Product 02 workflow engine. Mirrors meridianos.ai's
 * "看懂一家公司只是起点。下一步,是把整个市场*打下来*。" beat.
 */
export function TransitionSection() {
  const { t } = useLocale();
  const tr = t.transition;

  return (
    <section
      id="transition"
      className="bg-white py-24 text-[#0a0d12] sm:py-28 lg:py-32"
    >
      <div className="mx-auto max-w-[1120px] px-4 text-center sm:px-6 lg:px-8">
        <h2 className="font-semibold text-[2.2rem] leading-[1.1] tracking-[-0.02em] text-[#0a0d12] sm:text-[3rem] lg:text-[3.6rem]">
          {tr.headlineLine1}
          <br />
          <span className="text-[#0a0d12]/56">{tr.headlineLine2}</span>
        </h2>
        <p className="mx-auto mt-6 max-w-[640px] text-[14px] leading-7 text-[#0a0d12]/52 sm:text-[15px]">
          {tr.body}
        </p>
      </div>
    </section>
  );
}
