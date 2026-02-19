# UI Smoke Test Tool

Visual regression testing for Kubeflow Pipelines frontend. Captures screenshots of key UI pages and generates side-by-side comparisons between branches using a live Kind backend.

## Prerequisites

Install these before first use:

| Tool | Required For | Install |
|------|-------------|---------|
| Node.js >= 18 | All workflows | `brew install node` or [nodejs.org](https://nodejs.org/) |
| git >= 2.5 | All workflows | `brew install git` or [git-scm.com](https://git-scm.com/) |
| Docker | `--compare` | [Docker Desktop](https://www.docker.com/products/docker-desktop/) |
| kind | `--compare` | `brew install kind` or [kind.sigs.k8s.io](https://kind.sigs.k8s.io/docs/user/quick-start/#installation) |
| kubectl | `--compare` | `brew install kubectl` or [kubernetes.io](https://kubernetes.io/docs/tasks/tools/) |
| gh CLI | `--pr` | `brew install gh` or [cli.github.com](https://cli.github.com/) |

The tool checks these at startup and fails fast with actionable install commands if anything is missing.

## First-Time Setup

```bash
# 1. Navigate to the tool directory
cd frontend/scripts/ui-smoke-test

# 2. Install tool dependencies (playwright, sharp, looks-same)
npm install

# 3. Install Chromium for screenshot capture
npx playwright install chromium

# 4. (For --compare) Ensure Docker is running
open -a Docker  # macOS

# 5. Run your first comparison
node smoke-test-runner.js --compare master
```

The first `--compare` run takes 5-10 minutes because it creates a Kind cluster and deploys Kubeflow Pipelines. Subsequent runs reuse the existing cluster and are much faster.

## Quick Reference

```bash
# Most common: compare your branch against master with live backend
node smoke-test-runner.js --compare master

# Frontend-only PR? Skip the slow backend rebuild
node smoke-test-runner.js --compare master --skip-backend

# Fail if any non-zero visual diff is detected (default behavior)
node smoke-test-runner.js --compare master --fail-threshold 0

# Keep local HEAD but label screenshots as a specific PR
node smoke-test-runner.js --compare master --pr-number 12793

# Screenshot your running dev server (fastest)
node smoke-test-runner.js --current-only --use-existing --url http://localhost:3000
```

## `--compare` Workflow

The primary workflow. Compares your working tree (or a specific PR) against a base ref with a live Kind backend.
By default, any non-zero visual diff fails (`--fail-threshold 0`) so every change is reviewed.

```bash
# Compare against master (most common)
node smoke-test-runner.js --compare master

# Compare against the latest release tag
node smoke-test-runner.js --compare release

# Compare against a specific tag
node smoke-test-runner.js --compare 2.15.0

# Skip backend rebuild for frontend-only changes
node smoke-test-runner.js --compare master --skip-backend

# Label local HEAD comparisons as a PR in screenshot headers
node smoke-test-runner.js --compare master --pr-number 12793

# Delete the Kind cluster when done
node smoke-test-runner.js --teardown
```

### What happens (step by step)

The runner shows numbered progress like `[3/12] Building 2 backend component(s)...`:

1. **Check port availability** — fails fast if ports are in use
2. **Detect changes** — `git diff --name-only <base>..<head>` mapped to backend components
3. **Ensure Kind cluster** — starts one if not already running
4. **Build changed backend components** — only the ones that changed, loads into Kind, restarts deployments
5. **Re-apply manifests** — if `manifests/` files changed
6. **Set up port forwarding + frontend server** — proxies API calls to K8s services
7. **Seed test data** — creates sample pipelines, experiments, runs
8. **Fetch PR code** — if `--pr` is specified, fetches via git
9. **Build base frontend** — via git worktree (fast, offline)
10. **Build PR frontend** — current tree or fetched PR
11. **Start proxy servers** — ports 4001 (base) and 4002 (PR)
12. **Capture screenshots + generate comparison** — Playwright headless Chrome

Steps that aren't needed (e.g., no backend changes) are shown as `[4/12] Backend rebuild (skipped)`.

## Testing Someone Else's PR

Use `--pr` with `--compare` to test a PR you don't have checked out locally:

```bash
# Fetch PR #12756 and compare it against master
node smoke-test-runner.js --compare master --pr 12756

# Same but skip backend rebuild
node smoke-test-runner.js --compare master --pr 12756 --skip-backend
```

This fetches the PR ref via `git fetch origin pull/<N>/head:pr-<N>`, creates a git worktree for the PR code, builds its frontend, and uses it as the "PR" side of the comparison. The change detection diff is also done against the PR ref (not your local HEAD).

## Command Reference

### Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--compare <ref>` | **Primary workflow.** Compare HEAD (or `--pr`) against a base ref with live backend | (none) |
| `--fail-threshold <percent>` | Exit non-zero when any page diff is above this percentage | `0` |
| `--diff-threshold <percent>` | Only mark pages as `[diff]` in summary when above this percentage | `0` |
| `--skip-backend` | Force-skip backend rebuild and manifest re-apply even if changes are detected (auto-skipped when no backend changes) | off |
| `--pr <number>` | In `--compare` mode: fetch and test this PR instead of local HEAD. In legacy mode: PR number for branch comparison | (none) |
| `--pr-number <number>` | Label screenshots as `PR #<number>` without fetching that PR (useful when comparing local HEAD). If omitted, runner tries best-effort GH auto-detection from current `HEAD` SHA. | (none) |
| `--teardown` | Delete the Kind cluster and exit | off |
| `--current-only` | Screenshot current build only (no branch comparison) | off |
| `--use-existing` | Use an already-running server instead of building/serving | off |
| `--url <url>` | URL of existing server (requires `--use-existing`) | `http://localhost:3000` |
| `--proxy` | Proxy API calls to a real backend when serving static builds | off |
| `--backend <url>` | Backend URL for proxy mode | `http://localhost:3000` |
| `--base <branch>` | Base branch for legacy comparison | `master` |
| `--repo <owner/repo>` | GitHub repository | `kubeflow/pipelines` |
| `--mode <mode>` | Backend mode: `auto`, `cluster`, `mock`, `static` | `auto` |
| `--start-cluster` | Start a Kind cluster if none is running (legacy mode) | off |
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
| `UI_SMOKE_PR_NUMBER` | PR number used for screenshot labels when `--pr` is not used | (none) |
| `UI_SMOKE_FAIL_THRESHOLD` | Default fail threshold percentage for comparison | `0` |
| `UI_SMOKE_DIFF_THRESHOLD` | Default summary marker threshold percentage | `0` |
| `API_BASE` | API base URL for data seeding | `http://localhost:3001` |

### Individual Scripts

Each script can be run standalone:

```bash
# Detect which backend components changed vs master
node detect-changes.js --base master

# Capture screenshots from a running server
node capture-screenshots.js --port 3000 --output ./my-screenshots --label "my-branch"

# Generate side-by-side comparisons from two sets of screenshots
node generate-comparison.js --main ./screenshots/main --pr ./screenshots/pr --output ./comparison --fail-threshold 0

# Post comparison results to a GitHub PR
node upload-to-pr.js --pr 12756 --repo kubeflow/pipelines --screenshots ./comparison

# Start the proxy server standalone
node proxy-server.js --build ../../build --port 4001 --backend http://localhost:3000

# Seed test data into a running KFP API
node seed-data.js              # skip if data exists
node seed-data.js --force      # overwrite existing data

# Filter specific pages
UI_SMOKE_PAGES=pipelines,runs node capture-screenshots.js --port 3000 --output ./screenshots
```

## Troubleshooting

### Port conflicts

```
Port 3001 is in use by node (PID 12345)
```

The tool checks all required ports before starting and fails fast if any are in use. Fix by killing the process:

```bash
kill 12345
# or find what's using a port:
lsof -i :3001
```

Ports used: 3001 (frontend server), 3002 (ml-pipeline proxy), 4001 (base proxy), 4002 (PR proxy), 9000 (minio), 9090 (metadata-envoy).

### Stale worktrees from a previous failed run

The tool auto-cleans stale worktrees at startup. If you see git worktree errors:

```bash
git worktree remove .ui-smoke-test/base --force
git worktree remove .ui-smoke-test/pr-branch --force
git worktree prune
```

### Docker is not running

```
Docker is not running. Start Docker Desktop or run: sudo systemctl start docker
```

The `--compare` workflow requires Docker for Kind. Start Docker Desktop or your Docker daemon. If you only need screenshots without a backend, use `--current-only`.

### Slow backend rebuild

Backend rebuilds (`make image_apiserver`, etc.) can take several minutes each. If your PR only changes frontend code:

```bash
node smoke-test-runner.js --compare master --skip-backend
```

This forces steps 4-5 (build/deploy, manifests) to be skipped. Note: these steps auto-skip when change detection finds no backend changes, so you typically only need `--skip-backend` when change detection incorrectly flags backend files (e.g., during a rebase).

### Kind cluster in bad state

If the cluster is misbehaving, tear it down and start fresh:

```bash
node smoke-test-runner.js --teardown
node smoke-test-runner.js --compare master
```

### Ctrl+C doesn't clean up

The tool registers cleanup actions (worktree removal, ml-pipeline-ui restore) that run on SIGINT. If cleanup didn't complete, manually fix:

```bash
# Remove worktrees
git worktree remove .ui-smoke-test/base --force 2>/dev/null
git worktree remove .ui-smoke-test/pr-branch --force 2>/dev/null
git worktree prune

# Restore ml-pipeline-ui
kubectl -n kubeflow scale deployment/ml-pipeline-ui --replicas=1
```

## Pages Captured

| Page | Route | Wait Condition | Description |
|------|-------|----------------|-------------|
| pipelines | `/#/pipelines` | Table rows + pipeline links | Pipeline list |
| pipeline-details-seeded | `/#/pipelines/details/{seed.pipelineId}` | Root + details content | Seeded pipeline details (default view) |
| pipeline-details-seeded-sidepanel | `/#/pipelines/details/{seed.pipelineId}` | Side panel close button visible | Seeded pipeline details with side panel open |
| experiments | `/#/experiments` | Table rows + experiment links | Experiment list |
| runs | `/#/runs` | Table rows + run links | Run history |
| run-details-seeded | `/#/runs/details/{seed.runId}` | Root + graph/details content | Seeded run details (default view) |
| run-details-seeded-sidepanel | `/#/runs/details/{seed.runId}` | Side panel close button visible | Seeded run details with side panel open |
| recurring-runs | `/#/recurringruns` | Table rows | Scheduled runs |
| artifacts | `/#/artifacts` | Table rows | ML artifacts |
| executions | `/#/executions` | Table rows + execution links | Execution history |
| pipeline-create | `/#/pipeline/create` | Input field | Create pipeline form |
| experiment-create | `/#/experiments/new` | Input field | Create experiment form |

## Output

Results are saved to `.ui-smoke-test/` at the repo root (gitignored):

```
.ui-smoke-test/
  base/                      # git worktree checkout (--compare mode)
  pr-branch/                 # git worktree for --pr mode
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

## Architecture

### Port Layout (`--compare` mode)

```
Kind Cluster (K8s)
  ├── ml-pipeline service :8888
  ├── metadata-envoy-service :9090
  └── minio-service :9000

Port Forwards (kubectl)
  ├── localhost:3002 → ml-pipeline:8888
  ├── localhost:9090 → metadata-envoy:9090
  └── localhost:9000 → minio:9000

Node.js Frontend Server (localhost:3001)
  └── Proxies API calls to :3002, :9090, :9000

proxy-server.js × 2
  ├── localhost:4001 → static base build + API → :3001
  └── localhost:4002 → static PR build + API → :3001

Playwright captures screenshots from :4001 and :4002
```

### Component Change Detection

The `detect-changes.js` script maps changed files to backend components using 2-dot diff (`base..head`), which shows only what the PR changed:

| File Path | Component | Make Target | K8s Deployment |
|-----------|-----------|-------------|----------------|
| `backend/src/apiserver/**` | apiserver | `image_apiserver` | `ml-pipeline` |
| `backend/src/agent/persistence/**` | persistence-agent | `image_persistence_agent` | `ml-pipeline-persistenceagent` |
| `backend/src/cache/**` | cache-server | `image_cache` | (varies) |
| `backend/src/crd/controller/scheduledworkflow/**` | scheduledworkflow | `image_swf` | `ml-pipeline-scheduledworkflow` |
| `backend/src/crd/controller/viewer/**` | viewercontroller | `image_viewer` | `ml-pipeline-viewer-crd` |
| `backend/src/apiserver/visualization/**` | visualization | `image_visualization` | `ml-pipeline-visualizationserver` |
| `backend/src/v2/cmd/driver/**` | driver | `image_driver` | (runtime image) |
| `backend/src/v2/cmd/launcher-v2/**` | launcher | `image_launcher` | (runtime image) |
| `backend/src/common/**` | ALL Go components | | |
| `go.mod`, `go.sum` | ALL Go components | | |

### Scripts Reference

| Script | Purpose |
|--------|---------|
| `smoke-test-runner.js` | Main orchestrator — `--compare` (primary) and legacy modes |
| `detect-changes.js` | Maps `git diff` to backend components for selective rebuild |
| `cluster-manager.js` | Kind cluster lifecycle, component build/deploy, port forwarding, frontend server |
| `capture-screenshots.js` | Playwright-based screenshot capture with configurable wait conditions |
| `generate-comparison.js` | Side-by-side image generation with labels and pixel diff percentage |
| `proxy-server.js` | Static file server that proxies API calls to a real backend |
| `upload-to-pr.js` | Posts text summary of results to a GitHub PR comment |
| `seed-data.js` | Creates sample pipelines, experiments, runs via KFP API |

### Dependencies

- **playwright** — headless Chrome for screenshot capture
- **sharp** — image processing for side-by-side layout images
- **looks-same** — mature visual diff engine (diff percentages + clusters)

## Lessons Learned

These are practical findings from building and using this tool.

### 3-dot diff includes unwanted files (fixed)

`git diff --name-only base...HEAD` (3-dot) shows all changes since the merge base, including files changed on the base branch since divergence. For a frontend-only PR, this could flag backend files that changed on master, triggering unnecessary rebuilds. Fixed by switching to 2-dot diff (`base..HEAD`) which shows only what HEAD added.

### Backend rebuild auto-skips when no backend changes are detected

The tool automatically skips backend rebuild (step 4) and manifest re-apply (step 5) when change detection finds no backend file changes. The `--skip-backend` flag exists as a manual override for cases where detection is wrong (e.g., during a rebase that touches `go.mod`).

### Static serving without a backend is limited

Using `npx serve -s` (static mode) gives you the app shell, but all data-driven pages show error banners because there's no API. This is fine for layout/styling regression checks, but useless for verifying data-driven UI. The `--proxy` flag and `proxy-server.js` were built to address this.

### Wait selectors must target real data, not just DOM structure

Initially we waited for `[class*="tableRow"]` which fires when the empty/skeleton table renders. Screenshots captured loading or error states. We added `waitForData` selectors (e.g., `a[href*="pipeline"]`) that only match when actual data rows with links are present.

### `--use-existing` is the most useful quick workflow

Run your dev server (`npm start` in `frontend/`), then point the tool at it. No build step, no server management, instant screenshots.

### `--compare` is the most useful full workflow

For PR review, `--compare master` gives you the full picture: change detection, selective backend rebuild, both frontends built and screenshotted against a live backend. It uses `git worktree` (fast, offline) instead of `git clone`.

### Cleanup must be guaranteed, not conditional

Early versions only cleaned up worktrees and restored ml-pipeline-ui on the success path. A Ctrl+C or error mid-workflow would leave stale worktrees and ml-pipeline-ui scaled to 0. Fixed with a LIFO resource tracker that runs on all exit paths (success, error, SIGINT).

### GitHub cannot accept image uploads via CLI or API

Neither `gh` nor the GitHub REST API supports image attachments programmatically. Images must be uploaded via drag-drop in the browser, or hosted externally. The `upload-to-pr.js` script posts a text summary.

### Hash-based routing requires specific URL patterns

The KFP frontend uses hash-based routing (`/#/pipelines`, not `/pipelines`). The page definitions in `capture-screenshots.js` use `/#/` prefixed paths. If the app moves to history-based routing, those paths need updating.
