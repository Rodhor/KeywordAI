# Navigation

> ⚠ **Superseded — 2026-03-13**
> The Templ/HTMX navbar was replaced by `frontend/src/lib/components/NavBar.svelte` during the Wails migration. The new navbar uses Skeleton UI v4's `AppBar` component, SvelteKit's `$app/state` page store for route detection, and includes a `LightSwitch` button. See `NavBar.svelte` for the current implementation.

- **Status:** Removed
- **Created:** 2026-03-02 (onboarded — code predates this record)
- **Last updated:** 2026-03-13
- **Touches:** ~~`ui/customComponents/Navigation/NavBar.templ`~~ · ~~`ui/layouts/base.templ`~~ (all removed)

---

## What It Does

A sticky top navigation bar that appears on every page. Contains the application logo/name on the left and a "Create Another Campaign" home button and the theme toggle on the right. The home button resets the main content area to the upload form without a full page reload.

---

## Why It Was Built

Users need a consistent way to start over after reviewing results, without losing the nav bar or triggering a full page reload. A sticky navbar keeps navigation accessible even when scrolling through a long keyword results table.

---

## Implementation Checklist

### Initial implementation — (onboarded 2026-03-02)
- [x] `NavBar` Templ component renders sticky header
- [x] Logo / app name on the left
- [x] Home button with HTMX `hx-get="/api/v1/home"` to reset main content
- [x] `ThemeToggle` component embedded on the right
- [x] Included in `base.templ` layout so it appears on all pages

---

## Technical Notes

- **Home button behaviour:** Uses HTMX `hx-get` to fetch the landing page content partial and swap it into the main content target. The navbar itself is outside the swap target and remains untouched — this gives an SPA-like feel without a page reload.
- **Sticky positioning:** CSS `position: sticky; top: 0` keeps the bar visible while scrolling. Important for the results table which can be long.
- **ThemeToggle composition:** The theme toggle is a separate Templ component rendered inside NavBar. This keeps the toggle logic self-contained and reusable if it were ever needed elsewhere.
- **Pattern used:** HTMX partial rendering — the home button triggers a partial swap of the main content, not a full navigation.

---

## Change History

*(no changes since initial implementation)*
