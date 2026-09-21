"use client";
import { use, useEffect, useState } from "react";
import Link from "next/link";
import { REASONS, displayTitle } from "../../../lib/labels";

const API = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8081";

type Item = {
  id: string;
  source_type: string;
  title: string;
  raw_ref: string;
  domain: string;
  captured_at: string;
  status: string;
  merged_into: string;
  original: Record<string, unknown>;
  topics: { id: string; name: string; confidence: number }[];
  entities: { id: string; type: string; name: string; confidence: number }[];
  asset: { kind: string; mime: string; byte_size: number; width: number | null; height: number | null; page_count: number | null } | null;
  content: string;
  truncated: boolean;
};

type Related = { id: string; title: string; domain: string; reasons: string[] };

function isHttp(s: string) {
  return /^https?:\/\//i.test(s || "");
}

// Same voice as the home screen: mono archival stamps.
const stampCls =
  "font-mono text-[10px] font-medium uppercase tracking-[0.16em] text-muted/80";

function savedOn(iso: string) {
  const d = new Date(iso);
  return Number.isNaN(d.getTime()) ? "" : d.toISOString().slice(0, 10);
}

export default function ItemPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const [item, setItem] = useState<Item | null>(null);
  const [related, setRelated] = useState<Related[]>([]);
  const [missing, setMissing] = useState(false);

  useEffect(() => {
    let alive = true;
    fetch(`${API}/v1/items/${id}`)
      .then((r) => {
        if (!r.ok) throw new Error();
        return r.json();
      })
      .then((j) => {
        if (alive) setItem(j);
      })
      .catch(() => {
        if (alive) setMissing(true);
      });
    fetch(`${API}/v1/events`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ item_id: id, kind: "opened" }),
    }).catch(() => {});
    fetch(`${API}/v1/items/${id}/related`)
      .then((r) => (r.ok ? r.json() : { related: [] }))
      .then((j) => {
        if (alive) setRelated((j.related ?? []).slice(0, 6));
      })
      .catch(() => {});
    return () => {
      alive = false;
    };
  }, [id]);

  async function correct(itemId: string, kind: string, targetId: string) {
    try {
      const r = await fetch(`${API}/v1/corrections`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ item_id: itemId, kind, target_id: targetId }),
      });
      if (!r.ok) return;
    } catch {
      return;
    }
    setItem((prev) => {
      if (!prev) return prev;
      if (kind === "topic") return { ...prev, topics: prev.topics.filter((t) => t.id !== targetId) };
      return { ...prev, entities: prev.entities.filter((e) => e.id !== targetId) };
    });
  }

  if (missing)
    return (
      <main className="mx-auto w-full max-w-[660px] px-4 py-10">
        <p className={stampCls}>
          <Link href="/" className="transition-colors hover:text-accent-deep">
            ← archive
          </Link>
        </p>
        <p className="mt-6 font-brand text-[24px] font-semibold tracking-[-0.02em] text-deep">
          That is gone or was never here.
        </p>
        <p className="mt-2 text-[12.5px] leading-relaxed text-muted">
          It may have been deleted, or merged into another save.
        </p>
      </main>
    );
  if (!item)
    return (
      <main className="mx-auto w-full max-w-[660px] px-4 py-10">
        <p className={stampCls}>
          <Link href="/" className="transition-colors hover:text-accent-deep">
            ← archive
          </Link>
        </p>
        <p className={`mt-6 ${stampCls}`}>fetching…</p>
      </main>
    );

  return (
    <main className="mx-auto w-full max-w-[660px] px-4 pb-16 pt-6 md:px-6">
      <p className={stampCls}>
        <Link href="/" className="transition-colors hover:text-accent-deep">
          ← archive
        </Link>
      </p>

      <header className="animate-rise mt-5">
        <div className="flex items-center gap-2.5">
          <span className="rounded-full bg-hover px-1.5 py-px font-mono text-[9.5px] font-medium uppercase tracking-[0.08em] text-muted">
            {item.source_type}
          </span>
          <span className="h-px flex-1 bg-line" />
        </div>
        <h1 className="m-0 mt-3 font-brand text-[clamp(22px,3.4vw,28px)] font-semibold leading-[1.18] tracking-[-0.02em] text-deep [overflow-wrap:anywhere]">
          {displayTitle(item)}
        </h1>
        <p className={`m-0 mt-2.5 flex flex-wrap items-center gap-x-2.5 gap-y-1 ${stampCls}`}>
          {item.domain && <span className="max-w-[260px] truncate tracking-[0.04em]">{item.domain}</span>}
          {item.captured_at && <span className="tabular-nums">saved {savedOn(item.captured_at)}</span>}
          {(item.source_type === "url" || item.source_type === "bookmark") && isHttp(item.raw_ref) && (
            <a
              href={item.raw_ref}
              target="_blank"
              rel="noreferrer"
              className="font-medium text-accent-deep transition-colors hover:text-accent"
            >
              open original ↗
            </a>
          )}
          {(item.source_type === "pdf" || item.source_type === "image") && (
            <a
              href={`${API}/v1/items/${item.id}/file`}
              target="_blank"
              rel="noreferrer"
              className="font-medium text-accent-deep transition-colors hover:text-accent"
            >
              open file{item.asset?.page_count ? ` · ${item.asset.page_count} pages` : ""} ↗
            </a>
          )}
        </p>
      </header>

      {item.status === "merged" && item.merged_into && (
        <p className="m-0 mt-5 flex flex-wrap items-center gap-x-2 gap-y-1 rounded-2xl border border-warn/25 bg-warn-soft px-4 py-3 text-[12.5px] text-warn">
          Merged into another save.{" "}
          <Link
            href={`/item/${item.merged_into}`}
            className="font-semibold underline decoration-dotted underline-offset-2"
          >
            Open it
          </Link>{" "}
          ·{" "}
          <button
            type="button"
            onClick={async () => {
              await fetch(`${API}/v1/maintenance/unmerge`, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ ids: [item.id] }),
              });
              window.location.reload();
            }}
            className="font-semibold underline decoration-dotted underline-offset-2 hover:text-deep"
          >
            Undo
          </button>
        </p>
      )}

      {item.source_type === "text" || item.source_type === "note" ? (
        <p className="m-0 mt-6 whitespace-pre-wrap text-[14px] leading-[1.75] text-deep">{item.raw_ref}</p>
      ) : item.source_type === "image" ? (
        // eslint-disable-next-line @next/next/no-img-element
        <img
          src={`${API}/v1/items/${item.id}/file`}
          alt={item.title || "saved image"}
          className="animate-rise mt-6 max-w-full rounded-2xl border border-line bg-panel shadow-card"
        />
      ) : null}

      {item.source_type === "pdf" && item.content && (
        <div className="animate-rise mt-6 rounded-2xl border border-line bg-panel px-5 py-4 shadow-card">
          <p className={`m-0 mb-3 ${stampCls}`}>Extracted text</p>
          {item.content.split("\n").filter((p) => p.trim() !== "").map((p, i) => (
            <p key={i} className="m-0 mb-2.5 text-[13px] leading-[1.7] text-deep last:mb-0">
              {p}
            </p>
          ))}
          {item.truncated && (
            <p className={`m-0 mt-3 ${stampCls}`}>Cut short here · open the file for the rest</p>
          )}
        </div>
      )}

      {(item.topics.length > 0 || item.entities.length > 0) && (
        <section className="mt-10">
          <h2 className="m-0 flex items-center gap-2.5">
            <span className={stampCls}>What the system sees</span>
            <span className="h-px flex-1 bg-line" />
          </h2>
          <p className="m-0 mt-1.5 text-[11.5px] text-muted">Guesses, not facts. Remove any that are wrong.</p>
          {item.topics.length > 0 && (
            <p className="m-0 mt-3 flex flex-wrap items-center gap-1.5">
              {item.topics.map((t) => (
                <span
                  key={t.id}
                  className="inline-flex items-center gap-1.5 rounded-full border border-line bg-panel px-2.5 py-1 text-xs font-medium text-deep shadow-card transition-all duration-150 hover:border-accent/35"
                >
                  {t.name}
                  <button
                    type="button"
                    onClick={() => correct(id, "topic", t.id)}
                    className="text-muted transition-colors hover:text-accent-deep"
                    aria-label={`Remove topic ${t.name}`}
                  >
                    ×
                  </button>
                </span>
              ))}
            </p>
          )}
          {item.entities.length > 0 && (
            <p className="m-0 mt-2 flex flex-wrap items-center gap-1.5">
              {item.entities.slice(0, 12).map((e) => (
                <span
                  key={e.id}
                  className="inline-flex items-center gap-1.5 rounded-full border border-line bg-panel px-2.5 py-1 text-xs font-medium text-deep shadow-card transition-all duration-150 hover:border-accent/35"
                >
                  {e.name}
                  <button
                    type="button"
                    onClick={() => correct(id, "entity", e.id)}
                    className="text-muted transition-colors hover:text-accent-deep"
                    aria-label={`Remove ${e.name}`}
                  >
                    ×
                  </button>
                </span>
              ))}
            </p>
          )}
        </section>
      )}

      {related.length > 0 && (
        <section className="mt-10">
          <h2 className="m-0 flex items-center gap-2.5">
            <span className={stampCls}>Connected memories</span>
            <span className="h-px flex-1 bg-line" />
          </h2>
          <ul className="m-0 mt-3 grid list-none gap-2 p-0 sm:grid-cols-2">
            {related.map((r) => (
              <li key={r.id}>
                <Link
                  href={`/item/${r.id}`}
                  className="group flex h-full flex-col gap-1 rounded-2xl border border-line bg-panel px-3.5 py-3 shadow-card transition-all duration-200 hover:-translate-y-0.5 hover:border-accent/30 hover:shadow-lift"
                >
                  <span className="text-[13px] font-semibold leading-snug text-deep transition-colors group-hover:text-accent-deep [overflow-wrap:anywhere]">
                    {displayTitle(r)}
                  </span>
                  <span className={`${stampCls} !tracking-[0.08em] leading-relaxed`}>
                    {r.domain || ""}
                    {r.domain ? " · " : ""}
                    {r.reasons.map((rc) => REASONS[rc] ?? rc).join(", ").toLowerCase()}
                  </span>
                </Link>
              </li>
            ))}
          </ul>
        </section>
      )}
    </main>
  );
}
