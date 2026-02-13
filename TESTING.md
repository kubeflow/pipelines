# Testing Guide for Issue #12801

## Overview
This fix changes the `LocalStorage.hasKey()` method, which affects the auto-collapse behavior of the side navigation on mobile widths. This document explains how to test and verify the fix.

## Unit Tests

### Running the LocalStorage tests

```bash
cd frontend
npm install  # if not already done
npm test -- LocalStorage.test.ts
```

The test suite in `frontend/src/lib/LocalStorage.test.ts` verifies:
- `hasKey()` returns `false` when a key doesn't exist (the fix for #12801)
- `hasKey()` returns `true` when a key exists with any value
- Integration scenarios for auto-collapse behavior

## Visual Regression Testing

### Mobile Width Snapshots

The issue mentions that this fix will affect mobile layout broadly and requires rebaselining visual snapshots. The KFP project uses a smoke test framework for visual regression testing.

### To rebaseline mobile snapshots after this fix:

1. **Navigate to the smoke test directory:**
   ```bash
   cd frontend/scripts/ui-smoke-test
   npm install
   npx playwright install chromium
   ```

2. **Run comparison tests with mobile viewport (390x844):**
   ```bash
   UI_SMOKE_VIEWPORTS="390x844" node smoke-test-runner.js --compare master
   ```

   This will:
   - Capture screenshots from the base branch (master)
   - Capture screenshots from your branch with the fix
   - Generate side-by-side comparisons
   - Show which pages have visual differences

3. **Expected visual changes:**
   - On mobile widths (390x844), the side navigation should now be collapsed by default
   - Footer controls on Runs and Recurring Runs pages should no longer be clipped
   - More horizontal space available for table content

4. **Pages most likely to show differences:**
   - `/#/runs` (Runs page)
   - `/#/recurringruns` (Recurring Runs page)
   - Any page viewed at mobile width will show the collapsed sidebar

5. **Review and accept changes:**
   - Review the generated comparison images in `.ui-smoke-test/screenshots/comparison/`
   - Verify that the side nav is collapsed on mobile
   - Verify that table footer controls are fully visible
   - If changes look correct, these become the new baseline

### Additional viewport testing

You can also test multiple viewports at once:
```bash
UI_SMOKE_VIEWPORTS="390x844,768x1024,1280x800" node smoke-test-runner.js --compare master
```

This will capture screenshots at:
- 390x844 (mobile - should show collapsed nav)
- 768x1024 (tablet - may show collapsed nav)
- 1280x800 (desktop - should show expanded nav)

## Manual Testing

### Local development testing:

1. **Start the frontend development server:**
   ```bash
   cd frontend
   npm start
   ```

2. **Test in browser:**
   - Open the app at http://localhost:3000
   - Open browser DevTools (F12)
   - Use Device Toolbar to set viewport to 390x844 (or any width < 800px)
   - Clear localStorage: `localStorage.clear()` in console
   - Refresh the page
   - **Expected:** Side navigation should be collapsed
   - Navigate to `/#/runs` or `/#/recurringruns`
   - **Expected:** Table footer controls (pagination, rows-per-page) should be fully visible

3. **Test manual collapse/expand:**
   - Click the chevron button to expand the nav
   - Resize window to wide (>800px) then back to narrow (<800px)
   - **Expected:** Nav should stay expanded (manual state is preserved)
   - Click the chevron to collapse
   - Resize window again
   - **Expected:** Nav should stay collapsed (manual state is preserved)

4. **Test auto-collapse:**
   - Clear localStorage again: `localStorage.clear()`
   - Set viewport to wide (>800px)
   - **Expected:** Nav should be expanded
   - Resize to narrow (<800px)
   - **Expected:** Nav should auto-collapse
   - Resize back to wide
   - **Expected:** Nav should auto-expand

## Key Behavior Changes

### Before the fix:
- `LocalStorage.hasKey()` always returned `true` (bug)
- `manualCollapseState` was always `true` in SideNav
- Auto-collapse never triggered, even on mobile
- Result: Footer controls clipped on narrow screens

### After the fix:
- `LocalStorage.hasKey()` correctly returns `false` when key is missing
- `manualCollapseState` is `false` on first visit (no manual interaction yet)
- Auto-collapse triggers based on window width (< 800px)
- Result: Mobile layout has more horizontal space, footer controls visible

## Related Files

- `frontend/src/lib/LocalStorage.ts` - The fixed method
- `frontend/src/components/SideNav.tsx` - Uses `hasKey()` to determine `manualCollapseState`
- `frontend/src/lib/LocalStorage.test.ts` - Unit tests for the fix
- `frontend/scripts/ui-smoke-test/` - Visual regression test framework
