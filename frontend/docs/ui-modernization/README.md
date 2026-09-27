# UI modernization baseline

This baseline supports [issue #14572](https://github.com/kubeflow/pipelines/issues/14572) and [KEP #14574](https://github.com/kubeflow/pipelines/pull/14574). It records the existing UI before presentation changes, against application source and lockfiles at [`02cbc725ac9ddcd950f4400d8355dd78bfcd6c57`](https://github.com/kubeflow/pipelines/commit/02cbc725ac9ddcd950f4400d8355dd78bfcd6c57), captured on 2026-09-26.

The accompanying changes extend the existing screenshot harness and add the missing mock single-task lookup used by artifact lineage. Application code, backend contracts, dependencies and deployment configuration are unchanged. The baseline milestone remains open until the fixture, browser-performance and live compatibility gates below are satisfied.

## Evidence

[Browser performance measurements](performance-baseline.md) add nine fresh-context load samples, three filter trials and one exploratory run/task interaction trace under recorded CPU/network conditions. These extend the initial capture with controlled lab evidence.

| Artifact                                                                                        | Scope                                                                                                                                                                                                   |
| ----------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [Route/action/state inventory](route-inventory.md)                                              | 23 page routes, root redirect and not-found handling; artifact subviews, actions, state and URL/storage/integration contracts. Unchecked items are verification work, not missing product requirements. |
| [Deployment and browser contracts](deployment-baseline.md)                                      | Root and `/pipeline`, embedding, namespace/auth, probes, signing configuration, browser-floor decision and immutable rollback candidate.                                                                |
| [Screenshot index](screenshots/README.md) and [capture manifest](evidence/capture-results.json) | 25 native-mock routes at 1280×720 and 900×900; 50 full-page light-theme captures.                                                                                                                       |
| [Test/coverage results](evidence/test-results.json)                                             | Counts, commands and original coverage denominators.                                                                                                                                                    |
| [Coverage comparison input](evidence/coverage-baseline.json)                                    | Totals compatible with the existing `coverage:compare` script.                                                                                                                                          |
| [Production bundle measurements](evidence/bundle-sizes.json)                                    | Per-file sizes, compressed sizes and SHA-256 digests.                                                                                                                                                   |

Repeat capture results are in [repeatability.json](evidence/repeatability.json): **42 images are byte-identical; all eight graph images require manual review.** Two focused-task images have zero differences with the existing pixelmatch antialias handling; six unfocused graph images still have nonzero differences (maximum 0.2514% at threshold 0.1) after Fit View normalization. Existing initial-fit timing and intentional tiny position jitter make strict graph screenshot gates unreliable. Keep graph stabilization open before using those references as automated acceptance gates.

Screenshots are reference images for visual review. Scrollable panels remain at their captured scroll positions; full-page capture does not expose content inside every scroll container. They do not prove that displayed mutation controls, hidden states or deployment-specific behavior work. The current fixtures represent succeeded runs, sparse comparison data and one Dataset artifact; the pending fixture matrix is listed below.

## Verification and measurements

Toolchain: Node 24.14.0, npm 11.17.0, Vitest 4.1.11, Playwright Chromium 145.0.7632.6, macOS ARM64. Screenshot locale is `en-US`, timezone `UTC`, reduced motion enabled, and browser date fixed at `2026-09-26T12:00:00.000Z`. Each page uses a fresh browser context. Compare images with the same browser, operating system and font environment. The two configured creation forms receive fixed visible names; date freezing alone does not normalize their random name suffixes.

| Check                                                | Result                                                                                                               |
| ---------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------- |
| Existing UI suite with coverage                      | 127 files, 1,618 tests passed with two workers.                                                                      |
| Server suite with coverage                           | 29 files, 1,056 tests passed, including four new mock task lookup regressions.                                       |
| Capture harness regressions                          | 27 tests passed, including failed requests, missing ready data, stale screenshot removal and isolated page contexts. |
| Production build                                     | Passed, including UI/server lint and application typecheck.                                                          |
| Production bundle browser smoke                      | Passed; emitted bundle initializes and displays pipeline upload control without uncaught errors.                     |
| Format, mock-backend typecheck and React peer checks | Passed.                                                                                                              |

The initial combined `test:ci` invocation hit a timeout in `Trigger.test.tsx`. Its 29 tests passed in isolation, and the complete unchanged UI suite passed with two workers; no timeout was raised or test skipped. The successful component commands above are the recorded evidence, rather than a claim that the combined command passed.

| Coverage   | UI                    | Server               |
| ---------- | --------------------- | -------------------- |
| Lines      | 50.96% (5,922/11,620) | 81.32% (2,687/3,304) |
| Branches   | 43.50% (4,247/9,762)  | 78.72% (2,087/2,651) |
| Functions  | 58.14% (1,557/2,678)  | 73.90% (439/594)     |
| Statements | 51.28% (6,075/11,846) | 81.03% (2,744/3,386) |

Keep the original Vitest inclusion/exclusion rules when comparing. Coverage percentage alone does not establish workflow parity: preserve meaningful behavior assertions and explain denominator changes when obsolete presentation code is removed. To use these totals with the existing comparison tool, run `cp frontend/docs/ui-modernization/evidence/coverage-baseline.json frontend/.coverage-baseline.json` from the repository root, then run `npm run coverage:compare` inside `frontend`.

| Emitted assets        | Raw bytes | gzip bytes | Brotli bytes |
| --------------------- | --------: | ---------: | -----------: |
| JavaScript (one file) | 2,803,952 |    780,136 |      630,132 |
| CSS (one file)        |    28,104 |      5,521 |        4,879 |

Compression is an offline per-file calculation using Node `gzipSync(buffer, {level: 9})` and default `brotliCompressSync(buffer)`. Sourcemaps and analyzer reports are excluded. This is not HTTP transfer size or an interaction-performance measurement; Vite's own gzip summary uses a different compression setting. Hashes and other emitted assets are in the JSON evidence. Use the same toolchain, build mode and compression settings for comparisons.

## Reproduce tests and capture

Use the application revision above with the harness and mock-route changes accompanying this document. Install the pinned dependencies with `npm ci` in `frontend`; its postinstall installs the server and mock backend dependencies. Install the pinned browser with `npx playwright install chromium --only-shell`.

From `frontend`, before starting the mock server:

```sh
CI=true npm run test:ui:coverage -- --maxWorkers 2 --exclude scripts/visual-compare.test.mjs
npx vitest run scripts/visual-compare.test.mjs --maxWorkers 1
CI=true npm run test:server:coverage
npm run format:check
npm run typecheck:mock-backend
npm run check:react-peers
CI=true npm run build
node --test scripts/production-bundle.smoke.mjs
```

Server tests use mocked Kubernetes clients but require a valid context during client construction. For isolated execution, set `KUBECONFIG` for that command to a temporary credential-free config whose cluster server is `https://127.0.0.1:9`, with matching cluster/context/user names and an empty user object. This does not qualify a live deployment. Do not run the mock API server concurrently with server tests: both use port 3001.

After tests, run these in separate terminals from `frontend`:

```sh
npm run mock:api
```

```sh
NODE_OPTIONS=--dns-result-order=ipv4first npx vite preview --host 127.0.0.1 --port 4173 --strictPort
```

```sh
node scripts/visual-compare.mjs capture \
  --base-url http://127.0.0.1:4173 \
  --out-dir .visual/baseline \
  --fixed-time 2026-09-26T12:00:00.000Z \
  --viewports 1280x720,900x900
```

Use a fresh output directory. A failed capture removes that route's stale image and causes a nonzero exit; unrelated old files are intentionally retained, so mixing inventories in one directory can mislead the existing PNG-based diff command. The manifest records each attempted route, readiness checks, setup, viewport and status. Published paths are normalized to this document's directory and each screenshot has a SHA-256 digest.

The [route manifest](../../scripts/visual-compare.routes.json) is the executable capture definition. Optional `fitGraph` invokes the existing Fit View control after fonts and node measurements settle; the three unfocused graph captures use it, while the task-focused view retains its existing framing. `waitForSelectors` requires additional loaded data, `fillFields` normalizes accessible textboxes, and `failOnRequestErrors` checks same-origin pathname prefixes. These error checks and `failOnPageErrors` are opt-in so intentional error-state captures and deployment probes can be configured explicitly. Comparison requires both expected run rows and successful scoped data requests; an empty-parameters message alone is insufficient.

## Gates before workflow migration and release qualification

- [ ] Stabilize graph capture timing/position jitter and prove repeated equivalence before enabling strict graph image gates. Preserve task-focused framing.
- [ ] Add scenario fixtures/captures for loading, empty, backend error, partial response, recovery and 401/403 states; preserve the happy-path fixtures.
- [ ] Add running, failed, canceled/paused and archived runs with consistent tasks/retry attempts. Add long names, enough rows for pagination, overflow and mixed selection.
- [ ] Populate comparison parameters and metrics, ClassificationMetrics, HTML/Markdown/table viewers, multiple artifacts and missing/deleted provenance. Existing static viewer files are not wired native workflow evidence.
- [ ] Exercise graph/task navigation, logs/events, form validation, upload, creation/cloning, archive/restore/delete, retry/terminate and schedule enable/disable against a seeded backend. Reuse the existing [interactive smoke harness](../../scripts/ui-smoke-test/README.md) rather than growing a second general action framework.
- [ ] Qualify standalone and embedded multi-user namespace/permission behavior, shared pipelines and Kubernetes-backed pipeline storage. The default mock health response is single-user with database storage; artifact fixture endpoints ignore query filtering/paging.
- [ ] Agree a versioned browser floor and measured performance budgets. The [initial browser baseline](performance-baseline.md) covers loads, filtering and exploratory run/task navigation; extend it to realistic workload sizes, repeated task navigation, selection and populated comparison before closing this gate.
- [ ] Verify the previous UI image and modernized UI against the same backend, record immutable UI/backend identities, and rehearse rollback using the [deployment checklist](deployment-baseline.md). Registry resolution alone does not establish compatibility.

The 900-pixel capture documents existing narrow layout, not mobile acceptance. Dark theme, keyboard/focus, contrast and broader responsive acceptance remain implementation/qualification work. KFP Local execution, SDKs and backend schemas are unchanged by this baseline; the KEP's no-migration requirement still applies to the eventual UI cutover.
