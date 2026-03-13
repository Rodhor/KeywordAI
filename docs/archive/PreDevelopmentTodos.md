# Pre-Development TODOs

Items to resolve before adding new features. Completing these in order prevents refactoring half-finished work mid-feature.

---

## Master Checklist

### Bugs (fix first — active production breakage)

- [x] **Bug 1** — `FailJob` never called on pipeline error → job spins forever
- [x] **Bug 2** — Missing `return` after `renderStatus` error calls in upload handler → execution continues past errors
- [x] **Bug 3** — `http.MaxBytesReader` not set → 10 MB parse limit is unenforced

### Priority 1 — Refactors (unblock future features)

- [x] **P1a** — Pack `ProcessKeywords` 8 parameters into `KeywordJobInput` struct (depends on P1b + P1c)
- [x] **P1b** — Change `Server` + handlers to use `ai.KeywordEvaluator` interface instead of `*ai.Client`
- [x] **P1c** — Create `KeywordSource` interface + `CSVFileSource` implementation
- [x] **P1d** — Create composable pipeline with `Step` interface (depends on P1a)
- [x] **P1e** — Module-based route registration (depends on P1a)

### Quick Wins (no dependencies)

- [x] **QW1** — Wire `customInstructions` to `EvaluateKeywords` (depends on P1a)
- [x] **QW2** — Delete `internal/store/jobs.go` (dead code)

---

## Recommended Implementation Order

```
Bug 1  →  Bug 2  →  Bug 3   (independent, tackle sequentially — each is a small targeted fix)
P1b  +  P1c                 (independent of each other, can be done in parallel)
P1a                         (depends on P1b and P1c being done)
P1d  +  P1e  +  QW1        (all depend on P1a, then run independently)
QW2                         (anytime)
```

---

## Bug 1 — `FailJob` never called on pipeline error

**File:** [internal/processor/keywords.go](internal/processor/keywords.go)

**Problem:** `ProcessKeywords` calls `manager.FailJob(jobID, err)` nowhere. On any error (AI call failure, zip failure, etc.) the job stays in `StatusProcessing` forever. The HTMX spinner keeps polling with no way to stop — the user must reload the page.

**Where the bug is:**

```go
// Current — error is logged and returned, but the job is never marked failed
results, err := SendToAIForValidation(...)
if err != nil {
    log.Printf("...")
    return
}
```

**Fix:** Call `manager.FailJob(jobID, err)` on every early-return error path, then `manager.CompleteJob(jobID, result)` on success.

```go
results, err := SendToAIForValidation(...)
if err != nil {
    log.Printf("...")
    manager.FailJob(jobID, err)
    return
}

// ... (all other error paths get the same treatment)

manager.CompleteJob(jobID, result)
```

**Check:** After the fix, cause an intentional AI failure (bad API key). The HTMX spinner should be replaced by the error message fragment returned by `JobStatus.go`.

---

## Bug 2 — Missing `return` after `renderStatus` error calls

**File:** [internal/api/v1/upload_csv.go](internal/api/v1/upload_csv.go)

**Problem:** Two blocks in the upload handler call `renderStatus(false, ...)` to send an error response to the client, but then **continue executing** because there is no `return` after the call. On the `os.Create` failure path, the handler then tries to `io.Copy` into a nil file handle. On the `io.Copy` failure path, the handler continues as if the file was written successfully.

**Where the bug is (approximately lines 83–91):**

```go
file, err := os.Create(tmpPath)
if err != nil {
    renderStatus(false, "Failed to save file")   // ← response sent
    // ← MISSING return — continues to io.Copy(file, ...) with nil file
}

_, err = io.Copy(file, src)
if err != nil {
    renderStatus(false, "Failed to write file")  // ← response sent
    // ← MISSING return — continues as if write succeeded
}
```

**Fix:** Add `return` after every `renderStatus(false, ...)` call inside the error handling blocks.

```go
file, err := os.Create(tmpPath)
if err != nil {
    renderStatus(false, "Failed to save file")
    return
}

_, err = io.Copy(file, src)
if err != nil {
    renderStatus(false, "Failed to write file")
    return
}
```

**Also check:** The `renderStatus` closure itself calls `.Render(w)` but does not check the render error. This is lower priority but should be noted — a render failure will silently produce an empty or partial HTTP response.

---

## Bug 3 — 10 MB file size limit is not hard-enforced

**File:** [internal/api/v1/upload_csv.go](internal/api/v1/upload_csv.go)

**Problem:** `r.ParseMultipartForm(10 << 20)` uses 10 MB as the in-memory buffer size, but does **not** cap the total request body size. A client can send an arbitrarily large body. `http.MaxBytesReader` wraps the request body before parsing and causes `ParseMultipartForm` to return an error once the limit is exceeded.

**Fix:** Wrap the request body before calling `ParseMultipartForm`.

```go
r.Body = http.MaxBytesReader(w, r.Body, 10<<20)

if err := r.ParseMultipartForm(10 << 20); err != nil {
    renderStatus(false, "File too large (max 10 MB)")
    return
}
```

`MaxBytesReader` causes the read to fail with a `*http.MaxBytesError` once the limit is exceeded, which `ParseMultipartForm` surfaces as an error. This turns a soft hint into a hard reject.

---

## P1b — Use `ai.KeywordEvaluator` interface instead of `*ai.Client`

**Files to change:**

- [internal/services/server.go](internal/services/server.go) — `Server` struct field
- [internal/api/v1/KeyWordsSubmit.go](internal/api/v1/KeyWordsSubmit.go) — handler receives `*ai.Client`
- [internal/processor/keywords.go](internal/processor/keywords.go) — `ProcessKeywords` parameter

**Problem:** The `ai.go` file already defines the clean `KeywordEvaluator` interface, but the `Server` struct and every handler holds `*ai.Client` (the concrete Gemini type). This makes swapping providers (e.g. OpenAI) impossible without changing call sites.

**Fix:**

In `server.go`, change the field type:

```go
// Before
type Server struct {
    AIClient *ai.Client
    ...
}

// After
type Server struct {
    AIClient ai.KeywordEvaluator
    ...
}
```

Propagate through the handler and `ProcessKeywords`. The Gemini `*ai.Client` satisfies `ai.KeywordEvaluator` already, so no other changes are needed at this step.

**Verify:** The app should build and behave identically. The only change is the static type at the declaration sites.

---

## P1c — Create `KeywordSource` interface + `CSVFileSource`

**Create:** `internal/source/source.go`

**Problem:** The upload handler writes a tmp file and passes its path around as a raw `string`. The processor calls `os.Open(path)` directly, and there is no defined contract for how keyword data enters the system. Adding non-CSV sources (clipboard, direct paste, database) requires modifying the processor.

**Fix:** Define a thin interface:

```go
package source

import "io"

type KeywordSource interface {
    Open() (io.ReadCloser, error)
    Cleanup() error
}
```

Implement it for the CSV case:

```go
type CSVFileSource struct {
    Path string
}

func (s *CSVFileSource) Open() (io.ReadCloser, error) {
    return os.Open(s.Path)
}

func (s *CSVFileSource) Cleanup() error {
    return os.Remove(s.Path)
}
```

The `os.Remove(tmpPath)` call currently scattered in the processor moves into `CSVFileSource.Cleanup()`, which `ProcessKeywords` calls via `defer src.Cleanup()`.

---

## P1a — Pack `ProcessKeywords` parameters into `KeywordJobInput`

**File:** [internal/processor/keywords.go](internal/processor/keywords.go)

**Depends on:** P1b (so the struct can hold `ai.KeywordEvaluator`) and P1c (so it can hold `source.KeywordSource`)

**Problem:** `ProcessKeywords` currently takes 8 positional parameters. Adding any new input (custom instructions, user ID, campaign settings) means adding another parameter and updating every call site.

**Fix:** Define an input struct (location: `internal/processor/keywords.go` or a new `internal/processor/input.go`):

```go
type KeywordJobInput struct {
    JobID              string
    Source             source.KeywordSource
    ExcludeKeywords    []string
    CampaignDesc       string
    CustomInstructions string
    Evaluator          ai.KeywordEvaluator
    Manager            *jobs.Manager
}
```

Change the signature:

```go
// Before
func ProcessKeywords(jobID string, filePath string, excludeKeywords []string, ...) { ... }

// After
func ProcessKeywords(input KeywordJobInput) { ... }
```

Update the single call site in `KeyWordsSubmit.go`.

**Note:** `KeywordJobPayload` (currently defined in `internal/jobs/manager.go`) should either be removed or kept only as the async-manager's internal type. `KeywordJobInput` is the processor's concern; the manager's payload is the manager's concern.

---

## P1d — Composable pipeline with `Step` interface

**Create:** `internal/pipeline/pipeline.go`

**Depends on:** P1a (so steps can receive `KeywordJobInput` or a derived context struct)

**Problem:** `ProcessKeywords` is one large function. Adding, removing, or reordering steps (e.g. a deduplication step, a column-filter step) requires editing the function body. `DropUnwantedColumns` is already a dead no-op because there is nowhere clean to plug it in.

**Fix:** Define a step interface:

```go
package pipeline

type Context struct {
    Headers []string
    Rows    [][]string
}

type Step interface {
    Run(ctx *Context) error
}
```

Wrap existing cleaning functions as steps:

```go
type RemoveEmptyStep struct{}
func (s RemoveEmptyStep) Run(ctx *Context) error { ... }

type DeduplicateStep struct{}
func (s DeduplicateStep) Run(ctx *Context) error { ... }

type RemoveExcludedStep struct{ Excluded []string }
func (s RemoveExcludedStep) Run(ctx *Context) error { ... }
```

Add a runner:

```go
func Run(ctx *Context, steps []Step) error {
    for _, step := range steps {
        if err := step.Run(ctx); err != nil {
            return err
        }
    }
    return nil
}
```

`ProcessKeywords` builds the step slice and calls `pipeline.Run`. Adding a new cleaning step is now a one-liner in the slice.

---

## P1e — Module-based route registration

**File:** [internal/services/server.go](internal/services/server.go)

**Depends on:** P1a (so modules can receive the input struct rather than individual fields)

**Problem:** `registerV1Routes` is a long hardcoded function. Every new endpoint requires editing it. There is no pattern for grouping related routes or mounting feature modules independently.

**Fix:** Define a module interface:

```go
package module

type Module interface {
    Register(mux *http.ServeMux)
}
```

Each feature creates its own module type (e.g. `KeywordModule`, `HomeModule`) and registers its own routes. `registerV1Routes` becomes:

```go
func (s *Server) registerV1Routes(mux *http.ServeMux) {
    modules := []module.Module{
        home.NewModule(s.AIClient),
        keywords.NewModule(s.AIClient, s.JobManager),
        assets.NewModule(s.AssetFS),
    }
    for _, m := range modules {
        m.Register(mux)
    }
}
```

---

## QW1 — Wire `customInstructions` to `EvaluateKeywords`

**Depends on:** P1a (so `customInstructions` is available on `KeywordJobInput`)

**Files:** [internal/processor/keywords.go](internal/processor/keywords.go), [internal/ai/gemini.go](internal/ai/gemini.go)

**Problem:** The submission form has a `customInstructions` textarea. The value is parsed by the handler but silently dropped — it is never passed to the AI call.

**Fix (after P1a):**

In `gemini.go`, `EvaluateKeywords` builds a `prompt` string. Append custom instructions if provided:

```go
if input.CustomInstructions != "" {
    prompt += "\n\nAdditional instructions from the user:\n" + input.CustomInstructions
}
```

No interface changes needed — the field is already on `KeywordJobInput` after P1a.

**Note:** The Gemini client's system prompt is set at construction time (`NewClient`). Per-call customisation is added to the user-turn `prompt` string, not the system prompt. This is the correct approach — the system prompt defines the model's role; user instructions are per-request.

---

## QW2 — Delete `internal/store/jobs.go`

**File:** `internal/store/jobs.go`

**Problem:** This file is a deprecated duplicate of job-related types that now live in `internal/jobs/`. It has no callers. It adds confusion about which package owns job state.

**Fix:** Delete the file. If `internal/store/` becomes empty, delete the directory too.

**Verify:** `go build ./...` passes with no import errors after deletion.
