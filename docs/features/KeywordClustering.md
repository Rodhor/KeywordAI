# KeywordClustering

- **Status:** Complete
- **Created:** 2026-03-09
- **Last updated:** 2026-03-13
- **Touches:** `internal/ai/gemini.go` · `internal/processor/cluster.go` · `internal/processor/types.go` · `backend/commands/keywords/keywords.go` · `frontend/src/lib/components/ClusterResults.svelte` · `frontend/src/routes/keywords/+page.svelte`

---

## What It Does

Takes the user-selected "Good" keywords from the evaluation step, sends them to Gemini with a prompt that groups each keyword under a short thematic cluster name, and returns a table with all evaluated columns plus a new "Cluster" column. The user sees the clustered table and can start over when done.

---

## Why It Was Built

The evaluation step tells the user which keywords are worth keeping. Clustering organises those keywords into themes so they can be structured into ad groups or content categories. It is the second half of the keyword cleaning pipeline.

---

## Implementation Checklist

### Initial implementation — 2026-03-13

- [x] Add `organizerConfig *genai.GenerateContentConfig` field to `ai.Client` in `gemini.go`
- [x] Build organizer schema (`Cluster` column only) and system instruction in `NewClient`
- [x] Implement `OrganizeKeywords` using the organizer config — groups by search intent, 2–4 word title-cased cluster names
- [x] Implement `ProcessClustering` in `processor/cluster.go` — AI call + `ZipResults` + `CompleteJob`
- [x] Add `ClusterJobPayload` to `internal/processor/types.go`
- [x] `StartClustering` in `keywords.go` — reads eval job payload, filters to selected rows, creates cluster job
- [x] `GetClusterStatus` in `keywords.go` — returns `job.Payload` directly (already `ClusterJobPayload`)
- [x] `ClusterResults.svelte` — read-only table, "Start Over" button calls `onReset`
- [x] State machine in `keywords/+page.svelte` updated: `clustering` → `clustered` phases

---

## Technical Notes

- **Two separate Gemini configs:** `gemini.go`'s `NewClient` builds `evaluatorConfig` (schema: Evaluation + Reasoning columns) and `organizerConfig` (schema: Cluster column only) separately. They share the same model but have different system instructions and schemas — keeping them separate avoids a single overloaded config and makes swapping either independently possible.
- **Cluster payload field casing:** `ClusterJobPayload` uses exported Go field names (`Header`, `Content`, `OriginalRows`). The frontend receives these as capitalised JSON keys (`raw.Header`, `raw.Content`) — note the capitalisation; this is a common gotcha when bridging Go structs to JS.
- **No CSV download (deferred):** The original plan included a download button. This was deferred for the Wails migration — the clustering result is displayed as a read-only table. A download can be added later as a `runtime.SaveFileDialog` bound method.
- **`OriginalRows` is carried through** the payload but not displayed in `ClusterResults`. It is available for a future Sheets export.
- **Pattern used:** Async job (Pattern 2) + Composable pipeline (Pattern 3) + Wails command binding (Pattern 1).

---

## Change History

### 2026-03-13 — Reimplemented for Wails (replaces HTMX plan)

**Reason:** The original plan targeted HTMX HTTP handlers and Templ components. The project migrated to Wails v2 + SvelteKit before clustering was implemented. The checklist above reflects the Wails implementation — the old plan (cluster_status.go, cluster_download.go, ClusterSuccess.templ) was never built.

**What changed from the original plan:**
- HTTP handler + Templ approach dropped entirely
- Implemented as `StartClustering`/`GetClusterStatus` on `KeywordsCommand` (Wails binding)
- `ClusterResults.svelte` replaces the planned `ClusterSuccess.templ`
- CSV download deferred (no `cluster_download.go` built)
