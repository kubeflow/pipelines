# Stored workload assessment

The `--include-workloads` option extends the read-only scanner to experiments,
pipelines, pipeline versions, runs and recurring runs on an existing 2.17 source.
It never submits runs, fetches package/artifact URLs or issues authorization reviews.
Reports remain incomplete: a traversed inventory and a conditional policy prediction
are evidence for migration planning, not proof that an installation can upgrade.

## Collect or import

```bash
python3 tools/upgrade-readiness/readiness.py \
  --context source-cluster --system-namespace kubeflow --namespace team-a \
  --source-version 2.17.2 --include-workloads --include-shared-pipelines \
  --kfp-endpoint https://kubeflow.example/pipeline \
  --kfp-token-file /secure/source-token \
  --target-policy target-workloads.json --format json > readiness.json
```

The target policy is optional. Without it, the report still inventories resources,
classifies stored templates and identifies missing evidence. The source endpoint
uses the same verified HTTPS, no redirects, token-file and bounded-read transport
as schedule collection. All records and raw specifications stay in memory; reports
contain reviewed summaries, IDs and namespaces. Treat those reports as private.

Every selected namespace is explicit, including the installation namespace. The
collector paginates each resource category, hydrates missing run/version details,
checks parent experiments/pipelines, and resolves stored version references.
Disabled recurring runs and empty experiments are included. Duplicate list rows
are deduplicated; conflicting IDs/parents/scopes, repeated page tokens, inaccessible
references and truncated traversal remain collection failures. Earlier valid
evidence survives a later failure. Reads are not an atomic snapshot.

Shared pipeline/version reads need `--include-shared-pipelines`. This requests
only the historical empty/`-` shared pipeline scopes; it never performs an
unscoped experiment/run/recurring-run scan. A reference outside the chosen scope
remains unresolved. A latest-version reference is an observation of a completed,
newest-first list, not a promise about a later execution. Pin versions where that
distinction matters.

For a single-user source whose API omits namespace fields, add
`--source-single-user` and select exactly one namespace. The report labels that
namespace binding as an operator assertion. It is never inferred from empty
fields or applied to multiple namespaces.

The combined inventory limit is 10000 records; each list has a 100-page ceiling.
The shared HTTP client limits the whole collection to 200 requests and 16 MiB of
responses, with a 20-second per-request read deadline. These are scanner budgets,
not target upload limits. Split large installations into explicit scopes.
Migration and authorization assessment each have a 10000-finding budget; a
truncation finding makes remaining work explicit. Inline identity inspection also
bounds visited nodes, tasks and step groups to 10000 and literal accounts to 100.
Split reports that reach these limits before drawing conclusions.

For offline assessment replace the endpoint with
`--workload-inventory source-workloads.json`. That file contains five raw arrays:

```json
{"experiments": [], "pipelines": [], "pipeline_versions": [], "runs": [], "recurring_runs": []}
```

Use source API records, not a prior report. The importer revalidates scope and
parents and strips all incoming `_readiness_*` annotations. Missing references
remain unknown. Offline array order cannot establish the latest version or
complete collection. Each JSON input is capped at 16 MiB.

## Supply the exact target contract

The following is an illustrative target bundle. Replace the revision, settings,
identities and RBAC with the actual candidate deployment evidence. An arbitrary
revision string does not prove that the candidate implements this modeled policy.

```json
{
  "policy_contract": "2.18-workloads-preview.1",
  "target_revision": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
  "runtime": "legacy-and-v2",
  "multi_user": true,
  "shared_read": false,
  "service_account_mode": "enforce",
  "workflow_identity_mode": "enforce",
  "default_service_account": "pipeline-runner",
  "allowed_service_accounts": ["training-runner"],
  "compiler_patch_empty": true,
  "plugins_disabled": false,
  "rbac_complete": false,
  "rbac_only": true,
  "rbac": [],
  "identities": [
    {"resource_kind": "run", "resource_id": "existing-run-id", "user": "intended-user"},
    {"resource_kind": "recurring_run", "resource_id": "existing-schedule-id", "user": "actual-controller-caller"}
  ],
  "access_checks": [
    {"operation": "read_logs", "namespace": "team-a", "user": "log-reader", "resource_name": "exact-run-k8s-name"},
    {"operation": "manage_viewers", "namespace": "team-a", "user": "tensorboard-manager"},
    {"operation": "upload_pipeline", "namespace": "team-a", "user": "pipeline-writer", "resource_name": "new-pipeline-name"}
  ],
  "http_base_url": "https://files.example/approved/",
  "limits": {
    "upload_bytes": 33554432,
    "spec_bytes": 33554432,
    "update_body_bytes": 33554432,
    "parameter_bytes": 10000,
    "metrics_bytes": 1048576
  },
  "resource_evidence": []
}
```

`runtime` describes the actual target: the MLMD-retaining 2.18 release line differs
from post-MLMD-removal master. A legacy Argo template aimed at `v2-only` is rejected;
otherwise compatibility remains conditional. Structural V2 recognition does not
replace SDK compilation or validate arbitrary platform/compiler extensions.

Identity rows describe intended submissions/replays, not discovered historical
ownership. Recurring runs need the actual authenticated controller caller,
separately from human submitters. KFP's modeled SAR sends `User` without groups;
the evaluator does not invent Kubernetes service-account or authenticated groups.
It considers Role/ClusterRole plus RoleBinding/ClusterRoleBinding evidence, exact
resource names and additive grants. Missing/ambiguous/aggregated roles stay
unknown. A denial requires an explicit complete RBAC-only snapshot; a partial
snapshot can prove only a scoped grant. Authentication, scoped-token restrictions,
authorizer transport/evaluation errors, admission and workload success are separate.

Main-account and additional-workflow-identity modes are independent. Literal
embedded identities are inspected conservatively; lifecycle hooks, resource templates, external templates, dynamic
patches, registered plugins and retained execution state prevent exhaustive
identity coverage. Even audit still requires valid authorization-service evidence.
No finding recommends broad controller rights or cross-tenant access.

Access operations map to the actual request boundary:

| Operation | Target request | Resource name |
| --- | --- | --- |
| `read_logs` | `pipelines.kubeflow.org` / `runs` / `readLog` | Run Kubernetes name, not run ID; shared-read does not bypass this verb |
| `viewer_logs` | `kubeflow.org` / `viewers` / `get` | Empty |
| `manage_viewers` | `kubeflow.org` / `viewers` / `get`, `create`, `delete` | Empty |
| `upload_pipeline` | `pipelines.kubeflow.org` / `pipelines` / `create` | Requested new pipeline name |
| `upload_version` | `pipelines.kubeflow.org` / `pipelines` / `create` | Requested new version name; use the parent pipeline namespace |

For omitted upload namespace in multi-user mode, check the installation namespace
as a shared upload. Prefer explicit tenant namespace for private uploads. Merely
setting a client's default namespace does not supply the upload method argument.
If an applicable resource name is absent, named grants cannot establish a denial.

## Evidence limits and remediation

Artifact inspection reads source `RuntimeConfig.pipeline_root` and
`PipelineSpec.default_pipeline_root`. HTTP roots outside an explicitly supplied
absolute `HTTP_BASE_URL` produce conditional policy rejections. Encoded/ambiguous
URL forms, gateway mappings, redirects and effective provider endpoints stay
unknown; the scanner never fetches them. Source 2.17 task artifact records contain
MLMD IDs, not URIs, so complete task artifacts/archive credentials require
separately authorized metadata evidence and representative UI reads. Reports omit
paths, query strings and credentials.

To assess size ceilings, add `resource_evidence` rows with `resource_kind`,
`resource_id`, and independently measured `upload_bytes`, `spec_bytes`,
`update_body_bytes`, `parameter_bytes` or `metrics_bytes`. Only measured values are
compared with corresponding supplied limits. Original compressed sizes cannot be
reconstructed by serializing stored JSON. Missing measurements stay unknown;
values below a limit do not prove a future request will use those bytes. Pipeline
upload/spec/update limits accept at most 128 MiB. Account for concurrent memory
use and upstream request limits before raising them.

Legacy-cache findings require cold-run duration/cost measurement and scoped
cache-warming checks. SDK/private package mirrors, dynamic inputs, external
clients, ownership recovery and infrequent schedules need representative use;
an inventory cannot establish those behaviors.

JSON reports include resource counts, traversal failures, template summaries and
`control_coverage` with confidence and finding counts. Markdown includes the same
control summary. All material unknowns remain visible and exit status stays `2`.
Use the final-candidate conformance and isolated upgrade procedure in
[LIVE_VALIDATION.md](LIVE_VALIDATION.md) before claiming release acceptance.
