# Results Table

> ⚠ **Superseded — 2026-03-13**
> The Templ results table was replaced by `frontend/src/lib/components/KeywordTable.svelte` during the Wails migration. The new component is a Svelte 5 component that receives `data: EvaluationData` as a prop, pre-checks rows where `Evaluation === 'Good'`, and emits `onCluster(selectedIndices)` and `onReset` callbacks. The Templ component and HTTP status handler no longer exist.

- **Status:** Removed
- **Created:** 2026-03-02 (onboarded — code predates this record)
- **Last updated:** 2026-03-13
- **Touches:** ~~`ui/customComponents/DataDisplay/KeywordTable.templ`~~ · ~~`internal/api/v1/JobStatus.go`~~ (removed) · `frontend/src/lib/components/KeywordTable.svelte` (new Wails equivalent)

---

## What It Does

Displays the processed keyword results as a styled table. Each row shows the original keyword data alongside the AI-generated evaluation and reasoning. Every row has a checkbox (checked by default) so the user can select which keywords to keep. A "Create Another Campaign" button resets the view back to the upload form.

---

## Why It Was Built

The AI evaluation is only useful if the user can read and act on it. The table provides a scannable overview of all keywords with their ratings, and checkboxes anticipate a future export feature where the user selects which rows to download.

---

## Implementation Checklist

### Initial implementation — (onboarded 2026-03-02)
- [x] `KeywordTable` Templ component accepts `header []string` and `rows [][]string`
- [x] Renders TemplUI table components for consistent styling
- [x] Per-row checkboxes, all checked by default
- [x] "Create Another Campaign" button with HTMX `hx-get` to reset to home form
- [x] Returned from `JobStatus.go` when job status is `completed`

---

## Technical Notes

- **Data shape:** The component receives the merged result: original headers + `["Evaluation", "Reasoning"]` appended. Original data rows + AI columns appended to each row. The component does not know or care which columns are AI-generated.
- **TemplUI components:** The table uses TemplUI's `table`, `table-header`, `table-row`, and `table-cell` components for base styling. Some custom CSS is applied on top via `customCSS.css`.
- **Checkboxes:** Currently cosmetic only — no form submit or download action is wired to them. They exist in anticipation of an export feature. See `docs/Architecture.md` Extensibility Roadmap → Priority 2 (`CSVDownloadSink`).
- **Reset flow:** The "Create Another Campaign" button uses HTMX to fetch the landing page partial and swap it into the main content area. The navbar and theme toggle remain untouched.
- **Pattern used:** HTMX partial rendering — `JobStatus.go` returns this component as an HTML fragment that HTMX swaps in place of the polling spinner.

---

## Change History

### 2026-03-02 — Overhaul to TemplUI components
**Reason:** Previous version used hand-styled HTML; replaced with TemplUI components for consistency with the rest of the UI.

**What changed:**
- [x] Replaced custom HTML table markup with TemplUI table components
- [x] Retained custom CSS overrides where TemplUI defaults did not fit

**Notes:** Some styling is still applied via `customCSS.css` — the TemplUI base styles handle the majority.
