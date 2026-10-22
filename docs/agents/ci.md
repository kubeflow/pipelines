# CI and Workflows

GitHub Actions workflows are in `.github/workflows/`; reusable composite actions are in `.github/actions/`.

- Update this guide when changing workflows, CI matrices, commands, generated outputs, or common failure handling.
- `presubmit-backend.yml` exercises the self-service namespace transfer engine against disposable MySQL and MLMD services, using `KFP_TRANSFER_MYSQL_DSN` and `KFP_TRANSFER_MLMD_ADDRESS`. `KFP_TRANSFER_REQUIRE_INTEGRATION=1` prevents silent skips. SQL tests create/drop isolated databases; MLMD fixtures use unique resource names. The MLMD image matches the release deployment manifest. HTTP/authorization and frontend tests cover the end-user transfer boundary; no Python operator tool is required.
- `presubmit-backend.yml` runs recurring-run transaction concurrency tests against disposable MySQL 8.0 and PostgreSQL 16 services using `KFP_RECURRING_MYSQL_TEST_DSN` and `KFP_RECURRING_POSTGRES_TEST_DSN`. Tests allocate isolated databases/schemas and independent connections; the configured test roles need permission to create and drop their test database/schema and observe session lock metadata. These checks cover storage transaction behavior, not a live KFP upgrade.
- The KFP SDK and Kubernetes SDK plugin require Python 3.10+ so Requests and urllib3 security fixes can be installed. SDK and integration lanes exercise that minimum; the syntax-only lane retains Python 3.9 grammar checks.
- CI covers supported Kubernetes and Argo versions, database and Kubernetes pipeline stores, proxy/cache variants, GPU scheduling, and DRA (Dynamic Resource Allocation). Preserve the relevant matrix coverage when changing a lane. Release-2.18 retains the Argo 3.7.x compatibility lane alongside the default 4.1.x runtime; exact patches come from `third_party/argo/COMPATIBILITY_VERSION` and `VERSION`. The master branch's 4.0.x floor must not be backported to this release branch.
- The `dra-check` E2E lane runs on a Kind cluster with Kubernetes 1.34 (DRA GA minimum). It generates both Python and Go pipeline-spec bindings, then uses `kfp component build` with the checked-out SDK, Kubernetes plugin, and generated pipeline-spec packages to create the `kfp-dra-sdk:test` component image, then loads that image into Kind. `.github/actions/install-dra-driver` installs dra-example-driver via Helm, runs a smoke test that verifies a pod can schedule with a DRA ResourceClaim, then the E2E suite uploads DRA pipelines and validates pod-level `resourceClaimStatuses`. The driver version is pinned in `action.yml`; v0.2.1 targets K8s 1.34, while v0.4.0+ requires K8s 1.36.
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
- `docs.yml` builds the Sphinx documentation site with warnings treated as errors and runs a non-blocking external link check; Read the Docs (`.readthedocs.yml`) handles publishing, not this workflow. Before installing `docs/requirements.txt` through `setup-python-pip-cache` under the `docs` cache scope, both jobs generate the Python API protobuf bindings and install the source-built `kfp-pipeline-spec` package through `.github/actions/protobuf`.
- `osv-scanner.yml` runs on every push to master, weekly, and on manual dispatch. It scans supported dependency manifests and lockfiles and separately scans the container images rendered by the standard standalone and multi-user Kustomize overlays, removing structurally identical SARIF results before uploading both result sets to code scanning. Compiled Python requirements are scanned without dependency re-resolution; generated API-client constraint directories are excluded because they specify compatibility minimums rather than installed versions. Dependabot remains responsible for remediation PRs.
- `ai-analyzer.md` is the source for the AI issue quality analyzer; `ai-analyzer.lock.yml` is generated by GitHub Agentic Workflows. The analyzer runs when an issue is opened. Invalid-title runs post guidance and attempt to self-cancel so a correction stays quota-free. A title edit retries analysis only while the current title is valid and no prior quality verdict exists; other title edits also attempt to self-cancel before inference, and successful cancellations do not count toward the user rate limit. Body-only edits use isolated concurrency groups and skip without provisioning a runner, while analyzed per-issue runs use a full queue so later edits cannot evict a pending title correction. After editing the source, run `gh aw compile .github/workflows/ai-analyzer.md` and commit both files.
- Workflows that deploy CI-built images overlap Kind setup with current-branch image builds. The shared deploy action waits for the complete artifact inventory; its normal window extends while a missing artifact's producer is queued or running, and completed producers receive a short publication grace.
- CI Docker-sensitive paths use retry wrappers for image builds, BuildKit bootstrap, and runtime-image pulls. GitHub-hosted Docker jobs configure `mirror.gcr.io` as a Docker Hub pull-through cache, docker-container builders use the same mirror through `buildkitd.toml`, and uncached images still fall back to Docker Hub. Build jobs explicitly try and locally tag the mirrored BuildKit bootstrap image before falling back to Docker Hub within the shared retry budget.
- `runtime-base-images.yml` refreshes the shared runtime-image artifact. `image-builds.yml` prefers a matching current-source or trusted-master artifact, but rebuilds locally after a bounded lookup miss or failed download. Both workflows use `build-runtime-base-images.sh` to archive test-task, MySQL, and supported Argo images plus the Modelcar fixture. This keeps release branches and older PR snapshots working after shared artifacts expire without permanent release-specific producer lanes. Generation failures stop the job before upload; opted-in E2E lanes still require the Modelcar archive during deployment.
- For workflow-only changes, verify referenced working directories, Docker contexts/files, scripts, and local action paths exist.
- The KFP Manifests suite also checks the rendered profile-controller NetworkPolicy for the base multi-user install and the plain, platform-agnostic, and PostgreSQL multi-user overlays. These selector/namespace/port assertions run through `manifests/kustomize/hack/test.sh`; they do not verify live CNI enforcement or override additive allowances from other policies.
- Frontend CI also runs `frontend/scripts/check-spec-generation.sh` after installing `protoc`: pipeline and Kubernetes platform specs are generated into temporary directories and typechecked. Changes to either source proto directory trigger this workflow.
- The frontend protobuf installation scopes both APT commands to `/etc/apt/sources.list.d/ubuntu.sources`, excluding unrelated runner repositories such as Chrome without changing their configuration or bypassing package verification. If the runner's source layout changes, update this explicit path; the step checks that it exists before running APT. Browser integration tests continue to use their separate Selenium container.

- The shared runtime-image inventory pins acquisition of the Python 3.12 image used by the cache and proxy fixtures to a Docker Hub digest. Archive creation and direct Kind loading restore its runtime tag before save/load; keep fixture image references and source declarations aligned with that tag. CI script tests check this contract whenever those fixtures change.

- Frontend Vitest snapshots preserve rendered whitespace. The trailing-whitespace hook excludes `frontend/**/*.snap`; update snapshots through Vitest so formatting hooks do not alter expected UI text.
- `manifests/kustomize/hack/test.sh` runs `tensorboard_signing_key_test.py` to verify the shared Secret, least-privilege initializer permissions, migration-safe `Recreate` and documented opt-in `RollingUpdate`, generated Job names, and the actual CI and OpenShift overlays. The suite renders the exact rolling-update patch from `docs/operator-guides/server-config.md`, so changes to that guide also trigger `kubeflow-pipelines-manifests.yml`. The suite requires Python 3, `kustomize`, and mikefarah `yq` 3.4.1 (using its v3 command syntax); `manifests/kustomize/hack/presubmit.sh` already installs the pinned `kustomize` and `yq` versions. Keep coverage for CI's preloaded initializer image and OpenShift's namespace-assigned UID policy when changing these manifests.

## Common CI failures

- SDK imports are checked by isort 5.10.1 in both lanes: `sdk/python/requirements-dev.txt` for the SDK presubmit, and the pre-commit hook's `additional_dependencies`. Keep the two pinned together; a newer pre-commit hook wraps long imports differently and conflicts with the SDK formatter.
- Keep docformatter on v1.7.7 until its v1.7.8 tokenization regression is fixed: v1.7.8 crashes on explicit continuations and rewrites SDK blank lines in conflict with YAPF. Verify formatter upgrades by running the full hook chain twice on the updater and SDK structures files.
- Registry pull failures for Kind, BuildKit, Python, or Alpine images are usually transient; retry before changing code.
- A Kind checksum mismatch after cache restore means no tests or deployment ran; retry the job.
- SeaweedFS `PutObject` timeouts are artifact-store instability; retry rather than weakening assertions or increasing pipeline timeouts.
- For proxy failures, inspect the `tinyproxy` namespace pods, events, services, endpoints, and endpoint slices.

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
  failures remain failures and the disposable cluster is always deleted.
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

- Multi-user CI pins Kubeflow Community Distribution at `c5e3896566d52e02090f15b476f4528605a9c35f`
  (Istio 1.31.1), whose manifests use `docker.io/istio` for control-plane,
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
