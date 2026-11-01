# CI and Workflows

GitHub Actions workflows are in `.github/workflows/`; reusable composite actions are in `.github/actions/`.

- Update this guide when changing workflows, CI matrices, commands, generated outputs, or common failure handling.
- `presubmit-backend.yml` exercises the self-service namespace transfer engine against disposable MySQL and MLMD services, using `KFP_TRANSFER_MYSQL_DSN` and `KFP_TRANSFER_MLMD_ADDRESS`. `KFP_TRANSFER_REQUIRE_INTEGRATION=1` prevents silent skips. SQL tests create/drop isolated databases; MLMD fixtures use unique resource names. The MLMD image matches the release deployment manifest. HTTP/authorization and frontend tests cover the end-user transfer boundary; no Python operator tool is required.
- `presubmit-backend.yml` runs recurring-run transaction concurrency tests against disposable MySQL 8.0 and PostgreSQL 16 services using `KFP_RECURRING_MYSQL_TEST_DSN` and `KFP_RECURRING_POSTGRES_TEST_DSN`. Tests allocate isolated databases/schemas and independent connections; the configured test roles need permission to create and drop their test database/schema and observe session lock metadata. These checks cover storage transaction behavior, not a live KFP upgrade.
- Release 2.18 supports Python 3.9+ for the KFP SDK and Kubernetes SDK plugin. SDK unit and integration lanes exercise 3.9, 3.10 and 3.13 on Ubuntu 24.04. The unit lane runs `sdk-installed-smoke.py` after installing local companion packages to verify installed package versions, pipeline compilation with Kubernetes configuration and CLI imports. Interpreter markers retain compatible Requests/urllib3 on 3.9 and patched releases on newer interpreters; 3.9 security findings require explicit release disposition. The syntax-only lane also retains Python 3.9 grammar checks.
- The shared `test-and-report` action exports `KFP_TEST_SOURCE_SHA` from the actual checkout. Go E2E component SDK URLs prefer this immutable commit over PR merge refs and local branch defaults, so merging a PR cannot invalidate ongoing SDK installs and manual release runs test their own SDK.
- CI covers supported Kubernetes and Argo versions, database and Kubernetes pipeline stores, proxy/cache variants, GPU scheduling, and DRA (Dynamic Resource Allocation). Preserve the relevant matrix coverage when changing a lane. Release-2.18 retains the Argo 3.7.x compatibility lane alongside the default 4.1.x runtime; exact patches come from `third_party/argo/COMPATIBILITY_VERSION` and `VERSION`. The master branch's 4.0.x floor must not be backported to this release branch.
- The `dra-check` E2E lane runs on a Kind cluster with Kubernetes 1.34 (DRA GA minimum). It generates both Python and Go pipeline-spec bindings, then uses `kfp component build` with the checked-out SDK, Kubernetes plugin, server API client, and generated pipeline-spec packages to create the `kfp-dra-sdk:test` component image, then loads that image into Kind. `.github/actions/install-dra-driver` installs dra-example-driver via Helm, runs a smoke test that verifies a pod can schedule with a DRA ResourceClaim, then the E2E suite uploads DRA pipelines and validates pod-level `resourceClaimStatuses`. The driver version is pinned in `action.yml`; v0.2.1 targets K8s 1.34, while v0.4.0+ requires K8s 1.36.
- Use `.github/actions/setup-python-pip-cache` for pip caching. Give each dependency set a distinct `cache-scope` and hash every installed requirements file; do not use `setup-python`'s built-in pip cache.
- `pre-commit.yml` runs the hooks pinned by `.pre-commit-config.yaml` over each base-to-head change. Its local isort hook pins isort 5.10.1 through `additional_dependencies`, matching `sdk/python/requirements-dev.txt`; do not replace it with a newer upstream hook without also updating the SDK formatter. Configuration-only updates also run each applicable content hook against a small set of clean representative files so a hook upgrade executes without requiring the repository's historical formatting backlog to be fixed in the same PR.
- `validate-generated-files.yml` validates backend-generated outputs. Go-based API generator versions come from the root or `backend/api/tools` Go modules; a lightweight change detector skips this expensive validation for unrelated Go module updates. API generator toolchain upgrades must include every affected committed output and a root `go mod tidy`. Regenerated clients can also change handwritten integration points; go-swagger v0.36 and later pass request contexts through the operation's `...Context` method instead of a public parameter `Context` field. `frontend.yml` runs `npm run apis:all` and rejects stale frontend clients.
- Presubmit enforces the bounded [Go version policy](go-version-policy.md): tracked modules, explicitly registered builder `FROM` lines, and the two setup-go actions must agree with the root compiler. A path-filtered job inspects each pinned builder digest and verifies its Linux AMD64 and ARM64 image configurations declare the expected Go version. Upstream tag changes do not invalidate existing pins; Dependabot proposes image updates.
- Changes to `frontend/server/handlers/artifacts.ts` trigger `e2e-test.yml` so the multi-user artifact-proxy and SeaweedFS namespace-isolation checks run before artifact authorization changes merge.
- `frontend.yml` installs the standalone mock backend's nested dependency tree, and its server integration suite constructs that app through `frontend/mock-backend` so router upgrades are tested against the package that actually runs locally.
- Frontend integration tests require Node >=22.12 because of the `@puppeteer/browsers` 3.x pin explained in `test/frontend-integration-test/README.md`. `e2e-test-frontend.yml` runs `npm run test:dependencies` once in a plain Node job alongside the Kind lanes; the checks cover configuration merging, the installer API WebdriverIO calls, and remote Selenium setup. Keep the pin, the README rationale, and the checks aligned.
- `frontend.yml` sets up Python for the server integration test that feeds actual profile-controller artifact repositories into archived-log retrieval. Changes to `pipelines-profile-controller/sync.py` also trigger this workflow so producer and consumer changes both exercise the test.
- Non-documentation changes under `third_party/metadata_envoy/` trigger `e2e-test-frontend.yml`, which builds and deploys the metadata-envoy image before exercising the frontend-to-MLMD path.
- `.github/dependabot.yml` schedules weekly updates for every supported dependency ecosystem present in the repository: Go modules, Docker and Kubernetes image references, npm, pip, GitHub Actions, and pre-commit. Every generated PR starts with `do-not-merge/hold`; remove it only immediately before merge after verifying CI for the current head, and restore it after any head change. npm and pip updates are grouped by dependency name across their monorepo directories; generated Python API-client compatibility constraints remain excluded.
- `python-syntax.yml` parses every tracked Python file with Python 3.9 grammar without importing it, so syntax errors fail presubmit without creating `__pycache__` directories.
- `docs.yml` builds the Sphinx documentation site with warnings treated as errors and runs a non-blocking external link check; Read the Docs (`.readthedocs.yml`) handles publishing, not this workflow. Before installing `docs/requirements.txt` through `setup-python-pip-cache` under the `docs` cache scope, both jobs generate the Python API protobuf bindings and install the source-built `kfp-pipeline-spec` package through `.github/actions/protobuf`. They explicitly install the checked-out SDK and server API client alongside the documentation requirements (which contain only external dependencies so Dependabot can evaluate them in isolation) so an unpublished release version does not require its companion wheel on PyPI. The SDK client lane likewise installs the local server API before resolving SDK requirements. DRA includes all four local KFP packages in both its host environment and component image; preserve both paths when changing version pins.
- `osv-scanner.yml` runs on every push to master, weekly, and on manual dispatch. It scans supported dependency manifests and lockfiles and separately scans the container images rendered by the standard standalone and multi-user Kustomize overlays, removing structurally identical SARIF results before uploading both result sets to code scanning. Compiled Python requirements are scanned without dependency re-resolution; generated API-client constraint directories are excluded because they specify compatibility minimums rather than installed versions. Dependabot remains responsible for remediation PRs.
- `ai-analyzer.md` is the source for the AI issue quality analyzer; `ai-analyzer.lock.yml` is generated by GitHub Agentic Workflows. The analyzer runs when an issue is opened. Invalid-title runs post guidance and attempt to self-cancel so a correction stays quota-free. A title edit retries analysis only while the current title is valid and no prior quality verdict exists; other title edits also attempt to self-cancel before inference, and successful cancellations do not count toward the user rate limit. Body-only edits use isolated concurrency groups and skip without provisioning a runner, while analyzed per-issue runs use a full queue so later edits cannot evict a pending title correction. After editing the source, run `gh aw compile .github/workflows/ai-analyzer.md` and commit both files.
- Workflows that deploy CI-built images overlap Kind setup with current-branch image builds. The shared deploy action waits for the complete artifact inventory; its normal window extends while a missing artifact's producer is queued or running, and completed producers receive a short publication grace.
- CI Docker-sensitive paths use retry wrappers for image builds, BuildKit bootstrap, and runtime-image pulls. GitHub-hosted Docker jobs configure `mirror.gcr.io` as a Docker Hub pull-through cache, docker-container builders use the same mirror through `buildkitd.toml`, and uncached images still fall back to Docker Hub. Build jobs explicitly try and locally tag the mirrored BuildKit bootstrap image before falling back to Docker Hub within the shared retry budget.
- `runtime-base-images.yml` refreshes the shared runtime-image artifact. `image-builds.yml` prefers a matching current-source or trusted-master artifact, but rebuilds locally after a bounded lookup miss or failed download. Both workflows use `build-runtime-base-images.sh` to archive test-task, MySQL, and supported Argo images plus the Modelcar fixture. This keeps release branches and older PR snapshots working after shared artifacts expire without permanent release-specific producer lanes. Generation failures stop the job before upload; opted-in E2E lanes still require the Modelcar archive during deployment.
- Runtime base-image pulls and exports select the Docker server's platform in producers. The Kind fallback selects the actual node image platform and imports a temporary platform-scoped archive; its temporary file is removed on success or failure. This avoids incomplete multi-platform indexes that Kind imports with `--all-platforms`. Shared producers and consumers run on the same platform; heterogeneous Kind nodes are rejected. The generation fingerprint includes `helper-functions.sh`, so changes to this export logic invalidate old artifacts and caches.
- The DRA E2E lane loads its locally built `kfp-dra-sdk:test` component image the same way: `docker save --platform` for the Kind node platform from `runtime_image_platform` to a temporary archive, then `kind load image-archive`; the archive is removed on exit. `kfp component build` uses Docker's classic builder, whose images record no platform on the containerd image store (the default from Docker 29, used by `ubuntu-26.04` runners), so the lane first re-exports the image through BuildKit for that platform.
- For workflow-only changes, verify referenced working directories, Docker contexts/files, scripts, and local action paths exist.
- The KFP Manifests suite also checks the rendered profile-controller NetworkPolicy for the base multi-user install and the plain, platform-agnostic, and PostgreSQL multi-user overlays. These selector/namespace/port assertions run through `manifests/kustomize/hack/test.sh`; they do not verify live CNI enforcement or override additive allowances from other policies.
- Frontend CI also runs `frontend/scripts/check-spec-generation.sh` after installing `protoc`: pipeline and Kubernetes platform specs are generated into temporary directories and typechecked. Changes to either source proto directory trigger this workflow.
- The frontend protobuf installation scopes both APT commands to `/etc/apt/sources.list.d/ubuntu.sources`, excluding unrelated runner repositories such as Chrome without changing their configuration or bypassing package verification. If the runner's source layout changes, update this explicit path; the step checks that it exists before running APT. Browser integration tests continue to use their separate Selenium container.

- The shared runtime-image inventory pins acquisition of the Python 3.12 image used by the cache and proxy fixtures to a Docker Hub digest. Archive creation and direct Kind loading restore its runtime tag before save/load; keep fixture image references and source declarations aligned with that tag. CI script tests check this contract whenever those fixtures change.

- Frontend Vitest snapshots preserve rendered whitespace. The trailing-whitespace hook excludes `frontend/**/*.snap`; update snapshots through Vitest so formatting hooks do not alter expected UI text.
- `manifests/kustomize/hack/test.sh` runs `tensorboard_signing_key_test.py` to verify the shared Secret, least-privilege initializer permissions, migration-safe `Recreate` and documented opt-in `RollingUpdate`, generated Job names, and the actual CI and OpenShift overlays. The suite renders the exact rolling-update patch from `docs/operator-guides/server-config.md`, so changes to that guide also trigger `kubeflow-pipelines-manifests.yml`. The suite requires Python 3, `kustomize`, and mikefarah `yq` 4.54.1 (using its v4 command syntax); `manifests/kustomize/hack/presubmit.sh` installs pinned `kustomize` and builds pinned `yq` from `backend/api/tools` with the repository Go toolchain. Keep coverage for CI's preloaded initializer image and OpenShift's namespace-assigned UID policy when changing these manifests.

- Cache-deployer Dockerfile changes trigger the CI script suite. Its kubectl build must retain the verified Kubernetes Git tag and commit; build-time assertions reject missing or incorrect embedded module versions and revisions.

## Common CI failures

- Read the Docs release-readiness builds limit `sphinx_immaterial` font fetching to four workers with `SPHINX_IMMATERIAL_FONT_FETCH_MAX_WORKERS`, rather than its default 128. This reduces connection pressure on Google Fonts after observed TLS EOF and remote-disconnect failures. Both SDK documentation steps inherit the limit; fonts are still bundled, TLS verification stays enabled, and Sphinx errors still fail the job.

- The `sdk-viewer-assets` pre-commit hook rebuilds the SDK bundle on frontend, bundled-asset, or hook/workflow configuration changes; modified outputs fail the check and must be committed. Pre-commit CI installs the pinned Node/npm toolchain and locked dependencies only for these changes or configuration smoke tests. Local contributors must run `npm ci` in `frontend` first, and again after lockfile changes. See [viewer build instructions](../../frontend/src/standalone/README.md#rebuild-the-sdk-assets).

- The standalone viewer's generated `sdk/python/kfp/cli/viewer_assets/viewer.js` and `viewer.css` are excluded only from pre-commit's end-of-file and trailing-whitespace rewriting hooks. Vite owns those bytes, including whitespace in bundled third-party template strings; rebuilding would undo manual whitespace edits. Source files, licenses, and non-rewriting checks remain covered. `.gitattributes` does not control pre-commit hook exclusions.

- SDK imports are checked by isort 5.10.1 in both lanes: `sdk/python/requirements-dev.txt` for the SDK presubmit, and the pre-commit hook's `additional_dependencies`. Keep the two pinned together; a newer pre-commit hook wraps long imports differently and conflicts with the SDK formatter.
- Keep docformatter on v1.7.7 until its v1.7.8 tokenization regression is fixed: v1.7.8 crashes on explicit continuations and rewrites SDK blank lines in conflict with YAPF. Verify formatter upgrades by running the full hook chain twice on the updater and SDK structures files.
- Registry pull failures for Kind, BuildKit, Python, or Alpine images are usually transient; retry before changing code.
- A Kind checksum mismatch after cache restore means no tests or deployment ran; retry the job.
- SeaweedFS `PutObject` timeouts are artifact-store instability; retry rather than weakening assertions or increasing pipeline timeouts.
- For proxy failures, inspect the `tinyproxy` namespace pods, events, services, endpoints, and endpoint slices.
- Tinyproxy runs in the foreground under Kubernetes supervision. Keep its PID file disabled; health probes use its listening socket, and no fixture consumes a PID file. This avoids requiring a writable runtime path without changing its service identity or security profiles.

## Release 2.18 stabilization

- `release-2.18` preserves the MLMD-based release line. Backport selected fixes through reviewed PRs; do not merge the post-MLMD `master` branch wholesale.
- Merge queue checks run against the queued commit from the `merge_group` event. GitHub does not apply pull-request path filters to this event, so queued changes run the full release CI suite and use more runner time than ordinary PRs. The aggregate gate must fail closed when any release PR workflow lacks a successful check on the queued SHA.
- Both release E2E workflows run for `release-2.18` merge groups so the aggregate gate can verify the queued commit, including the backend Kind matrix and frontend integration lanes. Their workflow and reusable image-build tokens have read-only repository permissions because merge-group code may include changes from untrusted PRs.
- Queue-only tool and runtime image workflows build local images and upload test artifacts with read-only repository permissions. Registry publishing remains in the push/dispatch workflow; runtime cache pruning remains in the existing master-only workflow. SDK package references and remote test fixtures use the immutable queued SHA rather than a moving PR or branch ref.
- Release deployment CI uses MySQL. PostgreSQL production support is deferred until 3.0 because MLMD can create duplicate metadata types under concurrent PostgreSQL runs ([#14353](https://github.com/kubeflow/pipelines/issues/14353)). Keep KFP `pgx` deployments out of the release matrices while this limitation remains. Preserve both MySQL cache variants, multi-user artifact-proxy coverage, and MLflow's own PostgreSQL backend and registry stores. The aggregate CI gate still requires all applicable jobs to pass.
- Frontend and pre-commit workflows accept pushes and PRs targeting `release-2.18`. Backend and E2E PR workflows already accept this target subject to their path filters; their push triggers remain scoped to master.
- Require the unconditional pre-commit check and DCO on release PRs. Path-filtered jobs must pass when applicable but should not be required unconditionally, because GitHub would wait for checks that do not run.
- Tool-image builds publish `kfp-api-generator:release-2.18` and `kfp-release:release-2.18` on release-branch pushes. This branch uses those images for generation/release work to avoid consuming post-MLMD master tools.
- The manual image-release workflow grants its reusable build caller `contents: read`, `packages: write`, `attestations: write`, and `id-token: write`; manifest promotion needs only `contents: read` and `packages: write`. The callees use the automatic `GITHUB_TOKEN`, not inherited repository secrets. Manifest digest references must stay separate `imagetools` arguments; keep them in a Bash array rather than a space-delimited string. Preserve the legacy image matrix and dry-run behavior. Do not lower the repository token default until this caller contract is merged on every supported release branch and representative CI and release dry runs pass.
- A branch cut is not a release. Do not publish `2.18.0`, move `latest`, update documentation defaults, or sync the 2.18 version back to a divergent master as part of stabilization.

- Release merge-group runtime-image consumers retain exact queue SHA, branch, upstream repository, and successful producer-workflow validation. After three unsuccessful lookups or a failed download, they use the shared archive builder on the queued checkout. Queue and regular producers fingerprint the same generation inputs, including that builder; queue jobs remain read-only and do not publish images or prune caches.

- The release-2.18 upgrade workflow always upgrades from 2.17.2, including merge-group runs, and fails before target deployment if source preparation fails. The independent readiness schedule lane is off by default. After candidate policy review, activate it with repository variable `KFP_218_READINESS_SCHEDULES=enabled` or the manual `run_readiness_schedules=true` dispatch input, then record live acceptance in #14421. Its preflight rejects missing prerequisites. The isolated fixture verifies source reporting permissions, correlates 2.17.2 runs with their Workflows, verifies automatic adoption of the original legacy schedule IDs before policy observation, and checks successful enforce/audit runs plus scoped audit emission. It then verifies return to enforcement, named controller-grant revocation and restoration, and V1 jobs submitted through the retained V1 API. Each phase uses fresh identity-correlated evidence and a working default-account control; prior runs do not satisfy a later phase. The opt-in lane has a 90-minute ceiling to accommodate controller retry backoff and the additional bounded observation windows. It does not introduce master MLMD-to-native migration.
- `upgrade-readiness.yml` runs standalone preview tests and conformance against the explicitly pinned proposed scheduling policy; neither substitutes for a populated candidate upgrade run. Python 3.11 and 3.13 tests use synthetic inventories, subprocesses and loopback HTTP servers, never cluster credentials. Slow-header/chunk-framing probes enforce the response deadline, and fake-clock observations cover the final collection interval.
- CI Scripts Tests installs PyYAML 6.0.3 for Dependabot configuration validation. Argo dependency updates use `third_party/argo/Makefile` and the coordinated updater; update both runtime slots only after checking compatibility, module major version, and manifest downloads.

- The live readiness fixture installs `tools/upgrade-readiness/requirements.txt`
  and uses its hash for the pip cache. The read-only scanner and unit tests remain
  standard-library-only. `kubectl_inventory.py` owns bounded read-only subprocess
  collection; `fixture_http.py` owns the mutating fixture's loopback HTTP client.
- Upgrade CI uploads JSON evidence from the isolated fixture for seven days.
  There is no upload-time redaction step: collectors omit tokens, raw API/log
  payloads, and workflow specifications, but reports retain namespace, resource,
  service-account and run identifiers. Use only the synthetic CI cluster and
  review report contents before sharing; these artifacts are not anonymized.

- `artifact-storage-acceptance.yml` runs on explicit dispatch and non-documentation
  changes under `frontend/**`, its workflow, the selected SeaweedFS deployment,
  and the local cluster-creation/artifact-upload actions and their runtime helpers.
  This includes dependency locks, Node/Vitest configuration, shared application
  code, and test helpers; filtering only artifact handlers would miss these inputs.
  Markdown and `OWNERS` changes alone do not trigger this lane. It exercises the real standalone UI handlers, HTTP fetch, Kubernetes
  Secret/Workflow reads and S3 client against the stock SeaweedFS image. Stored
  artifact and archived-log bytes are seeded once, then read under default and
  explicitly migrated origin/base configuration. Rejections assert policy errors
  and zero outbound requests, not incidental network failures. The retained
  Vitest JSON and revision identify the exact candidate. This component lane
  does not run a 2.17-to-2.18 installation upgrade, authenticate tenants, execute
  the synthetic Workflow records, or reconcile profile-controller proxies; those
  remain separate live acceptance scopes. Its fixtures use only disposable
  credentials and a dedicated Kind namespace.

- The manual `run_readiness_adoption=true` input adds a separate populated
  2.17.2-to-candidate adoption job. It retains real completed source history,
  one enabled schedule with a persisted suspended Workflow, and two disabled
  schedules. It applies candidate images through the managed rolling deployment
  while writers remain running, then compares automatic adoption receipts,
  progress and stored definitions. Workflow evidence is read in pages of 100
  with a consistent list resource version and the existing per-response size
  limit; snapshots retain compact identity/progress fields rather than manifests.
  Missing or inconsistent pages cannot establish successful adoption.
  The fixture restarts the upgraded API and
  scheduled controller while the active run holds its concurrency slot, requires
  fresh ready Pod identities with no old or terminating Pods, reopens its local
  forwarding connection, and verifies preserved state and successful continuation.
  The separate scheduling-policy lane also adopts original source schedule IDs.
  A suspended Workflow tests nonterminal run accounting; it does not establish
  transparent retries in arbitrary task containers or zero downtime for every
  API operation. Rejected handoff mutations and successful retries are covered
  by deterministic resource tests. Only sanitized phase summaries under
  `reports/` are uploaded; SQL snapshots, manifests, credentials and raw adoption
  logs remain in the disposable runner.

- Manual `run_readiness_reporting=true` adds a separate disposable 2.17.2 upgrade
  lane. It retains suspended V1 immediate and recurring runs, captures read-only
  ownership findings, and verifies that the actual persistence worker catches up
  after API-server Workflow GET permission is restored. The fault does not remove
  the worker's informer access. Its scoped observer preflight submits explicit
  SubjectAccessReview attributes for the selected virtual KFP run resources;
  `kubectl auth can-i` discovery fallback does not preserve their API group.
  A test-only reporting interceptor exercises the
  deletion-after-capture race; it does not establish recovery when no terminal
  snapshot was captured. The lane must restore injected permissions on failure,
  upload only explicit sanitized report files, and delete its isolated cluster.
  Raw manifests, token files and restoration inputs are not acceptance artifacts.

- `upgrade-test.yml` has a manual, default-off `run_custom_roles` input. Its
  isolated `kind-kfp-custom-roles` cluster runs candidate API/UI images against
  namespaced custom-role users: private/shared upload and version inheritance,
  stored-reference confinement, `runs/readLog`, and TensorBoard viewer/manager
  permissions. Denials require the exact status and upload authorization
  diagnostic; the retained JSON report includes positive controls and revision.
  Trusted identity headers reach localhost-only forwards, and the log source is
  a controlled real Pod associated with an API-created run. This validates
  handler authorization and Kubernetes SAR, not ingress authentication, Argo
  execution, archived logs, browser navigation, or TensorFlow execution. The job
  deletes its entire disposable cluster even on failure.

- The pagination upgrade fixture retains its source API through mixed-version
  verification and evidence upload, then removes that deployment before general
  upgrade verification. Cleanup runs after failures and cancellation, tolerates
  an absent fixture, and waits for foreground deletion of its Pods. General
  upgrade verification uses an explicit non-cancellation condition so pagination,
  evidence-upload, or cleanup failure cannot silently skip it; those failures
  still fail the job.

- Pagination acceptance additionally creates suspended synthetic V1 runs to test
  mixed-case V1/V2 run lists and nullable metric cursors. A separate always-run
  cleanup deletes only Workflows labeled
  `pipelines.kubeflow.org/pagination-fixture=true`, including after failure or
  cancellation. Source baselines and independent complete inventories distinguish
  old defects from changed comparison order; fresh candidate omissions always fail.

- `cache-rolling-upgrade.yml` validates the legacy cache server separately from
  native V2 caching. It loads the published 2.17.2 cache image and a source-built
  candidate into disposable Kind with real MySQL, TLS admission and production
  Pod watchers. It proves a populated pre-namespace schema migrates while an old
  process remains active, old admissions and writes continue, and namespace-empty
  writes cannot satisfy enforcing scoped lookups. A temporary Deployment
  `minReadySeconds` and per-revision Services make overlap observable;
  both watchers remain running. The ordinary rolling Deployment then converges
  before cold-to-warm, cross-namespace and retained old-format completion checks.
  Retained completion requires the candidate watcher’s explicit rejection log,
  with pod-label and database invariants rechecked after subsequent writes.
  Synthetic Pods replace Argo's completion label, so this is cache component
  acceptance, not full pipeline or audit-mode acceptance. The JSON artifact
  records exact source/target revisions, live image IDs and successful phases;
  failures remain failures and the disposable cluster is always deleted. The
  candidate removes the source CACHE_IMAGE override, verifies the default helper
  completes a real cache-hit Pod, and records its image ID and log output. Helper
  qualification retains its Alpine package database and pinned OSV JSON/SARIF
  reports; missing package metadata or scanner errors fail coverage checks.
  Advisory counts are evidence, not automatic release exceptions.
- Docker mirror configuration uses `/usr/bin/python3` on Ubuntu runners for its
  privileged standard-library-only helper. Do not select a setup-python runtime
  from `PATH` or preserve loader environment variables through `sudo`: tool-cache
  interpreters can require `LD_LIBRARY_PATH`, which elevation removes.

- Failed Kubernetes-auth MLflow deployments collect bounded CR status, migration
  Job/Pod status and logs, operator logs and warning events. A zero-replica
  Deployment can mean operator-managed database migration is blocked; do not
  force replicas or treat `Available=True` at zero replicas as successful setup.
  Diagnostic failures retain partial evidence and do not alter the failed step.

- Public Kubernetes-auth MLflow CI pins the server and operator by immutable
  multi-platform digests. Update both as a compatible pair with the deploy-action
  revision: the operator's `config/component_metadata.yaml` supported version
  must match the server's `mlflow/version.py`. The selected server is 3.15.2
  (source `ce140990e1c4951de597c33a16a9f0e05a4c9e9a`); the operator and action
  are `45bce547f680ad310d9253bf8e3f77d370de5328`, supporting 3.15.2. Registry
  provenance and supported platforms were checked before selection. Retain real
  migration/serving acceptance; a static pin guard does not prove live upgrades.
  The separate `red-hat-data-services` RHOAI image selection is unchanged.

- Docker mirror setup only modifies and restarts an active local systemd Docker
  service. Containerized/provider-managed runners without that service retain
  their daemon configuration; the helper still requires `docker info` to succeed.
  Configure daemon mirrors in the runner provider when local service management
  is unavailable. BuildKit mirror configuration remains independent.
- Frontend image builds use `build_frontend_image.py`: each of two attempts has an eight-minute deadline covering builder setup and build/export. A failed attempt records bounded BuildKit process/log diagnostics, removes its builder, and retries once with a fresh builder and no cache. Cancellation stops the build and cleans up without retrying. Only a successful nonempty export is renamed for artifact upload. The overall 30-minute job guard remains; other image builds retain their existing recovery path. This addresses installs/build finalization that stall without exiting, which bypass command-exit retries.

- The readiness policy-conformance lane prefetches Go modules from its pinned
  `policy-source` checkout before running tests. Downloads have three attempts,
  each capped at 120 seconds, with 5- and 10-second backoffs. Exhaustion fails
  the job; normal Go checksum verification remains enabled. Conformance runs
  once after a successful prefetch, so test failures are never retried.

- Multi-user CI pins Kubeflow Community Distribution at `5f0edd216742560ba205775e17b6b522cee53ae2`
  (Istio 1.31.2), whose manifests use `docker.io/istio` for control-plane,
  gateway, CNI and injected sidecar images. Apply the upstream manifests directly.
  When advancing this pin, review and render all four consumed paths, including
  the profile controller; the pin selects more than Istio images.

- The shared deploy action provisions GitHub CLI before image artifact polling. It reuses an installed CLI or downloads a checksum-pinned Linux AMD64/ARM64 release into runner temporary storage and exports its path for later steps. Missing CLI prerequisites fail immediately instead of consuming the artifact wait window.

- Adoption acceptance also exercises bounded server-side handoff waiting against
  a replacement API Pod during a real rollout. A fixture-only, bounded controller
  preStop hook holds a retiring writer alive; releasing it must let one pending
  HTTP create succeed without retry. A separate request exhausts the 30-second
  server wait, and a canceled request must not create SQL or CR state, including
  after release. Health and list requests remain responsive. This controlled
  overlap complements the ordinary populated 2.17.2 upgrade; it does not replace
  that upgrade or change production manifests. The fixture restores its original
  controller template and records results in `reports/handoff-wait.json`.

- `image-builds.yml` loads the built visualization-server and metadata-writer archives and runs their runtime smoke/helper tests in disposable containers before uploading them. It checks that pip and ensurepip are absent. Visualization runs on native AMD64 and ARM64 runners; the ARM64 build uses a separate artifact name. Tests do not run during production Docker builds or write test artifacts into published layers. `make test-backend-visualization` also exercises the dependency-light notebook smoke test.
- The legacy cache rolling-upgrade workflow runs helper-image OSV coverage in a separate job so a scanner outage does not skip functional upgrade validation. Keep its scanner version/checksum aligned with `osv-scanner.yml`; the CI-script test checks both. The helper uses the digest-identical Google-hosted Alpine mirror to avoid Docker Hub rate limits (#4099). Coverage derives the Alpine ecosystem from the inspected image and requires BusyBox and musl records.

- Cache rolling-upgrade acceptance pulls published baseline and fixture images directly through the Kind node CRI runtime with bounded retries. Only the locally built candidate uses `kind load docker-image`; exporting published multi-platform images through host Docker can leave missing index content during containerd import.

- The TensorBoard runtime is a separate KFP image built by the CI and release matrices. Its hash-locked standalone dependencies include S3 and GCS readers; do not replace it with a TensorFlow image without verifying that TensorBoard and both readers are present. `manifests/kustomize/hack/release.sh` stamps the UI/controller image defaults through `hack/update_tensorboard_image.py`. Its regression tests require both paths in the CI Scripts Tests filter.
- `tensorboard-acceptance.yml` builds the candidate runtime explicitly on AMD64 and ARM64. The direct AMD64 runtime lane reads event files through S3/GCS protocol emulators and tests scalar/image/graph navigation under the scoped proxy prefix in Chromium. This bypasses published default-image tags so presubmit exercises the candidate build. The separate deployed lane below verifies Kubernetes proxy authorization; neither lane establishes real cloud IAM integration.

- TensorBoard deployed acceptance uses a disposable Kind cluster with candidate UI/API/controller images and the same candidate viewer reference requested by the UI. It uses real KFP AuthService and Kubernetes SubjectAccessReview through localhost trusted identity headers; it does not test ingress authentication. The fixture verifies mounted AWS credential discovery, S3/GCS event reads, growing and rotated event-object reload without pod or container restarts, proxy browser rendering, denied/reader/manager permissions, deletion/recreation and explicit custom-image selection. GCS uses an anonymous emulator, so real cloud IAM remains outside this lane. No production images are published.

- Both TensorBoard Chromium acceptance lanes use Ubuntu 24.04, supported by their pinned Playwright 1.56.1. Keep these runner pins until the browser tooling is upgraded and validated on a newer Ubuntu image; `ubuntu-latest` can advance independently of Playwright support.

- SDK YAPF, isort and docformatter jobs retain Python 3.9 on Ubuntu 24.04. Do not move these lanes to `ubuntu-latest` without validating interpreter availability: setup-python provides no Python 3.9 build for Ubuntu 26.04.

## PostgreSQL image patch acceptance

`postgresql-image-upgrade.yml` is an isolated image-compatibility check for the
shipped PostgreSQL manifests. It does not enable PostgreSQL deployment in the
2.18 release matrices or resolve the MLMD concurrency limitation in #14353.
The path-filtered PR workflow and manual dispatch use Kind, render the candidate
manifests, and retain their PVC, PGDATA, service-account and security settings.
The only initial deployment override is the old same-major database image:
18.3 for the default database and 14.7 for the legacy metadata overlay, both
pinned by digest in `PROFILES`. For every later image bump, advance each affected
baseline to the previously shipped manifest image and digest so this lane tests
the immediately preceding supported image. Do not refresh baseline digests
independently of that transition.

Each lane retains the same PVC and database identity while replacing the image
with the digest-pinned candidate from the manifest. It checks records, JSONB
queries, indexes, a subsequent write and a further pod restart. The 18 lane calls
KFP's real database initialization/migration code and preserves an experiment;
the 14 lane uses the actual MLMD deployment and verifies a metadata artifact
through gRPC before and after the database replacement and restart. This checks
single-client persistence, not concurrent MLMD type creation. Both lanes upload
rendered manifests, a success receipt, pod logs and volume events. Command or
application failures cannot create a passing receipt, and pipeline `pipefail`
preserves errors when saving console output.

## JavaScript dependency security guards

`frontend/scripts/dependency-security.test.mjs` checks patched dependency ranges in
the frontend, UI server, mock backend and frontend integration-test lockfiles,
including nested copies. Keep these guards aligned when regenerating locks; they
supplement fresh advisory scans.

The integration-test runner overrides Mocha 10.8.2's `serialize-javascript` 6.x
requirement with 7.1.3 to address GHSA-m5mm-p8h7-m29v, which has no patched 6.x
release. Retain the override until Mocha resolves a patched version itself, and
verify runner compatibility when changing it.

## Third-party image upgrade checks

`third-party-image-upgrades.yml` reads baseline images from the PR base commit
(or the explicit manual baseline ref) and candidate images from the tested head.
The Kind component check preserves seeded S3 objects, admin access and lifecycle
rules on the same PVC across a SeaweedFS upgrade and restart. It issues reusable
profile IAM credentials on the candidate and verifies them after restart. It invokes
the actual profile webhook without Metacontroller; it does not run an MLMD,
whole-installation or Argo upgrade. Existing API/E2E lanes cover Argo behavior.
The helper phases retain command stderr and make cluster deletion best-effort.
Profile credential selection requires the named artifact Secret; service-account
token Secrets can precede it in reconciliation output.

The scan matrix also covers the four cert-manager images from each checkout's
`CERT_MANAGER_VERSION`. It preserves the downloaded upstream installation
manifest and derives the ACME solver image from the controller arguments.
TLS-enabled frontend tests on Kubernetes 1.33 and 1.36 and the API TLS lane
exercise the candidate certificate installation and KFP-issued certificates.
Changes to `backend/Makefile` trigger these frontend compatibility checks.
These fresh-install checks do not validate an in-place cert-manager minor-version
jump; existing clusters must follow upstream's supported upgrade sequence.

Linux AMD64 scan jobs use pinned upstream release OSV tooling
(the release branch does not contain that package) and retain raw JSON,
SARIF, immutable image identities and advisory-alias comparisons. Success means
complete evidence, not zero findings, applicability exemptions or release approval.

Pinned upstream OSV tooling accepts classic Docker config IDs and containerd
manifest/index IDs only when SHA-256-verified archived descriptors bind the
selected native config. Missing, tampered, ambiguous or mismatched chains fail.

- TensorBoard deployed acceptance provisions Python 3.10 before deployment, matching the frontend acceptance lane. The shared deployment readiness helper invokes `python` and `pip`; do not rely on runner-provided `python3` alone.

- Validate Generated Files also builds standalone v1beta1/v2beta1 Python client wheels on Python 3.9 and 3.10 and checks interpreter markers, dependency floors and local HTTP success/error handling at the minimum supported urllib3 versions. Template changes trigger this workflow.

## Release Python dependency resolution

Dependabot does not read the interpreter in uv's generated requirements header.
Use project-scoped `.python-version` files for dependencies that cannot resolve on
its newest interpreter. Full patch versions must exist in pyenv; Dependabot then
uses its available patch within that minor line. These files select dependency
and local development tooling, not the package's supported Python range.

The backend, metadata writer and visualization projects select Python 3.11,
matching their runtime images. SDK and Kubernetes platform development tooling
also select 3.11: the pinned docformatter 1.4 metadata build fails on Python 3.14.
The PostgreSQL retained-data harness selects 3.10 for frozen MLMD 1.14. SDK and
platform package metadata, universal requirements markers and CI retain Python
3.9 support. MLMD updates remain ignored; package pins are unchanged.

### Release updater audit

All 20 configured projects were checked with uv 0.13.5 against
`x86_64-manylinux_2_39`. The table records the resolver interpreter, not the
minimum supported product Python. Pass means the existing requirements resolve;
it does not guarantee every future proposed upgrade or hosted updater run passes.

| Configured project | Resolver Python | Result |
| --- | --- | --- |
| `.github/resources/postgresql-upgrade` | 3.10 (explicit) | Pass |
| `.github/resources/scripts/kfp-readiness` | 3.14 | Pass |
| `api/v2alpha1/python` | 3.14 | Pass |
| `backend` | 3.11 (explicit) | Pass, including immediate metadata-writer requirements separately |
| `backend/metadata_writer` | 3.11 (explicit) | Pass |
| `backend/src/apiserver/visualization` | 3.11 (explicit) | Pass |
| `backend/src/tensorboard` | 3.14 | Pass |
| `backend/src/v2/test` | 3.14 | Pass; local SDK path handled separately |
| `components/google-cloud` | 3.14 | Pass, setup dependencies |
| `docs` | 3.14 | Pass, including `sdk` child requirements |
| `docs/sdk` | 3.14 | Pass |
| `kubernetes_platform/python` | 3.11 (explicit) | Pass, normal and development requirements |
| `kubernetes_platform/python/docs` | 3.14 | Pass |
| `manifests/kustomize/base/installs/multi-user/pipelines-profile-controller` | 3.14 | Pass |
| `proxy` | 3.14 | Pass |
| `release` | 3.14 | Pass, pyproject dependencies |
| `samples/core/artifact_previews` | 3.14 | Pass, including the PyTorch index |
| `sdk/python` | 3.11 (explicit) | Pass, normal and development requirements |
| `test/kfp-kubernetes-native-migration-tests` | 3.14 | Pass |
| `tools/upgrade-readiness` | 3.14 | Pass |

Reproduce each requirements file independently (use `pyproject.toml` for the
release tool, and the table's interpreter):

```sh
uv tool run --from uv==0.13.5 uv pip compile path/to/requirements.txt \
  --python-version 3.11 --python-platform x86_64-manylinux_2_39 \
  -o /tmp/resolver-proof.txt
```

Include development `.txt` files and source `.in` files. Dependabot fetches
requirements in the configured directory and its immediate child directories,
then follows `-r`/`-c` includes; child interpreter files do not override the
parent project's selection. Do not combine independent lockfiles into one
resolution: they may intentionally pin different versions. Check local path
handling against Dependabot's preparer rather than requiring a full checkout
for every isolated project. Keep the hosted post-merge check separate from these
local resolver checks.
