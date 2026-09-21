"use client";
import { HugeiconsIcon } from "@hugeicons/react";
import type { IconSvgElement } from "@hugeicons/react";
import {
  Link01Icon,
  Note01Icon,
  SourceCodeIcon,
  File01Icon,
  Image01Icon,
  Bookmark01Icon,
} from "@hugeicons/core-free-icons";

// Shared brand + result atoms. One place, so screens never disagree.

// Tile wash, ink and plain label per source_type. Lively tints, still quiet.
export const TYPE_STYLES: Record<string, { tint: string; ink: string; glyph: IconSvgElement; label: string }> = {
  url: { tint: "#f1f1f6", ink: "#2563eb", glyph: Link01Icon, label: "link" },
  text: { tint: "#f1f1f6", ink: "#b45309", glyph: Note01Icon, label: "note" },
  note: { tint: "#f1f1f6", ink: "#b45309", glyph: Note01Icon, label: "note" },
  file: { tint: "#f1f1f6", ink: "#16a34a", glyph: SourceCodeIcon, label: "code" },
  pdf: { tint: "#f1f1f6", ink: "#0d9488", glyph: File01Icon, label: "pdf" },
  image: { tint: "#f1f1f6", ink: "#9333ea", glyph: Image01Icon, label: "image" },
  bookmark: { tint: "#f1f1f6", ink: "#fc4c01", glyph: Bookmark01Icon, label: "saved" },
};

export const styleOf = (t: string): { tint: string; ink: string; glyph: IconSvgElement; label: string } =>
  TYPE_STYLES[t] ?? { tint: "#f1f1f6", ink: "#868593", glyph: File01Icon, label: t || "item" };

// Claim status → tinted chip classes.
export const CLAIM_CHIP: Record<string, string> = {
  backed: "bg-good-soft text-good",
  guess: "bg-warn-soft text-warn",
  contradicted: "bg-bad-soft text-bad",
  disputed: "bg-warn-soft text-warn",
  unknown: "bg-hover text-muted",
};

// Brand mark: two interlocked rings — the "oo" in loom, two memories linked.
export function Mark({ size = 22 }: { size?: number }) {
  return (
    <svg width={size} height={(size * 44) / 64} viewBox="0 0 64 44" aria-hidden="true" fill="none">
      <circle cx="41" cy="22" r="12.5" fill="none" stroke="currentColor" strokeWidth="5" />
      <path d="M29.44 32.71A12.5 12.5 0 1 0 33.93 28.06" fill="none" stroke="currentColor" strokeWidth="5" />
    </svg>
  );
}

// Result tile: tinted wash + colored glyph per source_type.
export function Tile({ type, size = 36 }: { type: string; size?: number }) {
  const s = styleOf(type);
  return (
    <span
      className="grid shrink-0 place-items-center rounded-[10px]"
      style={{
        background: s.tint,
        color: s.ink,
        width: size,
        height: size,
        boxShadow: "inset 0 0 0 1px rgba(15, 17, 21, 0.05)",
      }}
      aria-hidden="true"
    >
      <HugeiconsIcon icon={s.glyph} size={Math.round(size * 0.47)} />
    </span>
  );
}