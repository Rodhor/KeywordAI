# Feature — Checkbox Submission

**Mode:** FEATURE
**Date:** 2026-03-07
**Depends on:** Brainstorm-NextSteps.md
**Builds toward:** KeywordClustering feature

---

## What This Feature Does

Makes the checkboxes in the results table functional. After reviewing AI-evaluated keywords, the user selects the rows they want to keep, clicks "Proceed to Clustering", and the selected keywords are submitted to a new async clustering job. The HTMX polling pattern is reused — the user sees the spinner and then the clustering result.

This is the bridge between Job 1 (evaluation) and Job 2 (clustering). Without it, the results table is a dead end.

---

## Where It Fits in Architecture.md

- Follows **Core Pattern 2** (async job + HTMX polling) — new job, same pattern
- Follows **Core Pattern 3** (module registration) — new routes added to `keywords/module.go`, new dep threaded from `main.go`
- Follows **Core Pattern 1** (HTMX + Templ partials) — form submission returns `PollingUI`

---

## UI Overview

### What the results table becomes

Visually the table looks nearly identical to today. The changes are functional:

**Checkbox column**
All rows start pre-checked. The user unchecks keywords they want to discard before clustering. Each checkbox submits a row index as its value — this is invisible to the user but lets the server know which rows were selected.

**Action area below the table**
Two buttons sit below the card:

| Button | Style | Behaviour |
|--------|-------|-----------|
| Proceed to Clustering | Primary (filled) | Submits the form via HTMX. The entire main content area is replaced by the clustering spinner. |
| Create Another Campaign | Ghost (outline/text) | Navigates back to the upload form. Must NOT submit the form — requires `type="button"` attribute explicitly set. |

Without `type="button"`, the secondary button will trigger a form submit in HTML by default.

**Zero selection error**
If the user unchecks all rows and clicks "Proceed to Clustering", the server rejects the request and returns an error fragment inline. No clustering job is started. The error message should explain that at least one keyword is required. The "Create Another Campaign" option in the error fragment still works normally.

**Clustering spinner**
The same `PollingUI` spinner component as during evaluation, but the copy should distinguish this step (e.g. "Organising keywords into clusters..."). It polls a different URL: `/api/v1/submit/cluster/status`.

**Cluster result (stub during this feature)**
During this feature, the cluster result is the ErrorUI showing "Clustering not yet implemented". This is intentional — it confirms the full UI path works before the AI clustering logic is built.

---

### What the cluster results view should look like

This is the final destination of the whole flow. Even though the actual render is wired up in the KeywordClustering feature, the design is decided here so you build toward something specific.

**What data comes back**

`KeywordOrganizer.OrganizeKeywords` returns the same flat `(newHeaders, newRows)` shape as the evaluator — a table where each row gets an extra column, e.g. `"Category"`. A typical result looks like:

```
Keyword              | Match Type | Evaluation | Reasoning       | Category
---------------------|------------|------------|-----------------|------------------
buy running shoes    | Exact      | Good       | High intent...  | Purchase Intent
best running shoes   | Phrase     | Good       | Comparison...   | Research
nike running shoes   | Broad      | Good       | Brand search... | Brand
```

The data is flat. The Category column is what lets you group the display.

**Recommended display: grouped cards by cluster**

Rather than another flat table, group by the Category column. Each cluster becomes its own card with a title and a compact list of its keywords. This makes the organisation immediately legible — the whole point of clustering is to see groups, not a flat list.

```
┌─────────────────────────────────────────────┐
│ Purchase Intent                    3 keywords│
│─────────────────────────────────────────────│
│  buy running shoes                           │
│  order running shoes online                  │
│  running shoes cheap buy                     │
└─────────────────────────────────────────────┘

┌─────────────────────────────────────────────┐
│ Research                           2 keywords│
│─────────────────────────────────────────────│
│  best running shoes                          │
│  top rated running shoes 2024               │
└─────────────────────────────────────────────┘
```

Each card:

- Header: cluster name (from the Category column) + keyword count badge
- Body: one row per keyword — show the keyword text and optionally the match type
- No need to show Evaluation/Reasoning here — those served their purpose in step 1

**Grouping the data**

The AI returns a flat `[][]string`. Before rendering, you need to group it by the Category column. Do this in the `clusterStatus` handler (Go side) before passing to the template — not in the template itself. A simple loop that builds a `map[string][][]string` (category name → rows) is enough. Convert the map to an ordered slice for stable display order.

```go
// In the status handler, before rendering:
groups := make(map[string][][]string)
order := []string{}
for _, row := range payload.Content {
    category := row[categoryColIndex]
    if _, seen := groups[category]; !seen {
        order = append(order, category)
    }
    groups[category] = append(groups[category], row)
}
// Pass `order` + `groups` to the template
```

**Page layout**

```
[ Clustered Keywords          12 keywords across 4 clusters ]

[ Purchase Intent — 3 ]   [ Research — 2 ]
[ Brand — 4 ]             [ Generic — 3 ]

                    [ Export to Google Sheets ]   ← future
                    [ Create Another Campaign ]
```

- A summary line at the top: total keywords, number of clusters
- Cards in a responsive grid (2 columns on wide screens, 1 on narrow)
- Two buttons at the bottom: export (future, greyed out / not yet built) and reset

**Fallback: flat table**

If the grouped card layout feels complex to build alongside the AI wiring, a flat table identical to the evaluation results table works fine as a first pass — just add the Category column to the existing `ResultsTable`. The grouped view can be layered on top once the AI is returning real clusters. Note this in the feature record so the decision is tracked.

---

## Code Patterns

### Form submission with HTMX

```html
<!-- Why: The form wraps the whole table so checkboxes are included in the POST body.
     hx-post fires on submit. hx-target replaces the entire content area so the spinner
     takes over the full view — same visual transition as the evaluation step. -->
<form
    id="cluster-form"
    hx-post="/api/v1/submit/cluster"
    hx-target="#main-content-area"
    hx-swap="innerHTML"
>
    <input type="hidden" name="eval_job_id" value={ params.JobID }/>
    <!-- ... table ... -->

    <!-- "type=submit" is the default for buttons inside a form -->
    <button type="submit">Proceed to Clustering</button>

    <!-- Must be type="button" — without this it submits the form -->
    <button type="button" hx-get="/api/v1/home" hx-target="#main-content-area">
        Create Another Campaign
    </button>
</form>
```

### Checkbox value carries the row index

```html
<!-- Why: The server needs to know which rows were selected.
     Submitting the index is simpler and safer than submitting keyword text
     (which may contain commas, quotes, or other characters that complicate parsing).
     The handler looks up rows from the job payload using these indices. -->
<input
    type="checkbox"
    name="selected"
    value="3"      <!-- row index, set by the template loop variable -->
    checked
/>
```

### PollingUI with configurable status URL

```go
// Why: The cluster job polls a different endpoint than the evaluation job.
// Adding statusURL as a parameter makes PollingUI reusable for any async job.
templ PollingUI(jobID string, statusURL string) {
    hx-get={ fmt.Sprintf("%s?id=%s", statusURL, jobID) }
```

### Cluster handler reads job payload from the manager

```go
// Why: The form does not send row data — only the eval job ID and selected indices.
// The server fetches the full payload from the in-memory job manager.
// This keeps the form small and avoids encoding/decoding row data as form fields.
evalJob, exists := jobMgr.GetJob(evalJobID)
payload, ok := evalJob.Payload.(processor.KeywordJobPayload)
// filter payload.Content to selectedIndices → start cluster job
```

---

## Implementation Steps

Work through these in order. Each step keeps the build passing before the next.

---

### Step 1 — Add stub `OrganizeKeywords` to `internal/ai/gemini.go`

The `*ai.Client` type must satisfy the `ai.KeywordOrganizer` interface before it can be passed to the cluster module. The real clustering implementation comes in the KeywordClustering feature. For now, add a method that returns its inputs unchanged:

```go
func (c *Client) OrganizeKeywords(ctx context.Context, description string, headers []string, rows [][]string) ([]string, [][]string, error) {
    return headers, rows, nil
}
```

This is the only step that touches `gemini.go`. Do not change anything else there.

---

### Step 2 — Expand types in `internal/processor/types.go`

**Add `OriginalRows` to `KeywordJobPayload`:**

```go
type KeywordJobPayload struct {
    Header       []string
    Content      [][]string
    OriginalRows [][]string   // rows before any pipeline step ran
}
```

**Add the cluster result types:**

```go
type ClusterJobPayload struct {
    Header       []string     // headers from the evaluation step
    Content      [][]string   // rows with added Category/Cluster column
    OriginalRows [][]string   // carried through from evaluation
}
```

`ClusterJobPayload` uses the same flat `[][]string` shape as `KeywordJobPayload` because `KeywordOrganizer.OrganizeKeywords` returns `(newHeaders, newRows)` — a flat table with an extra column added, not a nested structure. This keeps the results table pattern consistent across both jobs.

---

### Step 3 — Populate `OriginalRows` in `internal/processor/keywords.go`

In `ProcessKeywords`, after `ExtractHeaderRow` and before `pipeline.Run`, capture the original content:

```go
header, content := ExtractHeaderRow(keywords)

// Capture before pipeline mutates content.
// copy() is a shallow copy — it copies row slice headers, not cell values.
// This is safe because pipeline steps replace whole rows, never mutate cells.
originalRows := make([][]string, len(content))
copy(originalRows, content)

ctx := &pipeline.Context{Headers: header, Rows: content}
```

Then in the `CompleteJob` call, add the field:

```go
result := KeywordJobPayload{
    Header:       ctx.Headers,
    Content:      ctx.Rows,
    OriginalRows: originalRows,
}
```

---

### Step 4 — Create stub `internal/processor/cluster.go`

New file. Define the params struct and a stub processor that fails immediately with a clear message. This is intentional — it allows the full UI flow to be tested before the AI logic exists.

```go
package processor

import (
    "Demo/internal/ai"
    "Demo/internal/jobs"
)

type ClusterProcessParams struct {
    JobID        string
    Headers      []string
    Rows         [][]string       // selected (Good) rows from evaluation
    OriginalRows [][]string       // full original list from evaluation job
    Organizer    ai.KeywordOrganizer
    Manager      *jobs.Manager
}

func ProcessClustering(params ClusterProcessParams) {
    // Stub: always fail with a visible message.
    // Replace with real implementation in the KeywordClustering feature.
    params.Manager.FailJob(params.JobID, "Clustering not yet implemented")
}
```

---

### Step 5 — Add `statusURL` parameter to `ui/customComponents/shared/PollingUI.templ`

Change the signature and the `hx-get` attribute:

```go
// Before
templ PollingUI(jobID string) {
    hx-get={ fmt.Sprintf("/api/v1/submit/status?id=%s", jobID) }

// After
templ PollingUI(jobID string, statusURL string) {
    hx-get={ fmt.Sprintf("%s?id=%s", statusURL, jobID) }
```

No other changes to this file. The spinner copy ("Processing Job", "Analysing data...") is fine as-is for now.

---

### Step 6 — Update `PollingUI` call in `internal/api/v1/keywords/submit.go`

This file calls `PollingUI` when it first starts the evaluation job. Add the status URL:

```go
// Before
shared.PollingUI(jobID).Render(r.Context(), w)

// After
shared.PollingUI(jobID, "/api/v1/submit/status").Render(r.Context(), w)
```

---

### Step 7 — Update `internal/api/v1/keywords/status.go`

Two changes in this file:

**1. Update the `PollingUI` call** (evaluation is still processing — keeps polling):

```go
// Before
shared.PollingUI(jobID).Render(r.Context(), w)

// After
shared.PollingUI(jobID, "/api/v1/submit/status").Render(r.Context(), w)
```

**2. Pass `JobID` to `ResultsTableParams`** (so the form knows which job to reference):

```go
keywordsui.ResultsTable(keywordsui.ResultsTableParams{
    Headers:      payload.Header,
    Rows:         payload.Content,
    WithCheckbox: true,
    JobID:        jobID,      // ← added
})
```

`jobID` is already in scope — it's read from the URL query param at the top of the handler.

---

### Step 8 — Update `ui/customComponents/keywords/KeywordTable.templ`

This is the largest single change. Four things to do:

**1. Add `JobID string` to `ResultsTableParams`:**

```go
type ResultsTableParams struct {
    Headers      []string
    Rows         [][]string
    Footer       []string
    WithFooter   bool
    WithCheckbox bool
    JobID        string    // ← added
}
```

**2. Wrap the card in a `<form>` with HTMX attributes and hidden input:**

```html
<form
    id="cluster-form"
    hx-post="/api/v1/submit/cluster"
    hx-target="#main-content-area"
    hx-swap="innerHTML"
>
    <input type="hidden" name="eval_job_id" value={ params.JobID }/>

    <!-- existing @card.Card(...) content goes here unchanged -->

    <!-- button row below the card (replaces the existing single button) -->
</form>
```

**3. Update checkboxes in `ResultsTableRow` to carry name and value:**

Check whether `checkbox.Props` supports `Name` and `Value` fields by reading `ui/components/checkbox/`. If it does, use them. If not, render a raw `<input type="checkbox" .../>` instead.

The checkbox needs:

- `name="selected"` — this is the POST field name
- `value=<row_index>` — the index of this row

```go
// If checkbox.Props supports these fields:
@checkbox.Checkbox(checkbox.Props{
    ID:      fmt.Sprintf("checkbox-%d", index),
    Name:    "selected",
    Value:   fmt.Sprintf("%d", index),
    Checked: true,
})

// If it does not, fall back to raw HTML:
<input
    type="checkbox"
    id={ fmt.Sprintf("checkbox-%d", index) }
    name="selected"
    value={ fmt.Sprintf("%d", index) }
    checked
    class="..."
/>
```

**4. Replace the single "Create Another Campaign" button with two buttons:**
Value

```go
// Primary: submits the form
@button.Button(button.Props{
    Variant: button.VariantDefault,
    Attributes: templ.Attributes{"type": "submit"},
}) {
    Proceed to Clustering
}

// Secondary: navigates away — must NOT submit the form
@button.Button(button.Props{
    Variant: button.VariantGhost,
    Attributes: templ.Attributes{
        "type":      "button",               // prevents form submission
        "hx-get":    "/api/v1/home",
        "hx-target": "#main-content-area",
    },
}) {
    Create Another Campaign
}
```

---

### Step 9 — Create `internal/api/v1/keywords/cluster.go`

New handler file. Pattern matches `submit.go`.

```go
package keywords

import (
    "Demo/internal/ai"
    "Demo/internal/jobs"
    "Demo/internal/processor"
    "Demo/ui/customComponents/shared"
    "log/slog"
    "net/http"
    "strconv"
)

func cluster(jobMgr *jobs.Manager, organizer ai.KeywordOrganizer, w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
        return
    }

    // 1. Retrieve the eval job payload from the manager
    evalJobID := r.FormValue("eval_job_id")
    if evalJobID == "" {
        w.Header().Set("Content-Type", "text/html")
        shared.ErrorUI("Missing evaluation job reference.").Render(r.Context(), w)
        return
    }

    evalJob, exists := jobMgr.GetJob(evalJobID)
    if !exists || evalJob.Status != jobs.StatusCompleted {
        w.Header().Set("Content-Type", "text/html")
        shared.ErrorUI("Evaluation job not found or not complete.").Render(r.Context(), w)
        return
    }

    payload, ok := evalJob.Payload.(processor.KeywordJobPayload)
    if !ok {
        slog.Error("Type assertion failed for eval job payload", "id", evalJobID)
        http.Error(w, "Internal data error", http.StatusInternalServerError)
        return
    }

    // 2. Parse selected row indices from form
    if err := r.ParseForm(); err != nil {
        http.Error(w, "Bad request", http.StatusBadRequest)
        return
    }
    selectedIndices := r.Form["selected"]

    // 3. Validate at least one row is selected
    if len(selectedIndices) == 0 {
        w.Header().Set("Content-Type", "text/html")
        shared.ErrorUI("Please select at least one keyword before proceeding.").Render(r.Context(), w)
        return
    }

    // 4. Filter payload rows to selected indices only
    selectedRows := make([][]string, 0, len(selectedIndices))
    for _, idxStr := range selectedIndices {
        idx, err := strconv.Atoi(idxStr)
        if err != nil || idx < 0 || idx >= len(payload.Content) {
            continue
        }
        selectedRows = append(selectedRows, payload.Content[idx])
    }

    // 5. Start clustering job
    clusterJobID := jobMgr.CreateJob()
    go processor.ProcessClustering(processor.ClusterProcessParams{
        JobID:        clusterJobID,
        Headers:      payload.Header,
        Rows:         selectedRows,
        OriginalRows: payload.OriginalRows,
        Organizer:    organizer,
        Manager:      jobMgr,
    })

    // 6. Return polling UI — note the different status URL
    w.Header().Set("Content-Type", "text/html")
    shared.PollingUI(clusterJobID, "/api/v1/submit/cluster/status").Render(r.Context(), w)
}
```

---

### Step 10 — Create `internal/api/v1/keywords/cluster_status.go`

New handler file. Pattern matches `status.go`.

```go
package keywords

import (
    "Demo/internal/jobs"
    "Demo/internal/processor"
    "Demo/ui/customComponents/shared"
    "fmt"
    "log/slog"
    "net/http"
)

func clusterStatus(jobMgr *jobs.Manager, w http.ResponseWriter, r *http.Request) {
    jobID := r.URL.Query().Get("id")

    job, exists := jobMgr.GetJob(jobID)
    if !exists {
        http.Error(w, "Job not found", http.StatusNotFound)
        return
    }

    w.Header().Set("Content-Type", "text/html")

    if job.Status == jobs.StatusProcessing {
        shared.PollingUI(jobID, "/api/v1/submit/cluster/status").Render(r.Context(), w)
        return
    }

    if job.Status == jobs.StatusCompleted {
        payload, ok := job.Payload.(processor.ClusterJobPayload)
        if !ok {
            slog.Error("Type assertion failed for cluster job",
                "id", jobID,
                "got", fmt.Sprintf("%T", job.Payload),
            )
            http.Error(w, "Internal data error", http.StatusInternalServerError)
            return
        }
        // TODO: render cluster results table (built in KeywordClustering feature)
        _ = payload
        return
    }

    if job.Status == jobs.StatusFailed {
        shared.ErrorUI(job.Error).Render(r.Context(), w)
    }
}
```

---

### Step 11 — Update `internal/api/v1/keywords/module.go`

Add the `Organizer` field and register the two new routes:

```go
type Module struct {
    JobMgr    *jobs.Manager
    Evaluator ai.KeywordEvaluator
    Organizer ai.KeywordOrganizer   // ← added
}

func (m Module) Register(mux *http.ServeMux) {
    mux.HandleFunc("/submit/keywordupload", func(w http.ResponseWriter, r *http.Request) {
        submit(m.JobMgr, m.Evaluator, w, r)
    })
    mux.HandleFunc("/submit/status", func(w http.ResponseWriter, r *http.Request) {
        status(m.JobMgr, w, r)
    })
    mux.HandleFunc("/submit/cluster", func(w http.ResponseWriter, r *http.Request) {
        cluster(m.JobMgr, m.Organizer, w, r)         // ← new
    })
    mux.HandleFunc("/submit/cluster/status", func(w http.ResponseWriter, r *http.Request) {
        clusterStatus(m.JobMgr, w, r)                // ← new
    })
}
```

---

### Step 12 — Update `internal/api/v1/v1.go`

Add `organizer ai.KeywordOrganizer` to `Modules()` and pass it to `keywords.Module`:

```go
// Before
func Modules(jobMgr *jobs.Manager, evaluator ai.KeywordEvaluator) []module.Module {
    return []module.Module{
        ...
        keywords.Module{JobMgr: jobMgr, Evaluator: evaluator},
    }
}

// After
func Modules(jobMgr *jobs.Manager, evaluator ai.KeywordEvaluator, organizer ai.KeywordOrganizer) []module.Module {
    return []module.Module{
        ...
        keywords.Module{JobMgr: jobMgr, Evaluator: evaluator, Organizer: organizer},
    }
}
```

---

### Step 13 — Update `internal/services/server.go`

Add `Organizer` to `ServerParams` and `routesParams`, and thread it to `apiv1.Modules`:

```go
type ServerParams struct {
    Port      string
    JobMgr    *jobs.Manager
    Evaluator ai.KeywordEvaluator
    Organizer ai.KeywordOrganizer   // ← added
}

type routesParams struct {
    jobMgr    *jobs.Manager
    evaluator ai.KeywordEvaluator
    organizer ai.KeywordOrganizer   // ← added
}

// In StartServer, pass it through:
setupRoutes(routesParams{
    jobMgr:    params.JobMgr,
    evaluator: params.Evaluator,
    organizer: params.Organizer,   // ← added
})

// In setupRoutes, pass it to Modules:
apiv1.Modules(params.jobMgr, params.evaluator, params.organizer)
```

---

### Step 14 — Update `cmd/main.go`

`aiClient` already satisfies both `ai.KeywordEvaluator` and `ai.KeywordOrganizer` (after Step 1 adds the stub method). Pass it as both:

```go
services.StartServer(services.ServerParams{
    Port:      ":8090",
    JobMgr:    jobMgr,
    Evaluator: aiClient,
    Organizer: aiClient,   // ← added — same client satisfies both interfaces
})
```

---

### Step 15 — Run `templ generate`

```
templ generate
```

Run this from the project root. Any `.templ` files changed in Steps 5 and 8 need their `_templ.go` files regenerated. The build will fail without this step.

---

### Step 16 — `go build ./...`

```
go build ./...
```

Must pass with zero errors before testing. Common issues at this step:

- Missing import for `ai` package in `server.go` or `v1.go` if the `Organizer` type wasn't imported yet
- `checkbox.Props` missing `Name`/`Value` fields if the TemplUI checkbox doesn't support them (fall back to raw `<input>` as described in Step 8)

---

### Step 17 — End-to-end verification

1. Start the app and upload a CSV, submit for evaluation as normal.
2. When the results table appears, verify checkboxes are rendered and pre-checked.
3. Uncheck a few rows.
4. Click "Proceed to Clustering" — the spinner should appear with a new job ID.
5. After ~1 second, the error fragment should appear: "Clustering not yet implemented". This is the expected and correct outcome at this stage.
6. Click "Create Another Campaign" — verify it returns to the upload form without errors.
7. Reload the page and repeat, but uncheck all rows before clicking submit. Verify the inline error appears immediately and no spinner is shown (no cluster job was started).

---

## What This Feature Does NOT Do

- Implement actual clustering logic — deferred to the KeywordClustering feature
- Filter checkboxes by Evaluation value automatically — all rows pre-checked for now; a user can manually uncheck
- Render cluster results — the `clusterStatus` handler has a `TODO` placeholder
- Export data — deferred to GoogleSheetsExport
- Persist job results across restarts — in-memory only throughout this feature
