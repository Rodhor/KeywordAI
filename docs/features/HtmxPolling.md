# HTMX Polling

> ⚠ **Superseded — 2026-03-13**
> The project migrated to Wails v2 + SvelteKit. HTMX polling was replaced by a JavaScript `setInterval` loop in `frontend/src/routes/keywords/+page.svelte`. The `PollingUI.templ`, `ErrorUI.templ`, and HTTP status handler no longer exist. This record is kept for historical context only. See `PollingSpinner.svelte` for the current implementation.

- **Status:** Removed
- **Created:** 2026-03-02 (onboarded — code predates this record)
- **Last updated:** 2026-03-13
- **Touches:** ~~`ui/customComponents/Loading/PollingUI.templ`~~ · ~~`ui/customComponents/Loading/ErrorUI.templ`~~ · ~~`internal/api/v1/keywords/status.go`~~ (all removed)

---

## What It Does

Shows a loading spinner after the user submits the keyword form. HTMX automatically polls the job status endpoint every second. When the job completes, the spinner is seamlessly replaced by the results table. If the job fails, an error fragment with a "Back to Home" button is shown instead, and polling stops.

---

## Why It Was Built

Gemini evaluation takes 5–30 seconds depending on keyword count. Without feedback the user sees nothing and cannot tell if the app is working or broken. The spinner communicates progress; automatic polling removes the need for a manual refresh.

---

## Implementation Checklist

### Initial implementation — (onboarded 2026-03-02)
- [x] `PollingUI` Templ component renders spinner with embedded `hx-trigger="every 1s"`
- [x] `hx-get` targets `/api/v1/submit/status?id={jobID}`
- [x] `hx-swap` replaces the spinner element with whatever the endpoint returns
- [x] `JobStatus.go` returns `PollingUI` again while status is `processing`
- [x] `JobStatus.go` returns `KeywordTable` when status is `completed`
- [x] `JobStatus.go` returns an error message when status is `failed`
- [x] Polling stops automatically when the returned HTML contains no `hx-trigger`

### Bug 1 fix + ErrorUI — 2026-03-07
- [x] `ErrorUI` Templ component created (`ui/customComponents/Loading/ErrorUI.templ`)
- [x] `ErrorUI` uses `id="job-display-node"` so HTMX `outerHTML` swap replaces the spinner
- [x] `ErrorUI` includes "Back to Home" button: `hx-get="/api/v1/home"` targeting `#main-content-area` with `hx-push-url="true"`
- [x] Status handler moved to `internal/api/v1/keywords/status.go` (P1e module restructure)
- [x] Failed job now returns HTTP 200 with `ErrorUI` instead of `http.Error` (500) — polling stops correctly

---

## Technical Notes

- **Why HTTP 200 for errors:** HTMX only performs DOM swaps on HTTP 2xx responses. A `http.Error(w, msg, 500)` call returns the error text as a plain string body but HTMX silently discards it — the spinner stays visible forever. Returning `ErrorUI` as HTTP 200 lets HTMX replace the spinner with the error fragment, which has no `hx-trigger`, so polling stops.
- **Self-stopping polling:** HTMX stops polling when the swapped-in HTML does not contain an `hx-trigger` attribute. Both `ResultsTable` and `ErrorUI` omit this attribute — polling ceases on either terminal state.
- **Job ID flow:** The submit handler embeds the job ID directly in the `PollingUI` component render call. The component hardcodes it into the `hx-get` URL. Each polling instance is scoped to exactly one job.
- **Pattern used:** Async job + HTMX partial rendering. This feature is the UI half of the async job pattern — it only works because `AsyncJobManager` tracks status independently of the HTTP request lifecycle.

---

## Change History

### 2026-03-07 — Bug 1 fix: spinner never stopped on job failure

**Reason:** `FailJob` was never called in `ProcessKeywords` (fixed in KeywordPipeline), and even when it was, the status handler returned `http.Error` (500) which HTMX ignores — the spinner looped forever.

**What changed:**
- [x] `ErrorUI` component created with correct `id` for HTMX swap target
- [x] Status handler returns HTTP 200 + `ErrorUI` for failed jobs instead of `http.Error`
- [x] Handler moved from `internal/api/v1/JobStatus.go` to `internal/api/v1/keywords/status.go` as part of P1e module restructure

**Notes:** The HTTP 200 requirement for HTMX swaps is a framework constraint, not a REST convention violation — the response is consumed entirely by HTMX and never seen as a "success" by the user.
