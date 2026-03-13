# Settings / API Key Management

- **Status:** Planned
- **Created:** 2026-03-13
- **Last updated:** 2026-03-13
- **Touches:**
  - `internal/config/config.go` (new)
  - `backend/commands/settings/settings.go` (new)
  - `backend/backend.go` (modified)
  - `main.go` (modified)
  - `internal/ai/gemini.go` (modified)
  - `frontend/src/routes/settings/+page.svelte` (new)
  - `frontend/src/lib/components/NavBar.svelte` (modified)
  - `frontend/src/routes/+layout.svelte` (modified)

---

## What It Does

Provides a persistent settings page where users enter their Gemini API key once. The key is saved to a JSON config file in the OS-standard user config directory (`~/.config/keywordai/config.json` on Linux/macOS) with file permissions set to owner-read/write only. On first launch — or any time the Gemini key is missing — the app redirects to the Settings page before the user can access any tool. The Settings page is also reachable at any time from a gear icon in the NavBar. A `google_credentials_json` field is present in the form and config but disabled and marked "Coming Soon"; it is a forward-declared placeholder for the future Google OAuth feature.

---

## Why It Was Built

The previous approach read `GEMINI_API_KEY` from a `.env` file via `godotenv`. This is appropriate for development but wrong for a shipped desktop app: end users cannot and should not be editing `.env` files. A config file in the user's profile directory is the standard approach for desktop application secrets that do not require encryption — the OS enforces access control via file permissions. The `.env` fallback is retained so existing development workflows continue to work without any changes.

---

## Implementation Checklist

### Phase 1: Go backend — 2026-03-13

- [ ] Create `internal/config/config.go`: `Config` struct, `ConfigPath()`, `Load()` (with env fallback), `Save()` (with chmod 600)
- [ ] Modify `internal/ai/gemini.go`: change `NewClient(ctx)` to `NewClient(ctx, apiKey string)`, remove `os.Getenv` call
- [ ] Create `backend/commands/settings/settings.go`: `SettingsCommand` with `GetSettings()`, `SaveSettings()`, `HasRequiredConfig()`, and injected `reinitAI` closure
- [ ] Update `backend/backend.go`: call `config.Load()` first; pass key to `ai.NewClient`; define `reinitAI` closure; add `Settings` field; call `b.Settings.Setup(cfg, reinitAI)`
- [ ] Update `main.go`: add `app.Backend.Settings` to the `Bind` slice

### Phase 2: Frontend — 2026-03-13

- [ ] Create `frontend/src/routes/settings/+page.svelte`: form with Gemini key input (password + show/hide toggle), disabled Google credentials field with "Coming Soon" badge, Save button with success/error feedback, pre-populated from `GetSettings()` on mount
- [ ] Update `frontend/src/lib/components/NavBar.svelte`: add `SettingsIcon` button in `AppBar.Trail` linking to `/settings`
- [ ] Update `frontend/src/routes/+layout.svelte`: on mount, call `HasRequiredConfig()`; if false and not on `/settings`, call `goto('/settings')`

### Verify

- [ ] First-launch redirect: no config file → app opens on Settings page
- [ ] Save writes correct JSON with mode 600
- [ ] After save, Tools page loads without redirect
- [ ] Key change takes effect immediately (no restart required)
- [ ] Google credentials field is visible, disabled, and labelled "Coming Soon"
- [ ] .env fallback still works in development

---

## Technical Notes

_To be filled in after implementation. Say "I finished Settings" to trigger this._

- **Config file location:** —
- **Reinit approach chosen (Option A vs B):** —
- **File permissions:** —
- **Env fallback behaviour:** —
- **Deviations from plan:** —

---

## Change History

_No changes yet — feature not yet implemented._
