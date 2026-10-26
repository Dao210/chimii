import * as React from "react";

import { cn } from "../../lib/utils";
import { CHIMII_BRAND_MARK_SRC } from "./brand-mark";

interface ChimiiInventorBrandProps extends React.ComponentProps<"span"> {
  inverse?: boolean;
}

/**
 * Shared CHIMII 奇觅 inventor lockup used by the marketing and auth surfaces.
 * The mascot tile reads on canvas, paper, and inverse backgrounds without any
 * surface-dependent cutout.
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
      <img
        src={CHIMII_BRAND_MARK_SRC}
        alt=""
        draggable={false}
        aria-hidden="true"
        className="block size-8 shrink-0 select-none"
      />
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
