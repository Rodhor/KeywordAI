# Async Job Manager

- **Status:** Modified
- **Created:** 2026-03-02 (onboarded — code predates this record)
- **Last updated:** 2026-03-13
- **Touches:** `internal/jobs/manager.go`

---

## What It Does

Tracks background processing tasks using UUID job IDs. Any part of the system can create a job, check its status, mark it complete (with a result payload), or mark it failed (with an error message). A background goroutine automatically removes jobs older than 30 minutes to prevent unbounded memory growth.

---

## Why It Was Built

AI evaluation calls take 5–30 seconds. Blocking a Wails-bound method call for that duration would freeze the UI. The job manager decouples the command method (which returns a job ID immediately) from the processing goroutine (which runs asynchronously).

---

## Implementation Checklist

### Initial implementation — (onboarded 2026-03-02)
- [x] `CreateJob()` — generates UUID, stores job with `StatusProcessing`
- [x] `GetJob(id)` — thread-safe read, returns job or `false` if not found
- [x] `CompleteJob(id, payload)` — sets status to `StatusCompleted`, stores typed payload
- [x] `FailJob(id, errMsg)` — sets status to `StatusFailed`, stores error string
- [x] `StartCleanup(interval, maxAge)` — launches background ticker goroutine
- [x] Auto-cleanup: removes jobs older than `maxAge` (default 30 min), runs every `interval` (default 10 min)
- [x] Thread safety via `sync.RWMutex` (read lock for reads, write lock for mutations)

### Refactor session — 2026-03-07
- [x] `KeywordJobPayload` moved out of `internal/jobs/manager.go` into `internal/processor/types.go`

---

## Technical Notes

- **Generic payload:** The `Payload` field is `any`, making the manager usable for every future job type without modification. Each caller is responsible for type-asserting the payload it expects.
- **`KeywordJobPayload` lives in `internal/processor/types.go`.** The manager has no knowledge of keyword-specific types — it stores and returns `any`. The status handler type-asserts to `processor.KeywordJobPayload` at read time.
- **Cleanup is not a guarantee.** Jobs are removed on a timer, not immediately on completion. A completed job is still readable for up to 30 minutes after creation. This is intentional — the polling UI may still need to read it.
- **No persistence.** All jobs are lost when the app closes. This is correct for the current desktop single-user use case. SQLite is the planned upgrade path if history or resume is ever needed — the manager interface would stay the same.
- **`internal/store/jobs.go`** is an older version of this feature. It is not used anywhere and can be deleted (QW2).

---

## Change History

### 2026-03-07 — `KeywordJobPayload` moved to processor package

**Reason:** The manager is a generic async primitive; it should not hold domain types. `KeywordJobPayload` belongs to the processor that creates it.

**What changed:**
- [x] `KeywordJobPayload` struct removed from `manager.go`, added to `internal/processor/types.go`
- [x] Import in `internal/api/v1/keywords/status.go` updated to use `processor.KeywordJobPayload`
