# Brainstorm — Next Steps After Keyword Evaluation

**Date:** 2026-03-07
**Status:** Direction agreed — proceeding to FEATURE

---

## Problem Statement

The app can evaluate keywords and present them in a results table with checkboxes. The table is static — the checkboxes do nothing. The user has no way to act on the evaluation results, export data, or hand off the validated keywords for further processing (clustering, Sheets export).

---

## What Success Looks Like

1. The user can select Good keywords from the results table using checkboxes.
2. The selected keywords are submitted for a second processing step: thematic clustering.
3. The final Google Sheets export contains the full keyword history — original list, post-evaluation state, and post-clustering state — so nothing is lost and discarded keywords can be reused.

---

## Considered Approaches

### Option A — Single pipeline, all steps sequential

Extend the existing pipeline with a clustering step that runs immediately after evaluation, without user interaction between steps.

**Trade-offs:**
- Simpler wiring — one job, one payload, one table render
- Removes user agency: the user cannot review evaluation results before clustering happens
- Clustering irrelevant keywords wastes AI quota
- Does not match stated goal of user selecting Good keywords first

**Verdict:** Rejected. User explicitly wants to review before clustering.

---

### Option B — Two independent async jobs, user-gated (chosen)

Step 1 (existing): Evaluate → present results table with checkboxes.
Step 2 (new): User selects Good keywords → submits selection → clustering job starts → second results view.

**Trade-offs:**
- Two round-trips, two job IDs, more wiring
- Matches the mental model: evaluate first, then organise what's worth keeping
- Checkbox submission needs a new endpoint and a new Templ form
- Clustering job needs a new payload shape and a new status handler

**Verdict:** Chosen. User confirmed: "The clustering is a second step."

---

### Option C — Client-side clustering in JavaScript

Group keywords in the browser after evaluation, no second API call.

**Trade-offs:**
- Eliminates Gemini latency and quota usage for clustering
- No AI quality — pure string matching or naive grouping
- Inconsistent with the pattern of using AI for quality judgements
- Contradicts the HTMX + server-rendered constraint

**Verdict:** Rejected.

---

## Chosen Direction

```
Evaluate (Job 1)
  └─ Results table with checkboxes
        └─ User selects Good keywords
              └─ POST /api/v1/submit/cluster
                    └─ Clustering (Job 2)
                          └─ Clustered results view
                                └─ Google Sheets export (future)
```

---

## Data Preservation Requirement

Every cleaning step must preserve the original keyword list so the final Google Sheet export contains the full history. The user explicitly confirmed:

> "I want the google sheet to later have all the original keywords as well as the individual versions after each step of cleaning. This makes it possible for the user to organise and if wanted, reuse keywords that were discarded at first."

**Implication:** The processor must store the original list (before any pipeline steps run) alongside the current `[][]string` payload. This needs to be threaded through the job payload before the Sheets export is built.

**Decision for now:** Add an `OriginalRows` field to `KeywordJobPayload` in `internal/processor/types.go`. Populate it once, before the pipeline runs. The clustering step receives the full evaluated payload (including originals) and carries it forward.

---

## Open Questions / Risks

| Question | Decision |
|----------|----------|
| Does clustering produce clusters as a flat table or a nested structure? | Flat table for now (cluster name + keyword rows). Nested is a display concern. |
| Does the clustering job receive only selected keywords, or the full evaluated set? | Only selected (Good) keywords — user selection is the gate. |
| What if the user selects zero keywords? | Validate at the endpoint — return an error fragment, do not start a job. |
| Should the original rows carry through both jobs, or just be stored once in Job 1? | Stored once in Job 1's payload; Job 2 receives them as part of the submission. |

---

## Agreed Implementation Order

1. **CheckboxSubmission** — wire the checkbox form to POST selected keywords and launch Job 2
2. **KeywordClustering** — implement the clustering job, Gemini call, and clustered results view
3. **OriginalKeywordPreservation** — add `OriginalRows` to the job payload and thread it forward
4. **GoogleSheetsExport** — write final data (original + evaluated + clustered) to a Google Sheet

Each step depends on the previous. Start with CheckboxSubmission.

---

## Transition

BRAINSTORM complete → proceeding to **FEATURE: CheckboxSubmission**
