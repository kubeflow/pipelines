### Direction

Keep **KFP’s Python authoring experience and UI**, but make Argo the execution model and source of truth.

The prototype should answer:

> Can an existing KFP pipeline compile into ordinary Argo resources, run without any KFP backend, and still look like an ML pipeline in the KFP UI?

That means preserving useful compatibility—not reproducing KFP’s internal architecture behind different endpoints.

```text
KFP Python DSL
      │
      ▼
Argo compiler
      │
      ├── WorkflowTemplate    reusable pipeline version
      ├── Workflow            individual run
      └── CronWorkflow        recurring run
                 │
                 ▼
        Argo workflow-controller
                 │
                 ▼
        Component containers
        + ordinary Argo executor


Adapted KFP UI
      ├── Kubernetes API      resources and metadata
      └── Argo Server         hydrated workflows, archive,
                              logs, artifacts, run operations
```

No KFP API server, ScheduledWorkflow controller, driver pods, or separate run database. Argo can still use SQL for its archive and supported caching configuration.

### 1. Set the compatibility contract

Separate three promises:

| Area | Prototype goal |
|---|---|
| Pipeline Python source | Existing decorators, components, typed artifacts, and task configuration work wherever their semantics can be preserved. |
| Compiled output | Intentionally changes: native Argo YAML rather than KFP IR. |
| KFP client/server interfaces | Not a compatibility goal initially. Use Kubernetes and Argo clients for submission and management. |

Keep the familiar entrypoint:

```python
compiler.Compiler().compile(
    pipeline_func=my_pipeline,
    package_path="pipeline.yaml",
)
```

Make the normal reusable output a `WorkflowTemplate`, with explicit ways to emit a standalone `Workflow` or `CronWorkflow`.

Unsupported constructs must produce actionable compile errors. Silently dropping a task setting would make “backwards compatible” misleading.

#### Important compiler detail

In [current KFP source](https://github.com/kubeflow/pipelines/blob/master/sdk/python/kfp/dsl/graph_component.py), `@dsl.pipeline` builds KFP IR before `Compiler.compile()` is called. Replacing the YAML writer alone won’t remove that dependency.

The intended path should be:

```text
Existing DSL → captured tasks, groups, and channels → Argo templates
```

Preserve the frontend’s type checking and graph validation, moving IR-dependent validation where necessary. Don’t make “KFP IR embedded inside an Argo manifest” the new architecture.

### 2. Map KFP concepts to native resources

| KFP concept | Proposed representation |
|---|---|
| Pipeline | Logical grouping of `WorkflowTemplate` versions using labels |
| Pipeline version | Content-addressed `WorkflowTemplate`, treated as immutable |
| Run | `Workflow`; Kubernetes UID is the run ID |
| Recurring run | `CronWorkflow`, including native suspension |
| Experiment membership | Label on workflows and schedules |
| Experiment name/description | Annotations; optionally a small ConfigMap for an independently existing experiment |
| Component | Argo container template |
| Nested pipeline | Argo DAG template |
| Scalar inputs/outputs | Argo parameters, with type-aware serialization |
| Dataset/model payload | Argo artifact |
| Artifact type and metadata | Versioned metadata convention understood by SDK and UI |
| Cache | Argo memoization |
| Run history | Argo workflow archive |
| Execution status | Argo workflow/node status—not a second status model |

Use labels for filterable identifiers and annotations for display names and descriptions.

For experiments, use a small ConfigMap if retaining KFP’s “create an empty experiment” flow matters. It requires neither a new CRD nor a controller. Otherwise, derive experiments from run labels and explicitly accept that empty experiments don’t exist.

Pipeline-version immutability is a convention unless enforced. Don’t imply Kubernetes makes `WorkflowTemplate` immutable.

### 3. Build one convincing vertical slice

Use a real, small KFP pipeline:

```text
prepare Dataset → train Model → evaluate Metrics
```

Run it twice with different training parameters.

The demonstration should show:

1. Existing KFP-style Python compiles to inspectable Argo YAML.
2. The pipeline runs with the KFP backend entirely absent.
3. Only expected task pods appear—no KFP driver pods.
4. The adapted KFP UI shows the experiment, runs, graph, logs, and typed artifacts.
5. The comparison view displays both runs’ evaluation metrics.
6. The second run reuses preparation through Argo memoization.
7. A recurring run is visibly an ordinary `CronWorkflow`.
8. `kubectl` or Argo CLI changes appear in the KFP UI without synchronization machinery.

That last check is important: it demonstrates that this is genuinely one resource model.

### 4. Implement in five stages

#### Stage A — Prove the runtime contract before writing the compiler

Hand-author the Argo workflow for the three existing Python components.

The main compatibility challenge is the component runtime. Existing functions expect things such as:

- `Input[Dataset]` and `Output[Model]`;
- `.path`, `.uri`, and `.metadata`;
- Python return values;
- `Metrics.log_metric()`.

Reuse KFP’s Python execution machinery where practical, with a small in-container bridge:

- construct typed arguments from Argo-staged inputs;
- map artifact objects to local paths and actual storage identities;
- invoke the original function;
- write scalar outputs to files Argo captures;
- write artifact metadata and metrics to declared output files.

Argo owns artifact transfer and output reporting. The bridge does not create workflows, reconcile tasks, or contact a KFP service. It runs inside the component container, not in another pod.

**Exit criterion:** unchanged component functions execute correctly from handwritten Argo YAML.

#### Stage B — Compile the existing DSL directly to that contract

Start with:

- Python and container components;
- parameters and typed artifacts;
- task dependencies;
- nested DAGs;
- resources, retries, and common `kfp-kubernetes` settings;
- caching enabled/disabled.

Strip local execution from the KFP fork: remove its SDK entrypoints, Docker/subprocess runners, local-only dependencies, tests, examples, and documentation. Remove local-execution branches from shared DSL code while preserving cluster execution. Do not retain a compatibility shim or fallback to local execution.

Then cover control flow against representative existing pipelines:

- `dsl.If` / `Else`;
- `dsl.ParallelFor`;
- collected outputs;
- exit handlers and task-final-status inputs.

These need semantic checks, not just syntax translation. Skipped branches, failed loop iterations, and exit-handler inputs are easy places to change behavior accidentally.

**Exit criterion:** a small compatibility corpus compiles, validates, and runs with the expected outputs and failure behavior.

#### Stage C — Reconnect the KFP UI

Keep the existing pages and visualization widgets. Replace their data access and IR-specific interpretation.

KFP’s [`frontend/src/lib/Apis.ts`](https://github.com/kubeflow/pipelines/blob/master/frontend/src/lib/Apis.ts) is a useful starting point, but graphs and artifact views also need attention.

Implement browser-side adapters for:

- pipelines and versions → `WorkflowTemplate`;
- runs → live and archived workflows;
- recurring runs → `CronWorkflow`;
- experiments → labels/configuration;
- graph → workflow templates plus node status;
- artifacts and metrics → declared Argo outputs.

Do not recreate KFP HTTP endpoints just to leave every frontend call unchanged.

Prefer Argo Server for workflow details because large workflow status may be compressed or offloaded. A raw Kubernetes read is not always sufficient. Deduplicate live/archive results by workflow UID.

For browser access, use an authenticated same-origin ingress/proxy where needed. No embedded cluster-admin token, no SQL access, and no new application backend disguised as a proxy.

**Exit criterion:** the retained KFP pages operate with no KFP backend requests.

#### Stage D — Prove the ML features

Define one small, versioned metadata format for each task’s outputs:

- artifact type and logical port;
- producing run/task identity;
- storage identity and input references;
- user metadata;
- scalar metrics and pointers to visualization data.

Store larger metadata and visualization payloads as artifacts, not giant workflow annotations. Preserve small metadata alongside outputs where downstream tasks need it.

For the prototype:

- derive within-run lineage from graph connections;
- retain producer identity when cached artifacts are reused;
- compare explicitly selected runs;
- limit artifact discovery to a bounded set of live/archived workflows.

**Exit criterion:** lineage and comparison work without ML Metadata or a custom indexing service.

This proves the UX, not scalable global artifact search. That is a separate design question.

#### Stage E — Prove the simplification and Argo access

Add one native capability beyond the original pipeline: for example, an approval gate before publishing the model.

Also verify:

- recurring runs survive UI restarts;
- archived runs remain viewable after live Workflow deletion;
- denied Kubernetes/Argo actions remain denied in the UI;
- deleting an experiment does not unexpectedly delete runs;
- cached results still reference available artifacts.

**Exit criterion:** one source of truth, fewer deployed services, and a demonstrated Argo capability—not just a different compiler output.

### 5. Treat caching as a correctness problem

Keep KFP’s boolean caching interface, but generate the Argo policy underneath it.

A cache key should account for:

- component implementation and image identity;
- relevant execution configuration;
- resolved parameter values;
- input artifact identities or versions.

Runtime parameter values must participate when Argo executes the template; they aren’t all known at compile time.

A mutable URI is not a content identity. Either require an explicit version, disable caching for that input, or document the limitation. Likewise, cached metadata must retain the original artifact producer rather than inventing new lineage.

Start with ConfigMap-backed memoization to prove the compiler contract. Validate the SQL-backed work separately rather than making the entire prototype depend on it. Cache retention and artifact retention also need to agree.

### Non-goals

- **Local execution.** Do not support or preserve KFP’s Docker/subprocess execution paths. Executing this plan includes stripping local execution from the KFP fork, not merely hiding it in the UI or excluding it from testing. Revisit local execution later with a more Argo-native design; no replacement is part of this prototype. Offline compilation and validation of Argo manifests remain in scope.
- Full KFP server/client compatibility.
- Migration of historical KFP runs.
- Global artifact indexing and metadata queries.
- Every cloud-specific SDK extension.
- New CRDs or controllers unless the prototype demonstrates a concrete need.

### Recommended first milestone

**Three existing Python components, one handwritten Argo workflow, and one adapted KFP run-details page.**

That tests the two hardest assumptions immediately: whether KFP components can run without its backend, and whether the KFP UI can explain an Argo run without reconstructing a separate KFP run model.

### Validation status

### Backend cleanup checkpoint — 2026-11-19

The user approved two commits: implementation checkpoint `2a1c1439a`, followed by
an isolated cleanup commit. The cleanup removes the KFP backend/API/controllers,
legacy clients/UI/server, Go stack, old manifests, catalog/sample/integration code,
and backend/release/merge-queue build machinery. Native-only CI replaces it;
security scanning and basic contributor/issue automation remain.

- The retained SDK/cleanup/deployment/governance tests pass on Python 3.11 and 3.13:
  981 tests. All 161 retained UI tests and both production/offline browser tests pass.
- Schema regeneration, wheel/sdist boundary checks, docs build and actionlint pass.
- A fresh disposable Kind deployment passed all nine compatibility cases and the
  full authenticated browser acceptance test. Its temporary kubeconfig leaves the
  user's global Kubernetes context unchanged.
- The cleaned UI/runtime are also redeployed to native OrbStack, namespace `kfp-argo`.
  Only five deployments remain: Argo Server/controller, static UI, PostgreSQL and
  object storage. The unused httpbin deployment/service/credential was removed.
  Existing runs, archives and persistent artifact data are retained.
- Current deployed UI image: `kfp-argo-ui:cleaned`; version
  `argo-ml-demo-0c0f13da152a`; component snapshot
  `kfp-argo-demo:kfp-894a37d992ea877832582479e0819bb5de22de92de450849f2c6c088ec01dc66`.
  OrbStack browser acceptance passed after redeployment. The URL remains
  `http://127.0.0.1:3003/argo.html?namespace=kfp-argo`.
- `docs/compatibility.md` compares unavailable features with stock baseline
  `7780f12acfbde9f16169c7fcbfdfb6c3ef5432ea`, excluding the execution mode the user
  did not want listed. It separates known gaps from unverified compatibility.
- The two shared authoring/Kubernetes schema bindings and a small **container
  definition** serializer remain intentionally. There is no pipeline IR builder,
  KFP REST client or hidden backend behind them.
- Validation logs: `/tmp/kfp-cleanup-python{311,313}.log`,
  `/tmp/kfp-cleanup-sdk-full.log`, `/tmp/kfp-cleanup-browser.log`,
  `/tmp/kfp-cleanup-docs.log`, `/tmp/kfp-cleanup-isolated-deployment.log`, and
  `/tmp/kfp-cleanup-orbstack.log`. The final response reports exact Git LOC deltas
  for the cleanup and the full branch, with generated/lockfile lines separated.

### Earlier deployment checkpoint (superseded) — 2026-11-18

The verified prototype scope is deployed and passing on native OrbStack Kubernetes.
This is not a claim of complete upstream SDK/client compatibility.

- Target: context `orbstack`, namespace `kfp-argo`, Kubernetes v1.36.5+orb1.
  Argo v4.1.6, PostgreSQL archive/offloaded status, persistent artifact storage,
  and static KFP UI/proxy are ready. No KFP backend/driver/metadata services.
- UI: `http://127.0.0.1:3003/argo.html?namespace=kfp-argo`. Login with a token from
  `kubectl --context orbstack -n kfp-argo create token argo-ui-editor --duration=2h`.
  Tokens stay in browser memory. `argo-ui-reader` demonstrates denied actions.
- Deployed UI image: `kfp-argo-ui:final`. Final reusable version:
  `argo-ml-demo-024a796656a9`. Its component image is
  `kfp-argo-demo:kfp-179ecf7abc219cc2e4e56ee30be805753dd7087d0f4324175be2670bfca18ebd`.
  Content-named image snapshots avoid following later `:dev` rebuilds.
- Direct native compiler and CLI, local-execution removal, native resource pages,
  offline previews, typed artifact/lineage metadata, comparison, archive access,
  recurring resources and approval/resume are implemented in the tested scope.
  Full metadata travels as artifacts; only small identities enter parameters.
- 352 focused Python tests pass on both Python 3.11 and 3.13. 57 focused frontend
  tests pass. Frontend build (ESLint/TypeScript) and seven existing browser smokes
  pass, including native offline/watch/sandbox graph and inspector behavior.
- Nine current-image compatibility cases pass on OrbStack: nested DAGs, both
  branches, loops/collection, failed iterations, successful/failing bodies and
  cleanup, nested native-container artifact transfer, 100 KB metadata, and actual
  retry attempts with resource requests/limits. See `verify_compatibility.py`.
- The deployed browser test passed after fixing and retrying surfaced problems.
  It covers UI version upload, empty experiments and filtered runs, submissions,
  actual image IDs, 3 then 2 task pods, cache producer identity, both scores,
  authenticated artifact downloads/logs, offloaded status, native approval,
  denied Kubernetes/Argo actions, schedule persistence and CLI changes, archive
  access after live deletion, non-cascading experiment deletion, and SQL/artifact
  payload survival after storage deployment restarts.
- Latest browser evidence: version `verification-a6d1058cd70c`, runs
  `verification-a6d1058cd70c-866sh` (UID `5cb7629e-1d2d-4bba-9931-e0f30bffa474`,
  archived after live deletion) and `verification-a6d1058cd70c-tlbv9` (UID
  `618492d6-4bee-462b-9108-d570e16eee8d`), schedule
  `schedule-73a3415e-a6cf-4a04-998f-033ebbba2170`, approval run
  `native-approval-jjvrc`. The test deletes its experiment but retains runs/schedule.
- Reproduction, architecture diagram, limitations and authentication instructions:
  `samples/argo-native/README.md`. Logs/evidence:
  `/tmp/kfp-argo-deployment-evidence.json`, `/tmp/kfp-argo-deployed-e2e.log`,
  `/tmp/kfp-argo-final-corpus.log`, `/tmp/kfp-native-focused-final-{311,313}.log`,
  `/tmp/kfp-native-browser-regressions.log`, and
  `/tmp/kfp-argo-{deployed-runs,denied,archived}.png`.
- Port-forward log/PID: `/tmp/kfp-argo-deployed-ui.log` and
  `/tmp/kfp-argo-deployed-ui.pid`. Restart after UI pod replacement. The native
  offline preview watcher is at `.preview/argo-native.html`, with log/PID under
  `/tmp/kfp-argo-preview.*`. Herdr's UI socket was unavailable, so no tab opened.
- Broad legacy SDK runs were attempted and are not green because their IR/client
  contracts changed. Those failures are documented, not hidden or reported as
  successful compatibility coverage. Remaining unsupported constructs and scale/
  production-auth limitations are explicit in the sample README.
- Changes remain uncommitted. Do not discard them or restart from the old Stage A
  setup. If continuing, start from the final deployed scope and its current tests.

### Earlier execution checkpoint (superseded) — 2026-11-18

- Stage A is verified: `sdk/python/kfp/dsl/argo_runtime.py` runs the unchanged
  functions in `samples/argo-native/components.py` using `handwritten.yaml`.
- Disposable cluster: `kind-kfp-argo`, namespace `argo`. No KFP backend is
  installed. The demo image is `kfp-argo-demo:dev`, loaded into kind.
- Verified runs: `argo-ml-demo-4m8pj` (factor 1.0, score 2/3, three task pods)
  and `argo-ml-demo-hjtp8` (factor 1.2, score 1.0, two task pods). Preparation
  was memoized, including its original producer metadata and storage URI.
- Repeatable verification:
  `uv run python samples/argo-native/verify.py argo-ml-demo-4m8pj argo-ml-demo-hjtp8`
  with `kubectl --context kind-kfp-argo -n argo port-forward svc/argo-server 2746:2746`.
- 80 focused Python tests pass (runtime adapter, existing executor, demo installer).
  `argo lint --offline samples/argo-native/handwritten.yaml` passes.
- Fixed interrupted setup: retired MinIO image replaced with the pinned Silo
  image used by current upstream Argo; controller granted update permission only
  on the pre-created demo cache. Added regression tests, observed red then green.
- Runtime serialization now rejects incorrect declared output types and nested
  non-finite JSON numbers (seven regression cases observed red then green).
- First run-details slice is verified: `frontend/argo.html` mounts
  `ArgoRunDetails.tsx` with retained KFP graph/log widgets and browser-side
  `Argo.ts` adapters. It reads hydrated Argo workflows, task metadata/artifacts
  and logs; compares up to five explicitly selected runs; makes no KFP backend
  requests. This isolated entrypoint does not yet replace all retained pages.
- 40 focused frontend tests pass, as do the production build (ESLint/TypeScript),
  all six existing production-bundle browser smokes, and the new real-Argo smoke.
  See `samples/argo-native/README.md` for exact commands and red/green evidence.
- Live page: `http://127.0.0.1:3002/argo.html?namespace=argo`, started with
  `KFP_ARGO_DEMO=true npm start -- --host 127.0.0.1 --port 3002 --strictPort`.
  Argo port-forward is at `https://localhost:2746`; restart it if it disconnects.
  Logs: `/tmp/kfp-argo-ui.log`, `/tmp/kfp-argo-port-forward.log`.
  Browser evidence: `/tmp/kfp-argo-ui.png`.
- Stage B compiler work is now implemented in `argo_compiler.py` and
  `argo_groups.py`: GraphComponent captures DSL objects without calling the IR
  builder; normal Compiler.compile emits content-addressed WorkflowTemplates,
  standalone Workflows, or suspended CronWorkflows. Mutable-image caching requires
  a supplied digest or explicit disabling; keys include resolved runtime parameters
  and artifact identities. Supported task settings include resources, retries,
  node selectors, labels, tolerations, secret/config-map env, PVC mounts and timeout.
- The compiled ML pipeline passed on kind (`argo-ml-demo-tk4lw`,
  `argo-ml-demo-p2n68`), including native preparation memoization. These are compiled
  resources, unlike the earlier handwritten demo.
- Seven real-cluster compatibility cases passed via
  `uv run python samples/argo-native/verify_compatibility.py --context kind-kfp-argo`:
  nested DAGs, both conditional branches, bounded loops and integer collection,
  failed loop iterations, successful/failing bodies and failing cleanup, and a
  BusyBox container artifact consumed by an unchanged Python component.
- Exit-handler testing found that Argo success-only leaf assessment can mask a
  failed body. The compiler now emits a failure-aware DAG with both `body` and
  `exit` as explicit targets, not a lifecycle hook. Cleanup still runs on failure;
  body or cleanup failure makes the run fail. No additional control/driver pod.
- 89 targeted Python tests pass. Legacy IR-oriented SDK tests still need to be
  reconciled with the intentional output-contract change; do not claim a green
  full SDK suite. Native container metadata also needs the frontend parameter
  fallback (Python metadata remains an Argo artifact).
- Current next work: complete local execution removal and the remaining adapters,
  then deployment/end-to-end verification. The user explicitly requested continuing
  through the ENTIRE plan, deploying on native OrbStack Kubernetes, and fixing and
  retrying any surfaced problems. Do not stop at another milestone.
- Native `orbstack` Kubernetes was stopped; `orbctl start k8s` brought it up.
  Node `orbstack` is Ready (v1.36.5+orb1), with only system deployments. Keep final
  prototype resources isolated and use explicit `--context orbstack` commands.
- Not yet implemented: removal of local execution, full page adapters,
  archive/authorization/recurring-run demonstrations, and approval gate.
  The handwritten cache key remains demo-only; the native compiler now uses
  image-digest, execution-configuration and resolved-input fingerprints.

Full SDK compatibility and scalable cross-run lineage remain unproven. The
existing `compiler.Compiler()` now emits native Argo, but only the explicitly
verified compatibility corpus is proven. Unsupported constructs produce compile
errors; legacy IR/client compatibility is not a goal.
