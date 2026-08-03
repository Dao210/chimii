import * as React from "react";

import { cn } from "../../lib/utils";

interface ChimiiInventorBrandProps extends React.ComponentProps<"span"> {
  inverse?: boolean;
}

/**
 * Shared CHIMII 奇觅 inventor lockup used by the marketing and auth surfaces.
 * The parent controls the centre-dot cutout with --chimii-brand-surface so the
 * same mark stays crisp on canvas, paper, and inverse backgrounds.
 */
export function ChimiiInventorBrand({
  className,
  inverse = false,
  ...props
}: ChimiiInventorBrandProps) {
  return (
    <span
      className={cn("inline-flex shrink-0 items-center gap-2.5", className)}
      {...props}
    >
      <span className="relative block size-7" aria-hidden="true">
        <span className="absolute left-0 top-0 size-3.5 rounded-[4px] bg-[#f05a3f] transition-transform duration-200 group-hover:-rotate-6" />
        <span className="absolute bottom-0 right-0 size-4 rounded-[4px] bg-[#4b79d8] transition-transform duration-200 group-hover:rotate-6" />
        <span className="absolute left-[9px] top-[8px] size-2.5 rounded-full border-2 border-[var(--chimii-brand-surface,#f5f0e6)] bg-[#f6c84a]" />
      </span>
      <span
        className={cn(
          "text-[15px] font-extrabold tracking-[0.13em] sm:text-base",
          inverse ? "text-white" : "text-[#12130f]",
        )}
      >
        CHIMII{" "}
        <span className="tracking-normal opacity-48 max-[359px]:hidden">
          奇觅
        </span>
      </span>
    </span>
  );
}
