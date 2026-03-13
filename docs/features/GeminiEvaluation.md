# Gemini Evaluation

- **Status:** Complete (OpenAI stub exists but is not implemented)
- **Created:** 2026-03-02 (onboarded — code predates this record)
- **Last updated:** 2026-03-13
- **Touches:** `internal/ai/ai.go`, `internal/ai/gemini.go`, `internal/ai/openai.go`

---

## What It Does

Sends a list of keyword rows and a campaign description to the Google Gemini API. Receives back a structured JSON response containing two new columns for each row: "Evaluation" (a rating) and "Reasoning" (a plain-language explanation). The result is merged into the original keyword table.

---

## Why It Was Built

Manual keyword evaluation for Google Ads campaigns is time-consuming and inconsistent. The AI can evaluate hundreds of keywords in seconds and provide reasoning that helps the user understand which keywords are relevant to their campaign.

---

## Implementation Checklist

### Initial implementation — (onboarded 2026-03-02)
- [x] Define provider interfaces in `ai.go`: `KeywordEvaluator`, `KeywordGenerator`, `KeywordOrganizer`
- [x] Implement `EvaluateKeywords()` on Gemini client
- [x] System prompt: positions model as SEO expert, instructs it to match input row count
- [x] Structured JSON output with schema validation (prevents unparseable responses)
- [x] Temperature 0.0 for deterministic, consistent results
- [x] N/A padding in `ZipResults()` for rows the AI skips or merges
- [x] Empty stub file `openai.go` created as placeholder

---

## Technical Notes

- **Structured output:** The Gemini call uses the `genai` SDK's schema-constrained output mode. This means the response is always valid JSON matching the expected shape — no regex parsing needed.
- **Temperature 0.0:** Chosen for consistency. The same input should always produce the same output, which makes debugging and testing predictable.
- **Row count contract:** The system prompt explicitly tells Gemini to return exactly one row per input keyword. `ZipResults` handles the case where it does not comply (pads with "N/A"). This is the primary source of potential data loss — if Gemini silently drops a row, the result table will have a mismatched row.
- **Interface is properly used:** `KeywordsCommand` holds `ai.KeywordEvaluator` and `ai.KeywordOrganizer` — never `*ai.Client`. The interface-over-concrete pattern is fully in place for the Wails migration.
- **Two separate configs on one client:** `gemini.go`'s `NewClient` builds both an `evaluatorConfig` and an `organizerConfig`. The `Client` struct implements both `KeywordEvaluator` and `KeywordOrganizer` using the appropriate config for each call.
- **OpenAI stub:** `internal/ai/openai.go` is intentionally empty. Implement it by satisfying the `ai.KeywordEvaluator` (and optionally `KeywordOrganizer`) interface, then pass it to `KeywordsCommand.Setup()` instead of the Gemini client.
- **`KeywordGenerator` is defined but has no callers.** It is the seam for a future keyword generation feature.

---

## Change History

*(no changes since initial implementation)*
