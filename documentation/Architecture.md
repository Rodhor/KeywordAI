# Architecture — Extensibility & Optimisation

**Goal:** Turn KeywordAI into a toolbelt of independent modules (keyword evaluation,
description generation, brainstorming, Google Ads API ingestion, Sheets export, …)
that share a common infrastructure but never get in each other's way.

**Guiding principle:** New functionality should not require editing existing files.
The one exception is a small amount of upfront groundwork (Priority 1 below) that
creates the seams — after that, every extension slots in without touching what is
already working.

---

## Table of Contents

1. [Current Structural Strengths](#1-current-structural-strengths)
2. [Current Pain Points](#2-current-pain-points)
3. [Priority 1 — Do Before Adding Anything Else](#3-priority-1--do-before-adding-anything-else)
4. [Priority 2 — When You Build the Next Module](#4-priority-2--when-you-build-the-next-module)
5. [Priority 3 — Down the Line](#5-priority-3--down-the-line)
6. [Proposed Directory Structure](#6-proposed-directory-structure)
7. [Extension Contract](#7-extension-contract)
8. [Things to Actively Avoid](#8-things-to-actively-avoid)
9. [Prioritised Checklist](#9-prioritised-checklist)

---

## 1. Current Structural Strengths

These parts are already well-designed. Do not overhaul them.

| Component | Why it is good |
|-----------|---------------|
| `internal/jobs/manager.go` | Generic `any` payload, thread-safe, auto-cleanup. Works for every future job type without changes. |
| `internal/ai/ai.go` | Defines three clean interfaces (`KeywordEvaluator`, `KeywordGenerator`, `KeywordOrganizer`). The abstractions are already there — they just need to be wired up. |
| `services/server.go` | `Server` struct with injected dependencies and a dedicated `registerV1Routes` function — easy to extend. |
| HTMX + Templ pattern | Each module can own its own templates and respond to HTMX requests without affecting others. |
| `jobs.KeywordJobPayload` | The `Header + Content [][]string` format is simple, serialisation-friendly, and a natural lingua franca between modules. |

---

## 2. Current Pain Points

These are the structural issues that will compound painfully as the project grows.
Each has a concrete fix described in Priority 1.

### 2a. `ProcessKeywords` has an exploding parameter list

```go
// internal/processor/keywords.go — current state
func ProcessKeywords(
    jobID string, filePath string, jobMgr *jobs.Manager,
    aiClient *ai.Client, description string, exclude []string,
    customInstructions string, keywords [][]string,
)
```

Eight parameters today. The next module will need different ones. Within two modules
this function will either fork into copies or accumulate a parameter list nobody can
follow. The `customInstructions` parameter is already accepted and silently ignored,
which is the first symptom.

### 2b. Concrete `*ai.Client` is passed everywhere

`Server`, `ProcessKeywords`, and all handlers hold a `*ai.Client` (the Gemini struct
directly). The interfaces in `ai.go` are never used. This means:

- You cannot swap the provider per-request or per-module.
- You cannot add an OpenAI path without touching every file that holds the client.
- You cannot test handlers without a live Gemini connection.

### 2c. The processing pipeline is a hardcoded sequence

`ProcessKeywords` calls `DropUnwantedColumns → ExtractHeaderRow → RemoveEmptyLines →
RemoveDuplicateLines → DropExcludedKeywords → SendToAIForValidation` in a fixed
order. Adding, removing, or reordering a step requires editing this function.
`DropUnwantedColumns` is already a no-op placeholder that will rot there forever
unless the pipeline becomes composable.

### 2d. The data source is a file path

`filePath string` is passed in and `os.Remove` is deferred inside the processor.
When the Google Ads API replaces the CSV upload, there is no file to remove — this
assumption breaks and must be patched every time a new source is added.

### 2e. Module routes are registered in one place

`registerV1Routes` in `server.go` will accumulate every route for every module.
At ten modules this becomes a maintenance problem and means editing the server file
for every new feature.

---

## 3. Priority 1 — Do Before Adding Anything Else

These changes touch existing files, but only minimally. Once done, every future
extension follows the "no modification" rule.

---

### 3a. Pack job input into a struct

**Problem:** 8-parameter `ProcessKeywords` signature.

**Fix:** One input struct per job type. Existing call sites pass the same values —
nothing changes in behaviour.

```go
// internal/processor/keywords.go
type KeywordJobInput struct {
    JobID              string
    JobMgr             *jobs.Manager
    AIClient           ai.KeywordEvaluator   // ← interface, not *ai.Client
    Description        string
    Exclude            []string
    CustomInstructions string
    Source             source.KeywordSource  // ← see 3c
}

func ProcessKeywords(input KeywordJobInput) { … }
```

New modules define their own input struct. `ProcessKeywords` stays untouched when
adding a description-generation job because that is a completely different struct.

---

### 3b. Use the AI interfaces that already exist

**Problem:** `*ai.Client` (concrete Gemini) is used everywhere, making the existing
interfaces in `ai.go` dead code.

**Fix:** Accept the narrowest interface the caller actually needs.

```go
// In Server struct — server.go
type Server struct {
    evaluator  ai.KeywordEvaluator   // only what keyword evaluation needs
    generator  ai.KeywordGenerator   // only what generation needs
    jobMgr     *jobs.Manager
}
```

`ai.Client` (Gemini) implements all three interfaces already. The single construction
in `main.go` stays exactly the same — just assign it to the interface fields instead
of the concrete type. New providers (OpenAI, Claude) need only implement the
interface they support; the rest of the program is unaffected.

---

### 3c. Introduce a `KeywordSource` interface

**Problem:** `filePath string` + `os.Remove` is hardcoded into the processor.

**Fix:** A one-method interface. The processor only calls `Fetch`, never touches files.

```go
// internal/source/source.go
type KeywordSource interface {
    Fetch(ctx context.Context) ([][]string, error)
    // Cleanup is called by the processor after Fetch, regardless of success.
    Cleanup() error
}
```

```go
// internal/source/csv.go  — wraps existing CSV logic, zero logic changes
type CSVFileSource struct{ path string }

func (s *CSVFileSource) Fetch(ctx context.Context) ([][]string, error) {
    return processor.ParseCSVToSlice(s.path)
}
func (s *CSVFileSource) Cleanup() error { return os.Remove(s.path) }
```

```go
// internal/source/googleads/googleads.go  — new, completely independent
type GoogleAdsSource struct{ … }

func (s *GoogleAdsSource) Fetch(ctx context.Context) ([][]string, error) { … }
func (s *GoogleAdsSource) Cleanup() error { return nil }  // no file to clean up
```

The `upload/csv` endpoint creates a `CSVFileSource`. The future Google Ads endpoint
creates a `GoogleAdsSource`. The processor never knows the difference.

---

### 3d. Make the processing pipeline composable

**Problem:** Fixed step sequence in `ProcessKeywords`.

**Fix:** A `Step` interface and a pipeline runner. Existing functions become steps
by wrapping them — their logic is not changed.

```go
// internal/pipeline/pipeline.go
type KeywordData struct {
    Header  []string
    Rows    [][]string
    Meta    map[string]any   // carry arbitrary context between steps
}

type Step interface {
    Name() string
    Run(ctx context.Context, data *KeywordData) error
}

func Run(ctx context.Context, steps []Step, data *KeywordData) error {
    for _, s := range steps {
        if err := s.Run(ctx, data); err != nil {
            return fmt.Errorf("%s: %w", s.Name(), err)
        }
    }
    return nil
}
```

```go
// internal/pipeline/steps/clean.go  — thin wrappers, zero new logic
type RemoveEmptyLinesStep struct{}

func (s RemoveEmptyLinesStep) Name() string { return "remove-empty-lines" }
func (s RemoveEmptyLinesStep) Run(_ context.Context, d *pipeline.KeywordData) error {
    d.Rows = processor.RemoveEmptyLines(d.Rows)
    return nil
}
```

The keyword workflow becomes:

```go
steps := []pipeline.Step{
    steps.RemoveEmptyLinesStep{},
    steps.RemoveDuplicatesStep{},
    steps.DropExcludedStep{Exclude: input.Exclude},
    steps.AIEvaluationStep{Client: input.AIClient, Description: input.Description},
}
pipeline.Run(ctx, steps, &data)
```

A future "clustering" or "bid suggestion" step is added by creating a new file in
`pipeline/steps/`. Nothing else is touched.

---

### 3e. Module-based route registration

**Problem:** All routes in `registerV1Routes`.

**Fix:** A `Module` interface. The server iterates registered modules; it never needs
to be edited for a new module.

```go
// internal/module/module.go
type Module interface {
    Name() string
    RegisterRoutes(mux *http.ServeMux)
}
```

```go
// internal/services/server.go  — only this block changes
var modules = []module.Module{
    keywordmodule.New(evaluator, jobMgr),
    // future: descriptionmodule.New(generator, jobMgr),
    // future: sheetsmodule.New(sheetsClient, jobMgr),
}

func (s *Server) registerV1Routes(mux *http.ServeMux) {
    for _, m := range modules {
        m.RegisterRoutes(mux)
    }
}
```

Adding a module = creating a new directory and appending one line to the `modules`
slice. Removing a module = deleting the line. Existing modules are never touched.

---

## 4. Priority 2 — When You Build the Next Module

Do these when you start building the Google Ads integration or the second distinct
workflow (description generation, brainstorming, etc.).

---

### 4a. Google OAuth as a shared service

Both the Google Ads API and Google Sheets API require OAuth 2.0. Plan the
architecture once, implement it once, and share it across all Google modules.

```
internal/
└── auth/
    └── google/
        ├── oauth.go      # token exchange, refresh loop
        └── store.go      # token persistence (file-backed for solo use)
```

Modules that need Google APIs receive a `*google.Client` (or an interface) injected
at startup — they do not each manage their own tokens. For a solo tool, storing the
token in `~/.config/keywordai/token.json` is sufficient; no database needed.

**Plan the token storage location before writing any Google API code.** Retrofitting
this is painful.

---

### 4b. `KeywordSink` for output destinations

Mirror the `KeywordSource` interface on the output side.

```go
// internal/sink/sink.go
type KeywordSink interface {
    Write(ctx context.Context, header []string, rows [][]string) error
}
```

Implementations:

| Sink | Notes |
|------|-------|
| `HTTPTableSink` | Renders the current `KeywordTable` template — wraps existing code |
| `CSVDownloadSink` | Sets `Content-Disposition: attachment` and writes CSV bytes |
| `GoogleSheetsSink` | Writes to a named sheet using the Sheets API |

The job-status handler calls `sink.Write(...)`. It does not know or care which sink
is active. The UI can pass a `format` query parameter to select the sink; no
processing code changes.

---

### 4c. Fix custom instructions (zero-effort quick win)

The form already captures `custom_instructions`. It is silently dropped in
`KeyWordsSubmit.go`. Adding it to `KeywordJobInput` (from Priority 1a) and then
passing it into the `AIEvaluationStep` is a two-line change — but only worth doing
after the struct refactor, otherwise it is another dangling parameter.

---

### 4d. Typed payloads per module, not per package

`KeywordJobPayload` currently lives in `internal/jobs/manager.go`. This means the
`jobs` package depends on domain knowledge it should not have. Move each payload type
to its own module:

```
internal/modules/keywords/payload.go  → KeywordJobPayload
internal/modules/descriptions/payload.go  → DescriptionJobPayload
```

The `jobs` package stays completely generic (`any` payload). Handlers that retrieve
jobs use a type assertion to their own payload type, not a shared one.

---

## 5. Priority 3 — Down the Line

These are real concerns, but acting on them before you need them creates premature
complexity.

---

### 5a. Persistent storage (when you need job history or cross-session state)

The in-memory job manager is correct for the current use case. Consider persistence
when:

- You want a history of past campaigns.
- You want to resume a job after a server restart.
- The Google Sheets sync needs to know what was last written.

**When that time comes:** SQLite (via `modernc.org/sqlite` — pure Go, no cgo) is the
right first step for a solo tool. It gives you full SQL with zero infrastructure.
Keep the `jobs.Manager` interface intact; swap the backing store to SQLite without
changing anything that calls the manager.

```
internal/
└── store/
    ├── store.go          # interface
    ├── memory.go         # current implementation, kept for tests
    └── sqlite.go         # future implementation
```

---

### 5b. Google Sheets as a live data layer

Google Sheets can eventually replace the CSV/API flow as the primary working surface
for a freelancer: write keywords to a sheet, trigger a run, get results back in the
same sheet. This is a natural extension of the `KeywordSource` + `KeywordSink`
pattern. No pipeline or job manager changes are needed.

Design consideration: the Sheets API is rate-limited and slow. Wrap it behind a
thin local cache (a `sync.Map` with a short TTL) to avoid hammering the API when
polling the same sheet repeatedly.

---

### 5c. Multi-step AI workflows (chaining)

The current flow is: clean → evaluate (one AI call). Future workflows may look like:
generate keywords → evaluate → organise → write descriptions. This is a natural
extension of the composable pipeline from Priority 1d. Each AI operation is a step.
No orchestration layer is needed unless steps need to run in parallel.

If parallel AI calls are ever needed (e.g. evaluate and organise simultaneously):
implement a `ParallelStep` that takes a slice of steps and runs them with a
`sync.WaitGroup`. The pipeline runner does not need to change.

---

### 5d. Google Docs integration

Lower priority than Sheets. Can be modelled as a `DocumentSink` that takes final
results and writes a formatted report. Shares the Google OAuth infrastructure from
Priority 2a.

---

### 5e. AI provider selection per request

Currently the provider is global. Once OpenAI is implemented, you may want per-module
or per-request provider selection (e.g. use Gemini for evaluation, OpenAI for
creative generation). The interface-based design from Priority 1b makes this
straightforward: the module receives the interface, `main.go` decides which
implementation to inject.

---

## 6. Proposed Directory Structure

The current layout is clean. The additions below extend it; nothing existing moves.

```
internal/
├── ai/
│   ├── ai.go                    # interfaces — unchanged
│   ├── gemini.go                # existing implementation — unchanged
│   └── openai.go                # implement when needed
│
├── source/                      # NEW — Priority 1c
│   ├── source.go                # KeywordSource interface
│   ├── csv.go                   # wraps existing CSV logic
│   └── googleads/
│       └── googleads.go         # Google Ads API source
│
├── sink/                        # NEW — Priority 2b
│   ├── sink.go                  # KeywordSink interface
│   ├── http.go                  # wraps existing table render
│   ├── csv.go                   # download endpoint
│   └── sheets/
│       └── sheets.go            # Google Sheets sink
│
├── pipeline/                    # NEW — Priority 1d
│   ├── pipeline.go              # Step interface + Run()
│   └── steps/
│       ├── clean.go             # RemoveEmptyLines, Duplicates, Exclusions
│       └── ai.go                # AIEvaluationStep, AIGenerationStep, etc.
│
├── auth/                        # NEW — Priority 2a
│   └── google/
│       ├── oauth.go
│       └── store.go
│
├── module/                      # NEW — Priority 1e
│   └── module.go                # Module interface
│
├── modules/                     # NEW — one sub-directory per feature
│   ├── keywords/                # current keyword workflow, extracted here
│   │   ├── handler.go
│   │   ├── payload.go
│   │   └── module.go            # implements module.Module
│   ├── descriptions/            # future
│   └── brainstorm/              # future
│
├── jobs/
│   └── manager.go               # unchanged — already well-designed
│
├── processor/
│   ├── keywords.go              # cleaning functions — unchanged
│   ├── ProcessAIResponse.go     # unchanged
│   └── ProcessFilesToSlices.go  # unchanged
│
├── api/v1/                      # can stay or migrate into modules/ over time
└── services/
    ├── server.go                # small change: module slice + loop
    ├── services.go
    └── middleware.go
```

---

## 7. Extension Contract

Once Priority 1 is complete, every new module follows this contract without touching
existing code:

1. **Create** `internal/modules/<name>/`
2. **Implement** `module.Module` — provide `Name()` and `RegisterRoutes(mux)`
3. **Define** an input struct and a payload struct inside the module
4. **Build** a `[]pipeline.Step` slice for the module's processing logic
5. **Add** a `KeywordSource` and/or `KeywordSink` appropriate to the workflow
6. **Append** the module to the `modules` slice in `server.go` — one line

If the module needs a Google API, it receives the shared `*google.Client` via
injection at startup.

That is the complete list. No existing file changes.

---

## 8. Things to Actively Avoid

| Anti-pattern | Why |
|---|---|
| Dynamic plugin loading (`plugin` package) | Overkill for a solo tool; compilation becomes painful |
| Message bus / event system | Adds invisible coupling; the request-response model is correct here |
| Global state / package-level vars | Makes testing and concurrency reasoning hard; use injection |
| One giant `utils` package | Becomes a dump for things nobody knows where to put; keep logic in its domain |
| Prematurely abstracting the UI | Templ components are cheap to copy; don't build a component registry before you have three modules using it |
| Storing tokens in `.env` | Fine for the Gemini key (no user identity); wrong for OAuth tokens, which must be refreshed and stored securely |
| Adding a database before needing one | The in-memory job manager is correct right now; add SQLite only when you have a concrete reason |

---

## 9. Prioritised Checklist

### Do now (before adding the next feature)

- [ ] Introduce `source.KeywordSource` interface + `CSVFileSource` wrapper
- [ ] Move `os.Remove` into `CSVFileSource.Cleanup()`, out of the processor
- [ ] Change `ProcessKeywords` signature to accept a struct (and `ai.KeywordEvaluator` interface, not `*ai.Client`)
- [ ] Change `Server` struct to hold AI interfaces, not `*ai.Client`
- [ ] Wrap existing cleaning functions as `pipeline.Step` implementations
- [ ] Add `module.Module` interface and update `registerV1Routes` to iterate a slice
- [ ] Extract keyword handlers + payload into `internal/modules/keywords/`

### Do when building the Google Ads or next module

- [ ] Implement `GoogleAdsSource` in `internal/source/googleads/`
- [ ] Design and implement shared Google OAuth in `internal/auth/google/`
- [ ] Implement `CSVDownloadSink` in `internal/sink/`
- [ ] Wire `custom_instructions` through to the AI step (now trivial via the input struct)

### Consider when the need arises

- [ ] `GoogleSheetsSink` + `GoogleSheetsSource` (after OAuth is working)
- [ ] SQLite-backed job store (when job history or persistence is needed)
- [ ] `ParallelStep` for concurrent AI calls (when chaining multiple AI operations)
- [ ] OpenAI provider implementation (implement `ai.KeywordEvaluator` etc.)
- [ ] Google Docs `DocumentSink`
- [ ] Apply middleware (`middleware.go` logger is already written, just not wired up)
- [ ] Remove `internal/store/jobs.go` (legacy, unused)
