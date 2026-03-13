# Brainstorm — Campaign Setup and Google Workspace Integration

**Date:** 2026-03-08
**Status:** Direction agreed — proceeding to FEATURE

---

## Problem Statement

The app evaluates and clusters keywords but has no concept of a campaign — a named entity that owns the Google Workspace artefacts (folder, Docs, Sheets) that the results are written to. Without it, there is nowhere to export results, and the user must re-enter the campaign description every time.

---

## What Success Looks Like

1. The user fills in a short campaign form (company name, short description, URL) once.
2. A Google Drive folder, a Google Docs file, and a Google Sheets file are created in the background.
3. An AI-generated long description is written to the Docs file in the background.
4. The user proceeds immediately to keyword upload — they are not blocked by the background work.
5. A toast notification appears when the Workspace setup is complete.
6. At the end of the clustering step, all keyword data (original + evaluated + clustered) is written automatically to the Sheets file.
7. The user sees a "Done" screen with a direct link to the Google Sheet.

---

## What Changes from the Current Flow

**Current:**
```
Upload form (user enters description) → eval → results table → cluster → [nothing yet]
```

**New:**
```
Campaign form (name, short desc, URL)
  → campaign job starts in background:
      - AI generates long description from short desc + URL
      - Google Drive: create folder
      - Google Drive: create Docs file, write long description
      - Google Drive: create Sheets file (empty, ready for data)
      - campaign job completes with payload: {FolderID, DocsID, SheetsID}
  → toast fires: "Your Google Workspace is ready"
  → user lands immediately on upload form (campaign job ID in hidden field)

Upload form (description comes from campaign, no re-entry needed)
  → eval job → results table → cluster submission
      → cluster + export job:
          - waits for campaign job if not yet complete (internal poll, shown as PollingUI)
          - runs AI clustering on selected rows
          - writes original + evaluated + clustered data to SheetsID
          - completes with payload: {SheetURL}
      → PollingUI resolves → "Done" screen with link to sheet
```

---

## Key Decisions

### Clustering results are not displayed
The clustered data goes directly to Google Sheets. There is no second results table.
The cluster + export step is a single job — `ProcessClustering` is extended to also run the export.

### Short description is used for keyword evaluation
The AI-generated long description is for the Docs file only. The existing eval job uses the
user's original short description, unchanged. No blocking dependency between jobs.

### Campaign setup uses the existing job manager
The campaign job is a standard async job. Its UUID is returned to the browser immediately
and threads through every subsequent form as a hidden field. Nothing new is needed
in the job system itself.

### Campaign job ID is the linking key
The campaign job payload (when complete) holds the three Google file IDs. The export step
retrieves them by looking up the campaign job ID — the same pattern `cluster.go` already
uses to look up the eval job payload.

### If the campaign job is still running at export time
The export job waits internally, not by blocking the HTTP response. The user always sees
the standard PollingUI. The "saving to Google Workspace" message is the same regardless
of whether the wait was zero seconds or thirty.

### Toast notification mechanism
A small HTMX polling div — invisible in the normal layout — polls the campaign job status.
When it detects `StatusCompleted`, it swaps in a toast fragment and stops polling.
No WebSockets, no SSE — stays within the existing HTMX pattern.

---

## Open Questions / Risks

| Question | Decision |
|----------|----------|
| Where does the campaign description come from in the upload form, if the user no longer types it? | The campaign job ID is in a hidden field; the upload handler reads the description from the campaign job payload. The upload form no longer shows a description input field. |
| What if Google OAuth is not yet implemented? | Campaign setup and Sheets export are blocked until OAuth is in place. Build them together as a single FEATURE session. |
| What if the campaign job fails (e.g. Google API down)? | The export step will receive a `StatusFailed` campaign job and return an appropriate ErrorUI. |
| Does the upload form change UI at all? | Yes — the description textarea is removed. Company name and URL are shown as read-only context instead. This is a UI change to `KeyWordsUploadForm.templ`. |
| In-memory only? | Yes, for now. Campaign job lives in `jobs.Manager` like any other job. Data is lost on server restart — acceptable for solo use. SQLite is the future path if persistence is needed. |

---

## Agreed Implementation Order

1. **KeywordClustering** — implement the actual AI call in `ProcessClustering` (stub already exists). Clustering must work end-to-end before the export is wired.
2. **GoogleOAuth** — set up the shared OAuth flow for Google APIs (`internal/auth/google/`). Required by both Workspace setup and Sheets export.
3. **CampaignSetup** — campaign form, campaign job, Google Drive folder + Docs + Sheets creation, AI description generation, toast notification, campaign ID threading.
4. **GoogleSheetsExport** — extend `ProcessClustering` to also write data to the Sheets file using the campaign job payload.

Each step depends on the previous. Start with KeywordClustering.

---

## Transition

BRAINSTORM complete → proceeding to **FEATURE: KeywordClustering**
