"use client";

import { useEffect, useState } from "react";
import { motion, useReducedMotion } from "motion/react";

const PHASE_COUNT = 4;
const PHASE_DURATION_MS = 2600;
const MORPH_EASE = [0.23, 1, 0.32, 1] as const;

type TileState = {
  x: number;
  y: number;
  width: number;
  height: number;
  rx: number;
  fill: string;
  opacity?: number;
  stroke?: string;
  strokeWidth?: number;
};

const PAPER = "#fffdf7";
const INK = "#12130f";
const CORAL = "#f05a3f";
const BLUE = "#4b79d8";
const YELLOW = "#f6c84a";
const GREEN = "#35a87d";
const PINK = "#e78ab2";
const MUTED = "#d8cfbf";

const hiddenTile = (fill: string): TileState => ({
  x: 248,
  y: 248,
  width: 24,
  height: 24,
  rx: 8,
  fill,
  opacity: 0,
  stroke: INK,
  strokeWidth: 0,
});

/**
 * Twelve persistent pieces reorganise into four product-shaped scenes:
 * Chimii mark → issue → agent hand-off → shared board. Keeping the same
 * SVG rectangles mounted lets Motion interpolate the geometry instead of
 * cross-fading between unrelated illustrations.
 */
const TILE_STATES: readonly (readonly TileState[])[] = [
  // 0 — Chimii mark.
  [
    { x: 158, y: 132, width: 170, height: 170, rx: 34, fill: CORAL, stroke: INK, strokeWidth: 3 },
    { x: 238, y: 230, width: 170, height: 170, rx: 34, fill: BLUE, stroke: INK, strokeWidth: 3 },
    { x: 232, y: 211, width: 76, height: 76, rx: 38, fill: YELLOW, stroke: PAPER, strokeWidth: 7 },
    hiddenTile(BLUE),
    hiddenTile(MUTED),
    hiddenTile(GREEN),
    hiddenTile(PINK),
    hiddenTile(YELLOW),
    hiddenTile(BLUE),
    hiddenTile(GREEN),
    hiddenTile(PAPER),
    hiddenTile(CORAL),
  ],
  // 1 — One issue becomes a structured, trackable object.
  [
    { x: 58, y: 88, width: 404, height: 344, rx: 32, fill: PAPER, stroke: INK, strokeWidth: 3 },
    { x: 92, y: 126, width: 64, height: 64, rx: 18, fill: CORAL, stroke: INK, strokeWidth: 3 },
    { x: 374, y: 138, width: 54, height: 22, rx: 11, fill: YELLOW, stroke: INK, strokeWidth: 2 },
    { x: 92, y: 224, width: 314, height: 20, rx: 10, fill: BLUE, opacity: 1 },
    { x: 92, y: 262, width: 248, height: 15, rx: 8, fill: MUTED, opacity: 1 },
    { x: 92, y: 322, width: 96, height: 40, rx: 20, fill: GREEN, stroke: INK, strokeWidth: 2 },
    { x: 207, y: 322, width: 72, height: 40, rx: 20, fill: PINK, stroke: INK, strokeWidth: 2 },
    { x: 298, y: 322, width: 72, height: 40, rx: 20, fill: YELLOW, stroke: INK, strokeWidth: 2 },
    hiddenTile(BLUE),
    hiddenTile(GREEN),
    hiddenTile(PAPER),
    hiddenTile(CORAL),
  ],
  // 2 — Human and agents hand the issue across one shared context.
  [
    { x: 176, y: 180, width: 168, height: 154, rx: 28, fill: PAPER, stroke: INK, strokeWidth: 3 },
    { x: 42, y: 116, width: 96, height: 96, rx: 26, fill: CORAL, stroke: INK, strokeWidth: 3 },
    { x: 382, y: 116, width: 96, height: 96, rx: 26, fill: BLUE, stroke: INK, strokeWidth: 3 },
    { x: 118, y: 226, width: 88, height: 12, rx: 6, fill: YELLOW, opacity: 1 },
    { x: 314, y: 226, width: 88, height: 12, rx: 6, fill: GREEN, opacity: 1 },
    { x: 212, y: 370, width: 96, height: 72, rx: 24, fill: GREEN, stroke: INK, strokeWidth: 3 },
    { x: 254, y: 318, width: 12, height: 72, rx: 6, fill: PINK, opacity: 1 },
    { x: 204, y: 214, width: 112, height: 16, rx: 8, fill: YELLOW, opacity: 1 },
    { x: 204, y: 250, width: 86, height: 14, rx: 7, fill: BLUE, opacity: 1 },
    { x: 204, y: 286, width: 64, height: 18, rx: 9, fill: GREEN, opacity: 1 },
    hiddenTile(PAPER),
    hiddenTile(CORAL),
  ],
  // 3 — The work settles into a visible board and reaches Done.
  [
    { x: 30, y: 84, width: 140, height: 354, rx: 28, fill: PAPER, stroke: INK, strokeWidth: 3 },
    { x: 190, y: 84, width: 140, height: 354, rx: 28, fill: PAPER, stroke: INK, strokeWidth: 3 },
    { x: 350, y: 84, width: 140, height: 354, rx: 28, fill: PAPER, stroke: INK, strokeWidth: 3 },
    { x: 50, y: 148, width: 100, height: 82, rx: 18, fill: CORAL, stroke: INK, strokeWidth: 2 },
    { x: 210, y: 136, width: 100, height: 96, rx: 18, fill: YELLOW, stroke: INK, strokeWidth: 2 },
    { x: 370, y: 184, width: 100, height: 94, rx: 18, fill: BLUE, stroke: INK, strokeWidth: 2 },
    { x: 210, y: 270, width: 100, height: 76, rx: 18, fill: GREEN, stroke: INK, strokeWidth: 2 },
    { x: 50, y: 286, width: 100, height: 72, rx: 18, fill: PINK, stroke: INK, strokeWidth: 2 },
    { x: 50, y: 108, width: 26, height: 12, rx: 6, fill: CORAL, opacity: 1 },
    { x: 210, y: 108, width: 26, height: 12, rx: 6, fill: YELLOW, opacity: 1 },
    { x: 370, y: 108, width: 26, height: 12, rx: 6, fill: GREEN, opacity: 1 },
    { x: 370, y: 322, width: 100, height: 54, rx: 27, fill: GREEN, stroke: INK, strokeWidth: 2 },
  ],
] as const;

export function InventionLoop() {
  const shouldReduceMotion = useReducedMotion() ?? false;
  const [phase, setPhase] = useState(shouldReduceMotion ? PHASE_COUNT - 1 : 0);
  const [isVisible, setIsVisible] = useState(true);

  useEffect(() => {
    const handleVisibilityChange = () => {
      setIsVisible(document.visibilityState !== "hidden");
    };
    handleVisibilityChange();
    document.addEventListener("visibilitychange", handleVisibilityChange);
    return () => document.removeEventListener("visibilitychange", handleVisibilityChange);
  }, []);

  useEffect(() => {
    if (shouldReduceMotion) {
      setPhase(PHASE_COUNT - 1);
      return;
    }
    if (!isVisible) return;

    const timer = window.setInterval(() => {
      setPhase((current) => (current + 1) % PHASE_COUNT);
    }, PHASE_DURATION_MS);
    return () => window.clearInterval(timer);
  }, [isVisible, shouldReduceMotion]);

  const tileStates = TILE_STATES[phase] ?? TILE_STATES[PHASE_COUNT - 1]!;

  return (
    <div className="w-full max-w-[520px]" aria-hidden="true">
      <div className="relative aspect-square overflow-hidden rounded-[40px] border-2 border-[#12130f] bg-[#fffdf7]/70 shadow-[10px_12px_0_rgba(18,19,15,0.16)]">
        <div className="chimii-invention-grid absolute inset-0 opacity-55" />
        <div className="absolute -left-12 top-10 size-32 rotate-12 rounded-[30px] border-[14px] border-[#f6c84a]/50" />
        <div className="absolute -right-10 bottom-8 size-28 rounded-full border-[14px] border-[#e78ab2]/40" />

        <svg
          viewBox="0 0 520 520"
          className="relative h-full w-full"
          focusable="false"
        >
          <motion.path
            d="M138 164 C176 164 176 208 206 226"
            fill="none"
            stroke={INK}
            strokeWidth="3"
            strokeDasharray="8 9"
            animate={{ opacity: phase === 2 ? 0.42 : 0 }}
            transition={{ duration: 0.35 }}
          />
          <motion.path
            d="M382 164 C344 164 344 208 314 226"
            fill="none"
            stroke={INK}
            strokeWidth="3"
            strokeDasharray="8 9"
            animate={{ opacity: phase === 2 ? 0.42 : 0 }}
            transition={{ duration: 0.35 }}
          />

          {tileStates.map((state, index) => (
            <motion.rect
              // The index is the stable identity of a physical module across
              // all four scenes; it must not change when a phase advances.
              key={index}
              initial={false}
              animate={{
                opacity: 1,
                stroke: "transparent",
                strokeWidth: 0,
                ...state,
              }}
              transition={{
                duration: shouldReduceMotion ? 0 : 0.9,
                ease: MORPH_EASE,
                delay: shouldReduceMotion ? 0 : index * 0.018,
              }}
            />
          ))}

          <motion.g
            animate={{ opacity: phase === 2 ? 1 : 0 }}
            transition={{ duration: 0.3, delay: phase === 2 ? 0.45 : 0 }}
          >
            <circle cx="76" cy="153" r="8" fill={PAPER} stroke={INK} strokeWidth="3" />
            <circle cx="105" cy="153" r="8" fill={PAPER} stroke={INK} strokeWidth="3" />
            <path d="M74 184 Q90 197 106 184" fill="none" stroke={INK} strokeWidth="4" strokeLinecap="round" />
            <circle cx="416" cy="153" r="7" fill={PAPER} />
            <circle cx="444" cy="153" r="7" fill={PAPER} />
            <path d="M414 184 H446" fill="none" stroke={PAPER} strokeWidth="4" strokeLinecap="round" />
          </motion.g>

          <motion.path
            d="M395 349 L414 366 L449 335"
            fill="none"
            stroke={PAPER}
            strokeWidth="8"
            strokeLinecap="round"
            strokeLinejoin="round"
            initial={false}
            animate={{
              opacity: phase === 3 ? 1 : 0,
              pathLength: phase === 3 ? 1 : 0,
            }}
            transition={{
              opacity: { duration: 0.2 },
              pathLength: { duration: 0.48, delay: phase === 3 ? 0.62 : 0 },
            }}
          />
        </svg>
      </div>

      <div className="mt-6 flex items-center justify-center gap-2">
        {Array.from({ length: PHASE_COUNT }, (_, index) => (
          <motion.span
            key={index}
            className="block h-2 rounded-full border border-[#12130f]/30"
            animate={{
              width: phase === index ? 28 : 8,
              backgroundColor:
                phase === index
                  ? [CORAL, YELLOW, BLUE, GREEN][index]
                  : "rgba(255,253,247,0.65)",
            }}
            transition={{ duration: shouldReduceMotion ? 0 : 0.35, ease: MORPH_EASE }}
          />
        ))}
      </div>
    </div>
  );
}
