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
dependency image for an explicit platform, and builds every reviewed first-party image required by
a locally built revision for the Kind node's native platform. On arm64, the two known amd64-only
workloads in the 2.17.1 manifest are pulled and loaded explicitly as amd64 without changing the
Kind node architecture. A Kubernetes canary verifies workload emulation before either revision is
deployed. Any other missing-platform image fails closed instead of silently falling back to a
foreign architecture.

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

To compare the MLMD-removal checkout against a non-release base such as current `main`, explicitly
trust both local revision inputs:

```bash
node smoke-test-runner.js \
  --compare origin/master \
  --full-stack \
  --head-checkout /path/to/reviewed/head \
  --trust-local-head \
  --trust-base-code \
  --pr-number 13986
```

The runner resolves the base ref to an immutable SHA before snapshotting the head, creates a
separate detached base worktree, and builds all first-party components used by each revision. The
extra base trust flag is required because a branch or arbitrary commit is executable input rather
than a verified published release. Each resulting UI is served by its own matching
frontend-server, backend, manifests, and isolated state.

To make an explicitly scoped browser-only comparison that ignores changed runtime surfaces:

```bash
node smoke-test-runner.js --compare origin/master --browser-only
```

The head label and report record every ignored surface. This result is a browser compatibility
signal only; it says nothing about the changed server, backend, deployment, or migration behavior.
In particular, it cannot validate pages that require #13986's native Task or Artifact endpoints;
use the revision-matched full-stack mode for those scenarios.

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
object. Full-stack semantic captures also pin Chromium through the nested lockfile, use a device
scale factor of 2, UTC, `en-US`, a light color scheme, reduced motion, and embedded Roboto 5.3.0
WOFF2 assets at weights 400, 500, and 700. Each font digest is attested in the capture manifest.
They freeze the browser clock, disable long polling timers, normalize rendered timestamps and
durations, and apply the same deterministic styles inside artifact frames. Semantic full-stack
captures also hide exactly the base revision's `#executionsBtn` and assert that the head revision
has no such element. This removes the reviewed sidebar/footer displacement without masking any
other navigation pixels; selector counts, the applied rule, and its expected-change annotation are
attested in every capture record. Scenario-declared run,
task, execution, Artifact, Artifact URI, pod-name, and pod-UID values are first validated at their
real generated values, then replaced inside narrowly scoped text nodes with stable semantic tokens
immediately before the screenshot. When 2.17.1 exposes repeated uncached `ParallelFor` task rows
without an execution or iteration identity, their task UUIDs use one explicit visual equivalence
token in both revisions. Every source task keeps its exact semantic path and raw-ID digest in the
capture evidence. Independently identified MLMD executions and native iteration scopes remain
distinct by iteration, and the separately observed parent DAGs, pods, and logs remain exact.
ROC series whose colors are derived from generated IDs are rebound to fixed palette slots by
semantic run identity after matching manifest-bound display names. Every declared ROC fixture must
be present. The capture manifest records the source kind and semantic path separately from the
cross-revision visual-token identity, plus replacement counts and SHA-256 digests of original
values and source colors without recording raw generated identities. It never applies a page-wide
UUID or numeric regex, and a missing, ambiguous, or unexpectedly repeated required replacement is
a capture failure.

Full-stack captures must explicitly request and attest `semantic-full-stack`; browser-only captures
must explicitly request `disabled-browser-compatibility` and cannot provide semantic or source
provenance. Comparison re-reads the attested semantic manifest, recomputes its fixture validation,
and verifies every normalized source-ID digest against it. A missing pinned font is an
infrastructure failure instead of a host-dependent screenshot.

For reviewed per-scenario exceptions, pass `--scenario-policy /path/to/policy.json`. The policy is
operator input and is not allowed for non-comparison workflows. The runner combines it with the
trusted semantic scenario catalog only after both captures finish, writes a run-scoped
`scenario-config.json`, and binds that config to both capture IDs and exact manifest SHA-256
digests. A stale policy binding is rejected before image analysis. Policy rules use schema
`ui-smoke-comparison-policy/v1` and may override `diffThreshold`, `failThreshold` (including
`null` to disable it), `looksSameTolerance`, `expectedChange`, and rectangular `masks` for a
semantic scenario. An optional `{ "width": 1280, "height": 800 }` viewport makes a rule specific
to that capture size. Mask coordinates are non-negative physical PNG pixels, must stay within the
image, and cannot cover the entire image. Viewport qualifiers use CSS pixels; masks use physical
pixels, so the default device scale factor of 2 makes a `1280x800` screenshot `2560x1600`. A
viewport-specific rule inherits the scenario-wide mask set when `masks` is omitted, clears it with
`"masks": []`, and replaces it when a non-empty mask array is supplied. `expectedChange` is an
annotation and does not waive a failure threshold; set `failThreshold` to `null` when the reviewed
change should remain informational.

```json
{
  "schemaVersion": "ui-smoke-comparison-policy/v1",
  "scenarios": [
    {
      "semanticScenario": "run-details-task-logs",
      "viewport": { "width": 1280, "height": 800 },
      "diffThreshold": 0.02,
      "failThreshold": 0.1,
      "looksSameTolerance": 2.3,
      "expectedChange": "Reviewed log-toolbar layout change",
      "masks": [{ "x": 2300, "y": 40, "width": 180, "height": 60, "reason": "provider badge" }]
    }
  ]
}
```

The clean-stack catalog keys are `executions-to-runs`, `artifact-list-evolution`,
`run-details-rich-graph`, `run-details-task-panel`, `run-details-task-logs`,
`run-details-scalar-metrics`, `run-details-html`, `run-details-markdown`, `run-details-roc`,
`compare-runs`, `compare-roc-selection`, `compare-html`, `compare-markdown`, `artifact-details`,
`artifact-related-tasks`, `topology-retried-task`, `topology-parallel-for`,
and `topology-nested-dag`.

Full-stack seeding creates the same logical pipeline, run, metrics, ROC data, artifacts, retry,
two-item `ParallelFor`, and nested DAG in each revision through that revision's supported APIs.
The artifact set includes deterministic scalar metrics, classification metrics, HTML, and Markdown
contents plus producer and consumer relationships. The retry fixture declares that it requires
Argo `retryPolicy: OnFailure`; the runner applies that requirement only to each rendered disposable
stack and verifies the target ConfigMap shape before deployment. Repository manifests are not
modified.

The 2.17.1 Argo reporter does not populate a complete task/Artifact projection. Legacy hydration
therefore resolves the exact `system.PipelineRun` MLMD context by the KFP run ID, loads every
execution, Artifact, and Event in that context, and reconstructs only the fixture's declared
producer/consumer ports from actual `INPUT` and `OUTPUT` Events. Any context ID exposed by GetRun is
treated as a cross-check, not as the source of truth. MLMD execution names, pod identities,
iteration indexes, parent DAG IDs, and executor-log Events provide complete task, retry, lineage,
and containment coverage without guessing between repeated task names. The legacy task API stores
dependency children as unjoinable Argo node IDs, so `depends-on` edges are explicitly marked as
`pipeline-version-spec` evidence parsed from the exact PipelineVersion referenced by the observed
run; containment and Artifact consumer edges remain independently backed by MLMD. Missing,
ambiguous, undeclared, version-mismatched,
cross-context, or GetRun/MLMD-conflicting evidence fails seeding. Executor-log Events are accepted
only when the referenced MLMD Artifact has type `system.Artifact`, custom display name
`executor-logs`, and a deterministic attempt-suffixed URI. Native runs page
through `/apis/v2beta1/runs/{run-id}/tasks` and preserve the returned Task and Artifact
relationships. Both revisions retain launcher-managed `executor-logs-N` Artifacts, order retry logs
by their URI suffix rather than API response order, and map their IDs and URIs to the same semantic
attempt identities without admitting other undeclared Artifacts.

The resulting `semantic-fixtures.json` maps stable fixture keys to each revision's generated IDs, so
routes and selectors do not need identical IDs.

Capture scenarios are semantic journeys rather than a shared list of URLs. The base and head may
use different routes, tabs, selectors, and actions for the same scenario. The clean-stack catalog
covers Executions to Runs, grouped to native Artifact lists, Run Details graph/task/logs and all
seeded visualizations, Compare selections, Artifact Details and relationships, retries,
`ParallelFor`, and nested DAGs. The clean-stack catalog deliberately omits the former historical
Artifact scenario because no historical identity exists there; it belongs in upgrade mode once an
adapter can discover and attest the migrated native Artifact identity.

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
        scenario-config.json      # policy bound to both exact capture manifests
        base/manifest.json        # includes seed, semantic, and source attestations
        head/manifest.json
        comparison/<scenario>-<viewport>--base.png
        comparison/<scenario>-<viewport>--head.png
        comparison/<scenario>-<viewport>--overlay.png
        comparison/<scenario>-<viewport>--raw-diff.png
        comparison/<scenario>-<viewport>.png # highlighted side-by-side diff
        comparison/summary.json
        comparison/report.html   # self-contained base/head/diff browser report
      worktrees/
```

`latest-run.txt` contains the absolute path of the newest run. Worktrees, temporary Git refs,
proxies, port-forwards, local servers, and owned clusters are cleaned up on ordinary success or
failure. The runner also requests cleanup on `SIGINT` and `SIGTERM`, but an uncatchable termination
can leave run-scoped resources that must be removed by exact name. Completed screenshots and
reports are retained. Other runs are never automatically deleted.

Comparison thresholds are evaluated only for complete, cryptographically attested semantic pairs.
Missing, degraded, corrupt, and stale results remain distinct from pixel-diff failures. A verified,
successfully captured expected removal is still analyzed and keeps all five image artifacts for
review. Its trusted scenario default disables the failure threshold, while a reviewed policy may
supply a numeric threshold that is enforced normally. Every emitted PNG is listed in the
managed-output marker and recorded with its SHA-256 digest and byte size in the summary and
self-contained report. Base and head capture manifests must also carry the same versioned semantic
ID normalization policy; malformed or incomplete per-screenshot replacement evidence is rejected
before pixel thresholds are evaluated.

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

1. Validates the reviewed checkout, dependencies, tools, both non-overlapping port sets, and either
   an exact release-tag base such as `2.17.1` or an explicitly trusted non-release base ref. A
   release base must use first-party images carrying that exact tag; a non-release base is pinned
   and built locally.
2. Creates unique run state and a detached base worktree, then renders only each revision's actual
   platform-agnostic overlay. Workload and optional-service discovery never scans unrelated YAML.
3. Verifies and exports every rendered dependency image and builds the selected head's—and, when
   applicable, the non-release base's—revision-compatible frontend, frontend-server, backend, and
   runtime images for the explicit Kind node platform. The known 2.17.1 amd64-only workloads use
   narrow workload-level overrides on arm64; unknown architecture or build failures occur before
   deployment. When a component declares its complete build inputs and those inputs are byte-for-byte
   identical across two local revisions, the exact base image is retagged for the head instead of
   being rebuilt.
4. Creates two run-scoped Kind clusters with separate kubeconfigs, then loads only the images
   preflighted for that revision. Exact local image overrides and runtime-image variables are
   applied to each locally built revision before any workload starts. After each run-scoped image
   is imported, its host-side tag is released so the two isolated stacks do not retain a third copy
   of every locally built image.
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
   seed manifests plus one `ui-smoke-semantic/v3` manifest keyed by logical fixtures. The manifest
   maps per-revision run, task instance, artifact, Artifact URI, pod identity, retry-attempt,
   iteration, and relationship IDs; unknown or incomplete semantic bindings stop the comparison.
9. Binds each capture to the revision-specific deployed UI URL, semantic manifest, and immutable
   source provenance.
10. Captures both revisions, compares only exact successful manifest pairs with pinned analysis
    settings, writes the report, and applies the exit policy.
11. If explicitly requested, posts the report even when visual differences make the run fail.

### Full-stack failure diagnostics

A full-stack setup, seed, fixture-validation, or capture failure writes both
`full-stack-diagnostics.json` and a self-contained `full-stack-diagnostics.html` in the run
directory before owned clusters are removed. Capture validity uses one explicit value:
`valid`, `ui_rendering_failure`, `api_incompatibility`, `seed_failure`, `missing_fixture`,
`selector_drift`, `expected_product_removal`, or `infrastructure_failure`. An asserted
`expected_product_removal` is an expected-change outcome, not a pixel-diff failure. Missing and
degraded captures are never converted into visual-difference percentages.

For each cluster created by the run, failure collection records bounded Deployment and Pod status,
namespace events, and tail-limited logs from known KFP service Pods. Every `kubectl` request carries
that stack's explicit run-scoped kubeconfig and context. Diagnostics never request Secret objects
or container environment values; common credentials, authorization headers, cookies, tokens, and
credential-bearing URLs are redacted. Individual text artifacts live under
`diagnostics/{base,head}` and the JSON record contains their relative paths and SHA-256 hashes.
The JSON and HTML also embed bounded, redacted log previews, so the HTML remains a useful single
entry point after cleanup while the full tail-limited files remain available for deeper inspection.
When a capture manifest provides browser diagnostics, its bounded console errors and failed network
requests are included in the same failure record.

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
  --normalization-mode semantic-full-stack \
  --revision-role base \
  --seed-manifest ./seed/base.json \
  --semantic-manifest ./semantic-fixtures.json \
  --source-provenance ./source-provenance.json

node capture-screenshots.js \
  --base-url http://127.0.0.1:3001 \
  --output ./screenshots/head \
  --label head \
  --normalization-mode semantic-full-stack \
  --revision-role head \
  --seed-manifest ./seed/head.json \
  --semantic-manifest ./semantic-fixtures.json \
  --source-provenance ./source-provenance.json

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
