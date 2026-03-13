# KeywordAI — Project Summary

## Overview

**KeywordAI** (also branded as "Google Ads Helper" in the PWA manifest) is a web application for validating and evaluating keyword lists for Google Ads campaigns. Users upload a CSV file containing keywords, provide a campaign description and optional exclusion rules, and receive an AI-enriched result table where each keyword has been rated and reasoned about by Google Gemini.

The application runs as a Go HTTP server on port `8090`. The frontend is rendered server-side using the Templ templating engine with HTMX for dynamic interactions — there is no JavaScript framework.

---

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Language | Go 1.24 |
| Web server | `net/http` (standard library) |
| Templates | [Templ](https://templ.guide) — type-safe Go templates |
| UI components | [TemplUI](https://templui.io) — Tailwind-based component library |
| Interactivity | [HTMX](https://htmx.org) — partial HTML swaps, polling |
| Styling | Tailwind CSS 3 + custom CSS |
| AI | Google Gemini API (`google.golang.org/genai`) |
| Dev tooling | Taskfile, godotenv |
| Deployment | Docker (multi-stage build) |

---

## Project Structure

```
Demo/
├── cmd/
│   └── main.go                        # Entry point: logger, env, AI client, server start
├── internal/
│   ├── ai/
│   │   ├── ai.go                      # Interfaces: KeywordEvaluator, KeywordGenerator, KeywordOrganizer
│   │   ├── gemini.go                  # Gemini API implementation (working)
│   │   └── openai.go                  # OpenAI stub (empty — not implemented)
│   ├── api/v1/
│   │   ├── home.go                    # GET /api/v1/home
│   │   ├── upload_csv.go              # POST /api/v1/upload/csv
│   │   ├── KeyWordsSubmit.go          # POST /api/v1/submit/keywordupload
│   │   └── JobStatus.go               # GET /api/v1/submit/status
│   ├── processor/
│   │   ├── keywords.go                # Keyword cleaning + AI call pipeline
│   │   ├── ProcessAIResponse.go       # ZipResults(): merges original + AI columns
│   │   └── ProcessFilesToSlices.go    # ParseCSVToSlice(): CSV to 2D string slice
│   ├── jobs/
│   │   └── manager.go                 # Async job manager (UUID, status, auto-cleanup)
│   ├── services/
│   │   ├── server.go                  # HTTP server setup, routing, graceful shutdown
│   │   ├── services.go                # Handler bindings for routes
│   │   └── middleware.go              # Middleware definitions (currently unused)
│   ├── store/
│   │   └── jobs.go                    # Legacy in-memory store (deprecated, not used)
│   └── database/                      # Placeholder — empty, no DB implemented
├── ui/
│   ├── layouts/
│   │   └── base.templ                 # Base HTML layout, service worker, dark mode
│   ├── pages/
│   │   └── landing.templ              # Landing page
│   └── customComponents/
│       ├── Forms/KeyWordsUploadForm.templ    # Main form (description, tags, file upload)
│       ├── FileUpload/dragNdrop.templ        # Drag-and-drop zone with validation states
│       ├── Loading/PollingUI.templ           # Spinner — HTMX polls every 1 second
│       ├── DataDisplay/KeywordTable.templ    # Results table with checkboxes
│       └── Navigation/
│           ├── NavBar.templ                  # Sticky navbar with logo and home button
│           └── ThemeToggle.templ             # Dark/light mode toggle
├── assets/
│   ├── manifest.json                  # PWA manifest
│   ├── css/                           # Tailwind output + custom CSS
│   ├── js/                            # HTMX library + component JS
│   └── assets.go                      # Embedded assets (Go embed)
├── documentation/
│   └── Planning.md                    # This file
├── tmp/                               # Temporary uploaded CSV files (gitignored)
├── Dockerfile
├── Taskfile.yml
└── .env                               # GEMINI_API_KEY, GO_ENV (gitignored)
```

---

## User Flow

```
1. User uploads a CSV file
       ↓ POST /api/v1/upload/csv
       ↓ Validates extension, non-empty, readable CSV
       ↓ Saves to tmp/<filename>
       ↓ Returns success/error feedback component

2. User fills in the form
       - Campaign description (required)
       - Keywords to exclude (optional, tag input)
       - Custom instructions (optional, currently unused)

3. User submits
       ↓ POST /api/v1/submit/keywordupload
       ↓ Job created (UUID), background goroutine starts
       ↓ Returns PollingUI component with job ID

4. HTMX polls every 1 second
       ↓ GET /api/v1/submit/status?id=<jobID>
       ↓ Returns spinner while processing
       ↓ Returns results table when done
       ↓ Returns error message on failure

5. User reviews results table
       - Headers: original columns + "Evaluation" + "Reasoning"
       - Each row has a checkbox (all checked by default)
       - "Create Another Campaign" button returns to the form
```

---

## Keyword Processing Pipeline

Executed in a background goroutine (`internal/processor/keywords.go`):

1. **DropUnwantedColumns** — placeholder, currently a no-op
2. **ExtractHeaderRow** — separates header row from data rows
3. **RemoveEmptyLines** — removes rows where the first column is empty
4. **RemoveDuplicateLines** — case-insensitive dedup on the first column
5. **DropExcludedKeywords** — removes rows containing any excluded keyword (substring, case-insensitive)
6. **SendToAIForValidation** — sends cleaned keywords + campaign description to Gemini
7. **ZipResults** — appends AI-returned columns to original rows

The Gemini call uses structured JSON output (schema-validated), temperature 0.0, and a system prompt that positions the model as an SEO expert. Missing AI rows are padded with "N/A".

---

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/home` | Landing page — returns full HTML or HTMX partial |
| POST | `/api/v1/upload/csv` | Upload and validate a CSV file |
| POST | `/api/v1/submit/keywordupload` | Start async keyword processing job |
| GET | `/api/v1/submit/status?id=<jobID>` | Poll job status (used by HTMX) |
| GET | `/api/v1/assets/` | Static assets (CSS, JS, images) |

---

## Async Job Manager

`internal/jobs/manager.go`

- Jobs are stored in a thread-safe in-memory map (`sync.RWMutex`)
- Each job has a UUID, a status (`processing` / `completed` / `failed`), and a typed payload
- A background goroutine runs every **10 minutes** and removes jobs older than **30 minutes**
- All job data is lost on server restart (no persistence)

---

## Configuration

| Variable | Description |
|----------|-------------|
| `GEMINI_API_KEY` | Required. Gemini API key loaded from `.env` |
| `GO_ENV` | Optional. Set to `production` to enable asset caching |

Server port is hardcoded to `8090`.

---

## What Is Already Implemented

- CSV file upload with drag-and-drop, extension validation, and visual feedback
- Keyword cleaning pipeline (empty lines, duplicates, exclusions)
- Gemini API integration returning structured "Evaluation" and "Reasoning" columns
- Async job processing with UUID job IDs
- HTMX polling with spinner UI
- Results table with per-row checkboxes
- Dark/light theme toggle (localStorage-persisted)
- PWA support (manifest, service worker, icons)
- Graceful server shutdown on SIGINT/SIGTERM
- Structured logging to stdout and `app.log`
- Navbar with logo and home/reset button
- Basic endpoint test (`internal/services/home/home_test.go`)

---

## What Is Not Yet Implemented

| Feature | Status | Notes |
|---------|--------|-------|
| **OpenAI provider** | Stub only | `internal/ai/openai.go` is empty |
| **Custom instructions** | Form field exists, value ignored | Never forwarded to AI call |
| **Column filtering** | Placeholder | `DropUnwantedColumns()` is a no-op |
| **Result export / download** | Not started | No CSV/JSON download from results table |
| **Database persistence** | Not started | `internal/database/` is empty; all state is in-memory |
| **Middleware (logging, auth)** | Defined, not applied | `middleware.go` exists but routes don't use it |
| **Keyword generation / organization** | Interfaces defined | `KeywordGenerator` and `KeywordOrganizer` have no implementation |
| **Input validation** | Minimal | No max description length, no CSRF protection, no rate limiting |
| **File size enforcement** | Partial | 10 MB stated in UI but not strictly enforced server-side |
| **AI call timeout** | Missing | Gemini requests can hang indefinitely |
| **Test coverage** | Very low | Only one test file; processor, AI layer, job manager untested |

---

## Known Technical Debt

- `internal/store/jobs.go` is a legacy in-memory store no longer used (superseded by `internal/jobs/manager.go`)
- Upload handler does not check errors returned by template render calls
- Concurrent file reads and deferred cleanup could produce a race condition under high load
- No secrets manager — API key lives in a plain `.env` file
