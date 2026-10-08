# UI capture checks

The existing screenshot harness captures fixture routes with loaded-data checks, isolated pages, deterministic form values and an optional fixed clock. The mock single-task endpoint supports artifact-to-task navigation. Interactive workflow testing belongs in the existing [smoke harness](../../scripts/ui-smoke-test/README.md).

Keep these workflows complementary: this harness checks layout and styling against mock data without a cluster. The smoke harness checks seeded runtime resources; its default comparison shares the base runtime, while `--full-stack` uses separate revision-matched deployments. Neither substitutes for component tests or certifies upgrades. Their capture manifests and report directories are separate formats and must not be interchanged.

## Retained historical evidence

The complete pre-trim baseline—50 screenshots, manifests, compact traces, inventories, measurements and corresponding tooling—is preserved in the [baseline evidence release](https://github.com/jeffspahr/jeffspahr-pipelines/releases/tag/ui-baseline-evidence-20261007), linked from [issue #14572](https://github.com/kubeflow/pipelines/issues/14572#issuecomment-6042440286).

[evidence-archive.json](evidence-archive.json) records the download URL, archive hash and source revision. It is an archival index, **not** input to the capture tool. The bundle contains its own per-file SHA-256 index and all referenced screenshots. Historical measurements retain their original application revision and limitations; they are not release qualification for the modernized UI.

## Contracts

- **Runtime:** Node ≥24.2.0; use `frontend/.nvmrc`. The script checks this before selecting its CLI entrypoint. Chromium must be available for captures.
- **CLI paths:** relative arguments and output defaults resolve from `process.cwd()`. `INIT_CWD` has no effect. npm `visual:*` scripts run in `frontend`; the comparison wrapper supplies absolute paths.
- **Capture directories:** `capture-results.json` is required. Screenshot paths resolve relative to that manifest. Keep the manifest and images together; stray PNGs are ignored.
- **Comparison:** every baseline entry is expected in current. Missing or failed captures fail with the affected side named. Additional successful current captures are informational; an explicitly failed additional capture still fails. Global capture errors always fail, while valid completed rows remain available for diagnostics.
- **Exit status:** input/capture/comparison errors always fail. Pixel changes alone fail only with `--fail-on-diff`.
- **Determinism:** the wrapper uses the same fixed UTC time for both sides. `FIXED_TIME=` selects real time; a nonempty override fixes both clocks. Use matching browser, fonts, OS, viewport and fixtures for meaningful visual comparisons.

## Use

From `frontend`, with the application and mock API already running:

```sh
node scripts/visual-compare.mjs capture \
  --base-url http://127.0.0.1:4173 \
  --out-dir .visual/current \
  --fixed-time 2026-09-26T12:00:00.000Z \
  --viewports 1280x720,900x900

node scripts/visual-compare.mjs diff \
  --baseline-dir .visual/baseline \
  --current-dir .visual/current
```

Capture `.visual/baseline` against the chosen baseline application first, or use a previously captured complete directory. The default [routes](../../scripts/visual-compare.routes.json) define readiness and normalization for mock fixtures. Unknown route keys fail validation. Browser captures belong in the configured CI environment; the regression tests below do not launch browsers.

`npm --prefix frontend run visual:current` from repository root writes to `frontend/.visual/current`. For direct `node frontend/scripts/visual-compare.mjs` invocation from repository root, pass `--out-dir frontend/.visual/current` explicitly. The cross-revision wrapper remains available as `frontend/scripts/visual-compare-run.sh <base-commit>`.

The default routes target an application containing the run Timeline and export/import pages. The route set includes run Timeline (retry and cache-hit fixtures) and the export/import form. Captures verify the loaded forms; they do not execute metadata exports or imports. For revisions before those pages existed, use a compatible routes file for the baseline only:

```sh
BASE_ROUTES=/absolute/path/to/historical-routes.json \
  frontend/scripts/visual-compare-run.sh <base-commit>
```

`ROUTES` selects the current routes; `BASE_ROUTES` defaults to the same file. Both accept paths relative to the invoking directory. Additional successful current screenshots are reported as added; failures still fail the comparison. A previously captured historical baseline can also be compared directly without recapturing unsupported routes.

## Verify

From `frontend`:

```sh
CI=true npx vitest run --coverage \
  scripts/visual-compare.test.mjs \
  scripts/visual-capture-flow.test.mjs \
  scripts/visual-compare-run.test.mjs
```

These tests use fake browser transport, tiny PNG fixtures and CLI boundary checks. The large historical artifacts are archived and are not processed in routine CI.
