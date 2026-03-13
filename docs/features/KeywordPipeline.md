# Keyword Pipeline

- **Status:** Modified
- **Created:** 2026-03-02 (onboarded — code predates this record)
- **Last updated:** 2026-03-13
- **Touches:** `internal/processor/keywords.go`, `internal/processor/types.go`, `internal/processor/ProcessAIResponse.go`, `internal/processor/ProcessFilesToSlices.go`, `internal/pipeline/pipeline.go`, `internal/pipeline/steps.go`, `internal/source/source.go`

---

## What It Does

Takes raw keyword data from a CSV file through a composable sequence of pipeline steps, then sends the cleaned data to the AI provider for evaluation. Returns a merged result set: the original columns plus AI-added "Evaluation" and "Reasoning" columns for each keyword row.

---

## Why It Was Built

Raw keyword exports from Google Ads contain noise — empty rows, duplicates, and terms the user explicitly wants to exclude. Sending dirty data to the AI wastes tokens, produces inconsistent results, and inflates costs. The pipeline removes noise before the AI call so it evaluates only meaningful keywords.

---

## Implementation Checklist

### Initial implementation — (onboarded 2026-03-02)
- [x] `ParseCSVToSlice()` — reads CSV file into `[][]string`
- [x] `DropUnwantedColumns()` — placeholder step (no-op)
- [x] `ExtractHeaderRow()` — splits header from data rows
- [x] `RemoveEmptyLines()` — removes rows where first column is blank
- [x] `RemoveDuplicateLines()` — case-insensitive dedup on first column
- [x] `DropExcludedKeywords()` — removes rows matching any excluded term (substring, case-insensitive)
- [x] `SendToAIForValidation()` — calls AI provider with cleaned data
- [x] `ZipResults()` — merges original rows with AI-returned columns
- [x] N/A padding — if AI returns fewer rows than sent, missing rows are padded with "N/A"
- [x] `os.Remove` deferred after processing completes

### Refactor session — 2026-03-07
- [x] **Bug 1 fix** — `FailJob` called on every error path so frontend polling terminates correctly
- [x] **P1c** — `internal/source/source.go` created: `KeywordSource` interface + `CSVFileSource` implementation; file cleanup moved into `CSVFileSource.Cleanup()`
- [x] **P1a** — `ProcessKeywords` parameters packed into `KeywordProcessParams` struct (JobID, Source, ExcludeKeywords, CampaignDesc, CustomInstructions, Evaluator, Manager)
- [x] **P1d** — `internal/pipeline/pipeline.go` created: `Context` struct + `Step` interface + `Run()` runner
- [x] **P1d** — `internal/pipeline/steps.go` created: `RemoveEmpty`, `DropIfEmpty`, `Deduplicate`, `ExcludeRows` — all parameterised on `col []int` with variadic constructors
- [x] `aiValidationStep` struct — domain-specific step kept in `internal/processor/keywords.go`; implements `pipeline.Step`
- [x] Old helper functions (`RemoveEmptyLines`, `RemoveDuplicateLines`, `DropExcludedKeywords`) removed from `keywords.go` — logic lives in pipeline steps
- [x] `KeywordJobPayload` moved from `internal/jobs/manager.go` to `internal/processor/types.go`

---

## Technical Notes

- **Step sequence:** `ProcessKeywords` builds a `[]pipeline.Step` slice and calls `pipeline.Run`. Adding or reordering a cleaning step is a one-liner change to that slice.
- **Generic steps in `pipeline/steps.go`:** `RemoveEmpty`, `DropIfEmpty`, `Deduplicate`, `ExcludeRows` are column-parameterised — they operate on whichever columns are passed via `col []int`. Constructors use variadic args: `NewDeduplicate(0)`, `NewExcludeRows(terms, 0)`, etc.
- **Null-byte separator for dedup keys:** `Deduplicate` builds composite keys by joining column values with `\x00` to prevent false collisions across column boundaries.
- **Domain-specific step stays in processor:** `aiValidationStep` depends on `ai.KeywordEvaluator` and the Gemini API response format — it would be wrong to put it in the generic `pipeline` package.
- **`KeywordSource` contract:** `ProcessKeywords` calls `params.Source.Open()` to get a reader, then `defer params.Source.Cleanup()` to remove the file. The processor never holds file paths.
- **`DropUnwantedColumns` is intentionally a no-op placeholder.** It exists to mark where column-selection logic will go when that feature is built.
- **`ZipResults` padding:** Gemini occasionally returns fewer rows than were sent. `ZipResults` fills the gap with "N/A" rather than panicking on an out-of-bounds index.
- **`customInstructions` is captured but not yet forwarded to the AI call.** Fixing it (QW1) requires only appending to the prompt string in `gemini.go` — no interface changes needed.
- **Pattern used:** Composable pipeline (Pattern 3 in Architecture.md) + Async job (Pattern 2).

---

## Change History

### 2026-03-07 — Full pipeline refactor (P1a, P1c, P1d + Bug 1 fix)

**Reason:** Pre-development refactor to unblock future features. The monolithic `ProcessKeywords` function made adding steps, swapping data sources, or testing steps in isolation impossible. Bug 1 caused the polling spinner to spin forever on any error.

**What changed:**
- [x] `FailJob` called on all error paths; `ErrorUI` component created and returned as HTTP 200
- [x] `internal/source/source.go` — `KeywordSource` interface + `CSVFileSource`
- [x] `ProcessKeywords` signature changed to accept `KeywordProcessParams` struct
- [x] `internal/pipeline/pipeline.go` — `Context`, `Step`, `Run`
- [x] `internal/pipeline/steps.go` — `RemoveEmpty`, `DropIfEmpty`, `Deduplicate`, `ExcludeRows` with `col []int`
- [x] `aiValidationStep` struct replaces inline AI call
- [x] Old helper functions removed from `keywords.go`
- [x] `KeywordJobPayload` moved to `internal/processor/types.go`

**Notes:** `customInstructions` is now on the struct and forwarded to the AI prompt via `aiValidationStep` (QW1 complete).
