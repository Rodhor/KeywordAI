# CheckboxSubmission

> ⚠ **Superseded — 2026-03-13**
> The HTMX form + HTTP cluster handler approach was replaced during the Wails migration. Row selection and cluster triggering now live entirely in `frontend/src/lib/components/KeywordTable.svelte` and `frontend/src/routes/keywords/+page.svelte`. The Templ components, HTTP handlers, and module registrations described below no longer exist. The Go types (`OriginalRows`, `ClusterJobPayload`) are still in use — see `internal/processor/types.go` and `backend/commands/keywords/keywords.go`.

- **Status:** Removed
- **Created:** 2026-03-07
- **Last updated:** 2026-03-13
- **Touches:** ~~`ui/customComponents/keywords/KeywordTable.templ`~~ · `ui/customComponents/shared/PollingUI.templ` · `internal/processor/types.go` · `internal/processor/keywords.go` · `internal/processor/cluster.go` (new) · `internal/api/v1/keywords/cluster.go` (new) · `internal/api/v1/keywords/cluster_status.go` (new) · `internal/api/v1/keywords/module.go` · `internal/api/v1/v1.go`

---

## What It Does

Makes the checkboxes in the evaluation results table functional. The user reviews AI-evaluated keywords, adjusts the selection, and clicks "Proceed to Clustering" to start a second async job. HTMX polling is reused — the spinner appears and then the clustering result is shown.

---

## Why It Was Built

The results table was a dead end — checkboxes rendered but submitted nothing. This feature is the gate between Job 1 (keyword evaluation) and Job 2 (keyword clustering). Without it, no further processing can happen on user-selected keywords.

---

## Implementation Checklist

### Initial implementation — 2026-03-07

- [x] Add `OriginalRows [][]string` to `KeywordJobPayload` in `processor/types.go`
- [x] Add `ClusterJobPayload` and `ClusterGroup` types to `processor/types.go`
- [x] Populate `OriginalRows` in `ProcessKeywords` before `pipeline.Run`
- [x] Create stub `processor/cluster.go` with `ProcessClustering` that calls `FailJob` immediately
- [x] Add `statusURL string` parameter to `PollingUI` in `shared/PollingUI.templ`
- [x] Update two `PollingUI` call sites in `keywords/submit.go` and `keywords/status.go`
- [x] Add `JobID string` to `ResultsTableParams` in `KeywordTable.templ`
- [x] Wrap table in `<form>` with `hx-post="/api/v1/submit/cluster"` and `eval_job_id` hidden input
- [x] Add `name="selected"` and `value=<row_index>` to checkboxes
- [x] Rename "Create Another Campaign" button to have `type="button"` to prevent accidental form submit
- [x] Add "Proceed to Clustering" submit button
- [x] Pass `JobID: jobID` in `status.go` where `ResultsTable` is called
- [x] Create `internal/api/v1/keywords/cluster.go` handler
- [x] Create `internal/api/v1/keywords/cluster_status.go` handler
- [x] Add `Organizer ai.KeywordOrganizer` to `keywords.Module`
- [x] Register `/submit/cluster` and `/submit/cluster/status` in `module.go`
- [x] Pass `Organizer` in `v1.go`
- [x] Run `templ generate`
- [x] Verify `go build ./...` passes
- [x] End-to-end test: form submission → spinner → "Clustering not yet implemented" error shown

---

## Technical Notes

- **Button component `Type` prop vs `Attributes`:** The templUI `Button` component defaults to `type="button"` when `Type` is not set, and renders that attribute before spreading `Attributes`. Passing `"type": "submit"` via `Attributes` produces a duplicate attribute — the first one wins in HTML. Fix: always use `Type: button.TypeSubmit` directly on the struct.
- **Submit button outside `<form>`:** Buttons placed after the closing `</form>` tag do not trigger form submission even with `type="submit"`. HTML5 `form="id"` association works in browsers but HTMX does not reliably intercept it. Fix: keep submit buttons inside the form.
- **`evalColIdx` lookup in Templ:** Go variable declarations inside a `templ` block require `{{ }}` wrapping. A bare `:=` outside a `for`/`if`/`switch` causes a parse error.
- **`JobID` threading:** `status.go` must pass `JobID: jobID` to `ResultsTableParams` so the hidden `eval_job_id` field in the form is populated. Missing this causes the cluster handler to reject every submission with "Missing evaluation job reference."
- **Checkbox default state:** `evalColIdx` is resolved at render time by scanning `params.Headers` for `"Evaluation"`. The index is passed to each row renderer; rows where `row[evalColIdx] == "Good"` are pre-checked.

---

## Change History

_No changes after initial implementation._
