# Upgrade to KFP 2.18

Use this checklist when planning an upgrade from an earlier KFP release to 2.18.
It describes the 2.18 release branch; select a published release tag from the
[releases page](https://github.com/kubeflow/pipelines/releases) for production
deployment. This guide is not a declaration that the release or its acceptance
testing is complete.

KFP 2.18 uses the normal installation upgrade process. The managed rollout
handles eligible legacy schedule adoption and UI signing-key initialization
automatically. No separate migration Job, manual sequencing of API/controller
updates, or closure of user access is needed for a standard managed upgrade.
Normal deployment availability still depends on replica counts and rollout settings.
For a resource-by-resource assessment, use the
[agent cluster-impact assessment guide](upgrade-2.18-agent-assessment.md). It is an
optional review procedure, not an additional upgrade prerequisite.

## What happens during a normal upgrade

| Area | Automatic behavior | What to expect |
| --- | --- | --- |
| Legacy multi-user schedules | The API adopts eligible database-stored definitions and retained progress, preserving enabled/disabled state. The controller handles the writer handoff automatically. | Schedule creation and enable/disable requests can temporarily return retryable `Unavailable`. Individual records with inconsistent state need investigation. See [schedule adoption](#standard-managed-installations-automatic-schedule-adoption). |
| API, controller, and RBAC | The release configuration routes multi-user scheduled executions through the API under the controller's identity. | If you use custom execution accounts or custom roles, check their scoped permissions. Standard deployment wiring is supplied by the release. See [recurring-run accounts](scheduled-service-accounts.md) and [RBAC migration](rbac-migration-2.18.md). |
| UI signing key | The initializer creates or reuses a persistent signing key; Kubernetes starts the UI when it is available. | First adoption briefly interrupts the UI and invalidates old TensorBoard URLs; reopen TensorBoard after the rollout. Existing valid keys are preserved. See [UI signing-key behavior](#prepare-the-ui-signing-key). |

## Check applicable compatibility changes

Review these before rollout when the corresponding features are in use:

| Area | Preparation |
| --- | --- |
| Service-account and workflow-identity policy | Inventory custom runner, template, and plugin identities and test their scoped grants. Both controls default to `enforce`; temporary `audit` modes restore the associated exposure and are planned for removal in 3.0. Keep modes consistent across API replicas. See [service-account migration](scheduled-service-accounts.md#temporary-audit-mode-for-migration) and [workflow-identity audit scope](server-config.md#audit-mode-for-rollout). Audit does not bypass schedule adoption or the independent [RBAC/client changes](rbac-migration-2.18.md). |
| Legacy V1/raw-Argo cache | For a 2.17.2 upgrade, apply the standard rolling update normally; no manual cache-server shutdown or submission pause is required. Wait for every old cache-server process to terminate before treating enforcement as fully active. Cache reuse can decrease during overlap. Default enforcement stops reusing historical entries of unknown ownership: extra computation occurs when steps next execute and miss the cache, not as a startup rebuild. Native V2 caching is unaffected. See [cache rollout and temporary audit mode](https://github.com/kubeflow/pipelines/blob/release-2.18/backend/src/cache/README.md#namespace-isolation-and-upgrades). |
| Multi-user artifact roots | Matching MLMD ownership alone no longer permits arbitrary object roots. Existing keys must include the namespace path component; configure a matching existing prefix or migrate objects and references. See [artifact-root migration](#artifact-root-migration). |
| Custom artifact storage | Configure exact trusted S3 origins and the HTTP artifact base where applicable; a permissive domain regex alone is insufficient for custom S3 origins. Propagate configuration to existing profile proxies and verify artifacts and archived logs in those profiles. See [endpoint configuration](server-config.md#artifact-storage-endpoint-allowlist) and [HTTP/profile migration](server-config.md#http-artifact-migration-for-218). |
| Pipeline size ceilings | Check uploads, extracted specifications (including existing object-store specifications used by later executions), and update bodies against their separate 32 MiB defaults. Supported overrides are bounded at 128 MiB; size memory and the entire request path before raising them. There is no audit/unlimited mode. See [settings and actionable errors](server-config.md#pipeline-size-limits). |
| Legacy metrics files | V1 metrics collection bounds each metrics file at 1 MiB by default. Check `MAX_METRICS_FILE_BYTES` on the collector; an oversized file can leave metrics missing from an otherwise completed task. This is not the native V2 metrics path. See [legacy metrics limits](server-config.md#legacy-metrics-file-limit). |
| Persisted run ownership and reporting | For an additional inventory or to investigate legacy ownership, use the optional read-only `--include-ownership` scan. Preserve Workflows until terminal status is recorded in KFP; investigate missing or inconsistent ownership without editing stored identities. See [reporting recovery](workflow-report-recovery.md). |
| Saved pagination tokens | Plan to restart affected listings after the API rollout. The updated UI retries once; SDK/custom clients must handle `PAGINATION_RESTART_REQUIRED` and discard or reconcile earlier traversal results before restarting. Updating the server does not update installed clients. See [pagination recovery](https://github.com/kubeflow/pipelines/blob/release-2.18/docs/operator-guides/pagination-recovery.md). |

## Artifact-root migration

This applies to authenticated multi-user artifact previews and downloads, including
requests through tenant proxies. Standalone mode is unchanged. In 2.18, a matching
MLMD namespace is necessary evidence but no longer sufficient for an arbitrary
object key: the key must also have the namespace prefix. The default is
`private-artifacts/<namespace>/...`.

Inspect representative existing artifact references, including old runs users still
open. Do not infer compatibility from new pipeline defaults alone:

- If keys already start with `private-artifacts/<namespace>/`, no root migration is
  needed. Verify real preview/download with the owning user's credentials.
- If keys consistently use another prefix followed by `/<namespace>/`, configure
  `ARTIFACT_NAMESPACE_KEY_PREFIX` to that prefix and verify the effective value on
  each authenticated serving UI replica. The setting does not insert a missing namespace component.
- For arbitrary legacy roots, configure namespace-prefixed roots for new runs,
  copy/move only authorized historical objects, and update their references using
  your storage/metadata migration process. Changing metadata does not move objects.
  Verify historical and new reads before removing old copies.

`KFP_SECURITY_ARTIFACT_OWNERSHIP_MODE=audit` is a temporary 2.18 migration option,
with `enforce` as the default. It permits only an absent prefix after successful
MLMD lookup and matching namespace evidence for every returned artifact. It does
not bypass user authorization, explicit namespace mismatches, unsafe paths, missing
context evidence or MLMD failures. An unavailable MLMD service can therefore deny
reads even when the key is correctly prefixed. Restore the service; do not disable
authentication to recover reads.

Audit is not complete security remediation: matching metadata can still reference
another tenant's object when downstream credentials are shared. Use it only after
independently verifying storage and proxy/network isolation, with a migration
owner and deadline. Keep the mode consistent across authenticated serving replicas, monitor
`artifact_ownership_audit` events, and return to `enforce` after verifying migrated
reads. An audit event is permission to attempt retrieval, not proof of a successful
download. This exception is absent from the 3.0 policy. See the
[artifact ownership details](https://github.com/kubeflow/pipelines/blob/release-2.18/frontend/README.md#artifact-ownership-migration-in-218).

Endpoint trust, credentials and object ownership are separate checks. Updating
`ALLOWED_ARTIFACT_ENDPOINTS` or `HTTP_BASE_URL` does not migrate object paths.
After changing environment-based storage settings, restart the UI and profile
controller as applicable. Existing profile proxies need a namespace reconciliation;
a controller restart alone does not enqueue them. Wait for the hourly resync or
annotate each affected namespace, then verify the new proxy pod template **before**
checking its rollout status. Follow the
[custom-store rollout procedure](https://github.com/kubeflow/pipelines/blob/release-2.18/manifests/kustomize/README.md#upgrade-example-custom-s3-storage).
For HTTP artifacts and archived logs, also verify that the shared UI has the
required network access and appropriate credentials; tenant-proxy access alone
does not establish either. Test an archived log after the original Pod is gone.

## Multi-user scheduling during the upgrade

### Standard managed installations: automatic schedule adoption

Upgrade through your normal deployment process. Schedule adoption runs in the
background while the API continues serving ordinary requests. KFP handles the
writer handoff automatically; you do not need to order API and controller
updates yourself or recreate eligible schedules.

While the managed writer handoff is pending, the upgraded API returns retryable
`Unavailable` for schedule creation and enable/disable requests before changing
the schedule's SQL or Kubernetes state. A create request may already have created
its default experiment before reaching this schedule guard. Retry them after the rollout; they are not queued as
accepted intent during the wait. Run operations, reads, reporting, and deletion
remain available. Existing legacy schedules retain old behavior until handoff,
while the new controller waits. This is an automatic scheduling-write pause, not
an operator-managed outage.

The server identifies this pre-mutation rejection with gRPC `Unavailable` (code
14), HTTP 503, and `google.rpc.ErrorInfo` reason `SCHEDULE_HANDOFF_IN_PROGRESS`
in domain `kubeflow.org`. The updated UI and Python SDK retry only this complete
condition, up to five attempts with 1, 2, 4 and 8 second delays. They stop on other
errors and surface exhaustion; older installed clients still need explicit retry.
Updating server images does not upgrade the SDK installed in users' environments.

An explicit handoff rejection means the operation was not applied: a rejected
**disable leaves the schedule enabled**. Retry with bounded exponential backoff,
then read the schedule back to confirm its state. Surface an exhausted retry
budget and investigate the rollout. A generic connection timeout is ambiguous;
check whether a create succeeded before resubmitting it.

The upgraded controller automatically waits for the managed API/controller
Deployments to finish rolling out and for incompatible or terminating writer Pods
to leave. Each upgraded process registers its own Pod; operators must not copy
registration annotations into Pod templates. After this handoff, adoption restores
API routing for legacy schedules while preserving their enabled/disabled state
and retained execution progress. Existing tasks do not need to finish.

Each process caches its first successful handoff; a new process verifies it again.
This assumes legacy writers will not be reintroduced after handoff. A downgrade
to legacy writers requires a separate coordinated rollback plan, not a live
rolling downgrade.

Migration includes disabled schedules. Background and submission-triggered retries
share per-record capped exponential backoff. Failed records are logged and retried
without preventing later records from migrating or blocking ordinary API
availability. A schedule whose state or Kubernetes synchronization is incomplete
remains blocked until its own migration succeeds; healthy and newly created
schedules are independent.

**No history cleanup is required before upgrading.** Adoption streams retained SQL
runs and reads all namespace Workflows in pages of 100 without a total-item cap.
It keeps compact identity/progress metadata rather than full manifests, so larger
histories still need more time and memory. The background worker can take longer
to reach later schedules while processing a large history. Stalled reads time out;
a progressing background inventory has no fixed total deadline.

An interrupted inventory retries from the beginning without committing partial
progress. After progress and its receipt are committed, retries resume
synchronization without importing history again. Preserve evidence if adoption
reports inconsistent records. Never delete active runs or Workflows, reset
counters, or remove scheduling state or receipts to force adoption; see
[retained-history adoption](scheduled-service-accounts.md#upgrade-and-revocation).

After handoff, each API process also performs one paginated startup repair of
existing Kubernetes schedule objects from trusted SQL state, including records
already marked ready. Each attempt is bounded. Repair does not recreate missing
objects, reseed progress, or acknowledge a pending execution; failed records remain
eligible for ordinary reconciliation. Legacy records without trusted SQL state
still use the 2.18 adoption checks before repair can proceed.

As with other upgrades, keep backups appropriate to your installation. After the
rollout, verify that enabled schedules submit their next ticks, disabled schedules
stay disabled, retained progress and concurrency limits are honored, and no
duplicate execution appears. Include both previously active and idle schedules.

If rollout or adoption stalls, inspect API/controller logs for the affected Pod
or schedule. Resolve failed rollouts, missing permissions or incompatible writers
identified by the diagnostics. Persistent record-specific errors need investigation;
do not reset counters or delete state/receipts to force migration. The controller
exits on invalid Pod identity or denied handoff permissions, so correct that
configuration and let Kubernetes restart it.

Adoption accepts **database-stored definitions as the existing installation's
baseline**. It does not prove that historical creators were authorized to submit
those definitions. Current pipeline, namespace, execution-policy, and
service-account checks still apply; adoption grants no new custom-account access.
If the installation may have been compromised or this baseline is unsuitable,
investigate and recreate affected definitions from independently reviewed sources.
CR-only schedules without an API job still require API creation by an authorized
caller. Account for existing executions before enabling replacements: disabling a
schedule does not stop its runs.

### Customized deployments and recovery

The automatic handoff covers the standard `ml-pipeline` and
`ml-pipeline-scheduledworkflow` Deployments and their labeled Pods in the
installation namespace. It does not discover separate installations or external
writers sharing the database. For renamed managed Deployments, configure the same
`KFP_SCHEDULE_WRITERS` identity inventory on every API/controller process and grant
the documented namespace-scoped ReplicaSet reads and reads of each configured
Deployment (update stock Role `resourceNames` restrictions). Include retiring writer identities
when renaming during upgrade; never omit an old writer to make handoff pass. See
[renamed managed Deployments](https://github.com/kubeflow/pipelines/blob/release-2.18/docs/operator-guides/scheduled-service-accounts.md#renamed-managed-deployments).
Unmanaged writers still need a migration plan covering every writer. When maintaining your own
deployment resources, incorporate the release changes to API/controller RBAC,
`--multiUser=true`, controller pipeline-read permissions and the removal of direct
controller `workflows/create`. Retain `POD_NAME` and `POD_NAMESPACE` from the
Kubernetes downward API and keep scheduling timezones consistent. These checks
apply to customized wiring; the standard deployment already supplies it.

Custom controller readiness probes must become ready independently of handoff or
adoption. The standard controller has no readiness probe; waiting for its
post-handoff HTTP listener creates a circular wait. API health starts independently
of background adoption.

The offline `--adopt-legacy-recurring-runs` command remains a recovery tool, not a
prerequisite for normal startup. Use its detailed recovery procedure only when
needed, with old API/controller writers stopped and outstanding transactions
drained. Never run it concurrently with uncontrolled writers. Missing or mismatched
execution records require investigation or [report recovery](workflow-report-recovery.md),
not counter resets.
Do not restore old writable APIs after adoption or restart an old controller
against adopted state. Adoption cannot detect or undo all changes from an earlier
unsafe mixed-version rollout.

The [detailed schedule guide](scheduled-service-accounts.md#upgrade-and-revocation)
includes inventory queries, account-specific grants, revocation behavior, and
recovery limits. KFP 2.18 retains V1 workflows where namespace policy permits;
this migration is not a requirement to convert every pipeline to V2.

(make-pipeline-lookup-namespaces-explicit)=
## Custom API integrations: pipeline-name lookup compatibility

This is a **targeted compatibility warning for custom API integrations**, not a
general operator upgrade task or a requirement to update ordinary SDK/CLI calls.
It applies to integrations calling the pipeline-by-name endpoint directly, or
through generated REST/gRPC clients, and relying on an omitted namespace:

- With multi-user Kubernetes-native storage, an omitted namespace is rejected
  instead of falling back to the KFP installation namespace.
- With SQL storage, an omitted namespace searches only shared pipelines, not
  private pipelines in other namespaces. Same-named private pipelines can no
  longer be returned by a request for a shared pipeline.

For custom integrations, search for `/apis/v2beta1/pipelines/names/`, generated
`pipeline_service_get_pipeline_by_name` methods, or `GetPipelineByName` calls.
Supply the owning namespace when requesting a private pipeline. A missing
namespace error requires correcting the request; a permission denial requires
reviewing access, not trying other namespaces. Check successful calls too, since
a same-named shared pipeline may exist. Verify the intended authorized result
using two profiles with the same pipeline name.

The high-level SDK's `get_pipeline_id()` and the CLI commands that use it call
filtered `ListPipelines`, **not this endpoint**. They did not rely on these unsafe
backend defaults and do not need a compatibility migration for these fixes.
Repository review found no handwritten UI caller of the affected endpoint; the
API integration tests already supply a namespace. This does not establish which
external integrations are deployed in a particular installation.

Existing pipeline and version IDs do not need to be recreated. Intentional shared
SQL lookups retain their scope, and ID-based calls retain normal authorization.
The independent `REQUIRE_NAMESPACE_FOR_PIPELINES=true` setting rejects omitted
namespaces rather than allowing shared lookup.

(private-pipeline-client-convenience)=
### Optional SDK/CLI convenience for private pipelines

Separate from the backend fixes, some clients' version-upload helpers and CLI
`--pipeline-name` operations search only shared pipelines and offer no pipeline
namespace selector. This is an existing usability limitation, not a new upgrade
breakage. The client/global experiment namespace does not select their lookup
scope. To use a private pipeline, resolve its ID explicitly and pass that ID:

```python
import kfp

namespace = "team-a"
client = kfp.Client(namespace=namespace)  # Configure host/credentials as usual.
pipeline_id = client.get_pipeline_id("training", namespace=namespace)
if pipeline_id is None:
    raise ValueError(
        f"Pipeline 'training' was not found in namespace {namespace!r}; "
        "check its name and namespace before uploading a version."
    )
client.upload_pipeline_version(
    pipeline_package_path="pipeline.yaml",
    pipeline_version_name="revision-2",
    pipeline_id=pipeline_id,
)
```

`get_pipeline_id()` does not inherit the client namespace. Omitting the argument
searches shared pipelines; it is not a cross-namespace private lookup. For CLI
commands, use the resolved `--pipeline-id`; a private run also needs an experiment
in the owning namespace. A wrong shared lookup can return a same-named shared
pipeline rather than `None`.

If a client offers `namespace=` on version uploads or `--pipeline-namespace` on
CLI by-name operations, it can make the scope explicit directly. Check that
client's API documentation or command `--help` before using those options;
server upgrades do not upgrade installed SDKs or CLI tools. The explicit lookup
followed by an ID-based call above works without those convenience options.
Adding those options is not a prerequisite for the backend security fixes.

Private uploads also need their own explicit `namespace=` argument, and stored
private references must remain in the owning namespace. See the
[RBAC and client migration guide](rbac-migration-2.18.md).

(prepare-the-ui-signing-key)=
## UI signing-key behavior

### Standard manifests: automatic setup

The standard deployment includes the Secret, initializer Job and its permissions.
The initializer generates the key only when it is absent and preserves an existing
valid key. The UI waits for the required Secret field and then starts automatically;
no manual key creation or startup ordering is needed.

To check UI rollout progress in the default `kubeflow` namespace:

```bash
kubectl -n kubeflow rollout status deployment/ml-pipeline-ui
```

Use your installation namespace if different. First adoption briefly interrupts
the UI and invalidates old TensorBoard proxy URLs; reopen TensorBoard from the UI
after the rollout. The default `Recreate` strategy handles this first adoption.
Subsequent restarts reuse the key; ordinary manifest reapplication does not rotate
it. `Recreate` remains the default for later UI updates too, so those updates still
interrupt the UI unless you opt into rolling updates after shared-key adoption. Include the signing Secret in your normal backups.

A brief `CreateContainerConfigError` while the initializer creates the key is
expected and clears automatically. If startup remains blocked, follow
[signing-key troubleshooting](server-config.md#signing-key-startup-and-recovery);
do not delete or regenerate the key as the first troubleshooting step.

### Additional checks only for customized deployments

- **Custom overlays or image-only upgrades:** include the signing Secret,
  initializer Job and RBAC, and required UI Secret reference from the complete
  release manifests. Retain `Recreate` for first adoption; changing only images
  is insufficient.
- **GitOps:** preserve the generated Secret data; do not prune, replace, or
  force-recreate the signing Secret. Use the
  [GitOps guidance](server-config.md#custom-overlays-and-gitops) if your reconciler
  treats generated key data as drift.
- **Externally managed keys:** use the
  [external-key configuration](server-config.md#externally-managed-signing-keys),
  not an object-store credential.
- **Rolling UI updates:** optional after shared-key adoption, not a prerequisite
  for this upgrade. Follow the separate
  [two-phase procedure](server-config.md#enabling-rolling-ui-updates).

## Optional ownership inventory and reporting recovery

For additional pre-upgrade inventory or to investigate reporting problems, use
the [read-only ownership scan](workflow-report-recovery.md). It is optional and
does not need to run before an ordinary managed upgrade. It inspects
active/unknown-state runs and all schedules through their API projections. It makes no writes and does not need
database access or Kubernetes Workflow reads. Keep its output protected because
it contains resource identifiers.

This is an inventory aid, not an upgrade pass/fail gate: `observed` means evidence
was returned, not that a report will be accepted. `review_required` and `unknown`
need investigation; a missing API response is not proof that a record is absent.
Collection is bounded and non-atomic, and namespace-scoped listings may omit
orphaned records. Reports remain incomplete (exit 2); compare them with your
existing inventory and repeat incomplete scopes after correcting collection errors.

Preserve original Workflows and persistence-agent evidence until KFP records their
terminal status. After restoring temporary API connectivity or scoped lookup
permissions, verify the same run IDs reach their terminal state without losing
experiment or recurring-run ownership. A deleted Workflow can be recovered only
when the reporter still has a trustworthy terminal snapshot matching persisted
identity. A UID alone cannot reconstruct lost results. Do not edit ownership
fields, reset schedule counters, or relax authentication to make recovery pass.
The {ref}`reporting recovery guide <before-and-after-the-upgrade>`
explains diagnosis, replacement limits, and when to escalate unresolved runs.

(verify-before-reopening-access)=
## Verify after the upgrade

Use the deployed revision, not only an earlier candidate. Check a real scheduled
execution and UI/TensorBoard startup. Where applicable, also verify custom-account
permissions and custom integrations that look up private pipelines by name.
Follow the applicable API checks in the
{ref}`RBAC migration guide <verify-before-releasing-the-upgrade-to-users>`.
Verify persistence reporting catches up for retained runs as well as confirming
that schedules fire. Health probes, a completed read-only scan, and passing unit
tests do not establish live upgrade acceptance. The separate
[reporting recovery acceptance fixture](https://github.com/kubeflow/pipelines/blob/release-2.18/tools/upgrade-readiness/REPORTING_RECOVERY.md)
is a mutating disposable-cluster test, never a production readiness command. It
checks recovery after a temporary lookup denial and the specific deletion race
where the worker has already captured a terminal report. Release qualification
should inspect its reports for the exact candidate.
Operators do not need to run this fixture as part of their upgrade; it does not
prove recovery when the terminal snapshot was lost before capture.

These checks supplement installation-specific workload and upgrade testing; they
do not establish that all security remediation is complete.

TaskService authorization remediation is deferred and is not included in 2.18.
The namespace-isolation and artifact-authorization changes described above do not
close that gap. Include this retained limitation in your deployment security
review and release communication.
