# Feature: Wails Migration

- **Mode:** FEATURE
- **Created:** 2026-03-13
- **Base template:** `~/Documents/Programming/keywordAutomator`
- **Pattern reference:** github.com/Rodhor/SubscriptionTracker

This document is the step-by-step migration guide from the current Go/Templ/HTMX PWA to a
Wails desktop app using SvelteKit and Skeleton UI v4.

---

## What This Migration Achieves

| Before | After |
|--------|-------|
| Local HTTP server on port 8090 | Native desktop app, no server |
| Browser tab as UI | Proper app window |
| HTMX for interactivity | Svelte 5 reactive state |
| Templ for HTML rendering | Svelte components |
| .env file for API key | Settings page in the app (future) |
| PWA install + background service | Standard app install |
| HTTP handlers in `internal/api/` | Command structs in `backend/` |

---

## What Transfers Unchanged

These packages have zero dependency on HTTP and move over without modification:

- `internal/ai/` — interfaces and Gemini client
- `internal/jobs/` — job manager
- `internal/processor/` — keyword and clustering logic
- `internal/pipeline/` — step interface and runner
- `internal/source/` — `KeywordSource` interface and `CSVFileSource`

The `[][]string` data contract is unchanged. The AI interfaces are unchanged.
Everything above the handler layer is reusable as-is.

---

## What Changes

| Layer | Before | After |
|-------|--------|-------|
| Entry point | `cmd/main.go` starts HTTP server | `main.go` starts Wails app |
| Lifecycle | `services.StartServer()` | `app.setupBackend()` / `app.shutdownBackend()` |
| Request handling | HTTP handlers in `internal/api/v1/` | Command methods in `backend/commands/` |
| UI rendering | Templ templates served over HTTP | Svelte components in `frontend/src/` |
| Interactivity | HTMX attributes on HTML | Svelte `$state` + `await` calls |
| Polling | `hx-trigger="every 1s"` | `setInterval` + Wails binding call |
| File uploads | Multipart HTTP POST | Wails file dialog or drag-and-drop |
| Dependencies | `godotenv`, `net/http`, `templ` | `wails/v2` replaces all of the above |

---

## New Project Structure

Start from the `keywordAutomator` template. The new layout is:

```
keywordAutomator/
├── main.go                          # Wails app bootstrap — no business logic
├── app.go                           # App struct, setupBackend, shutdownBackend
├── wails.json                       # Wails config (already set up)
├── go.mod                           # Add all dependencies from current project
├── backend/
│   ├── backend.go                   # Backend struct — single binding surface
│   └── commands/
│       ├── keywords/
│       │   └── keywords.go          # StartEvaluation, GetJobStatus, StartClustering, GetClusterStatus
│       └── upload/
│           └── upload.go            # UploadCSV (file path → tmp/, returns filename)
├── internal/                        # Moved from current project unchanged
│   ├── ai/
│   ├── jobs/
│   ├── processor/
│   ├── pipeline/
│   └── source/
└── frontend/                        # Already set up in keywordAutomator template
    └── src/
        ├── app.html                 # Add data-theme here
        ├── routes/
        │   ├── +layout.svelte       # Root layout, imports CSS
        │   ├── +layout.ts           # SSR: false, prerender: true (already set)
        │   ├── +page.svelte         # Landing page with upload form
        │   └── layout.css           # Skeleton + Tailwind imports (already set)
        └── lib/
            ├── components/
            │   ├── KeywordForm.svelte      # Upload form (replaces KeyWordsUploadForm.templ)
            │   ├── KeywordTable.svelte     # Results table with checkboxes
            │   ├── PollingSpinner.svelte   # Replaces PollingUI.templ
            │   └── NavBar.svelte          # App header
            ├── stores/
            │   └── app.svelte.ts          # Global app state
            └── wailsjs/                   # Auto-generated — never edit manually
                ├── go/
                │   ├── keywords/KeywordsCommand.d.ts
                │   ├── keywords/KeywordsCommand.js
                │   └── upload/UploadCommand.js
                └── runtime/
```

---

## Backend Architecture

### The Core Principle

Wails binds Go **structs** to the frontend. Every public method on a bound struct becomes
callable from JavaScript. Instead of binding individual functions or the App struct directly,
we bind a `Backend` struct whose fields are command handler structs — one per feature domain.

This means the frontend imports from predictable paths, the Go side stays organised by domain,
and adding a new feature means adding a new command struct with no changes to `main.go`.

---

### `backend/backend.go`

```go
package backend

import (
    "context"
    "io"
    "log/slog"
    "os"
    "time"

    "keywordAutomator/backend/commands/keywords"
    "keywordAutomator/backend/commands/upload"
    "keywordAutomator/internal/ai"
    "keywordAutomator/internal/jobs"

    "github.com/joho/godotenv"
)

// Backend is the single struct bound to the Wails frontend.
// Each field is a command handler for a feature domain.
// All public methods on these handlers are callable from Svelte.
type Backend struct {
    Keywords *keywords.KeywordsCommand
    Upload   *upload.UploadCommand
}

// New creates a Backend with empty command structs.
// Called before Wails starts — must not do any initialisation that requires context.
func New() *Backend {
    return &Backend{
        Keywords: &keywords.KeywordsCommand{},
        Upload:   &upload.UploadCommand{},
    }
}

// Setup is called from app.setupBackend once Wails has a context.
// This is where AI clients, job managers, and other services are initialised
// and wired into the command structs.
func (b *Backend) Setup(ctx context.Context) error {
    initLogger()
    loadDotEnv()

    aiClient, err := ai.NewClient(ctx)
    if err != nil {
        slog.Error("Failed to initialise AI client", "error", err)
        return err
    }

    jobMgr := jobs.NewManager()
    jobMgr.StartCleanup(10*time.Minute, 30*time.Minute)

    // Wire dependencies into each command struct.
    // The structs were pre-allocated in New() and already bound to Wails —
    // setting their fields here is safe because Wails calls OnStartup
    // before the frontend makes any calls.
    b.Keywords.Setup(jobMgr, aiClient, aiClient)
    b.Upload.Setup()

    slog.Info("Backend ready")
    return nil
}

// Shutdown is called from app.shutdownBackend when the app window closes.
func (b *Backend) Shutdown() {
    slog.Info("Backend shutting down")
    // Future: flush pending jobs, close DB connections, etc.
}

func initLogger() {
    file, _ := os.OpenFile("app.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    multiWriter := io.MultiWriter(os.Stdout, file)
    logger := slog.New(slog.NewTextHandler(multiWriter, nil))
    slog.SetDefault(logger)
}

func loadDotEnv() {
    if err := godotenv.Load(); err != nil {
        slog.Warn("No .env file found")
    }
}
```

---

### `app.go`

```go
package main

import (
    "context"

    "keywordAutomator/backend"
)

// App is the Wails application struct.
// It holds the context and the backend.
// It is also bound to Wails (see main.go) so any public methods here
// are callable from the frontend — keep it minimal, prefer backend commands.
type App struct {
    ctx     context.Context
    Backend *backend.Backend
}

func NewApp() *App {
    return &App{
        Backend: backend.New(),
    }
}

// setupBackend is called by Wails after the window is created.
// This is the single place where all services are initialised.
func (a *App) setupBackend(ctx context.Context) {
    a.ctx = ctx
    if err := a.Backend.Setup(ctx); err != nil {
        // The app will function but AI features will not work.
        // The frontend should handle missing-API-key state gracefully.
    }
}

// shutdownBackend is called by Wails just before the app exits.
func (a *App) shutdownBackend(ctx context.Context) {
    a.Backend.Shutdown()
}
```

---

### `main.go`

```go
package main

import (
    "embed"

    "github.com/wailsapp/wails/v2"
    "github.com/wailsapp/wails/v2/pkg/options"
    "github.com/wailsapp/wails/v2/pkg/options/assetserver"
    "github.com/wailsapp/wails/v2/pkg/options/mac"
)

//go:embed all:frontend/build
var assets embed.FS

func main() {
    app := NewApp()

    err := wails.Run(&options.App{
        Title:     "KeywordAI",
        Width:     1280,
        Height:    800,
        MinWidth:  900,
        MinHeight: 600,
        AssetServer: &assetserver.Options{
            Assets: assets,
        },
        OnStartup:  app.setupBackend,
        OnShutdown: app.shutdownBackend,
        // Bind the command structs, not the App struct.
        // The app struct is included only if you need methods callable from JS.
        // Each field here becomes a separate import namespace in the frontend.
        Bind: []interface{}{
            app.Backend.Keywords,
            app.Backend.Upload,
        },
        Mac: &mac.Options{
            TitleBar: mac.TitleBarHiddenInset(),
            About: &mac.AboutInfo{
                Title:   "KeywordAI",
                Message: "Keyword evaluation and clustering tool",
            },
        },
    })

    if err != nil {
        panic(err)
    }
}
```

**Why bind `app.Backend.Keywords` and not `app.Backend`?**
Wails generates one binding file per bound struct. If you bind `Backend` itself, all methods
appear in one file and the frontend has no namespace separation. Binding each command struct
separately keeps imports clean:

```typescript
// Clean — each import is a domain
import { StartEvaluation } from '$lib/wailsjs/go/keywords/KeywordsCommand';
import { UploadCSV }        from '$lib/wailsjs/go/upload/UploadCommand';
```

---

## Command Layer

Commands replace HTTP handlers. The difference is that instead of reading from `r.FormValue()`
and writing to `w http.ResponseWriter`, a command receives typed Go parameters and returns
typed Go values. Wails serialises the return value to JSON automatically.

### Return Type Convention

Use a consistent result wrapper across all commands. This mirrors what SubscriptionTracker
does with its `Response[T]` type:

```go
// backend/commands/result.go
package commands

// Result is the standard return type for all command methods.
// The frontend checks Status first, then reads Data or Error.
type Result[T any] struct {
    Status string `json:"status"` // "completed" | "processing" | "failed"
    Data   T      `json:"data"`
    Error  string `json:"error"`
}
```

---

### `backend/commands/upload/upload.go`

Replaces `internal/api/v1/upload/upload.go`.

```go
package upload

import (
    "fmt"
    "io"
    "os"
    "path/filepath"
    "strings"
)

type UploadCommand struct{}

func (u *UploadCommand) Setup() {}

// SaveCSV receives a local file path (from the Wails file dialog on the frontend),
// copies it to the tmp/ working directory, and returns the saved filename.
// The filename is then passed to StartEvaluation.
//
// Note: Wails file dialogs return absolute paths on the local filesystem —
// there is no multipart upload. This replaces the HTTP file upload handler entirely.
func (u *UploadCommand) SaveCSV(sourcePath string) (string, error) {
    if !strings.HasSuffix(strings.ToLower(sourcePath), ".csv") {
        return "", fmt.Errorf("only .csv files are supported")
    }

    if err := os.MkdirAll("tmp", 0755); err != nil {
        return "", fmt.Errorf("could not create tmp directory: %w", err)
    }

    src, err := os.Open(sourcePath)
    if err != nil {
        return "", fmt.Errorf("could not open file: %w", err)
    }
    defer src.Close()

    filename := filepath.Base(sourcePath)
    destPath := filepath.Join("tmp", filename)

    dst, err := os.Create(destPath)
    if err != nil {
        return "", fmt.Errorf("could not create destination file: %w", err)
    }
    defer dst.Close()

    if _, err = io.Copy(dst, src); err != nil {
        return "", fmt.Errorf("could not copy file: %w", err)
    }

    return filename, nil
}
```

---

### `backend/commands/keywords/keywords.go`

Replaces `internal/api/v1/keywords/` (submit.go, status.go, cluster.go, cluster_status.go).

```go
package keywords

import (
    "keywordAutomator/internal/ai"
    "keywordAutomator/internal/jobs"
    "keywordAutomator/internal/processor"
    "keywordAutomator/internal/source"
    "path/filepath"
)

type KeywordsCommand struct {
    jobMgr    *jobs.Manager
    evaluator ai.KeywordEvaluator
    organizer ai.KeywordOrganizer
}

func (k *KeywordsCommand) Setup(mgr *jobs.Manager, evaluator ai.KeywordEvaluator, organizer ai.KeywordOrganizer) {
    k.jobMgr = mgr
    k.evaluator = evaluator
    k.organizer = organizer
}

// --- Input types ---
// These are serialised from JSON by Wails when the frontend calls the method.

type EvaluationParams struct {
    Filename           string   `json:"filename"`           // from UploadCommand.SaveCSV
    CampaignDesc       string   `json:"campaignDesc"`
    ExcludeKeywords    []string `json:"excludeKeywords"`
    CustomInstructions string   `json:"customInstructions"`
}

type ClusterParams struct {
    EvalJobID    string `json:"evalJobId"`
    SelectedRows []int  `json:"selectedRows"` // indices of checked rows
}

// --- Result types ---
// These are serialised to JSON by Wails and received as typed objects in Svelte.

type JobRef struct {
    JobID string `json:"jobId"`
}

type JobStatus struct {
    Status string      `json:"status"` // "processing" | "completed" | "failed"
    Error  string      `json:"error,omitempty"`
    Data   interface{} `json:"data,omitempty"`
}

type EvaluationData struct {
    Headers      []string   `json:"headers"`
    Rows         [][]string `json:"rows"`
    OriginalRows [][]string `json:"originalRows"`
}

// --- Methods ---
// Each public method below becomes callable from the Svelte frontend.
// Wails generates a TypeScript binding for each one.

// StartEvaluation creates a background job and returns its ID immediately.
// The frontend polls GetEvaluationStatus until done.
// Replaces: submit.go HTTP handler.
func (k *KeywordsCommand) StartEvaluation(params EvaluationParams) (JobRef, error) {
    jobID := k.jobMgr.CreateJob()

    go processor.ProcessKeywords(processor.KeywordProcessParams{
        JobID:              jobID,
        Source:             &source.CSVFileSource{Path: filepath.Join("tmp", params.Filename)},
        CampaignDesc:       params.CampaignDesc,
        ExcludeKeywords:    params.ExcludeKeywords,
        CustomInstructions: params.CustomInstructions,
        Evaluator:          k.evaluator,
        Manager:            k.jobMgr,
    })

    return JobRef{JobID: jobID}, nil
}

// GetEvaluationStatus returns the current status of an evaluation job.
// Replaces: status.go HTTP handler.
func (k *KeywordsCommand) GetEvaluationStatus(jobID string) (JobStatus, error) {
    status, data, errMsg := k.jobMgr.GetJob(jobID)

    result := JobStatus{Status: string(status)}

    switch status {
    case jobs.StatusCompleted:
        if payload, ok := data.(processor.KeywordJobPayload); ok {
            result.Data = EvaluationData{
                Headers:      payload.Header,
                Rows:         payload.Content,
                OriginalRows: payload.OriginalRows,
            }
        }
    case jobs.StatusFailed:
        result.Error = errMsg
    }

    return result, nil
}

// StartClustering creates a clustering job from a completed evaluation.
// Takes the eval job's data and the user's selected row indices.
// Replaces: cluster.go HTTP handler.
func (k *KeywordsCommand) StartClustering(params ClusterParams) (JobRef, error) {
    // Retrieve eval job data
    _, data, _ := k.jobMgr.GetJob(params.EvalJobID)
    payload, ok := data.(processor.KeywordJobPayload)
    if !ok {
        return JobRef{}, fmt.Errorf("eval job not found or not completed")
    }

    // Filter to selected rows only
    selectedRows := make([][]string, 0, len(params.SelectedRows))
    for _, idx := range params.SelectedRows {
        if idx < len(payload.Content) {
            selectedRows = append(selectedRows, payload.Content[idx])
        }
    }

    clusterPayload := processor.ClusterJobPayload{
        Header:  payload.Header,
        Content: selectedRows,
    }

    jobID := k.jobMgr.CreateJob()
    go processor.ProcessClustering(processor.ClusterProcessParams{
        JobID:     jobID,
        Payload:   clusterPayload,
        Organizer: k.organizer,
        Manager:   k.jobMgr,
    })

    return JobRef{JobID: jobID}, nil
}

// GetClusterStatus returns the current status of a clustering job.
// Replaces: cluster_status.go HTTP handler.
func (k *KeywordsCommand) GetClusterStatus(jobID string) (JobStatus, error) {
    status, data, errMsg := k.jobMgr.GetJob(jobID)

    result := JobStatus{Status: string(status)}
    switch status {
    case jobs.StatusCompleted:
        result.Data = data
    case jobs.StatusFailed:
        result.Error = errMsg
    }
    return result, nil
}
```

> **Note on `jobs.Manager.GetJob`:** The current `manager.go` may not expose a method that
> returns all three values (status, data, errMsg) in one call. You may need to add or adjust
> a method. Check `internal/jobs/manager.go` and adjust accordingly before implementing the
> commands above.

---

## How Wails Generates Frontend Bindings

When you run `wails dev` or `wails build`, Wails reads every struct in the `Bind` slice and
generates TypeScript + JavaScript files in `frontend/src/lib/wailsjs/go/`.

The path follows the Go package name:

| Go struct | Package | Generated file |
|-----------|---------|----------------|
| `KeywordsCommand` | `keywords` | `wailsjs/go/keywords/KeywordsCommand.js` |
| `UploadCommand` | `upload` | `wailsjs/go/upload/UploadCommand.js` |

Each public method on the struct becomes a named export:

```typescript
// Auto-generated — do not edit
// wailsjs/go/keywords/KeywordsCommand.d.ts
export function StartEvaluation(arg1: EvaluationParams): Promise<JobRef>;
export function GetEvaluationStatus(arg1: string): Promise<JobStatus>;
export function StartClustering(arg1: ClusterParams): Promise<JobRef>;
export function GetClusterStatus(arg1: string): Promise<JobStatus>;
```

Import in Svelte:
```typescript
import { StartEvaluation, GetEvaluationStatus } from '$lib/wailsjs/go/keywords/KeywordsCommand';
import { SaveCSV } from '$lib/wailsjs/go/upload/UploadCommand';
```

All calls return `Promise<T>` — every Go method is async from the frontend's perspective.

---

## Frontend Architecture

### Replacing HTMX Polling with Svelte State

HTMX polling worked by re-rendering HTML from the server. In Svelte, the equivalent is
a `$state` variable and a `setInterval` that calls the status command.

**Before (HTMX in Templ):**
```html
<div hx-get="/api/v1/submit/status?id={{.JobID}}"
     hx-trigger="every 1s"
     hx-target="this"
     hx-swap="outerHTML">
  <spinner/>
</div>
```

**After (Svelte 5):**
```svelte
<script lang="ts">
  import { StartEvaluation, GetEvaluationStatus } from '$lib/wailsjs/go/keywords/KeywordsCommand';
  import type { EvaluationData } from '$lib/types';

  // Reactive state replaces server-rendered HTML swapping
  let status = $state<'idle' | 'processing' | 'completed' | 'failed'>('idle');
  let result = $state<EvaluationData | null>(null);
  let errorMsg = $state('');
  let pollInterval: ReturnType<typeof setInterval>;

  async function startEvaluation(params) {
      status = 'processing';
      const { jobId } = await StartEvaluation(params);
      pollInterval = startPolling(jobId);
  }

  function startPolling(jobId: string) {
      return setInterval(async () => {
          const jobStatus = await GetEvaluationStatus(jobId);

          if (jobStatus.status === 'completed') {
              clearInterval(pollInterval);
              result = jobStatus.data as EvaluationData;
              status = 'completed';
          } else if (jobStatus.status === 'failed') {
              clearInterval(pollInterval);
              errorMsg = jobStatus.error;
              status = 'failed';
          }
          // still 'processing' — do nothing, poll again next tick
      }, 1000);
  }
</script>

{#if status === 'idle'}
  <!-- show form -->
{:else if status === 'processing'}
  <PollingSpinner />
{:else if status === 'completed'}
  <KeywordTable data={result} />
{:else if status === 'failed'}
  <ErrorDisplay message={errorMsg} />
{/if}
```

This is the direct functional replacement for PollingUI.templ + the status handler pattern.

---

### File Upload — Replacing Drag-and-Drop

The current drag-and-drop uses HTMX to POST the file to `/api/v1/upload/csv`. In Wails,
there is no HTTP server. Instead, use the Wails runtime file dialog:

```svelte
<script lang="ts">
  import { OpenFileDialog } from '$lib/wailsjs/runtime/runtime';
  import { SaveCSV } from '$lib/wailsjs/go/upload/UploadCommand';

  let uploadedFilename = $state('');
  let uploadError = $state('');

  async function pickFile() {
      // Opens the native OS file picker — returns local path string
      const path = await OpenFileDialog({
          Title:   'Select keyword CSV',
          Filters: [{ DisplayName: 'CSV Files', Pattern: '*.csv' }],
      });

      if (!path) return; // user cancelled

      const result = await SaveCSV(path);
      if (result.error) {
          uploadError = result.error;
      } else {
          uploadedFilename = result.filename;
          uploadError = '';
      }
  }
</script>

<button class="btn preset-outlined" onclick={pickFile}>
  {uploadedFilename ? uploadedFilename : 'Choose CSV file'}
</button>
{#if uploadError}
  <p class="text-error-500">{uploadError}</p>
{/if}
```

If you still want drag-and-drop onto the window (dropping a file onto the app), Wails v2
supports this via `OnFileDrop` in options — but the file dialog approach above is simpler
to start with.

---

### Svelte Component Map

| Current Templ component | Svelte equivalent | Notes |
|------------------------|-------------------|-------|
| `KeyWordsUploadForm.templ` | `lib/components/KeywordForm.svelte` | Contains the whole form + state machine |
| `PollingUI.templ` | `lib/components/PollingSpinner.svelte` | Simple spinner, shown when `status === 'processing'` |
| `KeywordTable.templ` | `lib/components/KeywordTable.svelte` | Receives `EvaluationData` as a prop |
| `ErrorUI.templ` | Inline `{:else if status === 'failed'}` block | Too small to need its own component |
| `NavBar.templ` | `lib/components/NavBar.svelte` | App bar using Skeleton's CSS classes |
| `ThemeToggle.templ` | Skeleton mode toggle utility | See Skeleton docs `/guides/mode` |
| `base.templ` | `+layout.svelte` | Layout already exists in template |
| `landing.templ` | `+page.svelte` | Top-level page composition |

---

### `+page.svelte` — Top-Level Composition

```svelte
<script lang="ts">
  import KeywordForm from '$lib/components/KeywordForm.svelte';
  import KeywordTable from '$lib/components/KeywordTable.svelte';
  import PollingSpinner from '$lib/components/PollingSpinner.svelte';
  import NavBar from '$lib/components/NavBar.svelte';
  import { StartEvaluation, GetEvaluationStatus } from '$lib/wailsjs/go/keywords/KeywordsCommand';

  type Phase = 'form' | 'processing' | 'results' | 'error';
  let phase = $state<Phase>('form');
  let result = $state(null);
  let errorMsg = $state('');

  async function handleSubmit(params) {
      phase = 'processing';
      try {
          const { jobId } = await StartEvaluation(params);
          const interval = setInterval(async () => {
              const s = await GetEvaluationStatus(jobId);
              if (s.status === 'completed') {
                  clearInterval(interval);
                  result = s.data;
                  phase = 'results';
              } else if (s.status === 'failed') {
                  clearInterval(interval);
                  errorMsg = s.error;
                  phase = 'error';
              }
          }, 1000);
      } catch (e) {
          errorMsg = String(e);
          phase = 'error';
      }
  }

  function reset() {
      phase = 'form';
      result = null;
      errorMsg = '';
  }
</script>

<div class="h-screen flex flex-col">
  <NavBar />
  <main class="flex-1 overflow-auto p-8">
    {#if phase === 'form'}
      <KeywordForm onSubmit={handleSubmit} />
    {:else if phase === 'processing'}
      <PollingSpinner />
    {:else if phase === 'results'}
      <KeywordTable data={result} onReset={reset} />
    {:else if phase === 'error'}
      <div class="card preset-outlined-error-500 p-6 space-y-4">
        <p>{errorMsg}</p>
        <button class="btn preset-filled" onclick={reset}>Back</button>
      </div>
    {/if}
  </main>
</div>
```

---

### `app.html` — Theme Setup

In the keywordAutomator template, add the `data-theme` attribute to `<html>`:

```html
<!DOCTYPE html>
<html lang="en" data-theme="catppuccin">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    %sveltekit.head%
  </head>
  <body>
    <div id="svelte">%sveltekit.body%</div>
  </body>
</html>
```

---

## Migration Checklist

Work through this in order. Each section builds on the previous one.

### Phase 1 — Go setup

- [ ] Copy `internal/ai/`, `internal/jobs/`, `internal/processor/`, `internal/pipeline/`, `internal/source/` into the keywordAutomator project
- [ ] Copy `go.mod` entries for `godotenv` and `google.golang.org/genai` into the new `go.mod`
- [ ] Run `go mod tidy`
- [ ] Create `backend/commands/result.go` with the `Result[T]` type
- [ ] Create `backend/commands/upload/upload.go` — `SaveCSV` method
- [ ] Create `backend/commands/keywords/keywords.go` — all four methods
- [ ] Check `internal/jobs/manager.go` — add a `GetJob(id) (status, data, errMsg)` method if it does not already exist
- [ ] Create `backend/backend.go` — `Backend` struct, `New()`, `Setup()`, `Shutdown()`
- [ ] Replace `app.go` with the version from this document
- [ ] Replace `main.go` with the version from this document
- [ ] Run `wails dev` and confirm the app opens without panicking

### Phase 2 — Frontend skeleton

- [ ] Confirm `frontend/src/routes/layout.css` already imports Skeleton + Tailwind (it does in the template)
- [ ] Confirm `frontend/src/routes/+layout.ts` has `ssr = false` and `prerender = true` (it does)
- [ ] Confirm `frontend/src/app.html` has `data-theme` set
- [ ] Create `frontend/src/lib/components/NavBar.svelte` — app bar with title
- [ ] Create `frontend/src/lib/components/PollingSpinner.svelte` — spinner with message
- [ ] Run `wails dev` — confirm blank app opens with no console errors

### Phase 3 — File upload

- [ ] Create `frontend/src/lib/components/KeywordForm.svelte` — layout only, no submit logic yet
- [ ] Add file picker button using `OpenFileDialog` from Wails runtime
- [ ] Wire button to `SaveCSV` command
- [ ] Confirm file is copied to `tmp/` when picked
- [ ] Add form fields: campaign description, exclude keywords, custom instructions

### Phase 4 — Evaluation flow

- [ ] Wire form submit to `StartEvaluation`
- [ ] Implement polling loop with `setInterval` + `GetEvaluationStatus`
- [ ] Show `PollingSpinner` while processing
- [ ] Create `frontend/src/lib/components/KeywordTable.svelte`
- [ ] Render results table with checkboxes on completion
- [ ] Show error state on failure
- [ ] Test end-to-end: pick file → submit → poll → see results

### Phase 5 — Clustering flow

- [ ] Add "Proceed to Clustering" button in `KeywordTable.svelte`
- [ ] Wire button to `StartClustering` with selected row indices
- [ ] Implement cluster polling with `GetClusterStatus`
- [ ] Implement `processor.ProcessClustering` (currently stubbed)
- [ ] Show cluster results or error

### Phase 6 — Polish

- [ ] Dark/light mode toggle using Skeleton mode utilities
- [ ] App icon (`build/appicon.png`)
- [ ] macOS window title bar styling (already in `main.go` template above)
- [ ] Test macOS build: `wails build -platform darwin/arm64`
- [ ] Test Linux build: `wails build`

---

## Data Flow — End to End (After Migration)

```
1. App opens
   → main() creates App with Backend
   → Wails binds Keywords and Upload command structs
   → setupBackend(ctx) initialises AI client, job manager, wires commands
   → Frontend loads from embedded assets

2. User picks a CSV file
   → OpenFileDialog() returns OS file path
   → SaveCSV(path) copies file to tmp/, returns filename

3. User fills form and clicks submit
   → StartEvaluation(params) called — returns jobId immediately
   → Go goroutine starts ProcessKeywords in background
   → Frontend enters 'processing' phase

4. Frontend polls every 1 second
   → GetEvaluationStatus(jobId)
   → While 'processing': do nothing, wait for next tick
   → On 'completed': render KeywordTable with result data
   → On 'failed': render error card

5. User reviews table, checks/unchecks rows
   → All state lives in Svelte — no server round trip

6. User clicks "Proceed to Clustering"
   → StartClustering({ evalJobId, selectedRows })
   → Go goroutine starts ProcessClustering
   → Frontend polls GetClusterStatus
   → On complete: show cluster results

7. App closes
   → shutdownBackend() called
   → Backend.Shutdown() runs cleanup
```

---

## Key Differences to Keep in Mind

**No HTTP, no ports.** There is no `net/http` server running. If you see yourself reaching
for `http.Handler`, `http.Request`, or `http.ResponseWriter`, stop — that layer is gone.

**Wails bindings are async.** Every call to a Go method returns a `Promise`. Always `await`.

**Errors come back as strings.** Go `error` values are serialised to their `.Error()` string.
Check `result.error` in Svelte after every call.

**Bindings are generated, not written.** Never edit files in `wailsjs/go/`. Delete them and
re-run `wails dev` if they get out of sync.

**SSR is off.** `+layout.ts` disables server-side rendering. This is required for Wails —
there is no server to do SSR. All rendering happens in the browser window.

**File paths are OS-native.** `OpenFileDialog` returns paths like `/home/user/file.csv` on
Linux and `/Users/user/file.csv` on macOS. Use `filepath.Base` and `filepath.Join` in Go,
never string concatenation.
