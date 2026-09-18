# KFP upgrade-readiness preview

Assess selected deployment and RBAC configuration on an existing 2.17 installation
without upgrading it. This standalone Python tool uses `kubectl get` or an offline
JSON inventory and prints an actionable Markdown or JSON report. Optional authenticated
KFP API collection supplies recurring-run, experiment and pinned-template evidence.

**This first version is a partial migration-plan assessment, not an upgrade
certification.** Every report is marked `incomplete`. It can identify configuration
to review, but cannot establish that users or workloads will succeed on 2.18.
Ruleset `2.18-preview.4` includes proposed TensorBoard adoption behavior from
[#14362](https://github.com/kubeflow/pipelines/pull/14362), not a claim that this
change has shipped. Final release-candidate rules must be pinned and validated
before this tool can offer a readiness conclusion.

## Run alongside an existing installation

Requires Python 3.9+ and `kubectl` on Linux or macOS. No Python packages, KFP SDK,
server patch, database access or additional running service are needed.

```bash
python3 tools/upgrade-readiness/readiness.py \
  --context my-cluster \
  --system-namespace kubeflow \
  --namespace team-a --namespace team-b \
  --source-version 2.17.2 \
  --format markdown > readiness.md
```

The context and installation namespace are required. Additional namespaces are
explicitly selected; the tool does not discover or scan all tenant namespaces.
`--source-version` is an operator assertion, not a verified installed version;
mixed-version replicas and image digests are not assessed. Choose the actual
2.17 patch version; other versions require a new validated rule contract.

Custom deployment names can be selected with `--ui-deployment` and
`--cache-deployment` (defaults: `ml-pipeline-ui` and `cache-server`). A renamed UI
container is recognized only when it is the sole container; ambiguous multi-container
configuration remains unknown.

Use `--format json` for automation. Exit codes:

- `0`: help displayed (`--help`), not a readiness result.
- `1`: invalid input or tool failure; no assessment is available.
- `2`: an incomplete assessment was produced, including missing collection permissions.

There is deliberately no successful readiness exit code in this preview. Scripts
must not treat generating a report as passing an upgrade gate.

## Schedule inventory (optional)

Add `--include-schedules` to collect ScheduledWorkflow CRs in the same explicitly
selected namespaces. This needs `list` permission on `scheduledworkflows` in the
`kubeflow.org` API group. It works on the existing installation and does not
trigger runs or require an upgrade. The default scan does not request schedules.

Try the synthetic schedule inventory without cluster access:

```bash
python3 tools/upgrade-readiness/readiness.py \
  --inventory tools/upgrade-readiness/examples/schedules.json \
  --system-namespace kubeflow --namespace team-a \
  --source-version 2.17.2 --include-schedules --format markdown
```

Each schedule receives a resource-specific finding with its declared enablement
and, for the API submission path, explicit `spec.serviceAccount`. The report
identifies the specific account to check; it does **not** claim the controller is
allowed or denied. Confirm the actual controller caller, groups, target run
namespace and final 2.18 policy before adding narrowly scoped grants. An omitted
account remains unresolved because server defaults are not collected.

Schedules with embedded `spec.workflow.spec` use a different controller path;
the tool leaves their workflow identity unresolved instead of treating the
top-level account as authoritative. It does not resolve referenced pipeline
versions or reconcile CRs against database recurring runs. Disabled schedules
are included because they may be re-enabled later. A zero count, missing CRD or
permission failure never certifies that scheduling is unaffected.

## Collect source KFP evidence automatically

Combine the existing namespace scan and target policy file with a source API
endpoint. The target file still supplies the intended 2.18 settings, RBAC and
controller identity; its manual recurring-run/experiment arrays can be empty.

```bash
python3 tools/upgrade-readiness/readiness.py \
  --context my-cluster --system-namespace kubeflow --namespace team-a \
  --source-version 2.17.2 --include-schedules \
  --schedule-policy target-policy.json \
  --kfp-endpoint https://kubeflow.example/pipeline \
  --kfp-token-file /secure/path/kfp-token --format json
```

Use an existing bearer token accepted by that KFP installation, held in a local
file. Tokens never appear in command arguments or reports. `--kfp-ca-file` accepts
a custom HTTPS CA bundle; TLS verification cannot be disabled. HTTP is accepted
only for a literal loopback address, such as `http://127.0.0.1:8888`, for an existing
port-forward. The tool does not start one. Cookie login flows and arbitrary
identity headers are not supported. Redirects are refused, including login
redirects, and environment HTTP proxies are disabled. Choose the exact API base
URL, not a sign-in page.

Collection only performs V2 API GET requests:

- Paginate recurring runs in each selected namespace, including disabled runs.
- Fetch referenced experiments once and verify their namespaces.
- For omitted accounts, inspect inline V2 evidence or GET a pinned pipeline
  version and verify both returned IDs. Reused versions are fetched once.

Manual source records in the target file are **replaced**, even when collection
fails; stale records cannot mask permission failures. Returned records are reduced
to IDs, namespace, account and a template classification marker. Raw specifications,
parameters and token values are not reported or written to disk. Responses are
held temporarily in memory and may contain sensitive information.

Unpinned/latest versions remain unknown, as do unavailable templates and legacy
embedded workflows. No `package_url` or artifact endpoint is fetched. An omitted
account can be modeled as the target default only for collected V2 template
structure **and** an explicitly empty target `compiled_pipeline_spec_patch: {}`
in the target policy file. Omit that setting or supply a nonempty patch to leave
the default unresolved. Structural classification is not protobuf/compilation
validation, and plugin/additional identities are still unassessed.

Coverage reports include requested namespaces, completed list traversals, record
counts and failed checks. A completed traversal does not mean every experiment or
version was accessible. Reads are not an atomic snapshot; rerun near upgrade time.
Failures and repeated pagination tokens remain unknown, retaining earlier evidence.
Limits are 100 pages per namespace, 10000 recurring-run records, 200 HTTP requests,
16 MiB per response and cumulatively, and a 20-second request/body budget.
The 10000-record target budget is shared with experiments and target RBAC. DNS
resolution is subject to operating-system timeouts. Requests stop succeeding when
a budget is exhausted; reduce the explicit scope. Target bundle limits still apply.

The scan also reports whether the default-named controller Deployment declares a
Pod account or identity-header command flags, without exposing values. These are
source configuration hints only. Authentication headers can take precedence over
tokens, so the tool never converts these hints into `controller_user` automatically.

## Target main-account prediction (optional preview)

Use `--include-schedules --schedule-policy target-policy.json` to evaluate the
main service-account check using **persisted KFP recurring-run evidence and
explicit target settings/RBAC**. The operator still runs against 2.17. Without
`--kfp-endpoint`, this option makes no additional network requests. It never
submits authorization reviews. It does not change
any cluster permissions.

The JSON bundle has this shape (values are illustrative):

```json
{
  "policy_contract": "14363-main-account-preview.1",
  "target_revision": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
  "multi_user": true,
  "mode": "enforce",
  "default_service_account": "pipeline-runner",
  "allowed_service_accounts": ["training-runner"],
  "controller_user": "system:serviceaccount:kubeflow:ml-pipeline-scheduledworkflow",
  "rbac_complete": false,
  "rbac_only": false,
  "recurring_runs": [],
  "experiments": [],
  "rbac": []
}
```

Replace the revision with the exact proposed candidate commit. Supply
`recurring_runs` from authenticated V2 KFP API responses, retaining
`recurring_run_id`, `experiment_id`, `namespace` and `service_account`;
`experiments` retain `experiment_id` and `namespace`. Reconcile all pages in the
selected scope. Records are matched to `ScheduledWorkflow.metadata.uid`, never
by display name. Missing/duplicate records and inconsistent namespaces remain
unknown. Do not substitute the editable CR's account for the persisted account.
Use `--kfp-endpoint` to automate collection and inspect pinned/inline template evidence.
An omitted account is unresolved unless collected V2 evidence and an explicitly empty target compiler patch establish the conditional default. Embedded workflows remain unresolved.
An explicit persisted account can be checked without loading a referenced template;
template-dependent defaults and additional identities remain unknown.

Supply `rbac` as target Role, ClusterRole, RoleBinding and ClusterRoleBinding
objects, including retained custom grants as well as rendered candidate objects.
Never treat only the stock manifests as a complete snapshot. A missing grant is
reported as denial only when **both** `rbac_complete` and `rbac_only` are true:
these are explicit operator assertions that all applicable bindings/roles are
included and RBAC is the only relevant authorizer. Otherwise absence is unknown.
Missing referenced roles, ambiguous duplicate roles and unresolved aggregation
also remain unknown. Named-account restrictions, namespace scope and additive
grants are evaluated. No roles are created or recommended with wildcard access.

`controller_user` must be the caller the target API actually authenticates.
Do not infer it solely from a controller Pod: configured identity headers can
change it. This contract mirrors KFP's **user-only** SubjectAccessReview; it does
not add service-account or authenticated groups. A group-only grant therefore
does not satisfy this check. `allowed_service_accounts` contains the exact names
from target `ALLOWEDSERVICEACCOUNTS` (empty denies custom accounts; `*` is literal).
`default_service_account` comes from target `DEFAULTPIPELINERUNNERSERVICEACCOUNT`;
that account is exempt from this check. `mode` corresponds to target
`KFP_SECURITY_SERVICE_ACCOUNT_MODE`.

The contract is pinned to the proposed #14363 integration at
`698819580262320715ee616c7479b33c62e0a4b7`, not dynamically inferred from the supplied
candidate revision. The report records both revisions and labels target evidence
operator-supplied/unverified. Confirm the candidate has equivalent policy before
using its predictions; a changed policy needs a revised contract. Source 2.17
configuration and its current authorization decisions are not target evidence.

- `policy_rejection`: the supplied enforce policy would reject this main account.
- `operational_impact`: audit would record the modeled policy denial; transport,
  authentication and authorizer evaluation failures can still block execution.
- `no_issue_detected`: a main-account exemption or RBAC grant was found within this
  contract. This is not a schedule execution pass.
- `unknown`: evidence or supported identity resolution is insufficient.

Reports remain incomplete. They do not check controller run-creation permission,
pipeline access, account existence, tampering/replay protections, plugins or
additional workflow identities. Those checks and prediction-versus-execution
fixtures in existing upgrade CI remain open in #14421. Bundles may contain
sensitive workload data; keep them local and access-controlled. Reports omit raw
records, controller names, subjects, specifications and parameters. The bundle
shares the 16 MiB file limit and permits at most 10000 records across its lists.

## What it checks

| Rule | Evidence and limits | Action |
| --- | --- | --- |
| `tensorboard.key` | Explicit signing-key environment entry or unresolved environment imports in the selected UI deployment. Values are not reported; key existence, validity and equality across replicas remain unknown. | Verify persistent shared key configuration without exposing its value. |
| `tensorboard.rollout` | Declared UI rollout strategy. Proposed shared-key first adoption requires coordination even if the old deployment uses rolling updates. | Plan interruption and URL refresh; validate final target manifests and subsequent restarts. |
| `schedule.serviceAccount` / `schedule.coverage` | Optional ScheduledWorkflow inventory, explicit API-path account and declared enablement; embedded workflow identities and effective permissions remain unknown. | Review each account with the actual controller caller and reconcile with stored recurring runs. |
| `cache.legacy` | Presence/absence of the selected legacy cache deployment. Does not establish usage, cache ownership or cost. | Assess V1/raw-Argo cache data and representative task executions separately. |
| `rbac.readLog` | A namespace RoleBinding references a role declaring run read access; check whether that role also declares the `readLog` verb. | Verify effective caller permissions. Missing permission in one role is **not** proof of denial. |
| `rbac.tensorboard` | A bound role declares viewer reads; inspect create/delete declarations separately. | Keep readers read-only; verify scoped grants for intended managers. |
| `rbac.coverage` / `inventory.collection` | Missing referenced role, unresolved aggregate rules, failed/oversized/timed-out Kubernetes reads. | Repair scope/access or supply the missing evidence. Never count missing data as a pass. |

`no_issue_detected` means only that a checked role declares the relevant verbs.
It is not a subject-access decision: other bindings, groups, `resourceNames`,
cluster-wide bindings and the caller's identity are not evaluated. The tool never
automatically grants permissions or recommends making every reader an editor.

The report always lists unassessed areas: effective authorization, stored
pipelines/runs/schedules, workflow identities, artifact storage, size ceilings,
cache usage and cost, SDK compilation, actual client traffic and live upgrade
acceptance. A namespace with no collected bindings is not assumed unused.

## Access and data handling

The Kubernetes collector issues only `get` commands; optional KFP API collection is described above:

- List Deployments, Roles and RoleBindings in each explicitly selected namespace.
- Get each ClusterRole referenced by those RoleBindings, by name.
- With `--include-schedules`, list ScheduledWorkflows in those namespaces.

It does not read Secrets, ConfigMaps, pod logs, MLMD or the database, create
SubjectAccessReviews, execute workloads, mutate resources or export telemetry.
Use an existing read-only operator identity; no cluster-admin grant is required.
Permission failures become unknown findings. Referenced Secret/ConfigMap values
and `envFrom` imports are deliberately unresolved in this first implementation.

Kubernetes may return inline environment values inside Deployment objects. These
are held in memory for analysis but are not copied to reports; neither are binding
subjects or raw Kubernetes error messages. Schedule collection also reads
embedded workflow specifications and parameters into memory, but reports never
include their contents. Explicit API-path service account names are reported. Reports contain resource names and
namespace names and should remain access-controlled. Offline inventories can
contain more sensitive data than reports; handle them accordingly.

Each `kubectl` invocation has a 20-second request timeout, a 30-second process
budget and a streamed 16 MiB response cap. Retained inventory also has a
cumulative 16 MiB serialized-JSON budget; exceeding it stops collection and
reports the remaining scope as uncollected. Limits also include 100 selected
namespaces, 10000 inventory objects and 256 referenced ClusterRoles. Larger scans
should be split into explicit scopes. These caps bound collection; they are not
pipeline-size settings. Kubernetes exec credential plugins configured in the
selected kubeconfig run through `kubectl` in a private process group. On timeout
or excessive output, the tool kills that group, including inherited plugin
processes; it cannot contain a plugin that deliberately starts a new session.

## Offline analysis

`--inventory` accepts a JSON object with an `items` list containing only
Deployment, Role, RoleBinding, ClusterRole and ScheduledWorkflow objects.
ScheduledWorkflow objects are analyzed only with `--include-schedules`. It is a Kubernetes inventory,
not a previous report. Namespace-scoped objects need `metadata.namespace`; objects
outside the requested scope are excluded. Offline inventory completeness is
unknown even when no object is missing visibly.

```bash
python3 tools/upgrade-readiness/readiness.py \
  --inventory inventory.json --system-namespace kubeflow --namespace team-a \
  --source-version 2.17.2 --format json > readiness.json
```

Try the synthetic example without cluster access by passing
`--inventory tools/upgrade-readiness/examples/inventory.json --namespace team-a`
with the same installation namespace and source version above. It demonstrates
a custom reader role whose log and TensorBoard management permissions need review.

## Next stages

Tracked in [#14421](https://github.com/kubeflow/pipelines/issues/14421):

1. Expand authenticated KFP collection beyond recurring runs/experiments and
   pinned-template evidence to full stored workload, ownership and size coverage.
2. Add optional effective authorization checks with separately documented access
   requirements, including actual caller/group identities.
3. Assess artifact origins, profile reconciliation, archive credentials and
   HTTP configuration without exposing credentials or fetching arbitrary artifacts.
4. Add optional 2.17 observation telemetry for real requests. It must preserve
   existing enforcement, use bounded metric labels and distinguish evaluation
   failure from success. It must not replay requests or create runs.
5. Pin the final rules to the actual 2.18 candidate and test against populated
   upgrade fixtures; report the observation period and workload coverage.

Observation telemetry is **not** implemented here and is distinct from the
security audit/enforce controls. Zero findings over a quiet period cannot prove
that monthly schedules or unobserved clients will work.

## Development

```bash
python3 -m unittest discover -s tools/upgrade-readiness -p 'test_*.py' -v
```

Tests use synthetic inventories and subprocesses, never a real cluster. They cover
role scope/wildcards, unresolved aggregation, missing permissions, environment
imports, sensitive-value exclusion, output bounds, CLI errors and incomplete
assessment semantics, opt-in schedule scope, unresolved defaults and embedded
workflow identities, and exclusion of schedule parameters/specifications.

## Policy-code conformance versus upgrade acceptance

CI compares predictions with the actual backend `authorizeServiceAccount` method
at the policy source revision recorded above. It covers default exemption, a named
custom-account grant, denial, audit, exact allowlists, whitespace, wrong named
accounts and group-only grants. Controlled SAR responses are independent fixture
inputs; this does not verify a real Kubernetes authorizer or schedule firing.

To run locally, use a clean checkout at that exact revision:

```bash
python3 tools/upgrade-readiness/conformance.py --backend-source /path/to/policy-checkout
```

The runner checks the revision and clean working tree, generates predictions and
uses a Go overlay to add its test without editing the backend checkout. A policy
compile failure or prediction mismatch fails the job. It never silently substitutes
the current branch for the policy source.

The existing upgrade workflow separately tests a populated **2.17.2 → release-2.18**
installation when run on that release branch or a PR targeting it. Its master
MLMD migration gate remains in place. Preparation failures prevent target deployment.
Those existing tests cover single-user resource persistence, not multi-user schedule
firing. The final-candidate lane must still compare observed recurring-run execution
with predictions after the scheduling policy changes land; that work stays open
in #14421. Neither this conformance job nor a skipped upgrade job satisfies that gate.
