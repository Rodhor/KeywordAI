# Review — Context Threading

**Date:** 2026-03-14
**Severity:** Must fix — causes AI calls to fail on macOS

---

## Problem

Two processor functions pass `context.Background()` to the AI client instead of the Wails app context. On macOS the Gemini client's underlying HTTP transport is stricter about context validity, which surfaces the error.

The Wails context never enters the call chain because `StartEvaluation` and `StartClustering` do not declare `context.Context` as their first parameter. In Wails v2, a bound method only receives the app context if `context.Context` is its **first** parameter.

### Call chain showing where context is lost

```
Wails injects ctx
  → StartEvaluation(params)             ← no ctx param → Wails ctx discarded
      → go ProcessKeywords(params)      ← no ctx in params struct
          → SendToAIForValidation()     → aiClient.EvaluateKeywords(context.Background()) ← BUG

  → StartClustering(params)             ← same problem
      → go ProcessClustering(params)
          → organizer.OrganizeKeywords(context.Background()) ← BUG
```

---

## Changes

### 1. `backend/commands/keywords/keywords.go`

Add `context` to imports (it is not currently imported):

```go
import (
    "context"
    "fmt"
    "path/filepath"

    "keywordAutomator/internal/ai"
    "keywordAutomator/internal/jobs"
    "keywordAutomator/internal/processor"
    "keywordAutomator/internal/source"
)
```

Change `StartEvaluation` signature and pass ctx into params:

```go
// Before
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

// After
func (k *KeywordsCommand) StartEvaluation(ctx context.Context, params EvaluationParams) (JobRef, error) {
    jobID := k.jobMgr.CreateJob()

    go processor.ProcessKeywords(processor.KeywordProcessParams{
        Ctx:                ctx,
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
```

Change `StartClustering` signature and pass ctx into params:

```go
// Before
func (k *KeywordsCommand) StartClustering(params ClusterParams) (JobRef, error) {
    // ...
    go processor.ProcessClustering(processor.ClusterProcessParams{
        JobID:        jobID,
        Headers:      payload.Header,
        Rows:         selectedRows,
        OriginalRows: payload.OriginalRows,
        Organizer:    k.organizer,
        Manager:      k.jobMgr,
    })

    return JobRef{JobID: jobID}, nil
}

// After
func (k *KeywordsCommand) StartClustering(ctx context.Context, params ClusterParams) (JobRef, error) {
    // ...
    go processor.ProcessClustering(processor.ClusterProcessParams{
        Ctx:          ctx,
        JobID:        jobID,
        Headers:      payload.Header,
        Rows:         selectedRows,
        OriginalRows: payload.OriginalRows,
        Organizer:    k.organizer,
        Manager:      k.jobMgr,
    })

    return JobRef{JobID: jobID}, nil
}
```

---

### 2. `internal/processor/keywords.go`

Add `Ctx` to `KeywordProcessParams` and `aiValidationStep`, remove the `context` import's only usage of `context.Background()`:

```go
// Before
type KeywordProcessParams struct {
    JobID              string
    Source             source.KeywordSource
    ExcludeKeywords    []string
    CampaignDesc       string
    CustomInstructions string
    Evaluator          ai.KeywordEvaluator
    Manager            *jobs.Manager
}

// After
type KeywordProcessParams struct {
    Ctx                context.Context
    JobID              string
    Source             source.KeywordSource
    ExcludeKeywords    []string
    CampaignDesc       string
    CustomInstructions string
    Evaluator          ai.KeywordEvaluator
    Manager            *jobs.Manager
}
```

Update the step that is built inside `ProcessKeywords` to carry the context:

```go
// Before
aiValidationStep{Evaluator: params.Evaluator, Description: params.CampaignDesc, CustomInstructions: params.CustomInstructions},

// After
aiValidationStep{Ctx: params.Ctx, Evaluator: params.Evaluator, Description: params.CampaignDesc, CustomInstructions: params.CustomInstructions},
```

Add `Ctx` field to `aiValidationStep` and pass it to `SendToAIForValidation`:

```go
// Before
type aiValidationStep struct {
    Evaluator          ai.KeywordEvaluator
    Description        string
    CustomInstructions string
}

func (s aiValidationStep) Run(ctx *pipeline.Context) error {
    desc := s.Description
    if s.CustomInstructions != "" {
        desc += "\n\nFor this specific evaluation, also consider: " + s.CustomInstructions
    }
    newHeaders, newRows, err := SendToAIForValidation(s.Evaluator, ctx.Headers, desc, ctx.Rows)
    // ...
}

// After
type aiValidationStep struct {
    Ctx                context.Context
    Evaluator          ai.KeywordEvaluator
    Description        string
    CustomInstructions string
}

func (s aiValidationStep) Run(ctx *pipeline.Context) error {
    desc := s.Description
    if s.CustomInstructions != "" {
        desc += "\n\nFor this specific evaluation, also consider: " + s.CustomInstructions
    }
    newHeaders, newRows, err := SendToAIForValidation(s.Ctx, s.Evaluator, ctx.Headers, desc, ctx.Rows)
    // ...
}
```

Add `ctx` parameter to `SendToAIForValidation` and pass it to `EvaluateKeywords`:

```go
// Before
func SendToAIForValidation(aiClient ai.KeywordEvaluator, headers []string, description string, keywords [][]string) ([]string, [][]string, error) {
    newHeaders, newRows, err := aiClient.EvaluateKeywords(context.Background(), description, headers, keywords)
    // ...
}

// After
func SendToAIForValidation(ctx context.Context, aiClient ai.KeywordEvaluator, headers []string, description string, keywords [][]string) ([]string, [][]string, error) {
    newHeaders, newRows, err := aiClient.EvaluateKeywords(ctx, description, headers, keywords)
    // ...
}
```

---

### 3. `internal/processor/cluster.go`

Add `Ctx` to `ClusterProcessParams` and replace `context.Background()`:

```go
// Before
type ClusterProcessParams struct {
    JobID        string
    Headers      []string
    Rows         [][]string
    OriginalRows [][]string
    Organizer    ai.KeywordOrganizer
    Manager      *jobs.Manager
}

func ProcessClustering(params ClusterProcessParams) {
    newHeaders, newRows, err := params.Organizer.OrganizeKeywords(
        context.Background(), "", params.Headers, params.Rows,
    )
    // ...
}

// After
type ClusterProcessParams struct {
    Ctx          context.Context
    JobID        string
    Headers      []string
    Rows         [][]string
    OriginalRows [][]string
    Organizer    ai.KeywordOrganizer
    Manager      *jobs.Manager
}

func ProcessClustering(params ClusterProcessParams) {
    newHeaders, newRows, err := params.Organizer.OrganizeKeywords(
        params.Ctx, "", params.Headers, params.Rows,
    )
    // ...
}
```

The `"context"` import in `cluster.go` can be removed after this change since it is no longer used directly.

---

## Verification

After applying, confirm:
1. `go build ./...` passes with no errors
2. On macOS, start an evaluation job — the AI call should complete without a context error
3. On macOS, start a clustering job — same check
