// Jev wiring — mock-first, no API key needed for tests.
// Live path uses `experimental_evaluate` from the Vercel AI SDK only when
// explicitly called with a real evaluate fn (AI_GATEWAY_API_KEY required).
// Pure helpers below are fully tested without network or deps.
import { QueryRouterQuestions, routePolicy } from "./index.ts";

export const JEV_MODEL_ID = "typesafe-ai/jev";

export function buildRouterState(query: string, entities: string[] = []) {
  return { query, entities };
}

// Shape of what experimental_evaluate returns (minimal surface we use).
export type RouterAnswers = {
  operation: { choice: string; probabilities?: Record<string, number> };
  needs_synthesis: { probability: number };
};
export type ConfidenceMap = Record<string, number>;

// Pure: turn Jev answers + confidence map into a dispatch decision.
// Thresholds live here in code, never in the model.
export function applyRouterResult(answers: RouterAnswers, confidence: ConfidenceMap = {}) {
  const op = answers.operation.choice;
  const conf = confidence.operation ?? 0;
  const prob = answers.operation.probabilities?.[op] ?? 0;
  return { ...routePolicy(op, conf, prob), needsSynthesis: answers.needs_synthesis.probability >= 0.8 };
}

// Injectable evaluate fn so tests pass a mock and prod passes the real SDK call.
// evaluateFn(state, questions) -> { answers, confidence }
export async function routeQueryAmbiguous(
  query: string,
  entities: string[],
  evaluateFn: (state: unknown, questions: unknown) => Promise<{ answers: RouterAnswers; confidence?: ConfidenceMap }>,
) {
  const { answers, confidence } = await evaluateFn(buildRouterState(query, entities), QueryRouterQuestions);
  return applyRouterResult(answers, confidence ?? {});
}

// Live adapter (call only when AI_GATEWAY_API_KEY is set):
//   import { experimental_evaluate as evaluate } from 'ai';
//   routeQueryAmbiguous(q, ents, (state, questions) =>
//     evaluate({ model: JEV_MODEL_ID, state, questions }).then(r => ({
//       answers: r.answers,
//       confidence: r.providerMetadata?.typesafe?.confidence,
//     })));
