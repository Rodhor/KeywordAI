# Theme Toggle

> ⚠ **Superseded — 2026-03-13**
> The Templ theme toggle was replaced by `frontend/src/lib/components/LightSwitch.svelte` during the Wails migration. The new component initialises `isDark` directly from the `data-mode` DOM attribute (set by an inline `<svelte:head>` script) to avoid a "switch twice" bug caused by `$effect` re-initialisation. Theme is persisted to `localStorage`. See `LightSwitch.svelte` for the current implementation.

- **Status:** Removed
- **Created:** 2026-03-02 (onboarded — code predates this record)
- **Last updated:** 2026-03-13
- **Touches:** ~~`ui/customComponents/Navigation/ThemeToggle.templ`~~ · ~~`ui/layouts/base.templ`~~ (all removed)

---

## What It Does

A sun/moon icon button in the navbar that switches the application between light and dark mode. The chosen theme is persisted in `localStorage` so it survives page reloads and new sessions.

---

## Why It Was Built

The app is used repeatedly in extended work sessions. Dark mode reduces eye strain. Persisting the preference means the user does not have to toggle it on every visit.

---

## Implementation Checklist

### Initial implementation — (onboarded 2026-03-02)
- [x] Sun/moon icon toggle button in `ThemeToggle.templ`
- [x] JavaScript in `base.templ` reads `localStorage` on page load and applies theme before first render
- [x] Toggle click updates `localStorage` and switches the active CSS class
- [x] Tailwind `dark:` variant classes used throughout the UI for theme-aware styling
- [x] No server-side state involved — entirely client-side

---

## Technical Notes

- **No flash of wrong theme:** The `base.templ` layout includes an inline script in `<head>` that reads `localStorage` and applies the theme class to `<html>` before the page renders. This prevents the brief flash of the wrong theme that would occur if the script ran after the body renders.
- **No server-side state:** Theme preference lives entirely in `localStorage`. There is no cookie, session, or API call involved. This keeps the server stateless with respect to user preferences.
- **Tailwind integration:** All theme-sensitive components use Tailwind's `dark:` variant (e.g. `dark:bg-gray-900`). Adding `class="dark"` to the `<html>` element activates all dark variants simultaneously.
- **Pattern used:** Client-side only — the server has no awareness of this feature beyond rendering the toggle component into the layout.

---

## Change History

*(no changes since initial implementation)*
