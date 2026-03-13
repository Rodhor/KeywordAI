# Feature — Keyword Clustering (CSV Download)

**Mode:** FEATURE
**Date:** 2026-03-09
**Scope:** Run AI clustering on the selected Good keywords, produce a downloadable CSV, show a success screen. No Google integration yet.

---

## What It Does

After the user selects keywords from the evaluation results table and submits, a background job runs an AI clustering step that groups each keyword under a thematic cluster name. When complete, the user sees a success screen with a button to download the clustered keywords as a CSV file.

---

## Where It Fits

Follows the **Async job with UUID + polling** pattern (Architecture.md Core Pattern 2).
Extends the existing `ProcessClustering` stub in `internal/processor/cluster.go`.
New download endpoint follows the **Module pattern** (Core Pattern 3) — registered in `keywords/module.go`, no changes to `server.go`.

---

## Files That Change

| File | Change |
|------|--------|
| `internal/ai/gemini.go` | Rewrite `OrganizeKeywords` — new system instruction, new schema, new prompt |
| `internal/processor/cluster.go` | Implement `ProcessClustering` — AI call + ZipResults + CompleteJob |
| `internal/api/v1/keywords/cluster_status.go` | Fill in completed state — assert `ClusterJobPayload`, render success UI |
| `internal/api/v1/keywords/cluster_download.go` | New file — serves clustered data as a CSV download |
| `internal/api/v1/keywords/module.go` | Register the new download route |
| `ui/customComponents/keywords/ClusterSuccess.templ` | New component — success message + download button + start-over button |

---

## Implementation Steps

### Step 1 — Rewrite `OrganizeKeywords` in `gemini.go`

The current implementation is a copy of `EvaluateKeywords` — wrong schema, wrong prompt, wrong system instruction.

**New system instruction** (separate from the evaluator's):
```
// The organizer groups keywords by search intent and theme.
// It returns only a "Cluster" column — one cluster name per row.
// Cluster names must be short (2–4 words), title-cased, and descriptive of intent.
// Group by user intent first (e.g. "Buy Running Shoes"), then topic.
// A keyword with no clear cluster gets "Other".
```

**New schema** — only one column, `"Cluster"`:
```go
schema := &genai.Schema{
    Type: genai.TypeObject,
    Properties: map[string]*genai.Schema{
        "headers": { Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString} },
        "rows": {
            Type: genai.TypeArray,
            Items: &genai.Schema{
                Type:  genai.TypeArray,
                Items: &genai.Schema{Type: genai.TypeString},
            },
        },
    },
    Required: []string{"headers", "rows"},
}
```

The AI returns just the delta — one `["Cluster name"]` per input row, in the same order.
`ProcessClustering` then calls `ZipResults` to merge with the original headers and rows, exactly as the evaluator does.

**New prompt** in `OrganizeKeywords`:
```go
prompt := fmt.Sprintf(
    "Group each keyword into a short thematic cluster based on search intent.\n"+
    "Return only a 'Cluster' header and one cluster name per row, in the same order as input.\n"+
    "Campaign context (optional): %s\n\nHeaders: %v\nRows: %v",
    desc, headers, rows,
)
```

The `desc` parameter accepts an empty string for now — no campaign dependency in this feature.

**Important:** The organizer needs its own `*genai.Client` config (its own `GenerateContentConfig` with the clustering system instruction). The simplest approach: give `gemini.Client` a second config field `organizerConfig *genai.GenerateContentConfig` populated in `NewClient`, and use it in `OrganizeKeywords` instead of `c.config`.

---

### Step 2 — Implement `ProcessClustering`

Pattern: identical structure to `ProcessKeywords`, but simpler (no pipeline, just one AI call).

```go
// Why: same error-handling contract as every other processor — FailJob on every path
func ProcessClustering(params ClusterProcessParams) {
    newHeaders, newRows, err := params.Organizer.OrganizeKeywords(
        context.Background(), "", params.Headers, params.Rows,
    )
    if err != nil {
        params.Manager.FailJob(params.JobID, err.Error())
        return
    }

    finalHeaders, finalRows := ZipResults(params.Headers, params.Rows, newHeaders, newRows)

    params.Manager.CompleteJob(params.JobID, ClusterJobPayload{
        Header:       finalHeaders,
        Content:      finalRows,
        OriginalRows: params.OriginalRows,
    })
}
```

`ZipResults` is already in `internal/processor/ProcessAIResponse.go` — reuse it directly.

---

### Step 3 — Fix `cluster_status.go` completed state

Two fixes:

1. Assert `ClusterJobPayload` (not `KeywordJobPayload` — that was the wrong type).
2. Render the success component instead of discarding the payload.

```go
if job.Status == jobs.StatusCompleted {
    payload, ok := job.Payload.(processor.ClusterJobPayload)  // correct type
    if !ok {
        // ... error handling unchanged
    }
    _ = payload  // replace this line:
    keywordsui.ClusterSuccess(jobID).Render(r.Context(), w)  // ← render success UI
    return
}
```

---

### Step 4 — New download handler `cluster_download.go`

Endpoint: `GET /api/v1/submit/cluster/download?id=<jobID>`

```go
// Why: the browser hits this URL directly (not via HTMX) — normal file download
func clusterDownload(jobMgr *jobs.Manager, w http.ResponseWriter, r *http.Request) {
    jobID := r.URL.Query().Get("id")
    job, exists := jobMgr.GetJob(jobID)
    // ... not found → 404

    payload, ok := job.Payload.(processor.ClusterJobPayload)
    // ... type error → 500

    w.Header().Set("Content-Type", "text/csv")
    w.Header().Set("Content-Disposition", `attachment; filename="clustered-keywords.csv"`)

    writer := csv.NewWriter(w)
    writer.Write(payload.Header)     // header row
    writer.WriteAll(payload.Content) // data rows
    writer.Flush()
}
```

Uses `encoding/csv` from the standard library — no new dependency.

---

### Step 5 — New `ClusterSuccess.templ`

Location: `ui/customComponents/keywords/ClusterSuccess.templ`

```
ClusterSuccess(jobID string)
  Card:
    ✓ icon + "Clustering complete" heading
    "Your keywords have been grouped and are ready to download."
    [Download CSV]  → plain <a href="/api/v1/submit/cluster/download?id={jobID}" download>
    [Create Another Campaign]  → hx-get="/api/v1/home" hx-target="#main-content-area"
```

The download link is a plain `<a>` tag with the `download` attribute, not an HTMX call. The browser handles the file download natively — no swap needed.

---

### Step 6 — Register route in `module.go`

```go
mux.HandleFunc("/submit/cluster/download", func(w http.ResponseWriter, r *http.Request) {
    clusterDownload(m.JobMgr, w, r)
})
```

---

## One Decision to Flag

`OrganizeKeywords` currently shares the evaluator's `c.config` (system instruction + schema). The two configs are incompatible — the organizer must have its own. Two options:

- **A (recommended):** Add `organizerConfig *genai.GenerateContentConfig` to `ai.Client`, build it in `NewClient` alongside the existing `config`. Clean separation, no behaviour change for the evaluator.
- **B:** Make `OrganizeKeywords` build its config inline on every call. Simpler but allocates on each request.

Option A is the right long-term shape — the client already holds state, and two configs follow the same pattern.

---

## How to Test

1. Upload a CSV, submit for evaluation, select some Good keywords, click "Proceed to Clustering".
2. Polling spinner should appear. Watch the server logs for the Gemini call.
3. On completion: success screen with "Download CSV" link.
4. Click the link — browser downloads `clustered-keywords.csv`.
5. Open CSV: original keyword columns + new "Cluster" column with theme names.
6. "Create Another Campaign" returns to the home screen.
