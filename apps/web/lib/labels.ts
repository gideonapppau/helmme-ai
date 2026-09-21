// Shared plain words. One place, so screens never disagree.
// A raw URL is never a title. Show the domain, or a plain fallback.
export function displayTitle(h: { title: string | null; domain: string | null }) {
  const t = (h.title || "").trim();
  if (t && !/^https?:\/\//i.test(t)) return t;
  return h.domain || "Saved link";
}

export const REASONS: Record<string, string> = {
  "same-site": "Same site",
  "similar-words": "Similar words",
  "saved-together": "Saved together",
  "same-topic": "Same topic",
  earlier: "Saved just before",
  later: "Saved just after",
};

export const CLAIM_WORDS: Record<string, string> = {
  backed: "backed",
  guess: "guess",
  contradicted: "clashes with a source",
  disputed: "sources disagree",
  unknown: "unclear",
};

export const DUP_REASONS: Record<string, string> = {
  "same-file": "Same file, saved more than once",
  "same-link": "Same link, saved twice",
  similar: "Very similar",
};

// The archive is organized by time, not folders. Five quiet buckets.
export function timeBucket(iso: string, now: number | null): { key: string; label: string } {
  if (!now) return { key: "earlier", label: "Earlier" };
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return { key: "earlier", label: "Earlier" };
  const startToday = new Date(now);
  startToday.setHours(0, 0, 0, 0);
  const t = d.getTime();
  if (t >= startToday.getTime()) return { key: "today", label: "Today" };
  if (t >= startToday.getTime() - 86400000) return { key: "yesterday", label: "Yesterday" };
  if (t >= startToday.getTime() - 6 * 86400000) return { key: "week", label: "This week" };
  if (t >= startToday.getTime() - 29 * 86400000) return { key: "month", label: "This month" };
  return { key: "earlier", label: "Earlier" };
}
