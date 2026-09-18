# KFP upgrade-readiness preview

Assess selected deployment and RBAC configuration on an existing 2.17 installation
without upgrading it. This standalone Python tool uses `kubectl get` or an offline
JSON inventory and prints an actionable Markdown or JSON report.

**This first version is a partial migration-plan assessment, not an upgrade
certification.** Every report is marked `incomplete`. It can identify configuration
to review, but cannot establish that users or workloads will succeed on 2.18.
Ruleset `2.18-preview.1` includes proposed TensorBoard adoption behavior from
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

- `1`: invalid input or tool failure; no assessment is available.
- `2`: an incomplete assessment was produced, including missing collection permissions.

There is deliberately no successful readiness exit code in this preview. Scripts
must not treat generating a report as passing an upgrade gate.

## What it checks

| Rule | Evidence and limits | Action |
| --- | --- | --- |
| `tensorboard.key` | Explicit signing-key environment entry or unresolved environment imports in the selected UI deployment. Values are not reported; key existence, validity and equality across replicas remain unknown. | Verify persistent shared key configuration without exposing its value. |
| `tensorboard.rollout` | Declared UI rollout strategy. Proposed shared-key first adoption requires coordination even if the old deployment uses rolling updates. | Plan interruption and URL refresh; validate final target manifests and subsequent restarts. |
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

The tool issues only Kubernetes `get` commands:

- List Deployments, Roles and RoleBindings in each explicitly selected namespace.
- Get each ClusterRole referenced by those RoleBindings, by name.

It does not read Secrets, ConfigMaps, pod logs, MLMD or the database, create
SubjectAccessReviews, execute workloads, mutate resources or export telemetry.
Use an existing read-only operator identity; no cluster-admin grant is required.
Permission failures become unknown findings. Referenced Secret/ConfigMap values
and `envFrom` imports are deliberately unresolved in this first implementation.

Kubernetes may return inline environment values inside Deployment objects. These
are held in memory for analysis but are not copied to reports; neither are binding
subjects or raw Kubernetes error messages. Reports contain resource names and
namespace names and should remain access-controlled. Offline inventories can
contain more sensitive data than reports; handle them accordingly.

Each `kubectl` invocation has a 20-second request timeout, a 30-second process
budget and a streamed 16 MiB response cap. Retained inventory also has a
cumulative 16 MiB serialized-JSON budget; exceeding it stops collection and
reports the remaining scope as uncollected. Limits also include 100 selected
namespaces, 10000 inventory objects and 256 referenced ClusterRoles. Larger scans
should be split into explicit scopes. These caps bound collection; they are not
pipeline-size settings. Kubernetes exec credential plugins configured in the
selected kubeconfig still run normally through `kubectl`.

## Offline analysis

`--inventory` accepts a JSON object with an `items` list containing only
Deployment, Role, RoleBinding and ClusterRole objects. It is a Kubernetes inventory,
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

1. Add authenticated KFP inventory for stored specifications, schedules, ownership,
   service accounts and size limits, with explicit pagination/collection coverage.
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
assessment semantics.
