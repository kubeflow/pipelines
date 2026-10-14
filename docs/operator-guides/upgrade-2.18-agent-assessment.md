# Agent guide: assess a cluster's 2.18 upgrade impact

Use this procedure when an operator asks an agent to assess an existing KFP
installation before upgrading to 2.18. The deliverable is a scoped, evidence-backed
impact report: what will change for this installation, who is affected, what is
automatic, and what requires action. It is not a security certification or a
requirement that every operator run additional tooling.

Read the [upgrade guide](upgrade-2.18.md) first. Assess the exact target release
revision and rendered deployment, not current master: 2.18 retains MLMD and V1
support where policy permits; the 3.0 migration has different requirements.

## 1. Establish scope without changing the cluster

Record the explicitly selected Kubernetes context, installation namespace, tenant
namespaces, authentication mode, source image tags **and digests**, target revision,
manifest overlays and client versions. Ask for missing cluster identity before
accessing an ambiguous context. Do not infer the installed version from a scanner's
`--source-version` argument or one replica's tag. Include old/terminating replicas.

Default to read-only inspection. Do not restart deployments, annotate namespaces,
change RBAC, enable audit modes, copy objects, alter SQL/MLMD records, replay requests
or execute workloads as part of assessment. Propose those actions separately with
the affected resources and verification steps. An explicit authorized migration or
isolated test task can include writes within that scope.

Collect only needed fields. Deployment environment and specs can contain literal
credentials; do not dump complete manifests or Secret data into reports. Inspect
Secret reference names and presence through an authorized, redacted check without
printing values. Treat pipeline parameters, artifact URIs, endpoint query strings,
logs and user identifiers as sensitive. Keep detailed evidence in the operator's
approved location; publish only redacted examples. Never send it to external tools
or issue trackers without permission.

Create a coverage ledger: selected namespaces, resource/API types, pagination
completion, sample sizes, time interval, denied requests and uninspected scopes.
Empty results, unavailable APIs and truncated traversals are distinct. Read-only
collection is not an atomic snapshot; recheck drift near rollout.

## 2. Use the existing scanner as partial evidence

The [upgrade-readiness preview](https://github.com/kubeflow/pipelines/blob/release-2.18/tools/upgrade-readiness/README.md) runs beside
2.17 without a server upgrade. A scoped example is:

```bash
python3 tools/upgrade-readiness/readiness.py \
  --context my-cluster --system-namespace kubeflow \
  --namespace team-a --source-version 2.17.2 \
  --include-schedules --format markdown > readiness.md
```

Use only the operator-selected namespaces. Optional `--include-ownership` and
GET-only KFP collection require the documented endpoint/credential configuration.
Keep tokens in the supported token file, never command arguments. Pin the scanner
revision and target policy bundle; current permissions alone do not predict target
RBAC. Its preview rules can lag the final release and cannot overrule newer code.

Every assessment from this preview is incomplete (exit 2). It does not cover all
identities, artifact ownership, workload execution or mixed-version replicas.
Follow up on findings below; do not turn absence of scanner findings into a pass.
The `LIVE_VALIDATION` and `REPORTING_RECOVERY` fixtures deliberately mutate state
and inject failures. They belong in an authorized disposable environment, never
in a production read-only assessment.

## 3. Inspect the changes that can affect this installation

### Scheduling, identities and handoff

Inventory enabled **and disabled** API recurring runs, their owning experiments and
namespaces, corresponding ScheduledWorkflow CRs, retained execution identity and
progress. Paginate all relevant lists and record incomplete traversal. Distinguish
API-owned records from CR-only records; the presence of a CR is not evidence that
a schedule is unsupported. V2 continues using CRs for scheduling; the API now trusts
its stored definition instead of editable execution inputs from a CR.

For each schedule or representative workload class, inspect the resolved main
service account plus template, helper, ArtifactGC and plugin identities. Account
for compiler patches, referenced templates and moving/default pipeline versions.
If the effective identity cannot be resolved, mark it unknown rather than applying
the main-account result to the whole workflow. Compare the intended allowlist and
namespace/name-scoped `serviceaccounts/use` grants for both the initiating user and
the actual controller caller. Inspect groups and authentication/header precedence;
a Pod service account declaration alone may not identify the effective caller.

Use [recurring-run accounts](scheduled-service-accounts.md) and
[workflow policy](server-config.md#api-server-workflow-service-account-checks): inspect
`KFP_SECURITY_SERVICE_ACCOUNT_MODE` and `KFP_SECURITY_WORKFLOW_IDENTITY_MODE` on
**every** target API replica. Both default to enforcement, and audit relaxes only
its own policy. External/uninspectable templates and templated main-account values
need their specific policy assessment. Audit is not permission to bypass ownership,
adoption, namespace checks or authentication.

Inspect the API/controller Deployment names, `app` selectors, container names,
Pod downward-API identity, rollout state, handoff permissions and controller probes.
Identify every writer sharing the database, including separate installations or
external processes. Stock managed rollout adopts eligible schedules automatically:
no history deletion, blanket recreation, manual writer shutdown or active-run drain
is required. Renamed deployments need the same `KFP_SCHEDULE_WRITERS` inventory on all API/controller
processes, the documented namespace-scoped ReplicaSet reads, and Deployment read
grants covering the configured names (stock Role `resourceNames` are restricted); include any retiring
writer deployments. See [custom identities](https://github.com/kubeflow/pipelines/blob/release-2.18/docs/operator-guides/scheduled-service-accounts.md#renamed-managed-deployments).
Unmanaged/shared-database writers need a coordinated plan covering all writers.
A controller readiness probe tied to its post-handoff listener can deadlock rollout.
Do not copy handoff annotations into templates or fabricate adoption receipts.

Report temporary schedule-write rejection as user impact: create/enable/disable may
return explicit pre-mutation `Unavailable`; a rejected disable leaves the schedule
enabled. Inspect actual clients' versions and handling. Updated UI/Python SDK retry only
HTTP 503/code 14 with `google.rpc.ErrorInfo` reason `SCHEDULE_HANDOFF_IN_PROGRESS`
and domain `kubeflow.org`: five attempts with 1, 2, 4 and 8 second delays. Older
clients do not acquire this behavior from a server upgrade. An explicit rejection can be retried
with bounded backoff and read-back; a generic timeout on create is ambiguous and
must not be blindly replayed. Ordinary reads and run operations are separate.

For inconsistent per-record state, preserve evidence and use the documented recovery
procedure. Never propose counter resets, ownership edits or history deletion to force
adoption. CR-only multi-user schedules need an authorized API definition, with a
plan preventing duplicate runs. A rolling downgrade to legacy writers after adoption
is not supported by the forward-upgrade handoff.

### Artifact access, roots and proxy propagation

Inspect representative historical **and newly produced** artifact references using
the owning user's view. Classify object paths without exporting complete URIs:
namespace-prefixed, custom-prefix-plus-namespace, arbitrary legacy root, or unknown.
Compare with `ARTIFACT_NAMESPACE_KEY_PREFIX`; matching MLMD evidence alone no longer
authorizes an arbitrary root in multi-user mode. Standalone mode is unaffected by
this ownership middleware. See [root migration](upgrade-2.18.md#artifact-root-migration).

Check every authenticated serving UI replica (including custom authenticated
frontends) for
`KFP_SECURITY_ARTIFACT_OWNERSHIP_MODE`. Default enforcement is the target; temporary
audit only permits missing prefixes with successful matching MLMD evidence. Audit
cannot resolve missing evidence, cross-namespace evidence, unsafe paths or MLMD
outages. Audit counts mean permission to attempt retrieval, not successful reads.
If audit is proposed, require independent downstream storage/network isolation,
an owner, expiry and return-to-enforcement verification; it is not complete security
remediation and is absent from 3.0.

For S3-compatible providers, compare exact scheme/host/port origins with
`ALLOWED_ARTIFACT_ENDPOINTS` and configured trusted endpoints. DNS aliases need
explicit consideration; a regex match alone is insufficient. For HTTP artifacts,
check `HTTP_BASE_URL` origin/path, redirect destinations and shared-UI reachability.
An allowlist does not supply credentials. Standard tenant proxies rely on the
authenticated shared UI for ownership checks; they do not need the artifact audit
mode copied into their unauthenticated environment. Existing profile proxies need endpoint/base configuration
propagation and reconciliation, not merely a restarted controller. Verify desired
proxy template values before considering a completed rollout evidence of an update.
See [custom-store rollout](https://github.com/kubeflow/pipelines/blob/release-2.18/manifests/kustomize/README.md#upgrade-example-custom-s3-storage).

Assess archived logs separately from artifact proxies and live Pod logs. The shared
UI needs appropriate storage/network access; tenant-only credentials do not prove
it can read archives. A successful live log read is not archived-log evidence.
Custom NetworkPolicy must allow the actual metacontroller/profile-controller path.
Do not broaden tenant Secret access to restore functionality.

Also inspect custom consumers of download responses: hardened routes can return
raw archives and attachment headers. UI preview support does not imply a custom
client that relied on inline rendering or auto-extraction will behave identically.

### Authorization and client scope

Inventory custom roles, ordinary users, read-only users, designated shared publishers
and SDK workload identities. Evaluate target roles using exact SubjectAccessReview
attributes from [RBAC/client migration](rbac-migration-2.18.md), not discovery-based
`kubectl auth can-i` for virtual KFP resources. A current-role authorization probe is
only current-role evidence; it does not execute KFP's additional ownership checks.

Check private upload calls for explicit `namespace=`. `Client(namespace=...)` does
not automatically scope pipeline uploads. Check referenced private pipelines and
destination experiments share a namespace, even for users authorized in both.
Do not recommend broader shared-read policy as an upgrade shortcut. Verify custom
log roles have `runs/readLog`, and TensorBoard readers/managers have appropriate
`viewers` verbs.

Find direct/generated pipeline-by-name endpoint calls with omitted namespaces.
Do not flag every SDK `get_pipeline_id()` call as a newly broken endpoint; it uses
filtered listing. Inspect saved pagination-token consumers: updated UI restarts once,
whereas SDK/custom clients need to handle `PAGINATION_RESTART_REQUIRED`, discard or
reconcile prior traversal results, and restart deliberately. Mixed-version readers
may report generic invalid-token errors. A server rollout does not update clients.

### Limits, old specifications and workload behavior

Compare actual byte sizes with target `MAX_PIPELINE_UPLOAD_BYTES`,
`MAX_PIPELINE_SPEC_BYTES` and `MAX_PIPELINE_UPDATE_BODY_BYTES` (32 MiB defaults,
bounded overrides up to 128 MiB). Inspect **existing object-store specifications**
used by future runs, not just new uploads. Original compressed upload size cannot
be reconstructed from a stored extracted spec; record it unknown. Archive traversal
has an additional derived budget. Direct database-spec reads have a different path.
Consider ingress, gRPC, Kubernetes and memory limits separately; raising one KFP
limit does not prove the complete request will succeed.

For legacy V1 metrics collection, inspect file sizes and
`MAX_METRICS_FILE_BYTES` on the actual collector (1 MiB default). Oversized/malformed
metrics can leave a completed task without reported metrics. Changing the limit does
not automatically replay a completed node's permanent collection failure. Native V2
metrics are a different path; pre-existing count/one-file rules are not new 2.18 breaks.

Where used, inspect newly compiled lightweight components relying on pip trusted-host
behavior, embedded artifact symlinks, retry of archived runs, and literal scheduling
macro strings in nested parameters. Compare source and target behavior before labeling
an existing limitation a new regression. Record whether a change affects existing
compiled packages, recompilation, new runs, or a later user operation.

### Availability, cache and persistence

Check the UI signing Secret reference and initializer resources without printing the
key. First adoption interrupts UI access and invalidates old TensorBoard URLs;
reopening TensorBoard restores URLs. Later default `Recreate` updates still interrupt
the UI. Optional rolling updates require prior shared-key adoption. Inspect GitOps
Secret preservation, backup and restore behavior; do not rotate keys as a generic fix.

Identify actual legacy V1/raw-Argo cache users. Under
`KFP_SECURITY_LEGACY_CACHE_MODE=enforce`, unknown-owner historical entries are cold.
Extra runtime/compute is incurred when affected steps next execute, not in a startup
rebuild. Estimate from observed step durations, resources and expected cache hits,
not cluster size alone. Native V2 MLMD cache is separate. Standard rolling cache
updates are supported, but isolation is not fully effective while old writers remain.
Audit-to-enforce can produce another cold transition.

Inspect database size/engine and maintenance tolerance for automatic index/column
migration on `run_details`; large tables may rebuild or lock. This does not imply a
manual SQL migration is required. Check persistence-agent scoped identity lookup and
retained Workflow evidence for active runs. Follow [reporting recovery](workflow-report-recovery.md)
for missing/inconsistent ownership; a UID alone cannot recover lost terminal results.

Inspect actual retention settings; default zero disables automatic retention.
Do not invent a new deletion risk from an unchanged default. Record the deferred
TaskService authorization remediation as a retained security limitation, not a
resolved advisory or a new migration procedure. The 3.0 MLMD removal is separate.

## 4. Produce a report that supports decisions

For each applicable control, include:

| Field | Required content |
| --- | --- |
| Evidence and scope | Pinned source/target, inspected resources, observation time, pagination/sample limits; redact sensitive details. |
| Outcome | Confirmed impact, no impact observed within scope, unknown, or not applicable, with a concrete reason. |
| Who and when | Operator/user/custom-client owner; rollout, next scheduled tick, next cache miss, historical read, or recompilation. |
| Action | Exact scoped change and owner, or explicitly no manual action; identify sequencing only where required. |
| Verification | Safe read-only follow-up versus separately authorized workload/migration testing. |
| Confidence | Static configuration, exact policy evaluation, successful authenticated operation, or representative isolated upgrade evidence. |
| Temporary exception | Exact audit control, retained exposure, expiry and return-to-enforce check; otherwise none. |

Separate automatic stock rollout behavior from conditional operator changes and
user/client changes. Summarize unknown scopes prominently. Avoid “cluster ready”
when evidence is partial. Do not describe candidate tests as final-release proof.

For higher confidence, propose a populated isolated source-to-target upgrade with
representative workloads, users, custom storage and scheduling state, pinned to the
actual release candidate. Verify success **and** denied cross-namespace operations,
retained IDs/progress, no duplicates, artifacts/archived logs, and audit-to-enforce
where applicable. Do not ask production operators to run fault-injection fixtures.
After an authorized production upgrade, verify the affected real operations and
scheduled ticks with their actual clients; health checks alone are insufficient.
