# UI Smoke Test Tool

Visual regression testing for Kubeflow Pipelines frontend. Captures screenshots of key UI pages and generates side-by-side comparisons between branches.

## Quick Start

```bash
cd scripts/ui-smoke-test
npm install
npx playwright install chromium
```

## Common Workflows

### Screenshot your running dev server

The simplest and most useful workflow. Start your frontend (e.g. `npm start` in `frontend/`), then:

```bash
node smoke-test-runner.js --current-only --use-existing --url http://localhost:3000
```

Screenshots are saved to `.ui-smoke-test/screenshots/pr/`.

### Screenshot a static build (no backend)

Builds the frontend and serves it without a backend. API-dependent pages will show error banners, but layout, styling, and static content are captured.

```bash
cd ../../frontend && npm run build && cd -
node smoke-test-runner.js --current-only
```

### Screenshot a static build with a real backend (proxy mode)

Serves a static build while proxying API calls to a running backend. Useful for testing a production build against your dev cluster.

```bash
node smoke-test-runner.js --current-only --proxy --backend http://localhost:3000
```

### Full branch comparison

Compare the current branch against `master`:

```bash
node smoke-test-runner.js --pr 12756
```

This clones `master`, builds both branches, serves them on separate ports, captures screenshots, and generates side-by-side comparison images with diff percentages.

### Full comparison with proxy to real backend

```bash
node smoke-test-runner.js --pr 12756 --proxy --backend http://localhost:3000
```

## Command Reference

### Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--current-only` | Screenshot current build only (no branch comparison) | off |
| `--use-existing` | Use an already-running server instead of building/serving | off |
| `--url <url>` | URL of existing server (requires `--use-existing`) | `http://localhost:3000` |
| `--proxy` | Proxy API calls to a real backend when serving static builds | off |
| `--backend <url>` | Backend URL for proxy mode | `http://localhost:3000` |
| `--pr <number>` | PR number for full branch comparison | (none) |
| `--base <branch>` | Base branch to compare against | `master` |
| `--repo <owner/repo>` | GitHub repository | `kubeflow/pipelines` |
| `--mode <mode>` | Backend mode: `auto`, `cluster`, `mock`, `static` | `auto` |
| `--start-cluster` | Start a Kind cluster if none is running | off |
| `--seed-data` | Seed sample data (cluster mode) | off |
| `--skip-seed` | Skip automatic data seeding | off |
| `--skip-build` | Skip the `npm ci && npm run build` step | off |
| `--skip-upload` | Skip uploading results to PR | off |
| `--keep-servers` | Don't kill servers on exit (for debugging) | off |
| `--verbose` | Show all command output | off |

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `UI_SMOKE_PAGES` | Comma-separated list of pages to capture | all pages |
| `UI_SMOKE_VIEWPORT` | Viewport dimensions as `WIDTHxHEIGHT` | `1280x800` |
| `API_BASE` | API base URL for data seeding | `http://localhost:3001` |

### Individual Scripts

Each script can be run standalone:

```bash
# Capture screenshots from a running server
node capture-screenshots.js --port 3000 --output ./my-screenshots --label "my-branch"

# Generate side-by-side comparisons from two sets of screenshots
node generate-comparison.js --main ./screenshots/main --pr ./screenshots/pr --output ./comparison

# Post comparison results to a GitHub PR
node upload-to-pr.js --pr 12756 --repo kubeflow/pipelines --screenshots ./comparison

# Start the proxy server standalone
node proxy-server.js --build ../../frontend/build --port 4001 --backend http://localhost:3000

# Seed test data into a running KFP API
node seed-data.js              # skip if data exists
node seed-data.js --force      # overwrite existing data

# Filter specific pages
UI_SMOKE_PAGES=pipelines,runs node capture-screenshots.js --port 3000 --output ./screenshots
```

## Pages Captured

| Page | Route | Wait Condition | Description |
|------|-------|----------------|-------------|
| pipelines | `/#/pipelines` | Table rows + pipeline links | Pipeline list |
| experiments | `/#/experiments` | Table rows + experiment links | Experiment list |
| runs | `/#/runs` | Table rows + run links | Run history |
| recurring-runs | `/#/recurringruns` | Table rows | Scheduled runs |
| artifacts | `/#/artifacts` | Table rows | ML artifacts |
| executions | `/#/executions` | Table rows + execution links | Execution history |
| pipeline-create | `/#/pipeline/create` | Input field | Create pipeline form |
| experiment-create | `/#/experiments/new` | Input field | Create experiment form |

## Output

Results are saved to `.ui-smoke-test/` at the repo root (gitignored):

```
.ui-smoke-test/
  screenshots/
    main/                    # Screenshots from base branch
      pipelines.png
      experiments.png
      manifest.json          # Capture metadata (timestamp, viewport, etc.)
    pr/                      # Screenshots from PR / current branch
      pipelines.png
      experiments.png
      manifest.json
    comparison/              # Side-by-side comparisons
      pipelines.png          # Left = main, Right = PR, with labels
      experiments.png
      summary.json           # Diff percentages per page
```

The comparison images use `sharp` for pixel-level diff calculation (threshold: 5 per channel) and generate labeled side-by-side PNGs.

## Backend Modes

### `auto` (default)

Detects available infrastructure in order: Kind cluster > mock backend > static.

### `cluster`

Uses a real Kind cluster with the full KFP backend. Best for testing UI changes that depend on API data.

```bash
# Start cluster first (5-10 min on first run)
make -C ../../backend kind-cluster-agnostic

# Then run smoke test
node smoke-test-runner.js --current-only --mode cluster
```

### `static`

Serves the built frontend with `npx serve -s` (SPA mode). No backend — API-dependent pages show error states. Fast and useful for pure styling/layout changes.

### `proxy`

Serves static build files but proxies API paths (`/apis/`, `/system/`, `/k8s/`, `/artifacts/`, `/visualizations/`, `/apps/`, `/ml_metadata.`) to a real backend. Combines the speed of a static build with real API data.

```bash
node smoke-test-runner.js --current-only --proxy --backend http://localhost:3000
```

## Lessons Learned

These are practical findings from building and using this tool.

### Static serving without a backend is limited

Using `npx serve -s` (static mode) gives you the app shell, but all data-driven pages show error banners because there's no API. This is fine for layout/styling regression checks, but useless for verifying data-driven UI. The `--proxy` flag and `proxy-server.js` were built to address this: they serve static files while forwarding API calls to a real backend.

### Wait selectors must target real data, not just DOM structure

Initially we waited for `[class*="tableRow"]` which fires when the empty/skeleton table renders — before any data loads. Screenshots captured loading or error states. We added `waitForData` selectors (e.g., `a[href*="pipeline"]`) that only match when actual data rows with links are present, plus increased timeouts (5s to 10s) and settle time (1s to 2s).

### `--use-existing` is the most useful workflow

The most productive local workflow is: run your dev server (`npm start` in `frontend/`), then point the tool at it. No build step, no server management, instant screenshots. This is the workflow to reach for when making UI changes.

### GitHub cannot accept image uploads via CLI or API

We tried both `gh` and the GitHub REST API to attach comparison PNGs to PR comments. Neither supports image attachments programmatically. Images must be uploaded via drag-drop in the browser, or hosted externally and linked. The `upload-to-pr.js` script posts a text summary table to the PR; comparison images must be downloaded from the `.ui-smoke-test/` directory or CI artifacts.

### Branch comparison writes to the same output directories

Both `--current-only --use-existing` and the full comparison flow write to `.ui-smoke-test/screenshots/pr/`. Running a full comparison after a `--use-existing` capture will overwrite those screenshots. Be aware of this if you're comparing results across runs.

### Hash-based routing requires specific URL patterns

The KFP frontend uses hash-based routing (`/#/pipelines`, not `/pipelines`). The page definitions in `capture-screenshots.js` use `/#/` prefixed paths. If the app moves to history-based routing, those paths need updating.

## Architecture

```
                          ┌──────────────────────┐
                          │  smoke-test-runner.js │  Main orchestrator
                          │  (flags, modes, flow) │
                          └──────────┬───────────┘
                                     │
              ┌──────────────────────┼──────────────────────┐
              │                      │                      │
  ┌───────────▼──────────┐ ┌────────▼────────┐ ┌───────────▼──────────┐
  │ capture-screenshots.js│ │ proxy-server.js │ │generate-comparison.js│
  │ (Playwright capture)  │ │ (static + API   │ │ (sharp side-by-side) │
  │                       │ │  proxy)         │ │                      │
  └───────────────────────┘ └─────────────────┘ └──────────────────────┘
              │                      │
  ┌───────────▼──────────┐ ┌────────▼────────┐
  │ cluster-manager.js   │ │  seed-data.js   │
  │ (Kind detection,     │ │ (KFP API data   │
  │  port forwarding)    │ │  seeding)       │
  └──────────────────────┘ └─────────────────┘
```

### Dependencies

- **playwright** — headless Chrome for screenshot capture
- **sharp** — image processing for side-by-side comparisons and pixel diff

### Port Assignments

| Port | Used By |
|------|---------|
| 3000 | Dev server / existing backend (typical) |
| 4001 | Main branch static server |
| 4002 | PR branch static server |

## Scripts Reference

| Script | Purpose |
|--------|---------|
| `smoke-test-runner.js` | Main orchestrator — handles mode detection, build, serve, capture, compare |
| `capture-screenshots.js` | Playwright-based screenshot capture with configurable wait conditions |
| `generate-comparison.js` | Side-by-side image generation with labels and pixel diff percentage |
| `proxy-server.js` | Static file server that proxies API calls to a real backend |
| `upload-to-pr.js` | Posts text summary of results to a GitHub PR comment |
| `cluster-manager.js` | Kind cluster detection, startup, port forwarding |
| `seed-data.js` | Creates sample pipelines, experiments, runs via KFP API |
