# PDF extraction limits, measured, not guessed

Date: 2026-09-20. Method: word-level diff of extractor output against
ground truth (the same CV as `.docx`, 86 text runs, 5,782 chars).

## What the diff proves

Two dozen artifact tokens in PDF output, zero in ground truth:

- Trailing `a`: resume.a, data.a, storage.a, Universitya, APIs.a
- Leading `a`/`U`: agideonad, USoftware, UReact, UNode
- Trailing `n`: Presentn, Groqn, Queryn, Expectedn, 2022n
- Trailing `T`: PROJECTST, EDUCATIONT, SUMMARYT
- Trailing `U`: InternU, OptimizerU, TechnologyU, FellowshipU

Pattern: separators and markers (`|`, `•`, `-`, tab leaders) decode to
wrong ASCII letters that glue onto neighboring words. This is glyph
misdecoding inside the PDF library's font handling, not spacing, not
punctuation, not fixable by joining rules (those are already correct).

## Why regex cannot fix it

Every candidate rule eats real words. Stripping trailing `a` kills
Ghana, Accra, Java. Stripping leading `U` kills nothing common, but
the next font will misdecode to a different letter, and the rulebook
grows per font while precision never converges. Glyph identity needs
font tables or rendered pixels, not string patterns.

## Verdict: vision layer, with this evidence attached

Correct fix: render the page and read it (vision model) or run OCR,
then reconcile with extractor text. Needs a key and a pipeline, not this slice. Until then, containment stands: capped confidence,
visible artifacts, control-char cleaning, never asserted as fact.

Reproduce: `docs/eval` style, extract `.docx` word/document.xml
`w:t` nodes, diff token sets against `item_contents.extracted_text`.
