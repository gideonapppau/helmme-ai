import { NextResponse } from "next/server";
import { experimental_evaluate as evaluate } from "ai";
import { createTypeSafeAi } from "@ai-sdk/typesafe-ai";
import { fastPath, QueryRouterQuestions } from "@helmme/context-engine";

// Server-only route: the TypeSafe key never reaches the browser.
// POST { query } -> { action, operation?, needsSynthesis?, reason? }
//
// Rules (single source: @helmme/context-engine):
// - obvious queries are answered by cheap rules, no Jev call
// - Jev judges the closed operation set only, thresholds enforced in code
// - every judgment is logged to the API's decision_events (best effort,
//   never blocks the answer)

// logDecision files the judgment with the API so answers stay auditable.
// Plain words: the judge shows its work. Failures are swallowed on purpose.
function logDecision(input: {
  query: string;
  source: string;
  operation?: string;
  confidence?: number;
  probability?: number;
  action: string;
}) {
  const base = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8081";
  fetch(`${base}/v1/decision-events`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      component: "query-router",
      model: input.source === "jev" ? "jev-latest" : "rule",
      input: { query: input.query },
      result: {
        action: input.action,
        operation: input.operation ?? null,
        confidence: input.confidence ?? null,
        probability: input.probability ?? null,
      },
    }),
  }).catch(() => {});
}

export async function POST(req: Request) {
  let body: { query?: string };
  try {
    body = await req.json();
  } catch {
    return NextResponse.json({ error: "Send JSON with a query." }, { status: 400 });
  }
  const query = (body.query ?? "").trim();
  if (!query) return NextResponse.json({ error: "Type something first." }, { status: 400 });
  if (query.length > 500)
    return NextResponse.json({ error: "Shorter question, please." }, { status: 400 });

  const ruled = fastPath(query);
  if (ruled) {
    logDecision({ query, source: "rule", operation: ruled, action: "dispatch" });
    return NextResponse.json({
      action: "dispatch",
      operation: ruled,
      needsSynthesis: ruled !== "SEARCH",
      source: "rule",
    });
  }

  const apiKey = process.env.TYPESAFE_AI_API_KEY ?? process.env.TYPESAFE_API_KEY;
  if (!apiKey) {
    return NextResponse.json(
      { action: "human-review", reason: "Add TYPESAFE_AI_API_KEY to apps/web/.env.local and restart." },
      { status: 503 },
    );
  }

  try {
    const provider = createTypeSafeAi({ apiKey });
    const result = await evaluate({
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      model: provider.evaluationModel("jev-latest") as any,
      state: { query },
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      questions: QueryRouterQuestions as any,
    });
    const answers = result.answers as unknown as {
      operation: { choice: string; probabilities?: Record<string, number> };
      needs_synthesis: { probability: number };
    };
    const confidence =
      (result.providerMetadata?.typesafe?.confidence as Record<string, number> | undefined) ?? {};
    const op = answers.operation.choice;
    const conf = confidence.operation ?? 0;
    const prob = answers.operation.probabilities?.[op] ?? 0;
    if (conf < 0.6 || prob < 0.7) {
      logDecision({ query, source: "jev", operation: op, confidence: conf, probability: prob, action: "human-review" });
      return NextResponse.json({ action: "human-review", reason: "Not sure yet.", source: "jev" });
    }
    logDecision({ query, source: "jev", operation: op, confidence: conf, probability: prob, action: "dispatch" });
    return NextResponse.json({
      action: "dispatch",
      operation: op,
      needsSynthesis: answers.needs_synthesis.probability >= 0.8,
      source: "jev",
    });
  } catch {
    return NextResponse.json(
      { action: "human-review", reason: "The router is down. Plain search still works." },
      { status: 502 },
    );
  }
}
