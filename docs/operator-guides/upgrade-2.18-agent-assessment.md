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
is required. Deployment-name-only changes are discovered automatically when the
standard component `app` labels and writer container names are retained. Verify
namespace-scoped Deployment get/list and ReplicaSet get permissions from the release
RBAC, rather than requiring a manually supplied inventory for every rename. Check
that all old/new Deployments and their Pods participate, including retiring
zero-replica Deployments and terminating Pods; new process registration alone is
not a complete inventory.

Changed component labels/container names require the same complete
`KFP_SCHEDULE_WRITERS` inventory across API/controller processes. Inconsistent or
unresolved ownership must block handoff, not select a convenient subset of writers.
Flag in-place role-label/container changes while old Pods remain: the explicit
inventory has one identity per Deployment and cannot describe both generations.
See [discovery and custom identities](https://github.com/kubeflow/pipelines/blob/release-2.18/docs/operator-guides/scheduled-service-accounts.md#renamed-managed-deployments).
Unmanaged/shared-database or completely relabeled writers still need explicit
assessment and coordination; metadata discovery cannot prove they are absent.
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

Inventory unusual object keys as well as roots. Retained path hardening rejects
traversal, malformed percent escapes, encoded path separators (including literal
`%2F` segments), and internal empty segments. Test the authenticated serving path;
a successful standalone storage read does not establish compatibility. A tracked
directory prefix may retain one terminal `/`.

Assess compatibility against the existing 2.17 ownership policy. Complete artifact
ownership remediation is deferred to 3.0; do not report this gap as resolved by the
2.18 upgrade. Do not prescribe new prefixes, object migration, per-namespace
storage approval or adoption policies as a prerequisite for this release.

Inspect representative historical **and newly produced** artifact references using
the owning user's view. Classify URI scheme, bucket and object layout without
exporting complete URIs. Include old custom roots, pre-SeaweedFS V1 `artifacts/`
references, intentionally shared objects and roots that rely on rotating credentials.
Record the actual serving path and identity without collecting credentials.
Distinct Secret names do not prove restricted storage permissions.

Inspect existing `ARTIFACT_NAMESPACE_OWNERSHIP_MODE` (`mlmd-then-prefix` or
`mlmd-only`) and `ARTIFACT_NAMESPACE_KEY_PREFIX` settings. A successful MLMD lookup
with no record still uses that existing prefix/strict policy. Tracked artifacts do
not acquire a new mandatory prefix in 2.18. Known foreign MLMD namespace evidence
still denies access, but missing namespace evidence and unavailable MLMD validation
retain the older permissive behavior. `mlmd-only` does not make every metadata error
fail closed. Identify this retained exposure separately from a compatibility failure.

The unreleased `KFP_SECURITY_ARTIFACT_OWNERSHIP_MODE` setting is removed. Neither
`KFP_SECURITY_ARTIFACT_AUTHORIZATION_MODE` nor its proposed `auto`/`storage` modes,
external approval policies or readiness records ship in 2.18. Flag those settings
in pre-release overlays as inactive configuration to remove. Do not infer protection
from an unrecognized environment variable or ask an operator to enable artifact
audit. Service-account, workflow-identity and cache audit settings are separate.

For explicitly authorized disposable acceptance, require this evidence:

- Existing tracked custom roots, including multiple historical roots, remain
  readable by their existing authorized users through previews and downloads.
- Known foreign MLMD namespace evidence denies the request, including mixed
  matching/missing and foreign evidence. Namespace permission denial and namespace
  lookup failures also remain denied, independently of MLMD behavior.
- Missing tracked context, unavailable MLMD/protobufs and absent MLMD configuration
  retain 2.17 compatibility. A successful lookup with no record continues to reject
  foreign or absent prefixes, and `mlmd-only` still rejects untracked objects.
- Path/encoding, request-tampering, trusted-endpoint/redirect, credential, filesystem
  and response-header protections remain active. Do not interpret a test of these
  protections as proof of complete object ownership.
- Existing rotating-credential integrations survive renewal on their actual serving
  path. The upgrade does not add a new broker adapter or move credential placement.

Assess storage permissions independently. Review effective read/list permissions,
including bucket policies, administrator additions and historically shared grants.
Use synthetic tenant-owned objects for authorized cross-tenant tests; do not probe
other users' real data. If the current serving identity can retrieve objects whose
ownership is unknown or forgeable, record the unresolved security exposure rather
than marking the cluster fully isolated. Do not broaden grants, copy credentials
or edit metadata to make an acceptance test pass. See
[artifact compatibility and retained exposure](upgrade-2.18.md#artifact-compatibility-and-retained-exposure).

For S3-compatible providers, compare exact scheme/host/port origins with
`ALLOWED_ARTIFACT_ENDPOINTS` and configured trusted endpoints. DNS aliases need
explicit consideration; a regex match alone is insufficient. An allowlist does not
supply credentials. Namespace Secret-backed providers still require their supported
namespace-proxy path; do not grant the shared UI broad tenant Secret access.

For HTTP artifacts, inspect the actual serving process. With namespace artifact
proxying enabled, HTTP/HTTPS reads retain namespace receiver routing and credentials;
otherwise check the directly serving UI. Verify `HTTP_BASE_URL`, authentication,
redirect destinations and network reachability there. Updated receivers retain HTTP
destination and redirect protections: absolute `HTTP_BASE_URL` confines requests
to its configured origin/path, each redirect checks the domain allowlist, and
cross-origin redirects strip authentication headers. A scheme-less gateway may
redirect to other approved paths/origins; do not infer a same-origin or object-key
ownership guarantee from it. During rolling updates, old receivers keep
their older behavior until replaced; a successful read is not evidence that every
receiver has upgraded. Do not prescribe moving namespace HTTP credentials to the
shared UI. Verify custom HTTP authentication and Secret mounts are preserved by
the installation's provisioning source after controller reconciliation; the stock
controller does not provision arbitrary tenant HTTP credentials.

Existing profile proxies need environment-based endpoint/base configuration
propagation and reconciliation when those settings change. A controller restart
alone does not enqueue all namespaces: wait for hourly resync or use the documented
namespace annotation with operator authorization. Verify desired proxy template
values and actual image versions before treating rollout status as evidence.
See [custom-store rollout](https://github.com/kubeflow/pipelines/blob/release-2.18/manifests/kustomize/README.md#upgrade-example-custom-s3-storage).

Require actual Kubernetes NetworkPolicy enforcement when relying on those policies.
Inspect additive policies, installation namespace and shared-UI selectors, and custom
proxy routes. Live ingress/proxy probes need explicit authorization; otherwise report
the boundaries as unverified. Local port forwarding with a trusted user header can
exercise namespace checks, but cannot prove that real ingress authenticates users or
strips spoofed identity headers. Verify that path separately.

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

The accepted 2.18 SDK/Kubernetes SDK extension minimum is Python **>=3.10**,
delivered in [#14865](https://github.com/kubeflow/pipelines/pull/14865).
Prefer Python >=3.11 for newly migrated environments. Check whether the selected
candidate includes the change, then compare the SDK package's `Requires-Python` and dependency metadata against
notebooks, CI submitters and component base images that install that package. A
server image rollout does not upgrade those environments. Track pending dependency
PRs separately until selected; a proposed Python minimum is not evidence of a
landed release requirement. Distinguish SDK host requirements from user containers
that do not install the SDK.

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

Use the [SDK and custom-client migration examples](upgrade-2.18.md#sdk-and-custom-client-migration)
for private mirrors, embedded links, attachment rendering and archive consumers.
Inspect download consumers expecting inline HTML/SVG or automatically decompressed
S3 archives; verify actual response headers and bytes with authorized sample reads.

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

Distinguish the [database migration paths](upgrade-2.18.md#database-migration-paths).
Schema changes and the pipeline-reference backfill run synchronously before the new
API process serves requests. Inspect engine/version, `run_details`/`jobs` sizes,
backup/recovery readiness and evidence of migration/lock duration. Large-table DDL
and updates can block writes; the backfill's transactional completion marker does
not serialize all schema changes. Background schedule adoption is a later step,
not proof of a lock-free or zero-downtime database upgrade. Do not run DDL or mutate
migration markers as part of this read-only assessment.

For the recurring-run lookup index, verify the deployed image includes the
online JobUUID-index helper delivered by [#14874](https://github.com/kubeflow/pipelines/pull/14874). Inspect the existing `JobUUID` type and
`idx_run_details_job_uuid` definition without changing them. That helper preserves
existing column types and valid indexes; fresh tables use `varchar(255)`. Missing
indexes use MySQL `ALGORITHM=INPLACE, LOCK=NONE` with a 191-character prefix, or
PostgreSQL `CREATE INDEX CONCURRENTLY`. Check startup diagnostics for advisory-lock
contention, metadata-lock timeouts or build-deadline failures. It stops startup
rather than falling back to a copying or write-blocking build. Account for startup
duration, brief metadata locks and I/O; do not treat this one online operation as
proof that every startup migration is online.

Inspect actual retention settings; default zero disables automatic retention.
For pre-release images, verify inclusion of merged [#14877](https://github.com/kubeflow/pipelines/pull/14877)
before describing index preparation as automatic. In 2.18, retention or
`RUNS_GC_INDEX_PREPARE_ONLY=true` starts background preparation; both off does no
work. Inspect `idx_run_gc_lifecycle` and `idx_run_gc_archived`, including MySQL
visibility and PostgreSQL validity, plus preparation/timeout/retry diagnostics.
Check `RUNS_GC_INDEX_BUILD_TIMEOUT` (default `12h` per index attempt), resource
headroom and consistent prepare-only settings across replicas. Do not execute DDL
or activate retention as part of this assessment.

For retention indexes, interrupted PostgreSQL builds matching the exact managed
definition recover automatically. Active DDL defers recovery; conflicting index
definitions need inspection. Do not run manual DDL alongside the preparer. This
does not change the separate JobUUID startup-index recovery procedure.

Flag configured retention that was paused by missing indexes: automation can now
resume archival/deletion after preparation and a full GC interval. Recommend
prepare-only staging for backlog review; it blocks collection in updated replicas
but cannot pause an old leader before its rollout. Index creation and catch-up are
separate workloads. Conflicting definitions or custom index storage settings still
require diagnosis; do not
turn those exceptions into a manual-SQL requirement for ordinary upgrades.

Check persistence-agent scoped identity lookup and retained Workflow evidence for
active runs. Follow [reporting recovery](workflow-report-recovery.md) for inconsistent
ownership; a UID alone cannot recover lost terminal results. Record deferred
TaskService authorization remediation and the deferred artifact-ownership fix as
retained security limitations.

The 3.0 MLMD-to-native transition is separate. Replacement-history import does not
migrate active workloads, seed cache reuse or move artifact objects. Preserved
URIs do not themselves establish access. The
[master storage-authorization work](https://github.com/kubeflow/pipelines/pull/14883)
and [backend artifact API design](https://github.com/kubeflow/pipelines/issues/14031)
remain separate 3.0 work; the proposed 2.18 backport is excluded. Assess the exact
3.0 candidate's contract and require fresh storage/proxy isolation acceptance in
the replacement installation. Do not report successful import or table creation
as complete upgrade or artifact-access acceptance.

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
