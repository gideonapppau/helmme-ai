# Cleeve teardown, competitive note (v2, finished pass)

Date: 2026-09-20. Sources: homepage, Chrome Web Store listing, founder
Reddit/LinkedIn/Bluesky posts, store reviews metadata. No account created,
so in-app flows (onboarding, search quality, import) are reconstructed from
public material and marked as such. Hands-on protocol at the bottom.

## What it is

Bookmark saver/organizer/sharer. Free, beta. Chrome extension + web +
mobile. Built by a designer + developer in ~3 months. Explicitly
positioned as a **non-AI** tool.

Core loop: save link via extension → file into a Collection at save time →
find later with search + filters → share collections publicly or privately.

## By the numbers (honest base rates)

- ~500+ registered users (founder-reported, Feb 2025; growth via
  build-in-public only, no landing page, SEO, ads, or affiliates).
- Chrome extension: 355 users, 4.9/5 from **9 ratings**. Small, friendly
  base, not broad validation.
- Extension v1.1.0, last updated June 2025, 1.59MiB, English only.
  No update in 15 months = maintenance mode or focus shifted to web/mobile.
- Contact is a Gmail address; EU non-trader notice. Two-person project.

## Flow-by-flow

**Signup.** Email-first landing page, free. Keep ours equally fast.

**Capture (extension popup).** Their best work: small, non-intrusive,
quick access over extras, no media previews. Philosophy is ours inverted:
"if you're saving something, you likely already know where it belongs."
Capture **requires** a filing decision. Our thesis (§2.1) calls that the
tax we eliminate. Copy the popup discipline, reject the model.

**Organization.** Manual Collections (founder runs 46 of them) +
filters/sub-filters, source-aware cards (X, Dribbble, YouTube surface
extra info), YouTube watchable in-app. Works while libraries are small;
collapses at 5,000 items, the exact pain our progressive enrichment
(§16) exists for. No evidence of dedupe, canonicalization, or snapshots.

**Retrieval.** "Simple search" + filters; "find stuff even if you forget
the exact title" (fuzzy title match per store copy). Finds what you
filed, if you filed it. No semantic layer advertised, no synthesis, no
resurfacing, no temporal queries, no provenance story. Saved-but-forgotten
stays forgotten.

**Import.** No bulk-import story found anywhere. Onboarding is
save-as-you-go, which means no Day-1 archive moment (§38), our biggest
demo advantage over them.

**Sharing (their actual differentiator).** Public collection links
(`cleeve.app/p/...`), save-an-entire-collection, privately shared
collections. Real mechanics, not a button. Out of our scope until shared
context, and note the difference: they share *containers*, we would
share *context*.

## The objection they never answered

On Reddit: "my browser bookmarks already sync across devices, why do I
need this?" No crisp answer exists in their material. Their wedge is
cross-app centralization (Reddit/X/LinkedIn/Behance/Dribbble saves in one
place) + collections + sharing, not sync. Lesson for us: have the
one-line answer ready before anyone asks. Ours: bookmarks remember where
something was; we remember why it matters.

## The lane is crowded

The store page itself surfaces WebLinks, Lynkmark (5.0), CarryLinks,
Bookmarkify (4.6-4.8). "Save + organize links" is a commodity shelf.
Nobody there does synthesis, provenance, or resurfacing, the lane above
the shelf is empty.

## Privacy posture (for comparison)

Extension declares website-content access, Notion-hosted privacy policy,
no-sale/no-unrelated-use declarations. Fine for a side project; far from
our bar (§44-46: local-first, encrypted sync, provider registry). Privacy
architecture is a differentiator we should say out loud.

## Gaps that are our wedge

1. Organization tax at capture, every save demands a filing decision.
2. No intelligence over the archive, search finds, nothing explains.
3. No import/onboarding wedge, no "here are 37 things you forgot."
4. Links only, notes, screenshots, PDFs, ideas are outside the model.
5. "Non-AI" as identity, leaves the synthesis lane fully open.
6. Stale client (no update in 15 months) while we ship weekly.

## What to steal

- Result excerpts showing why something matched (shipped: `ts_headline`
  excerpts under every result, match in bold). Their cards proved the
  pattern; our excerpt is quieter and citation-oriented.
- Extension popup discipline: tiny, fast, invisible when unneeded.
- Source-aware extraction per adapter, maps to Tier-1 enrichment.
- Public build-in-public cadence: ship visible features, celebrate
  milestones, reply to users. Their 115-users-in-12-days came from this.
- Collection-sharing mechanics (save-entire-collection), file under
  future shared-context design, not folders.

## What to avoid

- Filing-at-capture. Never ask "which collection?" on save.
- Folders/collections as the mental model. Ours are living queries.
- Competing on sharing or star ratings. Different game.
- Judging by their 4.9, it is 9 friendly ratings, not market proof.

## Hands-on protocol (still open, needs an account)

1. Sign up, time-to-first-save. 2. Save 10 links across X/YouTube/articles
  , is collection choice forced? 3. Export browser bookmarks, is import
   offered? 4. Search a vague half-memory, does it resolve? 5. Share a
   collection, what does the recipient see? 6. Check network calls on
   save, full page content shipped? snapshots kept?

## One-line positioning vs Cleeve

Cleeve files your links. Helmme remembers what they meant.
