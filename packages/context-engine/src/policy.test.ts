// node:test suite for the Jev decision layer.
// Jev judges closed sets only; thresholds live in code, never in the model.
import { describe, it } from "node:test";
import assert from "node:assert/strict";
import {
  fastPath,
  routePolicy,
  labelClaim,
  MemoryStateOptions,
  QueryRouterQuestions,
} from "./index.ts";

describe("fastPath (deterministic before Jev)", () => {
  it("single token -> SEARCH, no AI call", () => {
    assert.equal(fastPath("postgres"), "SEARCH");
  });
  it("learn phrasing -> SYNTHESIS", () => {
    assert.equal(fastPath("What did I learn about Postgres?"), "SYNTHESIS");
    assert.equal(fastPath("How has my thinking on databases changed?"), "SYNTHESIS");
  });
  it("compare phrasing -> COMPARE", () => {
    assert.equal(fastPath("Compare Postgres and SQLite"), "COMPARE");
    assert.equal(fastPath("Postgres vs SQLite"), "COMPARE");
  });
  it("time phrasing -> FILTER", () => {
    assert.equal(fastPath("Postgres notes from last year"), "FILTER");
    assert.equal(fastPath("What did I save before Clerk?"), "FILTER");
  });
  it("half-memories -> RECALL", () => {
    assert.equal(fastPath("What connects Clerk and Redis?"), "RECALL");
    assert.equal(fastPath("What were those restaurants in Osu?"), "RECALL");
  });
  it("forgotten things -> REDISCOVER", () => {
    assert.equal(fastPath("What have I forgotten?"), "REDISCOVER");
    assert.equal(fastPath("Things I never revisit"), "REDISCOVER");
  });
  it("ambiguous -> null (Jev handles it)", () => {
    assert.equal(fastPath("Things I saved about restaurants in Osu"), null);
  });
});

describe("routePolicy (thresholds in code)", () => {
  it("low confidence -> human-review, never guess", () => {
    assert.equal(routePolicy("SEARCH", 0.4, 0.9).action, "human-review");
    assert.equal(routePolicy("SEARCH", 0.9, 0.5).action, "human-review");
  });
  it("confident -> dispatch", () => {
    const r = routePolicy("SYNTHESIS", 0.9, 0.85);
    assert.equal(r.action, "dispatch");
  });
});

describe("labelClaim (evidence-first)", () => {
  it("contradicted wins", () => assert.equal(labelClaim(0.9, 0.7), "CONTRADICTED"));
  it("conflict shows, even when support is strong", () => {
    assert.equal(labelClaim(0.7, 0.4), "DISPUTED");
    assert.equal(labelClaim(0.9, 0.3), "EVIDENCE_BACKED");
  });
  it("strong support -> EVIDENCE_BACKED", () => assert.equal(labelClaim(0.85, 0.1), "EVIDENCE_BACKED"));
  it("weak -> INFERENCE / UNKNOWN, never stated as fact", () => {
    assert.equal(labelClaim(0.6, 0.1), "INFERENCE");
    assert.equal(labelClaim(0.2, 0.1), "UNKNOWN");
  });
});

describe("question sets stay bounded (Jev constraint)", () => {
  it("memory states fit choice cardinality (<=255)", () => {
    assert.ok(MemoryStateOptions.length === 10 && MemoryStateOptions.length <= 255);
  });
  it("router exposes fixed taxonomy, no free extraction", () => {
    assert.ok("operation" in QueryRouterQuestions && "needs_synthesis" in QueryRouterQuestions);
  });
});
