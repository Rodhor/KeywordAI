# Feature Plan — Settings / API Key Management

**Mode: FEATURE**
**Date:** 2026-03-13
**Status:** Planning

---

## What the Feature Does and Why It Belongs Here

Currently `ai.NewClient(ctx)` reads `GEMINI_API_KEY` directly from the process environment via `os.Getenv`. This works during development with a `.env` file loaded by `godotenv`, but it is the wrong approach for a shipped desktop app — users cannot and should not be editing `.env` files.

This feature introduces a persistent config file at `~/.config/keywordai/config.json` (XDG standard location on Linux/macOS; `%AppData%\keywordai\config.json` on Windows, handled automatically by `os.UserConfigDir()`). A settings page in the frontend lets users enter their API keys once. If the Gemini key is absent on startup, the app redirects to the Settings page before any tool can be used.

The `.env` fallback is kept only for local development — `config.Load()` reads the file first and falls back to environment variables so existing dev workflows are undisturbed.

---

## Architecture — Where Each Piece Goes

This feature fits the existing dependency-injection pattern in `backend/backend.go`. No new architectural pattern is introduced: config is loaded at startup and the key is passed as a constructor argument, exactly like any other dependency.

```
internal/config/config.go
    owns: Config struct, Load(), Save(), ConfigPath()
    does NOT own: AI client lifecycle, UI concerns

backend/commands/settings/settings.go
    owns: SettingsCommand — the Wails-bound command that bridges config to the frontend
    does NOT own: reading/writing the file directly (delegates to internal/config)

backend/backend.go
    change: call config.Load() before ai.NewClient(); pass key as argument; wire b.Settings.Setup(cfg)
    change: store cfg on Backend so ReinitAI() can rebuild the client in place

main.go
    change: add app.Backend.Settings to the Bind slice

internal/ai/gemini.go
    change: NewClient(ctx, apiKey string) — accept the key as a parameter, remove os.Getenv call

frontend/src/routes/settings/+page.svelte
    owns: settings form — Gemini key input, Google credentials placeholder, Save button

frontend/src/lib/components/NavBar.svelte
    change: add a Settings (gear) icon button in AppBar.Trail

frontend/src/routes/+layout.svelte
    change: on mount, call HasRequiredConfig(); if false and not on /settings, goto('/settings')
```

---

## How It Works

### Config file

`internal/config/config.go` resolves the path using `os.UserConfigDir()`, which returns the platform-appropriate config directory:

- Linux/macOS: `~/.config/keywordai/config.json`
- Windows: `%AppData%\keywordai\config.json`

The directory is created with `os.MkdirAll` if it does not exist. After writing, `os.Chmod(path, 0600)` restricts the file to owner-read/write only. No encryption — the key is stored in plain text in a file only the OS user can read, which is the standard approach for API keys in desktop tools.

### JSON structure

```json
{
  "gemini_api_key": "AIza...",
  "google_credentials_json": ""
}
```

Both fields are always present in the file. An empty string means "not configured yet". The `google_credentials_json` field is stored but never used by any backend logic in this iteration — it is a forward-declared placeholder for the future Google OAuth feature.

### .env fallback for development

`config.Load()` reads the JSON file first. If the file does not exist or a field is empty, it falls back to `os.Getenv("GEMINI_API_KEY")`. This means developers who already have a working `.env` file do not need to change anything — the fallback fires automatically.

### First-launch detection

`SettingsCommand.HasRequiredConfig()` returns `true` only when `GeminiAPIKey` is non-empty. The frontend `+layout.svelte` calls this on mount and redirects to `/settings` if it returns `false` and the current route is not already `/settings`.

---

## Implementation Steps — Phase 1: Go Backend

### 1. Create `internal/config/config.go`

- [ ] Define `Config` struct with two exported string fields: `GeminiAPIKey` and `GoogleCredentialsJSON`
- [ ] `ConfigPath() (string, error)` — calls `os.UserConfigDir()`, appends `keywordai/config.json`, returns the full path
- [ ] `Load() (*Config, error)` — reads and JSON-unmarshals the file at `ConfigPath()`; if the file does not exist, returns an empty `Config{}` (not an error); after reading the file, for any field that is still empty, check the corresponding env var and use it if present (`GEMINI_API_KEY` for `GeminiAPIKey`)
- [ ] `Save(cfg *Config) error` — marshals to indented JSON, calls `os.MkdirAll` on the parent directory, writes the file with `os.WriteFile`, then calls `os.Chmod(path, 0600)`

### 2. Modify `internal/ai/gemini.go`

- [ ] Change `NewClient(ctx context.Context)` signature to `NewClient(ctx context.Context, apiKey string) (*Client, error)`
- [ ] Remove the `os.Getenv("GEMINI_API_KEY")` call — use the `apiKey` parameter directly
- [ ] Confirm nothing else in the codebase calls `ai.NewClient` (currently only `backend/backend.go` does)

### 3. Create `backend/commands/settings/settings.go`

- [ ] Define `Settings` struct (the frontend-facing type): `GeminiAPIKey string` and `GoogleCredentialsJSON string` — same shape as `config.Config`, intentionally separate so that the internal config type and the public Wails API can evolve independently
- [ ] Define `SettingsCommand` struct with fields: `cfg *config.Config` and `reinitAI func(apiKey string) error`
- [ ] `Setup(cfg *config.Config, reinitFn func(string) error)` — stores both on the struct
- [ ] `GetSettings() (Settings, error)` — reads from `config.Load()` and returns the frontend type; errors from the file read are returned verbatim
- [ ] `SaveSettings(s Settings) error` — writes to `config.Save()`, then calls `b.reinitAI(s.GeminiAPIKey)` to take effect immediately without a restart; returns any error from either step
- [ ] `HasRequiredConfig() bool` — calls `config.Load()` internally and returns `cfg.GeminiAPIKey != ""`

### 4. Update `backend/backend.go`

- [ ] Add `Settings *settings.SettingsCommand` field to `Backend` struct
- [ ] Initialise it in `New()`: `Settings: &settings.SettingsCommand{}`
- [ ] In `Setup()`: call `cfg, err := config.Load()` at the top, before `ai.NewClient`
- [ ] Pass `cfg.GeminiAPIKey` to `ai.NewClient(ctx, cfg.GeminiAPIKey)`
- [ ] Add a `reinitAI` closure that rebuilds the client and rewires `b.Keywords`: `func(apiKey string) error { client, err := ai.NewClient(ctx, apiKey); ...; b.Keywords.Setup(jobMgr, client, client); return err }`
- [ ] Call `b.Settings.Setup(cfg, reinitAI)` before the `slog.Info("Backend ready")` line
- [ ] Keep `loadDotEnv()` in place — it still runs before `config.Load()` so that env vars (set by dotenv) are available as fallbacks

### 5. Update `main.go`

- [ ] Add `app.Backend.Settings` to the `Bind` slice in `wails.Run`

---

## Implementation Steps — Phase 2: Frontend

### 6. Create `frontend/src/routes/settings/+page.svelte`

- [ ] Import `SettingsCommand` from `$lib/wailsjs/go/settings/SettingsCommand` (Wails generates this from the bound struct)
- [ ] On mount, call `GetSettings()` to pre-populate the form fields
- [ ] Gemini API key: `<input type="password">` with a show/hide toggle button (eye icon); field label "Gemini API Key"; marked required
- [ ] Google credentials JSON: `<textarea>` that is `disabled`; label "Google Credentials JSON"; include a "Coming Soon" badge next to the label; tooltip or hint text explaining it will be needed for Google Sheets export
- [ ] Save button: calls `SaveSettings({ GeminiAPIKey, GoogleCredentialsJSON })`; on success, show a brief success toast or inline message ("Settings saved"); on error, show the error message inline
- [ ] The page must work both as a first-launch onboarding screen and as a routine settings page reached from the NavBar — no special-case branching needed; the same form handles both

### 7. Add Settings icon to `frontend/src/lib/components/NavBar.svelte`

- [ ] Import `SettingsIcon` from `@lucide/svelte`
- [ ] Add an `<a href="/settings">` wrapping a `btn-icon hover:preset-tonal` inside `AppBar.Trail`, placed before `<LightSwitch />`
- [ ] The icon should not be shown as "active/selected" on non-settings pages — no special state needed; the back arrow in the Lead already handles navigation on any sub-page

### 8. Add first-launch guard to `frontend/src/routes/+layout.svelte`

- [ ] Import `goto` from `$app/navigation` and `onMount` from `svelte`
- [ ] Import `HasRequiredConfig` from the generated Wails bindings
- [ ] In `onMount`, call `HasRequiredConfig()`. If it returns `false` and `page.url.pathname !== '/settings'`, call `goto('/settings')`
- [ ] The guard runs on every page load — if the user navigates away from `/settings` before saving a key, they will be redirected back. This is intentional and correct behaviour.

---

## Key Decisions to Flag Before Implementation

### Decision 1 — AI client reinitialisation on Save

When `SaveSettings` is called, the new API key must take effect immediately. Two options:

**Option A — Restart required:** `SaveSettings` only writes to disk. The frontend shows "Restart the app to apply your new key." Simple, zero risk of race conditions, but bad UX.

**Option B — In-place reinit (recommended):** `SaveSettings` calls a `reinitAI(key string) error` closure that was injected into `SettingsCommand` during `Setup()`. The closure rebuilds the `*ai.Client` and re-calls `b.Keywords.Setup(jobMgr, newClient, newClient)`. The new client is live for the next job submitted.

The tradeoff with Option B: if a keyword evaluation job is in flight when the user saves a new key, that job will use whichever client was bound at job-launch time — it will not be affected mid-run. This is safe. New jobs after `SaveSettings` returns will use the new key. No locking is needed because `b.Keywords.Setup()` is only called from the main goroutine (startup and save), never concurrently.

**Recommendation: Option B.** Document the in-flight behaviour in Technical Notes after implementation.

### Decision 2 — `google_credentials_json` is a placeholder

The field is stored in `config.json` and returned by `GetSettings()` / accepted by `SaveSettings()`, but nothing in the backend reads or uses it yet. The UI field is disabled with a "Coming Soon" badge. This is intentional: the field exists in the config schema now so that when Google OAuth is implemented, the config file format does not break for users who already have a saved config. Do not add any validation or logic for this field in this iteration.

### Decision 3 — `config.Load()` is called multiple times

`GetSettings()` and `HasRequiredConfig()` each call `config.Load()` independently rather than reading from the in-memory `cfg` stored on `SettingsCommand`. This is correct: these calls are infrequent (once on app launch, once on Settings page open, once on Save), and always reading from disk avoids a stale-memory problem if the file is edited externally. This is not a performance concern for a desktop app.

### Decision 4 — Error handling when `HasRequiredConfig` fails

If `config.Load()` returns an error inside `HasRequiredConfig()`, return `false`. This redirects the user to Settings, where they can enter the key manually. Do not show a cryptic load error on the layout — the Settings page itself will surface errors via `GetSettings()`.

---

## How to Test

- [ ] Delete (or rename) `~/.config/keywordai/config.json` if it exists. Launch the app. Verify the browser redirects to `/settings` immediately and does not show the Tools page.
- [ ] Enter a valid Gemini API key in the Settings form and click Save. Verify a success message appears. Verify the file now exists at `~/.config/keywordai/config.json` with mode `600` (`ls -l`). Verify the JSON contains the key.
- [ ] Navigate to `/` after saving. Verify the Tools page loads without being redirected back to `/settings`.
- [ ] Click the Settings gear icon in the NavBar. Verify it navigates to `/settings` and the form pre-populates with the saved key (shown as a password field).
- [ ] Change the Gemini key to a different value and save. Without restarting the app, navigate to Keyword Evaluator and submit a job. Verify the job runs using the new key (if the new key is invalid, the job should fail — confirm the error comes from the AI call, not from a missing key).
- [ ] Verify the Google credentials JSON field is visible, disabled, and labelled "Coming Soon".
- [ ] On Linux: confirm `~/.config/keywordai/config.json` has permissions `600`. On macOS: confirm `~/Library/Application Support/keywordai/config.json` exists with permissions `600`.
- [ ] Rename `.env` to `.env.bak` and restart the app with no config file. Verify it redirects to Settings. Enter the key, save. Verify everything works. Restore `.env.bak`.
- [ ] With no `.env` and a valid config file: verify the app starts on the Tools page without error.
