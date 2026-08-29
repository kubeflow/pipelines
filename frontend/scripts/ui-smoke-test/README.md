# UI smoke-test utility

This utility compares fresh screenshots of the Kubeflow Pipelines UI at a base Git ref and a
local or fetched pull-request head. It uses one live Kind backend for both frontends, creates a
manifest for every capture, and fails closed when a required page is missing, degraded, stale,
corrupt, or different beyond the configured threshold.

## Prerequisites

- Node.js `24.14.0` and npm `11.17.0`, matching `frontend/.nvmrc` and
  `frontend/package.json`
- Git, Docker, Kind, and `kubectl` for comparisons
- `gh`, authenticated to the target repository, only when `--comment` is used

Install the utility's pinned dependencies and browser once:

```bash
cd frontend/scripts/ui-smoke-test
npm ci
npx playwright install chromium
```

The runner restores this nested package exactly with `npm ci` on every invocation and installs the
pinned Chromium build when it is absent. An explicit install is useful for warming those caches.

## Compare local changes

From `frontend/scripts/ui-smoke-test`:

```bash
node smoke-test-runner.js --compare origin/master
```

This comparison includes committed, staged, unstaged, and untracked local files. Change detection
uses the merge base with the selected base ref and handles rename sources as deletes, so moving a
file out of a sensitive tree cannot hide it.

The tool can attribute regressions only in the browser bundle. Both bundles therefore use the base
ref's `frontend/server`, manifests, and backend. A change under any of those surfaces stops the run
by default. To make an explicitly scoped browser-only comparison that ignores those changes:

```bash
node smoke-test-runner.js --compare origin/master --browser-only
```

The head label and report record every ignored surface. This result says nothing about the changed
server, backend, or deployment inputs.

To label local screenshots for an existing pull request:

```bash
node smoke-test-runner.js --compare origin/master --pr-number 12345
```

The label does not change the compared code and does not post anything to GitHub.

## Compare a fetched pull request

```bash
node smoke-test-runner.js \
  --compare origin/master \
  --pr 12345 \
  --repo kubeflow/pipelines \
  --trust-pr-code
```

The runner fetches the GitHub pull ref into a unique temporary ref and creates detached, per-run
worktrees. `--trust-pr-code` is required because the browser build executes scripts from the PR.
Host credentials and the Docker socket are not mounted. Containers have dropped capabilities,
resource limits, a read-only root filesystem, and only the fetched worktree as writable storage.

Dependency installation and code execution are separated. The online phase runs `npm ci
--ignore-scripts` for the root, server, and mock-backend packages into a run-scoped cache. The build
phase performs the exact npm install and build offline with `--network none`. A fetched PR that
changes an npm lockfile, `npm-shrinkwrap.json`, `.npmrc`, or `.corepack.env` is rejected before
installation. Review and check out such a PR locally instead.

Fetched server, backend, or manifest changes are never executed by this tool. They stop the run
unless the caller explicitly requests the same browser-only scope described above:

```bash
node smoke-test-runner.js \
  --compare origin/master \
  --pr 12345 \
  --repo kubeflow/pipelines \
  --trust-pr-code \
  --browser-only
```

## Capture an existing UI

To capture a development server without creating a cluster or building another ref:

```bash
node smoke-test-runner.js \
  --current-only \
  --use-existing \
  --url https://127.0.0.1:3000/my/base/path
```

The complete URL is preserved, including scheme, hostname, port, and path. The runner requires an
HTTP 2xx or 3xx response before capture. Seed-ID-dependent detail pages are omitted by default in
this mode; set `UI_SMOKE_PAGES` to select a different list.

## Thresholds and viewports

```bash
node smoke-test-runner.js \
  --compare origin/master \
  --viewports 1280x800,390x844 \
  --diff-threshold 0 \
  --fail-threshold 0.1
```

- `--viewports` is a comma-separated `WIDTHxHEIGHT` list. The default is `1280x800`.
- `--diff-threshold` controls when changed regions are drawn on a comparison image.
- `--fail-threshold` controls the maximum accepted changed-pixel percentage. The default is `0`, so
  every visual change requires review.

Each viewport is declared in the capture manifest. Comparison rejects missing pairs and dimension
mismatches. Before each screenshot, the browser disables animations and transitions, waits for web
fonts, and executes each configured readiness predicate rather than merely evaluating its function
object.

## PR comments

GitHub is never modified by default. Add `--comment` to a comparison that has `--pr` or
`--pr-number`:

```bash
node smoke-test-runner.js \
  --compare origin/master \
  --pr 12345 \
  --trust-pr-code \
  --comment
```

The reporter validates `summary.json`, uses argument-array subprocess calls, and creates or updates
only the uniquely marked comment authored by the authenticated GitHub user. The comment records the
full base and head SHAs, diff configuration, and failed, skipped, or threshold-exceeding results.
Immediately before posting, the runner verifies that the pull request still points to the captured
head SHA. Local `--pr-number --comment` runs additionally require a clean working tree whose HEAD
matches the PR. Images remain local; the utility does not claim that CI uploaded an artifact.

## Output and cleanup

Every invocation gets its own directory, so concurrent and previous runs cannot supply stale
screenshots:

```text
.ui-smoke-test/
  latest-run.txt
  runs/
    <timestamp>-<pid>-<random>/
      seed-manifest.json
      screenshots/
        base/manifest.json
        head/manifest.json
        comparison/<page>-<viewport>.png
        comparison/summary.json
      worktrees/
```

`latest-run.txt` contains the absolute path of the newest run. Worktrees, temporary Git refs,
proxies, port-forwards, and the local frontend server are cleaned up on success,
failure, `SIGINT`, or `SIGTERM`. Completed screenshots and reports are retained. Other runs are
never automatically deleted.

Comparisons use the dedicated `ui-smoke-test` Kind cluster and refuse to reuse it: carrying over a
database or locally loaded image could make a comparison falsely pass. The cluster remains after a
run for inspection. Teardown is therefore required before the next comparison:

To delete the managed Kind cluster:

```bash
node smoke-test-runner.js --teardown
```

## What the live comparison does

1. Validates arguments, dependencies, tools, Git refs, and all required local ports.
2. Creates unique run state and detached base/head worktrees.
3. Detects the merge-base change set, including the local working tree when applicable.
4. Builds both browser bundles; fetched builds use the two-phase constrained container flow.
5. Creates the clean, dedicated managed Kind cluster from the trusted base manifests without
   leaving the user's current Kubernetes context changed; an existing managed cluster is rejected.
6. Pulls and preloads the digest-pinned seed runtime, applies both manifest layers, and waits for
   every platform deployment. Any setup failure rolls back the newly created cluster.
7. Forwards the API, metadata, and SeaweedFS services and starts the frontend server with the same
   artifact-storage environment used by `frontend/scripts/start-proxy-and-server.sh`.
8. Creates or validates deterministic pipeline, experiment, run, and recurring-run resources,
   including populated scalar-metric and ROC artifacts, and writes their IDs to the per-run seed
   manifest. Every seeded run must reach `SUCCEEDED`.
9. Serves both static builds from loopback-only proxies against the same backend.
10. Captures every configured page for both revisions concurrently, compares only exact successful
    manifest pairs with pinned analysis settings, writes the report, and applies the exit policy.
11. If explicitly requested, posts the report even when visual differences make the run fail.

The local proxy pins API requests to the configured backend origin, rejects unsafe absolute-form
targets and path/symlink escapes, and returns real missing-asset errors instead of the SPA shell.
It permits read-only HTTP methods plus MLMD `Get*` RPCs, rejecting backend mutations from captured
frontend code.
The browser blocks service workers and all HTTP(S)/WebSocket traffic outside the exact capture
origin, so fetched frontend code cannot probe other localhost, LAN, or internet services.

## Direct utilities

The runner is the supported end-to-end entry point. The lower-level tools are useful for focused
debugging:

```bash
node capture-screenshots.js \
  --base-url http://127.0.0.1:3000 \
  --output ./screenshots/base \
  --label base \
  --seed-manifest ./seed-manifest.json

node generate-comparison.js \
  --main ./screenshots/base \
  --pr ./screenshots/head \
  --output ./screenshots/comparison \
  --fail-threshold 0

node upload-to-pr.js \
  --pr 12345 \
  --repo kubeflow/pipelines \
  --screenshots ./screenshots/comparison
```

`generate-comparison.js` requires matching version-2 capture manifests. It writes a summary even
when inputs are invalid, a screenshot is corrupt, image dimensions differ, or diff analysis fails,
and then exits nonzero.

Capture and comparison output directories carry ownership markers. Direct tools refuse to clean a
non-empty directory without a valid marker and remove only the files named by that marker.

The older `visual-compare.mjs`, `visual-compare-run.sh`, and `visual:*` npm commands were removed
because they did not enforce the live, versioned, fail-closed workflow. Use
`smoke-test-runner.js` for regression decisions.

## Tests

```bash
npm test
```

The nested tests use Node's built-in test runner and cover capture manifests, comparison failure
modes, change detection, cluster command construction, seeding, proxy boundaries, runner argument
validation, and GitHub reporting. They also run in the frontend CI workflow.
