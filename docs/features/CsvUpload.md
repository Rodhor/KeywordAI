# CSV Upload

> ⚠ **Superseded — 2026-03-13**
> The HTTP drag-and-drop upload was replaced during the Wails migration. File selection is now handled by `backend/commands/upload/upload.go`'s `PickAndSaveCSV()` method, which calls `runtime.OpenFileDialog` from the Go side (native OS file picker). The tmp/ caching and `CSVFileSource.Cleanup()` patterns are unchanged. The Templ components and HTTP handler no longer exist. See `UploadCommand` and `KeywordForm.svelte`.

- **Status:** Removed
- **Created:** 2026-03-02 (onboarded — code predates this record)
- **Last updated:** 2026-03-13
- **Touches:** ~~`internal/api/v1/upload/upload.go`~~ · ~~`ui/customComponents/FileUpload/dragNdrop.templ`~~ · ~~`ui/customComponents/Forms/KeyWordsUploadForm.templ`~~ (all removed) · `backend/commands/upload/upload.go` (new Wails equivalent) · `internal/source/source.go` (unchanged)

---

## What It Does

Users select or drag-and-drop a CSV file onto the upload zone. The file is validated server-side and cached temporarily on disk. A hidden form field carries the filename to the next step (form submission) without requiring another upload.

---

## Why It Was Built

Keywords must enter the system in a structured tabular format before any processing can occur. CSV is the standard export format from Google Ads and most keyword tools. The user should not need to re-upload the file when they submit the form — hence the cache-and-reference approach.

---

## Implementation Checklist

### Initial implementation — (onboarded 2026-03-02)
- [x] Drag-and-drop upload zone (Templ component)
- [x] File extension validation (.csv only)
- [x] Non-empty file check
- [x] First-row CSV readability check
- [x] Save to `tmp/<filename>` on disk
- [x] Return HTMX partial: success feedback with filename or error message
- [x] Hidden form field carries filename to submit handler

### Bug fixes + P1e module restructure — 2026-03-07
- [x] **Bug 2 fix** — `return` added after every `renderStatus(false, ...)` error call; execution no longer falls through
- [x] **Bug 3 fix** — `http.MaxBytesReader(w, r.Body, 200<<20)` added before `ParseMultipartForm`; 200 MB hard limit enforced
- [x] **P1e** — Handler moved from `internal/api/v1/upload_csv.go` into `internal/api/v1/upload/upload.go` as a `Module` struct

---

## Technical Notes

- **Cache mechanism:** The validated file is saved to `tmp/` with its original filename. The filename is returned to the browser and stored in a hidden `<input name="uploaded_file_name">` field in the form. On submit, the handler reads this filename to locate the file — no re-upload needed.
- **Validation sequence:** Extension check → empty check → `csv.NewReader` first-row parse. Fails fast and returns an error partial if any check fails.
- **File cleanup:** Handled by `CSVFileSource.Cleanup()` (P1c). The processor calls `defer params.Source.Cleanup()` — the upload handler is no longer responsible for deletion.
- **200 MB limit:** `http.MaxBytesReader` wraps the request body before `ParseMultipartForm`. This turns the soft in-memory hint into a hard reject — the body read fails with `*http.MaxBytesError` once the limit is exceeded. 200 MB was chosen to accommodate large keyword exports.
- **Pattern used:** HTMX partial rendering — the upload zone swaps itself with the feedback component on success or error.
- **Module pattern:** `upload.Module{}` implements `module.Module` and is registered via `v1.Modules()` in `internal/api/v1/v1.go`. The `Register` method mounts `/upload/csv`.

---

## Change History

### 2026-03-07 — Bug fixes (Bug 2, Bug 3) + module restructure (P1e)

**Reason:** Bug 2 caused execution to continue past error responses. Bug 3 left the server open to arbitrarily large uploads. P1e moved all handlers into per-feature module subfolders.

**What changed:**
- [x] Missing `return` statements added after all `renderStatus(false, ...)` calls
- [x] `http.MaxBytesReader` added, limit set to 200 MB
- [x] Handler extracted from `internal/api/v1/upload_csv.go` (deleted) into `internal/api/v1/upload/upload.go`
- [x] `upload.Module{}` created; registered via `v1.Modules()` collector

**Notes:** The original `upload_csv.go` file is deleted. The new `upload/upload.go` contains identical logic with the two bug fixes applied and the module wrapper added.
