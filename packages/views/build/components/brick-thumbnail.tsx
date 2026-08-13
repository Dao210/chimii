"use client";

import { useEffect, useState } from "react";
import { Box } from "lucide-react";
import { cn } from "@chimii/ui/lib/utils";
import { getLDrawThumbnail } from "../catalog/catalog-thumbnails.generated";

export function BrickThumbnail({
  ldrawID,
  colorCode,
  alt,
  eager = false,
  className,
  studsX = 0,
  studsZ = 0,
  platesY = 0,
}: {
  ldrawID: string;
  colorCode: number;
  alt: string;
  eager?: boolean;
  className?: string;
  studsX?: number;
  studsZ?: number;
  platesY?: number;
}) {
  const asset = getLDrawThumbnail(ldrawID, colorCode);
  const [status, setStatus] = useState<"loading" | "ready" | "failed">("loading");

  useEffect(() => setStatus("loading"), [asset?.src]);

  return (
    <div
      className={cn(
        "group/brick relative isolate aspect-[8/5] w-full overflow-hidden rounded-2xl border border-[#d8cfbf] bg-[linear-gradient(145deg,#f8f4e9_0%,#edf4ff_55%,#dce9f7_100%)]",
        className,
      )}
      data-thumbnail-source={asset ? "ldraw-glb" : "missing"}
      data-ldraw-id={ldrawID}
      data-color-code={colorCode}
    >
      <div className="absolute inset-0 opacity-35 [background-image:linear-gradient(rgba(60,111,198,.16)_1px,transparent_1px),linear-gradient(90deg,rgba(60,111,198,.16)_1px,transparent_1px)] [background-size:18px_18px]" />
      <div className="absolute bottom-[13%] left-1/2 h-[9%] w-[48%] -translate-x-1/2 rounded-full bg-[#1d241f]/20 blur-[5px]" />

      {asset && status !== "failed" ? (
        <>
          <div
            aria-hidden="true"
            className={cn(
              "absolute inset-3 rounded-xl bg-white/35 transition-opacity duration-300",
              status === "ready" ? "opacity-0" : "animate-pulse opacity-100 motion-reduce:animate-none",
            )}
          />
          <img
            src={asset.src}
            width={asset.width}
            height={asset.height}
            alt={alt}
            loading={eager ? "eager" : "lazy"}
            decoding="async"
            draggable={false}
            className={cn(
              "absolute inset-0 z-[1] h-full w-full select-none object-contain p-1.5 transition duration-300 ease-out group-hover/brick:scale-[1.035]",
              status === "ready" ? "translate-y-0 opacity-100" : "translate-y-1 opacity-0",
            )}
            onLoad={() => setStatus("ready")}
            onError={() => setStatus("failed")}
          />
        </>
      ) : studsX > 0 && studsZ > 0 && platesY > 0 ? (
        <svg viewBox="0 0 160 100" role="img" aria-label={alt} className="absolute inset-0 h-full w-full p-5">
          <g transform="translate(80 52)">
            <path d="M-44 -8 0 -30 44 -8 0 15Z" fill="currentColor" className="text-[#f28b80]" stroke="#1d241f" strokeWidth="2" />
            <path d="M-44 -8 0 15 0 36 -44 13Z" fill="currentColor" className="text-[#d85b50]" stroke="#1d241f" strokeWidth="2" />
            <path d="M44 -8 0 15 0 36 44 13Z" fill="currentColor" className="text-[#b7433a]" stroke="#1d241f" strokeWidth="2" />
            {Array.from({ length: Math.min(8, Math.max(1, studsX * studsZ)) }, (_, index) => {
              const columns = Math.min(4, Math.max(1, studsX));
              const row = Math.floor(index / columns);
              const column = index % columns;
              return <ellipse key={index} cx={(column - (columns - 1) / 2) * 16 + row * 4} cy={-20 + row * 8 - column * 4} rx="6" ry="3.2" fill="#ffaaa1" stroke="#1d241f" strokeWidth="1.4" />;
            })}
          </g>
        </svg>
      ) : (
        <div role="img" aria-label={alt} className="absolute inset-0 flex items-center justify-center text-[#69736d]">
          <Box className="size-9" aria-hidden="true" />
        </div>
      )}

      <div className="pointer-events-none absolute inset-x-0 bottom-0 h-10 bg-gradient-to-t from-white/40 to-transparent" />
    </div>
  );
}
