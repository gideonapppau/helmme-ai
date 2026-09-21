"use client";
import { useEffect, useRef, useState } from "react";
import { HugeiconsIcon } from "@hugeicons/react";
import {
  Search01Icon,
  Bookmark01Icon,
  Settings02Icon,
  Cancel01Icon,
  File01Icon,
  Layers01Icon,
  Delete02Icon,
  SparklesIcon,
  ArrowUpRight01Icon,
  LinkSquare02Icon,
  PlusSignIcon,
  ChevronDownIcon,
  AlertCircleIcon,
  ImportIcon,
  MergeIcon,
  HistoryIcon,
  ClipboardCheckIcon,
  Sun03Icon,
  Moon02Icon,
} from "@hugeicons/core-free-icons";
import { fastPath } from "@helmme/context-engine";
import { AnimatePresence, motion, useReducedMotion } from "motion/react";
import Link from "next/link";
import { CLAIM_WORDS, DUP_REASONS, REASONS, displayTitle, timeBucket } from "../lib/labels";
import { CLAIM_CHIP, Mark, Tile, styleOf, XLogoGlyph } from "../lib/ui";
import { AnimatedCounter } from "../components/ui/animated-counter";
import { DeleteButton } from "../components/ui/delete-button";
import { GooeyNav } from "../components/ui/gooey-nav";
import { Folder } from "../components/ui/folder-component";
import MatrixOrb from "../components/ui/matrix-orb";

const API = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8081";

// Shape returned by POST /v1/search (services/api/main.go).
type Hit = {
  id: string;
  title: string | null;
  source_type: string;
  domain: string | null;
  captured_at: string;
  raw_ref: string;
  excerpt: string;
  topics: string[];
  rank: number;
};

type Health = "unknown" | "up" | "down";
type Phase = "idle" | "loading" | "error" | "done";

// Sidebar sections map to the real source_type enums (db/migrations/001_init.sql).
// Filtering is honest: it narrows the results already returned by the api.
const FILTERS: Record<string, { label: string; types: string[] }> = {
  notes: { label: "Notes", types: ["text", "note"] },
  articles: { label: "Articles", types: ["url", "pdf"] },
  code: { label: "Code", types: ["file"] },
  media: { label: "Media", types: ["image"] },
  saved: { label: "Saved", types: ["bookmark"] },
  x: { label: "X", types: ["x"] },
};

function relTime(iso: string, now: number | null): string {
  if (!now) return "";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "-";
  const s = Math.max((now - d.getTime()) / 1000, 0);
  if (s < 60) return "just now";
  if (s < 3600) return `${Math.floor(s / 60)}m ago`;
  if (s < 86400) return `${Math.floor(s / 3600)}h ago`;
  if (s < 172800) return "yesterday";
  if (s < 604800) return `${Math.floor(s / 86400)}d ago`;
  return d.toISOString().slice(0, 10);
}

// Search-first omnibar: SEARCH default, SYNTHESIS explicit escalation, evidence-first.
export default function Home() {
  const [q, setQ] = useState("");
  const [hits, setHits] = useState<Hit[]>([]);
  const [lastQuery, setLastQuery] = useState("");
  const [phase, setPhase] = useState<Phase>("idle");
  const [error, setError] = useState("");
  const [health, setHealth] = useState<Health>("unknown");
  const [filter, setFilter] = useState<string | null>(null);
  const [showSettings, setShowSettings] = useState(false);
  const [showCleanup, setShowCleanup] = useState(false);
  const [showReview, setShowReview] = useState(false);
  const [review, setReview] = useState<{
    saved_this_week: number;
    top_topics: { name: string; count: number }[];
    top_entities: { name: string; count: number }[];
    resurfaced: { id: string; title: string; reason: string }[];
    duplicate_groups: number;
    unopened_bookmarks: number;
  } | null>(null);
  const [reviewLoading, setReviewLoading] = useState(false);
  const [dupeGroups, setDupeGroups] = useState<{ reason: string; items: { id: string; title: string; domain: string }[] }[]>([]);
  const [dupeLoading, setDupeLoading] = useState(false);
  const [views, setViews] = useState<{ id: string; name: string; query_text: string; count: number; new: number }[]>([]);
  const [resurf, setResurf] = useState<{ id: string; title: string; reason: string; seen: boolean }[]>([]);
  const [hideSeen, setHideSeen] = useState(true);
  // Constitution §3 nav: Search · Recent · Capture · Archive · Settings.
  const [showRecent, setShowRecent] = useState(false);
  const [showArchive, setShowArchive] = useState(false);
  const [recentItems, setRecentItems] = useState<{ id: string; title: string; source_type: string; domain: string; captured_at: string; excerpt: string }[]>([]);
  const [recentLoading, setRecentLoading] = useState(false);
  const [archiveView, setArchiveView] = useState("all");
  const [archiveData, setArchiveData] = useState<{ items?: { id: string; title: string; source_type: string; domain: string }[]; groups?: { name: string; count: number; id?: string; query_text?: string }[] }>({});
  const [archiveLoading, setArchiveLoading] = useState(false);
  // Constitution §11-§14 X connection + §38 resurfacing mode.
  const [xHandle, setXHandle] = useState("");
  const [xMsg, setXMsg] = useState("");
  const [resurfaceMode, setResurfaceMode] = useState<"quiet" | "balanced" | "proactive">("quiet");
  // Constitution §38: the mood lives on the server so every device agrees.
  // Plain words: read the saved mood when Settings opens, store it on change.
  useEffect(() => {
    if (!showSettings) return;
    fetch(`${API}/v1/settings`)
      .then((r) => (r.ok ? r.json() : null))
      .then((j) => {
        const m = j?.resurface_mode;
        if (m === "quiet" || m === "balanced" || m === "proactive") setResurfaceMode(m);
      })
      .catch(() => {});
  }, [showSettings]);

  async function saveResurfaceMode(m: "quiet" | "balanced" | "proactive") {
    setResurfaceMode(m);
    try {
      await fetch(`${API}/v1/settings`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ resurface_mode: m }),
      });
    } catch {
      /* local state already shows it; server syncs next open */
    }
  }
  // Theme: null until we know (SSR-safe), then mirrors the html.dark class.
  const [night, setNight] = useState<boolean | null>(null);

  useEffect(() => {
    setNight(document.documentElement.classList.contains("dark"));
  }, []);

  // Focus the omnibar on load, desktop only. On phones an autofocus
  // steals the first tap and pops the keyboard unprompted.
  useEffect(() => {
    if (window.matchMedia("(min-width: 768px)").matches) {
      inputRef.current?.focus();
    }
  }, []);

  function toggleTheme() {
    const root = document.documentElement;
    const next = !root.classList.contains("dark");
    root.classList.toggle("dark", next);
    try {
      localStorage.setItem("helmme-theme", next ? "dark" : "light");
    } catch {}
    setNight(next);
  }

  async function loadResurf(hide: boolean) {
    try {
      const r = await fetch(`${API}/v1/resurface${hide ? "" : "?show_seen=1"}`);
      const j = await r.json();
      setResurf(j.items ?? []);
    } catch {
      setResurf([]);
    }
  }
  const [now, setNow] = useState<number | null>(null);
  const [jevNote, setJevNote] = useState("");
  const [uploadMsg, setUploadMsg] = useState("");
  const [showCapture, setShowCapture] = useState(false);
  const [capText, setCapText] = useState("");
  const [capMsg, setCapMsg] = useState("");
  const [synPhase, setSynPhase] = useState<"idle" | "loading" | "done">("idle");
  const [synSummary, setSynSummary] = useState<string | null>(null);
  const [synNote, setSynNote] = useState("");
  const [synEv, setSynEv] = useState<{ id: string; title: string; source_type: string; excerpt: string }[]>([]);
  const [synClaims, setSynClaims] = useState<{ text: string; status: string; sources?: string[]; rejected?: boolean }[]>([]);
  const [synId, setSynId] = useState("");

  async function dismissClaim(index: number) {
    if (!synId) return;
    try {
      const r = await fetch(`${API}/v1/synthesis/${synId}/claims`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ index, rejected: true }),
      });
      if (!r.ok) return;
    } catch {
      return;
    }
    setSynClaims((prev) => prev.map((c, i) => (i === index ? { ...c, rejected: true } : c)));
  }
  const [openClaim, setOpenClaim] = useState<number | null>(null);
  const [litEv, setLitEv] = useState<string | null>(null);
  const litTimer = useRef<number | undefined>(undefined);

  function citeTo(n: number) {
    const ev = synEv[n - 1];
    if (!ev) return;
    document.getElementById(`ev-${n}`)?.scrollIntoView({ behavior: "smooth", block: "center" });
    setLitEv(ev.id);
    window.clearTimeout(litTimer.current);
    litTimer.current = window.setTimeout(() => setLitEv(null), 1600);
  }

  function renderProse(para: string, key: number) {
    const parts = para.split(/(\[\d+\])/g);
    return (
      <p key={key} className="m-0 mb-2.5 font-serif text-[14.5px] leading-[1.75] text-deep last:mb-0">
        {parts.map((part, i) => {
          const m = /^\[(\d+)\]$/.exec(part);
          const n = m ? parseInt(m[1], 10) : 0;
          if (n >= 1 && n <= synEv.length) {
            return (
              <button
                key={i}
                type="button"
                onClick={() => citeTo(n)}
                className="font-medium text-deep underline decoration-dotted underline-offset-2"
                aria-label={`Go to source ${n}`}
              >
                {part}
              </button>
            );
          }
          return <span key={i}>{part}</span>;
        })}
      </p>
    );
  }

  const inputRef = useRef<HTMLInputElement>(null);
  const fileRef = useRef<HTMLInputElement>(null);
  const importRef = useRef<HTMLInputElement>(null);

  const route = fastPath(q);
  const trimmed = q.trim();
  const visible = filter ? hits.filter((h) => FILTERS[filter].types.includes(h.source_type)) : hits;

  useEffect(() => {
    setNow(Date.now());
    let alive = true;
    fetch(`${API}/healthz`)
      .then((r) => {
        if (alive) setHealth(r.ok ? "up" : "down");
      })
      .catch(() => {
        if (alive) setHealth("down");
      });
    return () => {
      alive = false;
    };
  }, []);

  // Constitution §57: Cmd/Ctrl+K search · Cmd/Ctrl+Shift+S capture ·
  // arrows navigate · Enter open · Cmd/Ctrl+Enter synthesize · Esc close.
  // "/" focuses search; cmd/ctrl+K focuses and selects, from anywhere.
  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if ((e.metaKey || e.ctrlKey) && e.shiftKey && e.key.toLowerCase() === "s") {
        e.preventDefault();
        setShowCapture(true);
        setShowSettings(false);
        setShowCleanup(false);
        setShowReview(false);
        return;
      }
      if ((e.metaKey || e.ctrlKey) && e.key === "Enter") {
        e.preventDefault();
        summarize();
        return;
      }
      if (e.key === "Escape") {
        setShowCapture(false);
        setShowSettings(false);
        setShowCleanup(false);
        setShowReview(false);
        setShowRecent(false);
        setShowArchive(false);
        setRelOpen(null);
        return;
      }
      // Plain words: up/down walk through the result links. Typing stays typing.
      if (e.key === "ArrowDown" || e.key === "ArrowUp") {
        const el = e.target as HTMLElement | null;
        const tag = el?.tagName;
        if (tag === "INPUT" || tag === "TEXTAREA") return;
        const links = Array.from(
          document.querySelectorAll('a[href^="/item/"]'),
        ) as HTMLElement[];
        if (links.length === 0) return;
        e.preventDefault();
        const i = links.indexOf(document.activeElement as HTMLElement);
        let next: number;
        if (i === -1) {
          next = e.key === "ArrowDown" ? 0 : links.length - 1;
        } else {
          next =
            e.key === "ArrowDown"
              ? (i + 1) % links.length
              : (i - 1 + links.length) % links.length;
        }
        links[next]?.focus();
        return;
      }
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "k") {
        e.preventDefault();
        inputRef.current?.focus();
        inputRef.current?.select();
      } else if (e.key === "/" && !e.metaKey && !e.ctrlKey && !e.altKey) {
        const el = e.target as HTMLElement | null;
        const tag = el?.tagName;
        if (tag !== "INPUT" && tag !== "TEXTAREA") {
          e.preventDefault();
          inputRef.current?.focus();
        }
      }
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);

  async function search(override?: string) {
    const query = (override ?? q).trim();
    if (!query || phase === "loading") return;
    setPhase("loading");
    setError("");
    setJevNote("");
    setRelOpen(null);
    setRelaxed(false);
    setScopeEcho(null);
    setSynPhase("idle");
    setSynSummary(null);
    setSynNote("");
    setSynEv([]);
    setSynClaims([]);
    setLitEv(null);
    setOpenClaim(null);
    window.clearTimeout(litTimer.current);
    if (!route) {
      try {
        const jr = await fetch("/api/route-query", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ query }),
        });
        const jd = await jr.json();
        if (jd.action !== "dispatch" && jd.reason) {
          setJevNote(String(jd.reason));
        }
      } catch {
        setJevNote("");
      }
    }
    try {
      const r = await fetch(`${API}/v1/search`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ query }),
      });
      if (!r.ok) throw new Error(`api answered ${r.status}`);
      const j = await r.json();
      setHits(j.hits ?? []);
      setRelaxed(j.relaxed === true);
      setScopeEcho(j.scope ?? null);
      setLastQuery(query);
      setPhase("done");
      setShowSettings(false);
      setShowCleanup(false);
      setShowReview(false);
      setShowRecent(false);
      setShowArchive(false);
    } catch (e) {
      setHealth("down");
      setPhase("error");
      setError(e instanceof Error ? e.message : "unknown failure");
    }
  }

  function focusSearch() {
    setShowSettings(false);
    setShowCleanup(false);
    setShowReview(false);
    setShowRecent(false);
    setShowArchive(false);
    window.scrollTo({ top: 0, behavior: "smooth" });
    inputRef.current?.focus();
  }

  // Constitution §6 Recent captures, chronological, no ranking.
  async function openRecent() {
    setShowSettings(false);
    setShowCleanup(false);
    setShowReview(false);
    setShowArchive(false);
    setFilter(null);
    setShowRecent(true);
    setRecentLoading(true);
    try {
      const r = await fetch(`${API}/v1/recent?limit=20`);
      const j = await r.json();
      setRecentItems(j.items ?? []);
    } catch {
      setRecentItems([]);
    }
    setRecentLoading(false);
  }

  // Constitution §19 Archive, views over one graph, never containers.
  async function openArchive(view: string = archiveView) {
    setShowSettings(false);
    setShowCleanup(false);
    setShowReview(false);
    setShowRecent(false);
    setFilter(null);
    setShowArchive(true);
    setArchiveView(view);
    setArchiveLoading(true);
    try {
      const r = await fetch(`${API}/v1/archive?view=${encodeURIComponent(view)}`);
      const j = await r.json();
      setArchiveData(j);
    } catch {
      setArchiveData({});
    }
    setArchiveLoading(false);
  }

  // Constitution §11 X connect, handle only, never a password. OAuth lands here.
  async function connectX() {
    const h = xHandle.trim();
    if (!h) return;
    setXMsg("Connecting...");
    try {
      const r = await fetch(`${API}/v1/sources/x/connect`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ handle: h }),
      });
      if (!r.ok) throw new Error(await r.text());
      setXMsg("Connected. Your bookmarks can now be imported as normal memory.");
      setXHandle("");
    } catch {
      setXMsg("Could not connect. Try again.");
    }
  }

  async function uploadFile(f: File) {
    setUploadMsg("Saving...");
    try {
      const fd = new FormData();
      fd.append("file", f);
      const r = await fetch(`${API}/v1/items/upload`, { method: "POST", body: fd });
      if (!r.ok) throw new Error(await r.text());
      setUploadMsg("Saved. Search for it above.");
    } catch {
      setUploadMsg("Could not save that file. PDF, images, text, and Word files only, under 25MB.");
    }
  }

  async function importBookmarks(f: File) {
    setUploadMsg("Importing...");
    try {
      const fd = new FormData();
      fd.append("file", f);
      const r = await fetch(`${API}/v1/imports`, { method: "POST", body: fd });
      if (!r.ok) throw new Error(await r.text());
      const j = await r.json();
      const parts = [`${j.imported} saved`];
      if (j.duplicates > 0) parts.push(`${j.duplicates} already saved`);
      if (j.skipped > 0) parts.push(`${j.skipped} skipped`);
      setUploadMsg(parts.join(". ") + ".");
    } catch {
      setUploadMsg("Could not import that file. Use a browser bookmark export (.html).");
    }
  }

  async function summarize() {
    const query = q.trim();
    if (!query || synPhase === "loading") return;
    setSynPhase("loading");
    setSynSummary(null);
    setSynNote("");
    setSynEv([]);
    setSynClaims([]);
    setLitEv(null);
    setOpenClaim(null);
    try {
      const r = await fetch(`${API}/v1/synthesize`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ query }),
      });
      if (!r.ok) throw new Error(await r.text());
      const j = await r.json();
      setSynSummary(j.summary ?? null);
      setSynNote(j.note ?? "");
      setSynEv(j.evidence ?? []);
      setSynClaims(j.claims ?? []);
      setSynId(j.id ?? "");
      setSynPhase("done");
    } catch {
      setSynNote("The summary failed. Try again.");
      setSynPhase("done");
    }
  }

  function pickFilterValue(i: number) {
    setShowSettings(false);
    setShowCleanup(false);
    setShowReview(false);
    setShowRecent(false);
    setShowArchive(false);
    if (i === 0) {
      setFilter(null);
      return;
    }
    const key = Object.keys(FILTERS)[i - 1];
    setFilter((prev) => (prev === key ? null : key));
  }

  async function openCleanup() {
    setShowSettings(false);
    setShowReview(false);
    setShowRecent(false);
    setShowArchive(false);
    setFilter(null);
    setShowCleanup(true);
    setLastMerge(null);
    setDupeLoading(true);
    try {
      const r = await fetch(`${API}/v1/maintenance/duplicates`);
      const j = await r.json();
      setDupeGroups(j.groups ?? []);
    } catch {
      setDupeGroups([]);
    }
    setDupeLoading(false);
  }

  async function openReview() {
    setShowSettings(false);
    setShowCleanup(false);
    setShowRecent(false);
    setShowArchive(false);
    setFilter(null);
    setShowReview(true);
    setReviewLoading(true);
    try {
      const r = await fetch(`${API}/v1/review`);
      const j = await r.json();
      setReview(j);
    } catch {
      setReview(null);
    }
    setReviewLoading(false);
  }

  async function deleteDupe(id: string) {
    try {
      const r = await fetch(`${API}/v1/items/${id}`, { method: "DELETE" });
      if (!r.ok) return;
    } catch {
      return;
    }
    setDupeGroups((prev) =>
      prev
        .map((g) => ({ ...g, items: g.items.filter((it) => it.id !== id) }))
        .filter((g) => g.items.length > 1),
    );
  }

  const [lastMerge, setLastMerge] = useState<{ kept: string; merged: string[] } | null>(null);

  async function mergeGroup(ids: string[]) {
    setLastMerge(null);
    try {
      const r = await fetch(`${API}/v1/maintenance/merge`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ ids }),
      });
      if (!r.ok) return;
      const j = await r.json();
      setLastMerge({ kept: j.kept, merged: j.merged ?? [] });
    } catch {
      return;
    }
    openCleanupRefresh();
  }

  async function openCleanupRefresh() {
    try {
      const r = await fetch(`${API}/v1/maintenance/duplicates`);
      const j = await r.json();
      setDupeGroups(j.groups ?? []);
    } catch {
      /* panel keeps its last state */
    }
  }

  async function undoMerge() {
    if (!lastMerge) return;
    try {
      await fetch(`${API}/v1/maintenance/unmerge`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ ids: lastMerge.merged }),
      });
    } catch {
      return;
    }
    setLastMerge(null);
    openCleanupRefresh();
  }

  async function ignoreGroup(ids: string[]) {
    for (let i = 0; i < ids.length; i++) {
      for (let j = i + 1; j < ids.length; j++) {
        try {
          await fetch(`${API}/v1/maintenance/ignore`, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ a_id: ids[i], b_id: ids[j] }),
          });
        } catch {
          return;
        }
      }
    }
    openCleanupRefresh();
  }

  async function quickSave() {
    const t = capText.trim();
    if (!t) return;
    setCapMsg("Saving...");
    const isUrl = /^https?:\/\//i.test(t);
    const body = isUrl
      ? { source_type: "url", raw_ref: t.slice(0, 7000), title: t.slice(0, 500), original: { captured_from: "web" } }
      : {
          source_type: "text",
          raw_ref: t,
          title: t.split("\n")[0].slice(0, 80),
          original: { captured_from: "web" },
        };
    try {
      const r = await fetch(`${API}/v1/items`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      if (!r.ok) throw new Error(await r.text());
      setCapText("");
      setCapMsg("Saved.");
    } catch {
      setCapMsg("Could not save. Try again.");
    }
  }

  async function refreshViews() {
    try {
      const r = await fetch(`${API}/v1/queries`);
      const j = await r.json();
      setViews(j.views ?? []);
    } catch {
      setViews([]);
    }
  }

  async function saveView() {
    const query = (lastQuery || q).trim();
    if (!query) return;
    try {
      await fetch(`${API}/v1/queries`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name: query, query_text: query }),
      });
    } catch {
      return;
    }
    refreshViews();
  }

  async function openView(v: { id: string; query_text: string }) {
    setQ(v.query_text);
    setShowCleanup(false);
    setShowSettings(false);
    setShowReview(false);
    setShowRecent(false);
    setShowArchive(false);
    setFilter(null);
    await search(v.query_text);
    try {
      await fetch(`${API}/v1/queries/${v.id}/open`, { method: "POST" });
    } catch {
      /* counts refresh next visit */
    }
    refreshViews();
  }

  async function deleteView(id: string) {
    try {
      await fetch(`${API}/v1/queries/${id}`, { method: "DELETE" });
    } catch {
      return;
    }
    refreshViews();
  }

  async function clearScope() {
    const cleaned = q
      .replace(/\s*(in:\S+|topic:\S+|view:"[^"]+"|view:\S+)\s*/gi, " ")
      .replace(/\s+/g, " ")
      .trim();
    setQ(cleaned);
    setScopeEcho(null);
    if (cleaned) await search(cleaned);
  }

  useEffect(() => {
    refreshViews();
  }, []);

  useEffect(() => {
    loadResurf(true);
  }, []);

  const [relOpen, setRelOpen] = useState<string | null>(null);
  const [relaxed, setRelaxed] = useState(false);
  const [relBusy, setRelBusy] = useState(false);
  const [relMap, setRelMap] = useState<Record<string, { title: string; domain: string; source_type: string; reasons: string[] }[]>>({});
  const [scopeEcho, setScopeEcho] = useState<{ types: string[] | null; topic: string; view: string } | null>(null);

const SCOPE_NAMES: Record<string, string> = {
  text: "notes",
  note: "notes",
  url: "articles",
  pdf: "articles",
  file: "code",
  image: "media",
  bookmark: "saved",
  x: "x",
};

function scopeChips(scope: { types: string[] | null; topic: string; view: string }): string[] {
  const out: string[] = [];
  const names = [...new Set((scope.types ?? []).map((t) => SCOPE_NAMES[t] ?? t))];
  if (names.length > 0) out.push(`only showing ${names.join(", ")}`);
  if (scope.topic) out.push(`topic ${scope.topic}`);
  if (scope.view) out.push(`view ${scope.view}`);
  return out;
}

  async function toggleRelated(id: string) {
    if (relOpen === id) {
      setRelOpen(null);
      return;
    }
    setRelOpen(id);
    if (relMap[id]) return;
    setRelBusy(true);
    try {
      const r = await fetch(`${API}/v1/items/${id}/related`);
      const j = await r.json();
      setRelMap((m) => ({ ...m, [id]: (j.related ?? []).slice(0, 3) }));
    } catch {
      setRelMap((m) => ({ ...m, [id]: [] }));
    }
    setRelBusy(false);
  }

  // The quiet archive: hero mode when nothing is happening yet,
  // and the results sorted the way memory actually works, by time.
  const hero = phase === "idle" && !trimmed && !filter && !showCapture && !showCleanup && !showSettings && !showReview && !showRecent && !showArchive;

  // Motion pass: springy panel changes, silenced for reduced motion.
  const reduced = useReducedMotion() ?? false;
  const panelT = reduced ? { duration: 0 } : { type: "spring" as const, stiffness: 420, damping: 34 };
  const panelExit = reduced ? { opacity: 0 } : { opacity: 0, y: -8, transition: { duration: 0.14, ease: "easeOut" as const } };
  const groups = (["today", "yesterday", "week", "month", "earlier"] as const)
    .map((key) => {
      const items = visible.filter((h) => timeBucket(h.captured_at, now).key === key);
      const label = timeBucket(items[0]?.captured_at ?? "", now).label;
      return { key, label, items };
    })
    .filter((g) => g.items.length > 0);

  return (
    <div className="flex min-h-screen flex-col md:flex-row">
      {/* mobile top bar, brand, health, theme, settings */}
      <header className="glass sticky top-0 z-30 flex h-12 shrink-0 items-center justify-between border-b border-line px-3 pt-[env(safe-area-inset-top)] md:hidden">
        <div className="flex items-center gap-2 text-deep">
          <span className="grid h-8 w-8 place-items-center rounded-[10px] bg-accent-soft text-accent ring-1 ring-inset ring-accent/15">
            <Mark size={18} />
          </span>
          <span className="font-brand text-[15.5px] font-semibold tracking-[-0.01em]">helmme</span>
        </div>
        <div className="flex items-center gap-0.5">
          <i
            aria-hidden="true"
            title={`api ${health}`}
            className={`mr-1.5 h-[7px] w-[7px] rounded-full ${
              health === "up" ? "bg-good" : health === "down" ? "bg-bad" : "bg-line-strong"
            }`}
          />
          <button
            type="button"
            onClick={toggleTheme}
            className="grid h-10 w-10 place-items-center rounded-xl text-muted transition-colors hover:bg-hover hover:text-deep active:scale-95"
            aria-label="Switch theme"
          >
            <HugeiconsIcon icon={night ? Sun03Icon : Moon02Icon} size={18} />
          </button>
          <button
            type="button"
            onClick={() => {
              setShowSettings((s) => !s);
              setShowCleanup(false);
              setShowReview(false);
            }}
            className={`grid h-10 w-10 place-items-center rounded-xl transition-colors active:scale-95 ${
              showSettings ? "bg-accent-soft text-accent-deep" : "text-muted hover:bg-hover hover:text-deep"
            }`}
            aria-label="Settings"
          >
            <HugeiconsIcon icon={Settings02Icon} size={18} />
          </button>
        </div>
      </header>

      <div className="flex min-h-0 min-w-0 flex-1 flex-row">
      <aside className="glass sticky top-0 z-10 hidden h-screen shrink-0 flex-col border-r border-line px-2 pb-4 pt-4 md:flex md:w-[240px] md:px-3">
        <div className="flex items-center gap-2.5 px-1 pt-0.5 text-deep">
          <span className="animate-breathe grid h-9 w-9 shrink-0 place-items-center rounded-[11px] bg-accent-soft text-accent ring-1 ring-inset ring-accent/15">
            <Mark size={21} />
          </span>
          <span className="hidden font-brand text-[17.5px] font-semibold leading-none tracking-[-0.01em] md:inline">
            helmme
          </span>
        </div>

        <nav className="mt-4 flex min-h-0 flex-1 flex-col gap-0.5 overflow-y-auto pb-1" aria-label="Sections">
          {/* Constitution §3: Search · Recent · Capture · Archive · Settings. */}
          <button type="button" className={railCls} onClick={focusSearch}>
            <HugeiconsIcon icon={Search01Icon} size={17} className="shrink-0" />
            <span className="hidden md:inline">Search</span>
          </button>
          <button
            type="button"
            className={`${railCls}${showRecent ? railActive : ""}`}
            onClick={openRecent}
          >
            <HugeiconsIcon icon={HistoryIcon} size={17} className="shrink-0" />
            <span className="hidden md:inline">Recent</span>
          </button>
          <button
            type="button"
            className={`${railCls}${showArchive ? railActive : ""}`}
            onClick={() => openArchive("all")}
          >
            <HugeiconsIcon icon={Bookmark01Icon} size={17} className="shrink-0" />
            <span className="hidden md:inline">Archive</span>
          </button>
          <input
            ref={fileRef}
            type="file"
            accept=".pdf,.png,.jpg,.jpeg,.gif,.webp,.md,.markdown,.txt,.docx"
            className="hidden"
            onChange={(e) => {
              const f = e.target.files?.[0];
              if (f) uploadFile(f);
              e.target.value = "";
            }}
          />
          <input
            ref={importRef}
            type="file"
            accept=".html,.htm"
            className="hidden"
            onChange={(e) => {
              const f = e.target.files?.[0];
              if (f) importBookmarks(f);
              e.target.value = "";
            }}
          />
          <p className="mb-0 mt-4 hidden px-2.5 font-mono text-[10px] font-medium uppercase tracking-[0.16em] text-muted/70 md:block">
            Capture
          </p>
          <button type="button" className={railCls} onClick={() => fileRef.current?.click()}>
            <HugeiconsIcon icon={File01Icon} size={17} className="shrink-0" />
            <span className="hidden md:inline">Save a file</span>
          </button>
          <button type="button" className={railCls} onClick={() => importRef.current?.click()}>
            <HugeiconsIcon icon={ImportIcon} size={17} className="shrink-0" />
            <span className="hidden md:inline">Import bookmarks</span>
          </button>
          <button
            type="button"
            className={`${railCls}${showCapture ? railActive : ""}`}
            onClick={() => {
              setShowCapture((s) => !s);
              setShowSettings(false);
              setShowCleanup(false);
              setShowReview(false);
              setShowRecent(false);
              setShowArchive(false);
            }}
          >
            <HugeiconsIcon icon={PlusSignIcon} size={17} className="shrink-0" />
            <span className="hidden md:inline">Quick save</span>
          </button>
          <button
            type="button"
            className={railCls}
            onClick={() => {
              setShowSettings(true);
              setShowCleanup(false);
              setShowReview(false);
              setShowRecent(false);
              setShowArchive(false);
            }}
          >
            <HugeiconsIcon icon={XLogoGlyph} size={15} className="shrink-0" />
            <span className="hidden md:inline">Connect X</span>
          </button>
          <button
            type="button"
            className={`${railCls}${showCleanup ? railActive : ""}`}
            onClick={openCleanup}
          >
            <HugeiconsIcon icon={Layers01Icon} size={17} className="shrink-0" />
            <span className="hidden md:inline">Cleanup</span>
          </button>
          <button
            type="button"
            className={`${railCls}${showReview ? railActive : ""}`}
            onClick={openReview}
          >
            <HugeiconsIcon icon={ClipboardCheckIcon} size={17} className="shrink-0" />
            <span className="hidden md:inline">Review</span>
          </button>
          {views.length > 0 && (
            <p className="mb-1 mt-4 hidden px-2.5 font-mono text-[10px] font-medium uppercase tracking-[0.16em] text-muted/70 md:block">
              Saved searches
            </p>
          )}
          {views.map((v) => (
            <div key={v.id} className="group flex w-full items-center gap-1.5 rounded-[12px] px-2.5 py-[7px] text-left text-[13px] font-medium text-muted transition-all duration-150 hover:bg-hover hover:text-deep">
              <button type="button" onClick={() => openView(v)} className="min-w-0 flex-1 truncate text-left">
                <span className="hidden md:inline">{v.name}</span>
              </button>
              {v.new > 0 && (
                <span className="hidden shrink-0 rounded-full bg-accent-deep px-1.5 py-0.5 font-mono text-[9.5px] font-semibold text-white md:inline">
                  +{v.new}
                </span>
              )}
              <button
                type="button"
                onClick={() => deleteView(v.id)}
                className="grid place-items-center rounded-full text-muted opacity-0 transition-opacity duration-150 hover:bg-hover hover:text-deep group-hover:opacity-100 md:inline-grid"
                aria-label={`Remove ${v.name}`}
              >
                <HugeiconsIcon icon={Cancel01Icon} size={11} strokeWidth={2} />
              </button>
            </div>
          ))}
        </nav>

        <div className="mt-2 flex flex-col gap-1.5 border-t border-line pt-3">
          <span className="hidden items-center gap-2 px-2.5 md:flex">
            <i
              aria-hidden="true"
              className={`h-[7px] w-[7px] shrink-0 rounded-full ${
                health === "up" ? "bg-good" : health === "down" ? "bg-bad" : "bg-line-strong"
              }`}
            />
            <span className={stampCls}>api {health === "unknown" ? "checking" : health}</span>
          </span>
          <button
            type="button"
            className={railCls}
            onClick={toggleTheme}
            aria-label={night ? "Switch to light theme" : "Switch to dark theme"}
          >
            <HugeiconsIcon icon={night ? Sun03Icon : Moon02Icon} size={17} className="shrink-0" />
            <span className="hidden md:inline">{night === null ? "Theme" : night ? "Day paper" : "Night"}</span>
          </button>
          <button
            type="button"
            className={`${railCls}${showSettings ? railActive : ""}`}
            onClick={() => {
              setShowSettings((s) => !s);
              setShowCleanup(false);
              setShowReview(false);
            }}
          >
            <HugeiconsIcon icon={Settings02Icon} size={17} className="shrink-0" />
            <span className="hidden md:inline">Settings</span>
          </button>
        </div>
      </aside>

      <main className="flex min-w-0 flex-1 flex-col items-center px-4 pb-[calc(76px+env(safe-area-inset-bottom))] md:px-9 md:pb-16">
        <form
          className={`${
            hero
              ? "mt-[clamp(56px,15vh,140px)] h-[60px] rounded-[18px] border-line bg-panel pl-4 pr-3 shadow-lift"
              : "glass z-20 mt-1 h-[46px] rounded-2xl border-line pl-3.5 pr-2.5 shadow-card md:sticky md:top-2.5"
          } flex w-full max-w-[640px] items-center gap-3 border transition-all duration-[420ms] ease-[cubic-bezier(0.32,0.72,0,1)] focus-within:border-accent/45`}
          onSubmit={(e) => {
            e.preventDefault();
            search();
          }}
        >
          <span className={`grid place-items-center text-accent ${hero ? "" : "shrink-0"}`} aria-hidden="true">
            <HugeiconsIcon icon={Search01Icon} size={hero ? 20 : 17} />
          </span>
          <input
            ref={inputRef}
            value={q}
            onChange={(e) => setQ(e.target.value)}
            placeholder={hero ? "What are you looking for?" : "Search your memory..."}
            aria-label="Search your memory"
            autoComplete="off"
            spellCheck={false}
            className={`m-0 min-w-0 flex-1 border-0 bg-transparent p-0 font-medium text-deep placeholder:font-normal placeholder:text-muted/55 focus:outline-none ${
              hero ? "text-[17px]" : "text-[16px] md:text-[14.5px]"
            }`}
          />
          <kbd className={`${kbdCls} hidden sm:block`}>⌘K</kbd>
        </form>

        {scopeEcho && scopeChips(scopeEcho).length > 0 && (
          <div className="mt-2.5 flex w-full max-w-[640px] flex-wrap items-center gap-2 px-2.5">
            {scopeChips(scopeEcho).map((chip) => (
              <span
                key={chip}
                className="inline-flex items-center gap-1.5 rounded-full border border-line bg-panel px-2.5 py-1 font-mono text-[10.5px] uppercase tracking-[0.06em] text-deep"
              >
                {chip}
                <button type="button" onClick={clearScope} className="grid place-items-center text-muted transition-colors hover:text-accent-deep" aria-label="Clear search scope">
                  <HugeiconsIcon icon={Cancel01Icon} size={11} strokeWidth={2} />
                </button>
              </span>
            ))}
          </div>
        )}

        <div className="mt-3 w-full max-w-[640px] overflow-x-auto px-2 [scrollbar-width:none] [&::-webkit-scrollbar]:hidden">
          <div className="sm:hidden">
            <GooeyNav
              size="xs"
              activeColor="#CC3D00"
              items={["All", ...Object.values(FILTERS).map((f) => f.label)]}
              value={filter ? Object.keys(FILTERS).indexOf(filter) + 1 : 0}
              onChange={pickFilterValue}
            />
          </div>
          <div className="hidden sm:block">
            <GooeyNav
              size="sm"
              activeColor="#CC3D00"
              items={["All", ...Object.values(FILTERS).map((f) => f.label)]}
              value={filter ? Object.keys(FILTERS).indexOf(filter) + 1 : 0}
              onChange={pickFilterValue}
            />
          </div>
        </div>

        {showCapture && (
          <div className="mb-2 mt-3 flex w-full max-w-[640px] gap-2 px-2.5 md:hidden">
            <button
              type="button"
              onClick={() => fileRef.current?.click()}
              className="flex-1 rounded-2xl border border-line bg-panel px-3 py-2 text-xs font-medium text-muted shadow-card transition-colors hover:text-deep active:scale-[0.97]"
            >
              Save a file
            </button>
            <button
              type="button"
              onClick={() => importRef.current?.click()}
              className="flex-1 rounded-2xl border border-line bg-panel px-3 py-2 text-xs font-medium text-muted shadow-card transition-colors hover:text-deep active:scale-[0.97]"
            >
              Import bookmarks
            </button>
          </div>
        )}
        {showCapture && (
          <form
            className="animate-rise flex w-full max-w-[640px] gap-2 px-2.5"
            onSubmit={(e) => {
              e.preventDefault();
              quickSave();
            }}
          >
            <input
              value={capText}
              onChange={(e) => setCapText(e.target.value)}
              placeholder="Paste a link or type a note..."
              aria-label="Save a link or note"
              autoComplete="off"
              spellCheck={false}
              maxLength={7000}
              inputMode="text"
              enterKeyHint="done"
              className="m-0 min-w-0 flex-1 rounded-2xl border border-line bg-panel px-3.5 py-2.5 text-[16px] text-deep shadow-card placeholder:text-[13px] placeholder:text-muted/55 focus:border-accent/45 focus:outline-none md:text-[13px]"
            />
            <button
              type="submit"
              className="shrink-0 rounded-2xl bg-accent-deep px-4 py-2 text-xs font-semibold text-white shadow-card transition-all duration-150 hover:bg-accent active:scale-[0.97]"
            >
              Save
            </button>
          </form>
        )}
        {capMsg && (
          <p className="mt-2.5 w-full max-w-[640px] px-2.5 font-mono text-[10.5px] uppercase tracking-[0.08em] text-muted">{capMsg}</p>
        )}

        {(route === "SYNTHESIS" || jevNote) && (
          <div className="mt-3 w-full max-w-[640px] px-2.5">
            {route === "SYNTHESIS" && synPhase !== "done" && synPhase !== "loading" && (
              <button
                type="button"
                onClick={summarize}
                className="inline-flex items-center gap-2 rounded-full bg-grad px-4 py-2 text-xs font-semibold text-white shadow-card transition-all duration-150 hover:opacity-90 active:scale-[0.97]"
              >
                <HugeiconsIcon icon={SparklesIcon} size={14} className="shrink-0" />
                Summarize
              </button>
            )}
            {route === "SYNTHESIS" && synPhase === "loading" && (
              <MatrixOrb state="thinking" size={96} dots={9} color="#F75001" labels={{ thinking: "Reading your archive..." }} />
            )}
            {route === "SYNTHESIS" && synPhase === "done" && (
              <div className="overflow-hidden rounded-2xl border border-line bg-panel shadow-card">
                <div className="h-[3px] w-full bg-grad" aria-hidden="true" />
                <div className="px-4 py-3.5">
                {synSummary ? (
                  synSummary.split("\n\n").map((para, i) => renderProse(para, i))
                ) : (
                  <p className="m-0 text-[13px] text-muted">No summary. The evidence below still stands.</p>
                )}
                {synClaims.length > 0 && (
                  <ul className="m-0 mt-3 list-none space-y-2 border-t border-line pt-3 p-0">
                    {synClaims.map((c, i) => {
                      const linked = (c.sources ?? [])
                        .map((id) => synEv.find((e) => e.id === id))
                        .filter((e) => e !== undefined);
                      const open = openClaim === i;
                      const chip = c.rejected ? "bg-hover text-muted" : (CLAIM_CHIP[c.status] ?? "bg-hover text-muted");
                      return (
                      <li key={i} className={`text-xs leading-relaxed${c.rejected ? " opacity-50" : ""}`}>
                        <span className="flex flex-wrap items-center gap-x-1.5 gap-y-1">
                        {linked.length > 0 ? (
                          <button
                            type="button"
                            onClick={() => setOpenClaim(open ? null : i)}
                            className="text-left text-deep underline decoration-accent/40 decoration-dotted underline-offset-2 transition-colors hover:text-accent-deep"
                            aria-expanded={open}
                          >
                            {c.text}
                          </button>
                        ) : (
                          <span className="text-deep">{c.text}</span>
                        )}
                        <span
                          className={`shrink-0 rounded-full px-1.5 py-px text-[10px] font-semibold uppercase tracking-[0.04em] ${chip}`}
                        >
                          {c.rejected ? "you said not true" : (CLAIM_WORDS[c.status] ?? c.status)}
                        </span>
                        {!c.rejected && (
                          <button
                            type="button"
                            onClick={() => dismissClaim(i)}
                            className="text-[10.5px] text-muted underline decoration-dotted underline-offset-2 transition-colors hover:text-bad"
                          >
                            not true
                          </button>
                        )}
                        </span>
                          {open && (
                            <ul className="m-0 mt-1.5 list-none space-y-1.5 rounded-[10px] bg-accent-faint p-2.5 ring-1 ring-inset ring-accent/10">
                              {linked.map((e) => (
                                <li key={e!.id}>
                                  <span className="font-medium text-deep">{e!.title || "(no title)"}</span>
                                  {e!.excerpt && (
                                    <span
                                      className="excerpt block leading-relaxed text-muted"
                                      dangerouslySetInnerHTML={{ __html: e!.excerpt }}
                                    />
                                  )}
                                </li>
                              ))}
                            </ul>
                          )}
                        </li>
                      );
                    })}
                  </ul>
                )}
                {synEv.length > 0 && (
                  <ol className="m-0 mt-3 list-none space-y-0.5 border-t border-line pt-3 p-0">
                    {synEv.map((e, i) => (
                      <li
                        key={e.id}
                        id={`ev-${i + 1}`}
                        className={`rounded-[8px] px-1.5 py-1 text-xs text-muted transition-colors ${litEv === e.id ? "bg-accent-soft" : ""}`}
                      >
                        <span className="font-semibold text-accent-deep">[{i + 1}]</span>{" "}
                        <span className="text-deep">{e.title || "(no title)"}</span>
                      </li>
                    ))}
                  </ol>
                )}
                {synNote && <p className="m-0 mt-2.5 text-[11.5px] text-muted">{synNote}</p>}
                </div>
              </div>
            )}
            {route !== "SYNTHESIS" && jevNote && (
              <p className="m-0 text-[11.5px] text-muted">{jevNote}</p>
            )}
          </div>
        )}

        {uploadMsg && (
          <p className="mt-2.5 w-full max-w-[640px] px-2.5 font-mono text-[10.5px] uppercase tracking-[0.08em] text-muted">{uploadMsg}</p>
        )}

        <AnimatePresence mode="wait" initial={false}>
        {showReview ? (
          <motion.section
            key="review"
            initial={{ opacity: 0, y: 12 }}
            animate={{ opacity: 1, y: 0 }}
            exit={panelExit}
            transition={panelT}
            className="mt-8 w-full max-w-[640px]"
          >
            <h2 className="px-1 pb-1 text-[22px] font-semibold tracking-[-0.02em] text-deep">This week</h2>
            <p className="mb-4 px-1 font-mono text-[10px] font-medium uppercase tracking-[0.16em] text-muted/80">
              your archive, lately
            </p>
            {reviewLoading ? (
              <div className="grid gap-2 sm:grid-cols-3">
                {[0, 1, 2].map((i) => (
                  <div key={i} className="skeleton h-[84px] animate-shimmer rounded-2xl" />
                ))}
              </div>
            ) : !review ? (
              <div className="rounded-2xl border border-line bg-panel px-4 py-3.5 text-[13px] text-muted shadow-card">
                The review did not load. Try again from the sidebar.
              </div>
            ) : (
              <>
                <div className="grid gap-2 sm:grid-cols-3">
                  <div className="rounded-2xl border border-line bg-panel px-4 py-3.5 shadow-card">
                    <p className={`m-0 ${stampCls}`}>Saved this week</p>
                    <p className="m-0 mt-1.5 text-[26px] font-semibold leading-none tracking-[-0.02em] text-deep">
                      <AnimatedCounter value={review.saved_this_week ?? 0} />
                    </p>
                  </div>
                  <div className="rounded-2xl border border-line bg-panel px-4 py-3.5 shadow-card">
                    <p className={`m-0 ${stampCls}`}>Duplicates waiting</p>
                    <p className="m-0 mt-1.5 text-[26px] font-semibold leading-none tracking-[-0.02em] text-deep">
                      <AnimatedCounter value={review.duplicate_groups ?? 0} />
                    </p>
                    {(review.duplicate_groups ?? 0) > 0 && (
                      <button
                        type="button"
                        onClick={openCleanup}
                        className="mt-1.5 text-[11.5px] font-medium text-accent-deep transition-colors hover:text-accent"
                      >
                        open cleanup →
                      </button>
                    )}
                  </div>
                  <div className="rounded-2xl border border-line bg-panel px-4 py-3.5 shadow-card">
                    <p className={`m-0 ${stampCls}`}>Unopened saves</p>
                    <p className="m-0 mt-1.5 text-[26px] font-semibold leading-none tracking-[-0.02em] text-deep">
                      <AnimatedCounter value={review.unopened_bookmarks ?? 0} />
                    </p>
                  </div>
                </div>

                {(review.top_topics ?? []).length + (review.top_entities ?? []).length > 0 && (
                  <div className="mt-3 rounded-2xl border border-line bg-panel px-4 py-3.5 shadow-card">
                    <p className={`m-0 ${stampCls}`}>Mostly about</p>
                    <p className="m-0 mt-2 flex flex-wrap items-center gap-1.5">
                      {(review.top_topics ?? []).slice(0, 6).map((t) => (
                        <span
                          key={t.name}
                          className="inline-flex items-center gap-1.5 rounded-full bg-accent-faint px-2.5 py-1 font-mono text-[10.5px] font-medium uppercase tracking-[0.06em] text-accent-deep ring-1 ring-inset ring-accent/10"
                        >
                          {t.name}
                          <span className="text-muted tabular-nums">{t.count}</span>
                        </span>
                      ))}
                      {(review.top_entities ?? []).slice(0, 5).map((e) => (
                        <span
                          key={e.name}
                          className="inline-flex items-center gap-1.5 rounded-full bg-hover px-2.5 py-1 font-mono text-[10.5px] font-medium uppercase tracking-[0.06em] text-deep"
                        >
                          {e.name}
                          <span className="text-muted tabular-nums">{e.count}</span>
                        </span>
                      ))}
                    </p>
                  </div>
                )}

                {(review.resurfaced ?? []).length > 0 && (
                  <div className="mt-3">
                    <p className={`mb-2 px-1 ${stampCls}`}>Worth revisiting</p>
                    <ul className="m-0 grid list-none gap-2 p-0 sm:grid-cols-2">
                      {(review.resurfaced ?? []).map((r) => (
                        <li key={r.id}>
                          <Link
                            href={`/item/${r.id}`}
                            className="group flex items-start gap-3 rounded-2xl border border-line bg-panel px-3.5 py-3 shadow-card transition-all duration-200 hover:-translate-y-0.5 hover:border-accent/30 hover:shadow-lift"
                          >
                            <span className="grid h-8 w-8 shrink-0 place-items-center rounded-[10px] bg-accent-soft text-accent">
                              <HugeiconsIcon icon={HistoryIcon} size={15} />
                            </span>
                            <span className="min-w-0 flex-1">
                              <span className="block text-[13px] font-semibold leading-snug text-deep transition-colors group-hover:text-accent-deep [overflow-wrap:anywhere]">
                                {displayTitle({ title: r.title, domain: null })}
                              </span>
                              <span className="mt-0.5 block text-[11px] leading-relaxed text-muted">{r.reason}</span>
                            </span>
                          </Link>
                        </li>
                      ))}
                    </ul>
                  </div>
                )}
              </>
            )}
          </motion.section>
        ) : showCleanup ? (
          <motion.section
            key="cleanup"
            initial={{ opacity: 0, y: 12 }}
            animate={{ opacity: 1, y: 0 }}
            exit={panelExit}
            transition={panelT}
            className="mt-8 w-full max-w-[640px]"
          >
            <h2 className="px-1 pb-1 text-[22px] font-semibold tracking-[-0.02em] text-deep">Cleanup</h2>
            <p className="mb-4 px-1 font-mono text-[10px] font-medium uppercase tracking-[0.16em] text-muted/80">
              same thing, saved twice
            </p>
            {lastMerge && (
              <p className="m-0 mb-3 flex flex-wrap items-center gap-2 rounded-xl bg-good-soft px-3.5 py-2.5 text-[12px] text-good">
                Merged {lastMerge.merged.length + 1} into one.{" "}
                <button
                  type="button"
                  onClick={undoMerge}
                  className="font-semibold underline decoration-dotted underline-offset-2 hover:opacity-80"
                >
                  Undo
                </button>
              </p>
            )}
            {dupeLoading ? (
              <div className="space-y-1.5">
                <div className="h-[52px] rounded-2xl border border-line bg-panel skeleton animate-shimmer" />
                <div className="h-[52px] rounded-2xl border border-line bg-panel skeleton animate-shimmer" />
              </div>
            ) : dupeGroups.length === 0 ? (
              <div className="rounded-2xl border border-line bg-panel px-4 py-3.5 text-[13px] text-muted shadow-card">
                Nothing to clean. The archive looks tidy.
              </div>
            ) : (
              dupeGroups.map((g, gi) => (
                <div key={gi} className="mb-4 overflow-hidden rounded-2xl border border-line bg-panel shadow-card">
                  <p className="m-0 flex items-center justify-between gap-3 border-b border-line bg-accent-faint px-4 py-2.5 text-[11.5px] text-muted">
                    <span>{DUP_REASONS[g.reason] ?? g.reason}</span>
                    <span className="flex shrink-0 gap-2">
                      <button
                        type="button"
                        onClick={() => mergeGroup(g.items.map((it) => it.id))}
                        className="inline-flex items-center gap-1.5 rounded-full bg-panel px-2.5 py-1 text-[11px] font-semibold text-accent-deep shadow-card transition-colors duration-150 hover:bg-accent-soft"
                      >
                        <HugeiconsIcon icon={MergeIcon} size={12} className="shrink-0" />
                        Merge {g.items.length} into oldest
                      </button>
                      <button
                        type="button"
                        onClick={() => ignoreGroup(g.items.map((it) => it.id))}
                        className="rounded-full bg-panel px-2.5 py-1 text-[11px] font-semibold text-muted shadow-card transition-colors duration-150 hover:text-deep"
                      >
                        Not duplicates
                      </button>
                    </span>
                  </p>
                  <ul className="m-0 list-none p-0">
                    {g.items.map((it) => (
                      <li
                        key={it.id}
                        className="flex items-center justify-between gap-3 border-b border-line px-4 py-2.5 transition-colors duration-150 last:border-b-0 hover:bg-hover"
                      >
                        <span className="min-w-0 flex-1">
                          <span className="block truncate text-[13px] text-deep">{displayTitle(it)}</span>
                          {it.domain && (
                            <span className={`block ${stampCls} !tracking-[0.08em]`}>{it.domain}</span>
                          )}
                        </span>
                        <DeleteButton
                          onConfirm={() => deleteDupe(it.id)}
                          aria-label={`Delete ${displayTitle(it)}`}
                        />
                      </li>
                    ))}
                  </ul>
                </div>
              ))
            )}
          </motion.section>
        ) : showSettings ? (
          <motion.section
            key="settings"
            initial={{ opacity: 0, y: 12 }}
            animate={{ opacity: 1, y: 0 }}
            exit={panelExit}
            transition={panelT}
            className="mt-8 w-full max-w-[640px]"
          >
            <h2 className="px-1 pb-1 text-[22px] font-semibold tracking-[-0.02em] text-deep">Settings</h2>
            <p className="mb-4 px-1 font-mono text-[10px] font-medium uppercase tracking-[0.16em] text-muted/80">
              account · connections · privacy · memory
            </p>
            {/* Constitution §11-§14: X is a first-class source. Handle only, never password. */}
            <div className="mb-3 rounded-2xl border border-line bg-panel px-4 py-3.5 shadow-card">
              <div className="flex items-center gap-2.5">
                <span
                  className="grid h-7 w-7 shrink-0 place-items-center rounded-[8px]"
                  style={{ background: "var(--tile-x-bg)", color: "var(--tile-x-ink)" }}
                  aria-hidden="true"
                >
                  <HugeiconsIcon icon={XLogoGlyph} size={13} />
                </span>
                <p className={`m-0 ${stampCls}`}>Connections · X</p>
              </div>
              <p className="m-0 mt-1.5 text-[12.5px] leading-relaxed text-muted">
                Bring your X bookmarks into your memory. They become normal searchable
                memory, not a separate silo.
              </p>
              <form
                className="mt-2.5 flex gap-2"
                onSubmit={(e) => {
                  e.preventDefault();
                  connectX();
                }}
              >
                <input
                  value={xHandle}
                  onChange={(e) => setXHandle(e.target.value)}
                  placeholder="@handle (optional)"
                  aria-label="X handle"
                  autoComplete="off"
                  spellCheck={false}
                  maxLength={120}
                  className="m-0 min-w-0 flex-1 rounded-xl border border-line bg-panel px-3 py-2 text-[13px] text-deep placeholder:text-muted/55 focus:border-accent/45 focus:outline-none"
                />
                <button
                  type="submit"
                  className="inline-flex shrink-0 items-center gap-1.5 rounded-xl bg-accent-deep px-3.5 py-2 text-xs font-semibold text-white transition-all duration-150 hover:bg-accent active:scale-[0.97]"
                >
                  <HugeiconsIcon icon={XLogoGlyph} size={12} aria-hidden="true" />
                  Connect X
                </button>
              </form>
              {xMsg && <p className="m-0 mt-2 text-[11.5px] text-muted">{xMsg}</p>}
            </div>
            {/* Constitution §38: Quiet / Balanced / Proactive are behavior controls. */}
            <div className="mb-3 rounded-2xl border border-line bg-panel px-4 py-3.5 shadow-card">
              <p className={`m-0 ${stampCls}`}>Resurfacing</p>
              <p className="m-0 mt-1.5 text-[12.5px] leading-relaxed text-muted">
                Quiet by default. The archive returns only when it has a reason.
              </p>
              <div className="mt-2.5 flex gap-1.5">
                {(["quiet", "balanced", "proactive"] as const).map((m) => (
                  <button
                    key={m}
                    type="button"
                    onClick={() => saveResurfaceMode(m)}
                    aria-pressed={resurfaceMode === m}
                    className={`rounded-full px-3 py-1.5 font-mono text-[10.5px] font-medium uppercase tracking-[0.06em] transition-all duration-150 active:scale-[0.97] ${
                      resurfaceMode === m ? "bg-accent-deep text-white" : "bg-hover text-muted hover:text-deep"
                    }`}
                  >
                    {m}
                  </button>
                ))}
              </div>
            </div>
            <ul className="m-0 list-none overflow-hidden rounded-2xl border border-line bg-panel shadow-card p-0">
              {(
                [
                  ["status", <>{health === "unknown" ? "starting..." : health === "up" ? "working" : "not working, start the app and try again"}</>],
                  ["search", <>looks through the words in everything you saved.</>],
                  ["tricky questions", <>hard questions get a second look before answering.</>],
                  ["summaries", <>built from your evidence, with sources shown.</>],
                  ["shortcuts", <><kbd className={kbdCls}>/</kbd> focus · <kbd className={kbdCls}>⌘K</kbd> search · <kbd className={kbdCls}>⌘⇧S</kbd> capture · <kbd className={kbdCls}>⌘↵</kbd> synthesize · <kbd className={kbdCls}>esc</kbd> close</>],
                  ["your stuff", <>what you save stays as it is. nothing is rewritten. disconnecting X keeps your memories unless you choose delete.</>],
                ] as [string, React.ReactNode][]
              ).map(([label, value], i, all) => (
                <li
                  key={label}
                  className={`grid grid-cols-[104px_1fr] gap-3.5 px-4 py-3 text-[13px] text-muted ${
                    i < all.length - 1 ? "border-b border-line" : ""
                  }`}
                >
                  <span className={`pt-0.5 ${stampCls} !tracking-[0.1em]`}>
                    {label}
                  </span>
                  <span className="text-xs leading-relaxed">{value}</span>
                </li>
              ))}
            </ul>
          </motion.section>
        ) : showRecent ? (
          <motion.section
            key="recent"
            initial={{ opacity: 0, y: 12 }}
            animate={{ opacity: 1, y: 0 }}
            exit={panelExit}
            transition={panelT}
            className="mt-8 w-full max-w-[640px]"
          >
            <h2 className="px-1 pb-1 text-[22px] font-semibold tracking-[-0.02em] text-deep">Recent</h2>
            <p className="mb-4 px-1 font-mono text-[10px] font-medium uppercase tracking-[0.16em] text-muted/80">
              latest captures, in order
            </p>
            {recentLoading ? (
              <div className="space-y-1.5">
                <div className="h-[52px] rounded-2xl border border-line bg-panel skeleton animate-shimmer" />
                <div className="h-[52px] rounded-2xl border border-line bg-panel skeleton animate-shimmer" />
              </div>
            ) : recentItems.length === 0 ? (
              <div className="rounded-2xl border border-line bg-panel px-4 py-3.5 text-[13px] text-muted shadow-card">
                Nothing yet. Save something, it will appear here.
              </div>
            ) : (
              <ol className="m-0 flex list-none flex-col gap-1.5 p-0">
                {recentItems.map((h) => (
                  <li
                    key={h.id}
                    className="rounded-2xl border border-line bg-panel px-3.5 py-3 shadow-card transition-all duration-200 hover:-translate-y-px hover:border-accent/30"
                  >
                    <h3 className="m-0 text-[14px] font-semibold leading-snug text-deep [overflow-wrap:anywhere]">
                      <Link href={`/item/${h.id}`} className="transition-colors hover:text-accent-deep">
                        {displayTitle(h)}
                      </Link>
                    </h3>
                    <p className="m-0 mt-1 flex flex-wrap items-center gap-x-2 gap-y-1 text-[11.5px] text-muted">
                      <span className="rounded-full bg-hover px-1.5 py-px font-mono text-[9.5px] font-medium uppercase tracking-[0.08em]">
                        {styleOf(h.source_type).label}
                      </span>
                      <span className="font-mono text-[10.5px] tabular-nums">{relTime(h.captured_at, now)}</span>
                    </p>
                  </li>
                ))}
              </ol>
            )}
          </motion.section>
        ) : showArchive ? (
          <motion.section
            key="archive"
            initial={{ opacity: 0, y: 12 }}
            animate={{ opacity: 1, y: 0 }}
            exit={panelExit}
            transition={panelT}
            className="mt-8 w-full max-w-[640px]"
          >
            <h2 className="px-1 pb-1 text-[22px] font-semibold tracking-[-0.02em] text-deep">Archive</h2>
            <p className="mb-4 px-1 font-mono text-[10px] font-medium uppercase tracking-[0.16em] text-muted/80">
              views over one memory, not folders
            </p>
            <div className="mb-3 flex flex-wrap gap-1.5 px-1">
              {(["all", "sources", "topics", "people", "projects", "collections"] as const).map((v) => (
                <button
                  key={v}
                  type="button"
                  onClick={() => openArchive(v)}
                  aria-pressed={archiveView === v}
                  className={`rounded-full px-3 py-1.5 font-mono text-[10.5px] font-medium uppercase tracking-[0.06em] transition-all duration-150 active:scale-[0.97] ${
                    archiveView === v ? "bg-accent-deep text-white" : "bg-hover text-muted hover:text-deep"
                  }`}
                >
                  {v}
                </button>
              ))}
            </div>
            {archiveLoading ? (
              <div className="space-y-1.5">
                <div className="h-[52px] rounded-2xl border border-line bg-panel skeleton animate-shimmer" />
                <div className="h-[52px] rounded-2xl border border-line bg-panel skeleton animate-shimmer" />
              </div>
            ) : archiveData.groups ? (
              <ul className="m-0 list-none space-y-1.5 p-0">
                {archiveData.groups.map((g) => (
                  <li
                    key={(g.id ?? g.name) + g.name}
                    className="flex items-center justify-between gap-3 rounded-2xl border border-line bg-panel px-4 py-2.5 shadow-card"
                  >
                    <span className="min-w-0 flex-1 truncate text-[13px] font-medium text-deep">{g.name}</span>
                    <span className="shrink-0 font-mono text-[10.5px] tabular-nums text-muted">{g.count}</span>
                  </li>
                ))}
                {archiveData.groups.length === 0 && (
                  <li className="rounded-2xl border border-line bg-panel px-4 py-3.5 text-[13px] text-muted shadow-card">
                    Nothing here yet under this view.
                  </li>
                )}
              </ul>
            ) : (archiveData.items ?? []).length > 0 ? (
              <ol className="m-0 flex list-none flex-col gap-1.5 p-0">
                {(archiveData.items ?? []).map((h) => (
                  <li
                    key={h.id}
                    className="rounded-2xl border border-line bg-panel px-3.5 py-3 shadow-card transition-all duration-200 hover:-translate-y-px hover:border-accent/30"
                  >
                    <h3 className="m-0 text-[14px] font-semibold leading-snug text-deep [overflow-wrap:anywhere]">
                      <Link href={`/item/${h.id}`} className="transition-colors hover:text-accent-deep">
                        {displayTitle(h)}
                      </Link>
                    </h3>
                    <p className="m-0 mt-1 text-[11.5px] text-muted">
                      <span className="rounded-full bg-hover px-1.5 py-px font-mono text-[9.5px] font-medium uppercase tracking-[0.08em]">
                        {styleOf(h.source_type).label}
                      </span>
                    </p>
                  </li>
                ))}
              </ol>
            ) : (
              <div className="flex flex-col items-center gap-3 rounded-2xl border border-line bg-panel px-4 py-6 text-center shadow-card">
                <div className="h-[150px] w-[180px]">
                  <Folder color="white" size="sm" />
                </div>
                <p className="m-0 text-[13px] text-muted">Nothing here yet.</p>
                <p className="m-0 max-w-[260px] text-[11.5px] leading-relaxed text-muted/80">
                  Save a link, a note, a file, the archive fills as you live.
                </p>
              </div>
            )}
          </motion.section>
        ) : hero ? (
          <motion.section
            key="hero"
            initial={{ opacity: 0, y: 12 }}
            animate={{ opacity: 1, y: 0 }}
            exit={panelExit}
            transition={panelT}
            className="flex w-full max-w-[640px] flex-col items-center gap-5 text-center"
          >
            <h1 className="animate-rise m-0 mt-[clamp(20px,6vh,64px)] max-w-[520px] text-[clamp(26px,4.4vw,38px)] font-semibold leading-[1.14] tracking-[-0.025em] text-deep" style={{ animationDelay: "60ms" }}>
              Your information is scattered.{" "}
              <span className="text-grad">Save it here.</span>
            </h1>
            <p className="animate-rise m-0 text-[12.5px] leading-relaxed text-muted" style={{ animationDelay: "120ms" }}>
              Save anything. Find what you meant later. We&apos;ll make sense of it.{" "}
              <kbd className={kbdCls}>/</kbd> to focus,{" "}
              <kbd className={kbdCls}>enter</kbd> to search.
            </p>
            {resurf.length > 0 && (
              <div className="animate-rise mt-4 w-full text-left" style={{ animationDelay: "180ms" }}>
                <p className="m-0 mb-2 flex items-baseline justify-between px-1">
                  <span className={stampCls}>Worth revisiting</span>
                  <button
                    type="button"
                    onClick={() => {
                      const next = !hideSeen;
                      setHideSeen(next);
                      loadResurf(next);
                    }}
                    className="text-[11.5px] text-muted underline decoration-dotted underline-offset-2 hover:text-deep"
                    aria-pressed={hideSeen}
                  >
                    {hideSeen ? "Show seen" : "Hide seen"}
                  </button>
                </p>
                <ul className="m-0 grid list-none gap-2 p-0 sm:grid-cols-2">
                  {resurf.map((r, i) => (
                    <li key={r.id} className="animate-rise" style={{ animationDelay: `${220 + i * 50}ms` }}>
                      <Link
                        href={`/item/${r.id}`}
                        className="group flex h-full items-center gap-3 rounded-2xl border border-line bg-panel px-3.5 py-3 shadow-card transition-all duration-200 hover:-translate-y-0.5 hover:border-accent/30 hover:shadow-lift"
                      >
                        <span className="grid h-8 w-8 shrink-0 place-items-center rounded-[10px] bg-accent-soft text-accent">
                          <HugeiconsIcon icon={HistoryIcon} size={15} />
                        </span>
                        <span className="min-w-0 flex-1">
                          <span className="block truncate text-[13px] font-semibold text-deep transition-colors group-hover:text-accent-deep">
                            {displayTitle({ title: r.title, domain: null })}
                          </span>
                          <span className="block truncate text-[11px] text-muted">
                            {r.reason}
                            {r.seen ? " · seen" : ""}
                          </span>
                        </span>
                        <span className="shrink-0 text-muted transition-all duration-150 group-hover:-translate-y-0.5 group-hover:-translate-x-0.5 group-hover:text-accent">
                          <HugeiconsIcon icon={ArrowUpRight01Icon} size={14} />
                        </span>
                      </Link>
                    </li>
                  ))}
                </ul>
              </div>
            )}
          </motion.section>
        ) : (
          <motion.section
            key="results"
            initial={{ opacity: 0, y: 12 }}
            animate={{ opacity: 1, y: 0 }}
            exit={panelExit}
            transition={panelT}
            className="mt-8 w-full max-w-[640px]"
            aria-live="polite"
          >
            <div className="flex items-baseline justify-between gap-3 px-1 pb-2.5">
              <h2 className="m-0 text-[22px] font-semibold tracking-[-0.02em] text-deep">
                {phase === "error"
                  ? "search failed"
                  : filter
                    ? FILTERS[filter].label.toLowerCase()
                    : lastQuery
                      ? `results for “${lastQuery}”`
                      : "results"}
              </h2>
              <span className="shrink-0 rounded-full bg-accent-soft px-2.5 py-0.5 text-right font-mono text-[10.5px] font-medium tabular-nums text-accent-deep [overflow-wrap:anywhere]">
                {phase === "loading"
                  ? "searching…"
                  : phase === "error"
                    ? "search did not work"
                    : phase === "idle"
                      ? trimmed
                        ? "press enter to search"
                        : filter
                          ? "waiting for a query"
                          : ""
                      : filter
                        ? <><AnimatedCounter value={visible.length} /> of <AnimatedCounter value={hits.length} /> things</>
                        : relaxed
                          ? "closest matches"
                          : <AnimatedCounter value={hits.length} suffix={` ${hits.length === 1 ? "thing" : "things"}`} />}
              </span>
            </div>

            {phase === "done" && lastQuery && (
              <div className="px-1 pb-2">
                <button
                  type="button"
                  onClick={saveView}
                  className="inline-flex items-center gap-1.5 rounded-full border border-line bg-panel px-2.5 py-1 text-[11.5px] font-medium text-muted shadow-card transition-all duration-150 hover:border-accent/40 hover:text-accent-deep active:scale-[0.97]"
                >
                  <HugeiconsIcon icon={Bookmark01Icon} size={12} className="shrink-0" />
                  Save this search
                </button>
              </div>
            )}

            {phase === "error" && (
              <div className="mt-3 flex flex-wrap items-center justify-between gap-3.5 rounded-2xl border border-bad/25 bg-bad-soft px-4 py-3.5 text-[13px] text-bad shadow-card">
                <span className="inline-flex items-center gap-2">
                  <HugeiconsIcon icon={AlertCircleIcon} size={15} className="shrink-0" />
                  search did not work. try again.
                </span>
                <button
                  type="button"
                  onClick={() => search()}
                  className="shrink-0 rounded-full bg-bad px-3.5 py-1.5 text-xs font-semibold text-white transition-opacity duration-150 hover:opacity-90"
                >
                  try again
                </button>
              </div>
            )}

            {phase === "done" && visible.length === 0 && (
              <div className="mt-3 flex flex-wrap items-center justify-between gap-3.5 rounded-2xl border border-line bg-panel px-4 py-3.5 text-[13px] text-muted shadow-card">
                <span>
                  {filter && hits.length > 0
                    ? `no ${FILTERS[filter].label.toLowerCase()} in these results.`
                    : "nothing found. try fewer words."}
                </span>
                {filter && hits.length > 0 && (
                  <button
                    type="button"
                    onClick={() => setFilter(null)}
                    className="shrink-0 rounded-full bg-accent-soft px-3 py-1.5 text-xs font-semibold text-accent-deep transition-colors duration-150 hover:bg-accent-deep hover:text-white"
                  >
                    show all
                  </button>
                )}
              </div>
            )}

            {phase === "loading" && (
              <div className="mt-3 space-y-1.5">
                {[0, 1, 2].map((i) => (
                  <div key={i} className="h-[64px] rounded-2xl border border-line bg-panel skeleton animate-shimmer" />
                ))}
              </div>
            )}

            {phase !== "loading" && visible.length > 0 && (
              <div className="m-0 flex list-none flex-col p-0">
                {groups.map((g) => (
                  <div key={g.key}>
                    <p
                      className={`paper-blur z-[1] m-0 mb-1.5 mt-3 flex items-center gap-2.5 px-1 py-1 md:sticky md:top-[62px] ${stampCls}`}
                      aria-hidden="true"
                    >
                      {g.label}
                      <span className="h-px flex-1 bg-line" />
                      <span className="tabular-nums">{g.items.length}</span>
                    </p>
                    <ol className="relative m-0 flex list-none flex-col gap-2.5 p-0">
                      {/* the thread: runs behind the tile column, felt in the gaps */}
                      <span
                        aria-hidden="true"
                        className="absolute inset-y-2 left-[31px] w-px"
                        style={{
                          background:
                            "linear-gradient(to bottom, transparent, var(--color-line-strong) 5%, var(--color-line-strong) 95%, transparent)",
                        }}
                      />
                      {g.items.map((h, i) => (
                        <li
                          key={h.id}
                          className="animate-rise group relative rounded-2xl border border-line bg-panel px-3.5 py-3 shadow-card transition-all duration-200 hover:-translate-y-px hover:border-accent/30 hover:shadow-lift"
                          style={{ animationDelay: `${Math.min(i * 30, 300)}ms` }}
                        >
                          <div className="flex items-start gap-3">
                            <Tile type={h.source_type} />
                            <div className="min-w-0 flex-1">
                              <h3 className="m-0 text-[14.5px] font-semibold leading-snug text-deep [overflow-wrap:anywhere]">
                                <Link href={`/item/${h.id}`} className="transition-colors hover:text-accent-deep">
                                  {displayTitle(h)}
                                </Link>
                              </h3>
                              <p className="m-0 mt-1 flex flex-wrap items-center gap-x-2 gap-y-1 text-[11.5px] text-muted">
                                {h.domain && (
                                  <span className="max-w-[220px] truncate font-mono text-[10.5px] tracking-[0.02em]">
                                    {h.domain}
                                  </span>
                                )}
                                <span className="rounded-full bg-hover px-1.5 py-px font-mono text-[9.5px] font-medium uppercase tracking-[0.08em]">
                                  {styleOf(h.source_type).label}
                                </span>
                                {h.topics && h.topics[0] && (
                                  <span className="rounded-full bg-hover px-1.5 py-px font-mono text-[9.5px] font-medium uppercase tracking-[0.08em]">
                                    {h.topics[0]}
                                  </span>
                                )}
                                <button
                                  type="button"
                                  onClick={() => toggleRelated(h.id)}
                                  aria-expanded={relOpen === h.id}
                                  className="inline-flex items-center gap-0.5 font-medium text-accent-deep transition-colors hover:text-accent"
                                >
                                  related
                                  <HugeiconsIcon
                                    icon={ChevronDownIcon}
                                    size={11}
                                    className={`shrink-0 transition-transform duration-200 ${relOpen === h.id ? "rotate-180" : ""}`}
                                  />
                                </button>
                                {(h.source_type === "url" || h.source_type === "bookmark") &&
                                  /^https?:\/\//i.test(h.raw_ref || "") && (
                                    <a
                                      href={h.raw_ref}
                                      target="_blank"
                                      rel="noreferrer"
                                      className="inline-flex items-center gap-1 rounded-full border border-line bg-panel px-2 py-px text-[10px] font-semibold tracking-[0.03em] text-accent-deep shadow-card transition-all duration-150 hover:border-accent/40 hover:bg-accent-faint active:scale-[0.96]"
                                    >
                                      open original
                                      <HugeiconsIcon icon={LinkSquare02Icon} size={10} strokeWidth={2} className="shrink-0" />
                                    </a>
                                  )}
                                {(h.source_type === "pdf" || h.source_type === "image") && (
                                  <a
                                    href={`${API}/v1/items/${h.id}/file`}
                                    target="_blank"
                                    rel="noreferrer"
                                    className="inline-flex items-center gap-1 rounded-full border border-line bg-panel px-2 py-px text-[10px] font-semibold tracking-[0.03em] text-accent-deep shadow-card transition-all duration-150 hover:border-accent/40 hover:bg-accent-faint active:scale-[0.96]"
                                  >
                                    open file
                                    <HugeiconsIcon icon={LinkSquare02Icon} size={10} strokeWidth={2} className="shrink-0" />
                                  </a>
                                )}
                              </p>
                              {h.excerpt && (
                                <p
                                  className="excerpt m-0 mt-1.5 text-[12.5px] leading-relaxed text-muted"
                                  dangerouslySetInnerHTML={{ __html: h.excerpt }}
                                />
                              )}
                              {relOpen === h.id && (
                                <div className="mt-2.5 rounded-[12px] bg-accent-faint px-3 py-2.5 text-xs text-muted ring-1 ring-inset ring-accent/10">
                                  {relBusy && !relMap[h.id] ? (
                                    <span>looking...</span>
                                  ) : (relMap[h.id] ?? []).length === 0 ? (
                                    <span>nothing closely related yet.</span>
                                  ) : (
                                    <ul className="m-0 list-none space-y-1.5 p-0">
                                      {(relMap[h.id] ?? []).map((rel) => (
                                        <li key={rel.title + rel.reasons.join()} className="flex items-center gap-2">
                                          <Tile type={rel.source_type} size={22} />
                                          <span className="min-w-0 flex-1 truncate font-medium text-deep">
                                            {displayTitle(rel)}
                                          </span>
                                          {rel.reasons.map((rc) => (
                                            <span
                                              key={rc}
                                              className="shrink-0 rounded-full bg-accent-soft px-1.5 py-px font-mono text-[9px] font-medium uppercase tracking-[0.06em] text-accent-deep"
                                            >
                                              {REASONS[rc] ?? rc}
                                            </span>
                                          ))}
                                        </li>
                                      ))}
                                    </ul>
                                  )}
                                </div>
                              )}
                            </div>
                            <span className="hidden shrink-0 pt-0.5 font-mono text-[10.5px] tabular-nums text-muted/80 sm:block">
                              {relTime(h.captured_at, now)}
                            </span>
                          </div>
                        </li>
                      ))}
                    </ol>
                  </div>
                ))}
              </div>
            )}
          </motion.section>
        )}
        </AnimatePresence>
      </main>
      </div>

      {/* mobile bottom bar, constitution §55: Search · Recent · Capture · Archive · Settings */}
      <nav
        className="glass fixed inset-x-0 bottom-0 z-30 flex items-stretch justify-around border-t border-line pb-[env(safe-area-inset-bottom)] md:hidden"
        aria-label="Primary"
      >
        <button
          type="button"
          onClick={focusSearch}
          className="flex flex-1 flex-col items-center justify-center gap-1 py-2 text-[10px] font-medium text-muted transition-colors active:scale-95"
        >
          <HugeiconsIcon icon={Search01Icon} size={20} />
          Search
        </button>
        <button
          type="button"
          onClick={openRecent}
          className={`flex flex-1 flex-col items-center justify-center gap-1 py-2 text-[10px] font-medium transition-colors active:scale-95 ${
            showRecent ? "text-accent-deep" : "text-muted"
          }`}
        >
          <HugeiconsIcon icon={HistoryIcon} size={20} />
          Recent
        </button>
        <button
          type="button"
          onClick={() => {
            setShowCapture((s) => !s);
            setShowSettings(false);
            setShowCleanup(false);
            setShowReview(false);
            setShowRecent(false);
            setShowArchive(false);
          }}
          className={`flex flex-1 flex-col items-center justify-center gap-1 py-2 text-[10px] font-medium transition-colors active:scale-95 ${
            showCapture ? "text-accent-deep" : "text-muted"
          }`}
        >
          <HugeiconsIcon icon={PlusSignIcon} size={20} />
          Save
        </button>
        <button
          type="button"
          onClick={() => openArchive("all")}
          className={`flex flex-1 flex-col items-center justify-center gap-1 py-2 text-[10px] font-medium transition-colors active:scale-95 ${
            showArchive ? "text-accent-deep" : "text-muted"
          }`}
        >
          <HugeiconsIcon icon={Bookmark01Icon} size={20} />
          Archive
        </button>
        <button
          type="button"
          onClick={() => {
            setShowSettings((s) => !s);
            setShowCleanup(false);
            setShowReview(false);
            setShowRecent(false);
            setShowArchive(false);
          }}
          className={`flex flex-1 flex-col items-center justify-center gap-1 py-2 text-[10px] font-medium transition-colors active:scale-95 ${
            showSettings ? "text-accent-deep" : "text-muted"
          }`}
        >
          <HugeiconsIcon icon={Settings02Icon} size={20} />
          Settings
        </button>
      </nav>
    </div>
  );
}

// Shared utility strings (Tailwind). The rail buttons, mono stamps, kbd caps.
const railCls =
  "flex w-full items-center justify-center gap-2.5 rounded-[12px] px-2.5 py-[9px] text-left text-[13.5px] font-medium text-muted transition-all duration-150 hover:bg-hover hover:text-deep active:scale-[0.97] md:justify-start";
const railActive = " bg-accent-soft text-accent-deep";
const stampCls =
  "font-mono text-[10px] font-medium uppercase tracking-[0.16em] text-muted/80";
const kbdCls =
  "select-none rounded-[6px] border border-line bg-panel px-[7px] py-[3px] font-mono text-[10px] font-medium text-muted shadow-[0_1px_0_rgba(28,25,23,0.10)]";
