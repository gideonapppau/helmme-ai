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

// The real X brand mark: filled, on the 24-grid, so it reads at 12px.
// Hugeicons' free "NewTwitterIcon" is a thin stroke approximation, // a brand mark must be its true filled shape, same as the helmme Mark.
export const XLogoGlyph: IconSvgElement = [
  [
    "path",
    {
      d: "M18.244 2.25h3.308l-7.227 8.26 8.502 11.24H16.17l-5.214-6.817L4.99 21.75H1.68l7.73-8.835L1.254 2.25H8.08l4.713 6.231 5.451-6.231Zm-1.161 17.52h1.833L7.084 4.126H5.117l11.966 15.644Z",
      fill: "currentColor",
      key: "0",
    },
  ],
] as const;

// Shared brand + result atoms. One place, so screens never disagree.

// Tile wash, ink and plain label per source_type. Lively tints, still quiet.
// Tile wash + ink per source_type. Colors live in CSS vars (globals.css)
// so the tiles restyle with the theme, light paper and dark night.
export const TYPE_STYLES: Record<string, { tint: string; ink: string; glyph: IconSvgElement; label: string }> = {
  url: { tint: "var(--tile-url-bg)", ink: "var(--tile-url-ink)", glyph: Link01Icon, label: "link" },
  text: { tint: "var(--tile-note-bg)", ink: "var(--tile-note-ink)", glyph: Note01Icon, label: "note" },
  note: { tint: "var(--tile-note-bg)", ink: "var(--tile-note-ink)", glyph: Note01Icon, label: "note" },
  file: { tint: "var(--tile-file-bg)", ink: "var(--tile-file-ink)", glyph: SourceCodeIcon, label: "code" },
  pdf: { tint: "var(--tile-pdf-bg)", ink: "var(--tile-pdf-ink)", glyph: File01Icon, label: "pdf" },
  image: { tint: "var(--tile-image-bg)", ink: "var(--tile-image-ink)", glyph: Image01Icon, label: "image" },
  bookmark: { tint: "var(--tile-bookmark-bg)", ink: "var(--tile-bookmark-ink)", glyph: Bookmark01Icon, label: "saved" },
  x: { tint: "var(--tile-x-bg)", ink: "var(--tile-x-ink)", glyph: XLogoGlyph, label: "x" },
};

export const styleOf = (t: string): { tint: string; ink: string; glyph: IconSvgElement; label: string } =>
  TYPE_STYLES[t] ?? { tint: "var(--tile-fallback-bg)", ink: "var(--tile-fallback-ink)", glyph: File01Icon, label: t || "item" };

// Claim status → tinted chip classes.
export const CLAIM_CHIP: Record<string, string> = {
  backed: "bg-good-soft text-good",
  guess: "bg-warn-soft text-warn",
  contradicted: "bg-bad-soft text-bad",
  disputed: "bg-warn-soft text-warn",
  unknown: "bg-hover text-muted",
};

// Brand mark: two interlocked rings, the "oo" in loom, two memories linked.
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