# Upgrade to KFP 2.18

Use this checklist when planning an upgrade from an earlier KFP release to 2.18.
It describes the 2.18 release branch; select a published release tag from the
[releases page](https://github.com/kubeflow/pipelines/releases) for production
deployment. This guide is not a declaration that the release or its acceptance
testing is complete.

:::{warning}
Do not upgrade only the container images. Multi-user installations need coordinated
API server, ScheduledWorkflow controller, and RBAC changes. Existing recurring
runs can stop submitting even while the UI shows them enabled. The complete
manifests also initialize the UI signing key automatically; image-only upgrades
omit that setup.
:::

## Required migration actions

| Area | What changes | Operator action |
| --- | --- | --- |
| Legacy multi-user schedules | Jobs without API-owned scheduling state no longer execute. This includes disabled jobs when later enabled. Audit mode does not bypass the requirement. | Apply the complete manifests for automatic adoption of stored definitions and progress. Investigate and recreate affected definitions if that baseline is unsuitable. See [schedule migration](scheduled-service-accounts.md#upgrade-and-revocation). |
| API, controller, and RBAC | Multi-user scheduled executions go through the API under the controller's identity. Custom accounts require scoped authorization for both the creator and controller. | Apply the complete release manifests together, retain `--multiUser=true` and removal of controller `workflows/create`, and update custom roles. See [recurring-run accounts](scheduled-service-accounts.md) and [RBAC migration](rbac-migration-2.18.md). |
| UI signing key | The complete default manifests automatically create or reuse a persistent signing key and start the UI when it is available. First adoption briefly interrupts the UI and invalidates old TensorBoard URLs. | Apply the complete manifests, wait for the UI rollout, then reopen TensorBoard. No manual key generation is needed. Custom overlays and GitOps have separate checks. See [UI startup](#prepare-the-ui-signing-key). |

## Plan the multi-user rollout

### Standard managed installations: automatic schedule adoption

Apply the complete 2.18 manifests through the normal managed rollout. **Ordinary
upgrades do not require an operator stop/migrate/restart sequence, a migration
Job, or per-user schedule recreation.** The API continues serving ordinary
requests while adoption runs in the background; normal deployment availability
still depends on your replica count and rollout settings.

While the managed writer handoff is pending, the upgraded API returns retryable
`Unavailable` for schedule creation and enable/disable requests before changing
SQL or Kubernetes state. Retry them after the rollout; they are not queued as
accepted intent during the wait. Run operations, reads, reporting, and deletion
remain available. Existing legacy schedules retain old behavior until handoff,
while the new controller waits. This is an automatic scheduling-write pause, not
an operator-managed outage.

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

Migration processes bounded pages, including disabled schedules, with a timeout
for each record. Background and submission-triggered retries share per-record
capped exponential backoff. Failed records are logged and retried without
preventing later records from migrating or blocking ordinary API availability. A schedule whose
state or Kubernetes synchronization is incomplete remains blocked until its own
migration succeeds; healthy and newly created schedules are independent.

After handoff, each API process also performs one paginated startup repair of
existing Kubernetes schedule objects from trusted SQL state, including records
already marked ready. Each attempt is bounded. Repair does not recreate missing
objects, reseed progress, or acknowledge a pending execution; failed records remain
eligible for ordinary reconciliation. Legacy records without trusted SQL state
still use the 2.18 adoption checks before repair can proceed.

1. Back up the database, ScheduledWorkflow objects, deployment configuration, and
   signing Secret. Inventory schedules, intended accounts, and custom roles.
2. Apply the complete release configuration, images, and RBAC. Retain the managed
   API/controller Pod identity (`POD_NAME` and `POD_NAMESPACE` from the Kubernetes
   downward API) and handoff permissions, controller
   `--multiUser=true`, restriction against direct `workflows/create`, and pipeline
   read permissions. Keep API/controller scheduling timezones consistent.
3. Wait for the Deployments to roll out. Inspect API/controller logs for handoff
   waits and per-schedule adoption retries. Resolve incompatible remaining Pods,
   missing RBAC, or failed rollouts rather than bypassing the handoff. The
   controller exits on invalid Pod identity or denied handoff permissions; correct
   the configuration or RBAC and allow Kubernetes to restart it.
4. Verify that enabled schedules submit their next ticks, disabled schedules stay
   disabled, retained progress and concurrency limits are honored, and no duplicate
   execution appears. Check both previously active and idle schedules. Investigate
   persistent record-specific migration errors; do not reset counters or delete
   state/receipts to force migration.

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
writers sharing the database. Custom overlays must retain the handoff configuration
and permissions and account for every writer; renamed or unmanaged deployments
need an equivalent coordinated migration plan. Image-only upgrades omit required
RBAC and configuration.

The offline `--adopt-legacy-recurring-runs` command remains a recovery tool, not a
prerequisite for normal startup. Use its detailed recovery procedure only when
needed, with old API/controller writers stopped and outstanding transactions
drained. Never run it concurrently with uncontrolled writers. Missing or mismatched
execution records require investigation or report recovery, not counter resets.
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

## Prepare the UI signing key

### Standard manifests: automatic setup

You do not need to generate a key, copy it into a Secret, or manually order the
initializer and UI startup. Applying the complete release manifests includes the
Secret, initializer Job, and its narrowly scoped permissions. The initializer
generates the key only when it is absent and preserves an existing valid key.
The UI waits for the required Secret field and then starts automatically.

After applying the complete manifests as part of the rollout above, wait for the
UI Deployment. For the default `kubeflow` namespace:

```bash
kubectl -n kubeflow rollout status deployment/ml-pipeline-ui
```

Use your installation namespace if different. First adoption briefly interrupts
the UI and invalidates old TensorBoard proxy URLs; reopen TensorBoard from the UI
after the rollout. Keep the default `Recreate` strategy for this first upgrade.
Subsequent restarts reuse the key; ordinary manifest reapplication does not rotate
it. Include the signing Secret in your normal backups.

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

## Verify before reopening access

Use the final deployment revision, not only an earlier candidate. Check the chosen schedule
migration path and a real scheduled execution, scoped custom-account authorization,
private pipeline lookups, and UI/TensorBoard startup. Follow the API checks in the
[RBAC migration guide](rbac-migration-2.18.md#verify-before-releasing-the-upgrade-to-users).
These checks supplement installation-specific workload and upgrade testing; they
do not establish that all security remediation is complete.

TaskService authorization remediation is deferred and is not included in 2.18.
The namespace-isolation and artifact-authorization changes described above do not
close that gap. Include this retained limitation in your deployment security
review and release communication.
