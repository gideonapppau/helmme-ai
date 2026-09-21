// Mock-first tests for the Jev router wiring. No network, no API key.
import { describe, it } from "node:test";
import assert from "node:assert/strict";
import { routeQueryAmbiguous, buildRouterState, applyRouterResult } from "./jev.ts";

const mockEvaluate = async () => ({
  answers: {
    operation: { choice: "SYNTHESIS", probabilities: { SYNTHESIS: 0.88, SEARCH: 0.12 } },
    needs_synthesis: { probability: 0.93 },
  },
  confidence: { operation: 0.81 },
});

const mockUnsure = async () => ({
  answers: {
    operation: { choice: "SEARCH", probabilities: { SEARCH: 0.52, SYNTHESIS: 0.48 } },
    needs_synthesis: { probability: 0.5 },
  },
  confidence: { operation: 0.2 },
});

describe("Jev router wiring (mocked)", () => {
  it("confident mock -> dispatch SYNTHESIS + needsSynthesis", async () => {
    const r = await routeQueryAmbiguous("Things I saved about restaurants in Osu", ["restaurants", "Osu"], mockEvaluate);
    assert.equal(r.action, "dispatch");
    assert.equal(r.needsSynthesis, true);
  });
  it("unsure mock -> human-review, never guess", async () => {
    const r = await routeQueryAmbiguous("stuff", [], mockUnsure);
    assert.equal(r.action, "human-review");
  });
  it("state carries query + entities for Jev to judge", () => {
    assert.deepEqual(buildRouterState("q", ["a"]), { query: "q", entities: ["a"] });
  });
  it("applyRouterResult thresholds in code", () => {
    const ok = applyRouterResult(
      { operation: { choice: "SEARCH", probabilities: { SEARCH: 0.9 } }, needs_synthesis: { probability: 0.1 } },
      { operation: 0.9 },
    );
    assert.equal(ok.action, "dispatch");
    assert.equal(ok.needsSynthesis, false);
  });
});
