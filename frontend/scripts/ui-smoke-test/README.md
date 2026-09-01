# UI smoke-test utility

This utility compares fresh screenshots of the Kubeflow Pipelines UI at a base Git ref and a
local or fetched pull-request head. Its full-stack mode runs each UI with its matching
frontend-server, backend images, manifests, database, object store, metadata system, cache, and
Kubernetes state. It creates a manifest for every capture and fails closed when a required page is
missing, degraded, stale, corrupt, or different beyond the configured threshold.

## Prerequisites

- Node.js `24.14.0` and npm `11.17.0`, matching `frontend/.nvmrc` and
  `frontend/package.json`
- Git, Docker, Kind, and `kubectl` for comparisons
- `gh`, authenticated to the target repository, only when `--comment` is used

Before creating a cluster, the runner renders both revision overlays, verifies and exports every
dependency image for the Kind node's explicit platform, and builds every reviewed first-party head
image for that platform. An amd64-only release image on an arm64 Kind node therefore fails before
cluster creation with instructions to use a matching image or an amd64 node with emulation.

Install the utility's pinned dependencies and browser once:

```bash
cd frontend/scripts/ui-smoke-test
npm ci
npx playwright install chromium
```

Before a capture run, the runner restores this nested package exactly with `npm ci` and installs
the pinned Chromium build when it is absent. Help, teardown, and an upgrade capability check that
fails before capture do not install dependencies. An explicit install is useful for warming those
caches.

## Compare local changes

From `frontend/scripts/ui-smoke-test`:

```bash
node smoke-test-runner.js --compare origin/master
```

This comparison includes committed, staged, unstaged, and untracked local files. Change detection
uses the merge base with the selected base ref and handles rename sources as deletes, so moving a
file out of a sensitive tree cannot hide it.

When only browser code changed, the default comparison uses the base runtime for both bundles. A
change to the frontend-server, backend, or manifests stops that compatibility workflow. For a
revision-matched comparison, explicitly select a reviewed local checkout:

```bash
node smoke-test-runner.js \
  --compare 2.17.1 \
  --full-stack \
  --head-checkout /path/to/reviewed/head \
  --trust-local-head \
  --pr-number 13986
```

The selected path must be the root of a worktree belonging to the same repository. Dirty local
changes are supported, including changed lockfiles, frontend-server, backend, and manifests. The
runner snapshots the selected commit plus its staged, unstaged, and non-ignored untracked files
into a detached run-scoped worktree, records a cryptographic source fingerprint, and aborts if the
source changes while the snapshot is being made. Symlinks cannot escape the checkout. The trust
flag is required because this mode installs dependencies, builds images, starts servers, and
deploys manifests from that immutable snapshot.

When the base is a release such as `2.17.1`, the runner resolves the fully qualified release tag
from the canonical `kubeflow/pipelines` repository, verifies that the local tag peels to the same
commit, and pins all base work to that verified commit SHA. A moved or counterfeit local release
tag is rejected.

To make an explicitly scoped browser-only comparison that ignores changed runtime surfaces:

```bash
node smoke-test-runner.js --compare origin/master --browser-only
```

The head label and report record every ignored surface. This result is a browser compatibility
signal only; it says nothing about the changed server, backend, deployment, or migration behavior.

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

Fetched PRs cannot use `--full-stack` or `--upgrade`. Review and check out the target locally, then
select it with `--head-checkout --trust-local-head`.

## Upgrade a populated installation

Upgrade mode exercises a different invariant from two clean stacks: base data and persistent
volume identities must survive while the same environment is migrated and upgraded.

```bash
node smoke-test-runner.js \
  --compare 2.17.1 \
  --upgrade \
  --head-checkout /path/to/reviewed/head \
  --trust-local-head \
  --pr-number 13986
```

The fail-closed lifecycle is: deploy base, seed base, capture base, freeze writers, inventory PVCs
and semantic fixtures, run and validate the migration's durable marker, deploy head into the same
environment, validate the startup gate, prove PVC and fixture continuity, prune only explicitly
allowed non-persistent resources, capture head, and generate the attested comparison and HTML
report. A reviewed target advertises matching migration and startup-gate versions in
`.ui-smoke-upgrade.json` and names an in-checkout adapter that supplies those lifecycle operations.
Adapter paths cannot escape the selected checkout. The adapter factory must be side-effect-free,
and the adapter must provide `cleanupEnvironment`; the runner registers cleanup before invoking
any deployment operation. A cleanup failure invalidates an otherwise successful result.

PR #13986 does not yet contain the MLMD-to-native migration or startup gate tracked by #14029.
Against that head, upgrade mode writes `upgrade-result.json` with
`captureValidity: "migration_unavailable"` and invokes no cluster or head-mutation callback. This
is an intentional release-blocker result, not a successful visual comparison. Capability
discovery happens before Docker, Kind, or nested package setup, so an unavailable migration cannot
mutate the host or cluster as a side effect of preflight.

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

Full-stack seeding creates the same logical pipeline, run, metrics, ROC data, artifacts, retry,
two-item `ParallelFor`, and nested DAG in each revision through that revision's supported APIs.
Legacy runs are hydrated from their MLMD-backed run response. Native runs page through
`/apis/v2beta1/runs/{run-id}/tasks` and preserve the returned Task and Artifact relationships. The
resulting `semantic-fixtures.json` maps stable fixture keys to each revision's generated IDs, so
routes and selectors do not need identical IDs.

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
      semantic-fixtures.json
      source-provenance.json
      seed/
        base.json
        head.json
      kubeconfigs/
        base.yaml
        head.yaml
      upgrade-result.json        # upgrade mode, including fail-closed blockers
      screenshots/
        base/manifest.json        # includes seed, semantic, and source attestations
        head/manifest.json
        comparison/<page>-<viewport>.png
        comparison/summary.json
        comparison/report.html   # self-contained base/head/diff browser report
      worktrees/
```

`latest-run.txt` contains the absolute path of the newest run. Worktrees, temporary Git refs,
proxies, port-forwards, local servers, and owned clusters are cleaned up on ordinary success or
failure. The runner also requests cleanup on `SIGINT` and `SIGTERM`, but an uncatchable termination
can leave run-scoped resources that must be removed by exact name. Completed screenshots and
reports are retained. Other runs are never automatically deleted.

Comparison thresholds are evaluated only for complete, cryptographically attested capture pairs.
Missing, degraded, corrupt, or stale captures remain capture-validity failures rather than being
reported as pixel differences.

Full-stack comparisons create unique `ui-smoke-base-*` and `ui-smoke-head-*` clusters. Each has its
own kubeconfig, context, database, object store, cache, Kubernetes resources, image scope, ports,
and child processes. The runner never changes or depends on the global `current-context`.
Run-scoped clusters are destroyed during cleanup so their state cannot leak into another run.

The compatibility workflow retains the historical fixed `ui-smoke-test` cluster behavior. To
delete that legacy managed cluster:

```bash
node smoke-test-runner.js --teardown
```

## What a full-stack comparison does

1. Validates the reviewed checkout, dependencies, tools, an exact release-tag base such as
   `2.17.1`, and both non-overlapping port sets. The rendered first-party base images must carry
   that exact release tag.
2. Creates unique run state and a detached base worktree, then renders only each revision's actual
   platform-agnostic overlay. Workload and optional-service discovery never scans unrelated YAML.
3. Verifies and exports every rendered dependency image and builds the selected head's
   revision-compatible frontend, frontend-server, backend, and runtime images for the explicit Kind
   node platform. Any architecture or build failure occurs before cluster creation.
4. Creates two run-scoped Kind clusters with separate kubeconfigs, then loads only the images
   preflighted for that revision. Exact local image overrides and runtime-image variables are
   applied to the rendered head before any workload starts.
5. Applies the manifests and waits for the deployments actually rendered by that revision.
6. Forwards each cluster's deployed `ml-pipeline-ui` service on a distinct loopback port. Seeding,
   readiness checks, and screenshots all use that deployed UI and its matching in-cluster
   frontend-server/backend; full-stack mode does not substitute a host-side server or static proxy.
7. Executes equivalent deterministic fixtures through each revision's supported API and runtime.
   They cover scalar and ROC metrics, artifact producer/consumer relationships, a retry, a
   two-item `ParallelFor`, and nested DAG parent/child relationships. List-filler runs use a small
   deterministic pipeline so the richer topology remains the single semantic source of truth.
   The base records legacy MLMD task/artifact data; a native head records Task/Artifact API data.
8. Discovers generated IDs from each revision's run details and writes separate capture-compatible
   seed manifests plus one `ui-smoke-semantic/v2` manifest keyed by logical fixtures. The manifest
   maps per-revision run, task instance, artifact, retry-attempt, iteration, and relationship IDs;
   unknown or incomplete semantic bindings stop the comparison.
9. Binds each capture to the revision-specific deployed UI URL, semantic manifest, and immutable
   source provenance.
10. Captures both revisions, compares only exact successful manifest pairs with pinned analysis
    settings, writes the report, and applies the exit policy.
11. If explicitly requested, posts the report even when visual differences make the run fail.

The compatibility workflow's local proxy pins API requests to the configured backend origin,
rejects unsafe absolute-form targets and path/symlink escapes, and returns real missing-asset errors
instead of the SPA shell. It permits read-only HTTP methods plus MLMD `Get*` RPCs, rejecting backend
mutations from captured frontend code. The browser blocks service workers and all
HTTP(S)/WebSocket traffic outside the exact capture origin, so captured frontend code cannot probe
other localhost, LAN, or internet services.

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
