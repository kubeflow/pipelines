# Feature gaps versus stock KFP

## Comparison boundary

The stock baseline is commit `7780f12acfbde9f16169c7fcbfdfb6c3ef5432ea`, immediately
before this branch's native implementation checkpoint (`2a1c1439a`). This compares
that checkout's features, not every historical KFP release. The inventory concerns
cluster execution, authoring compatibility and application features.

“Unavailable” means the current native implementation has no equivalent interface
or implementation. It does **not** mean Argo cannot provide that capability.
Some have an Argo/Kubernetes alternative, but are not drop-in KFP compatibility.
A separate section lists behavior not yet demonstrated end to end.

## Execution and management interfaces

| Stock capability | Current gap / alternative |
|---|---|
| KFP API server and its v2/v2beta1 REST/gRPC contracts | Removed. Browser management uses Kubernetes and Argo Server directly. Existing integrations calling KFP endpoints must change. |
| `kfp.Client`, generated `kfp.server_api`, registry/Kubeflow backend clients, server-management CLI groups | Removed. No KFP `run`, `pipeline`, `experiment` or `recurring-run` management commands. Use Argo/Kubernetes clients; the CLI retains compile, image-build and preview commands. |
| PipelineSpec execution IR and KFP Pipeline/PipelineVersion CRD output | No pipeline IR compiler or KFP CRDs. Compile original Python to WorkflowTemplate/Workflow/CronWorkflow. Portable **container definitions** still use shared schema messages, without an execution DAG builder. |
| Loading compiled KFP pipeline graphs for reuse or migration | Not supported. Historical graph IR and old backend runs are not imported. Existing container definitions remain loadable. |
| KFP pipeline/version/experiment/run ID conventions and server-side CRUD contracts | Replaced by resource names/labels and Workflow UIDs. The native UI creates content-named versions; it does not expose all old pipeline/version editing and sharing operations. |
| ZIP/tar pipeline upload and legacy upload forms | Native UI accepts WorkflowTemplate YAML/JSON text, not KFP package archives or old upload endpoints. |
| KFP ScheduledWorkflow controller and recurring-run API | Removed. Recurrence exists through native CronWorkflows, not the old controller/model. The UI exposes cron expressions and suspension, not all old periodic-interval, start/end, catch-up or concurrency controls. Operators can configure native CronWorkflow fields separately. |
| Manual KFP archive/unarchive and experiment archive workflows | Not ported. Argo archives completed workflows; deleting the experiment ConfigMap does not delete runs. The old archive state/CRUD interfaces are not emulated. |
| Full legacy dashboard actions, routes and bookmarks | Replaced by native versions/experiments/runs/schedules pages. Legacy detail URLs, clone-run flows, separate execution pages and the old feature/settings pages are not maintained. Argo CLI/server capabilities are available independently of UI buttons. |
| KFP backend plugin inputs/outputs | No `plugins_input` dispatch or KFP plugin lifecycle. Component code/native Argo facilities must provide any replacement. |
| Bundled legacy component/sample catalogue | The old `components/` catalogue and backend-oriented samples were removed. Authoring custom Python/container components remains supported; vendor packages and old catalogue entries are not shipped or qualified here. |
| KFP release/upgrade tooling, service image builds and migration jobs | Retired with the backend. Native CI builds/tests the SDK, UI and an isolated Argo stack. It does not migrate or operate a stock KFP installation. |

## Authoring and compiler limits

The DSL still captures many constructs so it can type-check source and report a
useful compiler error. Presence of an annotation or helper does not imply native
execution support.

| Stock capability | Current native limitation |
|---|---|
| Root artifact inputs and importer tasks | Rejected by the native compiler. Use a produced artifact in a supported graph or explicit native Argo artifact configuration outside this compiler. |
| Artifact-list fan-in / `dsl.Collected` artifact lists | Rejected. Scalar collection is implemented and tested; multi-artifact staging/aggregation is not. |
| Conditional output merges / `dsl.OneOf` execution lowering | Not implemented. Branches execute, but outputs cannot escape them through a conditional merge. Keep consumers inside the branch or restructure the pipeline. |
| Every nested collection/loop/output pattern | Only the supported scope/collection forms are implemented. Some nested fan-in and grouped artifact-output cases fail explicitly. This is not complete control-flow compatibility. |
| `.ignore_upstream_failure()` continuation | Not lowered. Failure-aware exit handlers are supported, but are not a universal substitute for this task setting. |
| Workspace provisioning, workspace placeholders, and KFP `CreatePVC`/`DeletePVC` intrinsics | Not implemented. The fake `argostub` component hooks were removed with their backend compiler. Existing PVC mounts are supported; create/manage PVCs explicitly with Kubernetes/Argo. |
| Dynamic images, resource quantities, PVC names and some dynamic Kubernetes settings | Rejected instead of leaving unresolved channels in manifests. Use fixed values for supported settings. |
| All `kfp-kubernetes` options | Native lowering covers literal resources/retries, node selectors, pod metadata, tolerations, image pull policy, secret/config-map environment values, existing PVC mounts and timeout. Other configuration keys, including several affinity, volume, init-container, security-context and resource-claim/DRA options, are not mapped. |
| Automatic TaskConfig/configuration passthrough to external workloads | Rejected. The annotation remains a component-definition type, not a driver-populated runtime context. Pass explicit inputs/configuration instead. |
| Retry/custom-cache/platform overrides on nested DAG tasks | Some are rejected; configure supported inner tasks instead. |
| Arbitrary container scalar output types | Non-Python container outputs currently require `String`. Use the Python bridge for validated numeric, boolean, list and dict returns. Typed artifact payloads are supported. |
| Fractional retry backoff factors | Argo's integer factor is enforced. Integral values such as `2.0` normalize to `2`; fractional values fail compilation. This differs from the stock SDK's float-valued API/IR field. |
| Disabling compiler type checks | Not supported by the native compiler. Type checking stays enabled. |
| KFP launcher debug-pause barriers and UI resume protocol | Removed with the launcher. Native Argo suspend/resume is supported, but does not reproduce before/after/on-error in-container debug barriers. |
| Stock launcher/runtime image compatibility | Python component images need this checkout's runtime bridge (or its wheel). An unmodified upstream SDK runtime is not a drop-in executor for the new contract. |

## Storage, caching and ML views

| Stock capability | Current gap / alternative |
|---|---|
| Every stock artifact-store/pipeline-root scheme | The compiler currently requires an S3-compatible `artifact_repository`. Other Argo artifact drivers are not exposed through this compiler. |
| KFP cache store/API semantics | Replaced by native ConfigMap memoization. There is no KFP cache service or cache API, and SQL memoization is not implemented here. |
| Caching mutable image tags, runtime-installed dependencies, secret/config-map/PVC-backed state without explicit version identity | The compiler conservatively rejects these cache configurations. Pin/snapshot the image and use versioned declared inputs, bake dependencies into it, or disable caching. |
| Scalable global artifact discovery/indexing and cross-run metadata queries | No indexing service or global metadata database. Discovery is bounded to live/archive workflows; producer/input references provide within-run lineage and cached origin identity. |
| Full artifact comparison and visualization dashboard | Only selected-run scalar metrics, typed artifact metadata/downloads and input lineage are exposed. ROC/confusion-matrix plots, embedded HTML/Markdown/table viewers, richer artifact comparison and other stock visualization panels are not ported. |
| Automatic Dataset/Model previews in the native execution path | The bridge currently disables automatic preview generation. Existing metadata and payloads are preserved; stock automatic preview production/viewing is not reproduced. |
| Managed TensorBoard viewers | Viewer controller, TensorBoard proxy and associated UI are removed. Run TensorBoard independently if required. |
| Namespace export/import and historical metadata transfer | Removed. There is no migration/export service for stock KFP runs, artifacts or experiments. Native resource/SQL/object-store backup is an operator responsibility. |
| Full list/search/pagination/comparison UX | Native discovery is capped at 100 results per live/archive source, and comparison at five selected runs. No stock global search or large-selection comparison equivalent is provided. |
| Full legacy logging UX | The current UI fetches the last 200 lines per task pod and offers refresh. Argo archive logs remain readable, but the old logging/proxy interfaces and all their options are not reproduced. |

## Authentication and operations

- The provided deployment is namespace-scoped development infrastructure, not the
  stock Kubeflow multi-user/profile/central-dashboard integration. It uses manual
  in-memory service-account tokens, not a complete SSO/session experience.
- Namespace RBAC and denied Kubernetes/Argo actions are tested. This does not
  establish production tenancy, network isolation, storage-credential isolation,
  OpenShift support or every stock installation mode.
- The old persistence agent, KFP run/metadata database, retention/backfill logic,
  custom controllers, backend health endpoints and backend-specific telemetry are
  gone. Argo owns workflow status, archive/offloading, TTL and scheduling instead;
  its operational settings and semantics are not the KFP contracts.
- Stock release/merge-queue workflows were intentionally retired with user approval.
  Security scanning and basic issue/contributor automation remain. External branch
  protection or repository settings are not changed by this commit.

## Not yet established, rather than declared unavailable

- Complete behavior parity for every cancellation/error-code field of
  `PipelineTaskFinalStatus`; success and failure are verified.
- Every notebook, custom artifact class, optional-input combination and third-party
  component package, even where its source compiles successfully.
- Large production DAG throughput, global lineage scale, HA deployments, storage
  retention beyond the demonstrated cache lifetime, and all Kubernetes/Argo versions.
- Every native Argo feature through the KFP UI. Native operators can use Argo
  resources/APIs directly without adding a KFP backend.

## Evidence and source anchors

Confirmed native behavior is exercised by `argo_compiler_test.py`,
`native_cli_test.py`, the retained SDK/widget tests, `verify_compatibility.py` and
`argo-deployment.smoke.mjs`. Compiler rejection paths are in `argo_compiler.py`
and `argo_groups.py`; UI capabilities are in `ArgoWorkspace.tsx`,
`ArgoRunDetails.tsx`, `Argo.ts` and `ArgoResources.ts`.

Stock capabilities were checked against the baseline paths below. They are now
removed where obsolete; inspect them with `git show BASE:path` using the commit
above:

- `sdk/python/kfp/client/client.py` and `sdk/python/kfp/cli/cli.py`
- `sdk/python/kfp/compiler/compiler.py` and `pipeline_spec_builder.py`
- `sdk/python/kfp/dsl/pipeline_task.py` and `pipeline_config.py`
- `frontend/src/components/Router.tsx` and `frontend/src/components/viewers/`
- `backend/src/apiserver/`, `backend/src/crd/controller/`, `backend/src/v2/`
- `manifests/`, `frontend/server/`, `sdk/python/kfp/registry/`

The retained component schema is not a hidden execution backend. The cleanup test
asserts that retired services, controllers, RPC clients and their build roots stay
absent.
