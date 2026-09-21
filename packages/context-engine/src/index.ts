// Context Engine — Jev decision layer (constitution APP_CONSTITUTION.md App. J).
// Memory answers "What do I know?"; Jev answers "Given what I know, what
// should I do next?" Separate brands, shared infra. Jev is the first major
// consumer of the context infrastructure, not a feature inside capture.
// Rule: LLM/code extracts open slots; Jev judges closed sets only.
// Vercel AI SDK: experimental_evaluate, model 'typesafe-ai/jev'.
// boolean answer -> probability; choice/score confidence lives in
// providerMetadata.typesafe.confidence.

// Deterministic fast-path (cheap parsing first). Intent taxonomy is the
// constitution's (§75–§76): LOOKUP finds one thing, SEARCH looks broadly, FILTER
// narrows, SYNTHESIS combines, COMPARE sets side by side, RECALL rebuilds
// half-memories, REDISCOVER resurfaces the forgotten.
export type Operation = 'LOOKUP' | 'SEARCH' | 'FILTER' | 'SYNTHESIS' | 'COMPARE' | 'RECALL' | 'REDISCOVER';

export function fastPath(query: string): Operation | null {
  const q = query.trim();
  if (!q) return null;
  const lower = q.toLowerCase();
  if (!/\s/.test(q)) return 'SEARCH'; // single token e.g. `postgres`
  if (/^(what did i learn|what have i learned|how has .* changed|what have i changed)/.test(lower)) return 'SYNTHESIS';
  if (/^(compare|.*\bvs\.?\b|.*\bversus\b)/.test(lower)) return 'COMPARE';
  if (/^(what connects|what were those|remind me)/.test(lower)) return 'RECALL';
  if (/^(what have i forgotten|.*\bnever\b.*\b(revisit|look|open)|surprise me)/.test(lower)) return 'REDISCOVER';
  if (/\b(before|after|last (year|month|week)|in 20\d\d|since|from last)\b/.test(lower)) return 'FILTER';
  return null; // ambiguous -> Jev
}

// Jev question sets. Call with experimental_evaluate; thresholds in code.
export const QueryRouterQuestions = {
  operation: {
    type: 'choice',
    instructions: 'Which operation should handle this personal-archive query?',
    criteria: {
      LOOKUP: 'Find one specific saved thing by name',
      SEARCH: 'General lookup across the archive',
      FILTER: 'Search narrowed by time, source, or project',
      SYNTHESIS: 'Combine many sources into an answer',
      COMPARE: 'Set two things side by side',
      RECALL: 'Rebuild something half-remembered from context',
      REDISCOVER: 'Resurface forgotten or unseen material',
    },
  },
  needs_synthesis: {
    type: 'boolean',
    instructions: 'Does answering require synthesizing multiple sources, not just listing hits?',
  },
} as const;

export const MemoryStateOptions = [
  'FACT', 'BELIEF', 'QUESTION', 'DECISION', 'PLAN',
  'OBSERVATION', 'EXPERIMENT', 'LESSON', 'REFERENCE', 'INFERENCE',
] as const;

export const SourceClassifierQuestions = {
  is_personal_experience: { type: 'boolean', instructions: 'Does `state` describe the user\'s own experience/decision, not saved external content?' },
  is_external_claim: { type: 'boolean', instructions: 'Is `state` primarily an external claim (article/bookmark) rather than user-authored?' },
  is_decision: { type: 'boolean', instructions: 'Does `state` record an explicit user decision?' },
  is_unresolved: { type: 'boolean', instructions: 'Does `state` contain an unresolved question or unfinished intent?' },
} as const;

export const ClaimVerifierQuestions = (claim: string) => ({
  explicitly_supported: { type: 'boolean', instructions: `Does \`evidence\` explicitly state: ${claim}?` },
  contradicted: { type: 'boolean', instructions: `Does any \`evidence\` contradict: ${claim}?` },
});

// Policy: numbers from Jev, decisions in code.
export function routePolicy(operation: string, confidence: number, prob: number) {
  if (confidence < 0.6 || prob < 0.7) return { action: 'human-review' as const, reason: 'ambiguous operation' };
  return { action: 'dispatch' as const, operation };
}
export function labelClaim(supported: number, contradicted: number) {
  if (contradicted > 0.6) return 'CONTRADICTED';
  if (supported >= 0.5 && contradicted > 0.3) return 'DISPUTED';
  if (supported >= 0.8) return 'EVIDENCE_BACKED';
  if (supported >= 0.5) return 'INFERENCE';
  return 'UNKNOWN';
}
