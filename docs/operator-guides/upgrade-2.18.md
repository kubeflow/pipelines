# Upgrade to KFP 2.18

Upgrade KFP through your usual deployment process using the
[2.18 release manifests](https://github.com/kubeflow/pipelines/releases/tag/2.18.0).
The standard multi-user deployment handles recurring-run adoption automatically,
and the UI manifests initialize the shared signing key. You do not need a separate
migration Job, manual API-server/controller ordering, or blanket schedule recreation.

This guide covers preparation, expected rollout behavior, and checks after the
upgrade. Use the feature-specific sections for the integrations you have enabled.

## Before upgrading

1. **Back up your databases and deployment configuration.** Include the UI signing
   Secret if it already exists. See [database preparation](#database-migration-paths)
   for the automatic index builds and the separate, one-time reference backfill.
2. **Review your customizations.** Compare custom overlays with the complete release
   manifests, including RBAC, environment variables, Secret references, and Jobs.
   Use the [compatibility checklist](#check-applicable-compatibility-changes) below
   for custom roles, storage, clients, and enabled retention.
3. **Expect a brief UI interruption.** The default `Recreate` strategy replaces
   the old UI Pods before starting the new ones. First shared-key adoption also
   invalidates old TensorBoard URLs; reopen viewers after the rollout.
4. **If you also upgrade SDKs, update those environments separately.** A server
   upgrade does not update installed SDK packages or previously compiled
   components. Before recompiling with the new SDK, review the
   [SDK changes](#sdk-and-custom-client-migration), particularly preview privacy
   and private package mirrors.

## What happens during a normal upgrade

Apply the release through the same installation or GitOps process you normally
use, preserving your storage configuration and installation-specific settings.
For custom overlays, incorporate the release's configuration changes as well as
its image versions.

| Component | Expected behavior | What to check |
| --- | --- | --- |
| API server and database | New API Pods finish startup schema checks, lookup-index preparation, and any pending reference backfill before becoming ready. Retention indexes are prepared afterward in the background. | New Pod readiness and database load. Online index construction does not require stopping existing API replicas. See [database migration paths](#database-migration-paths). |
| Multi-user recurring runs | The API and controller coordinate the writer handoff and adopt eligible stored schedules in the background. Enabled/disabled state and retained execution progress are preserved. | Enabled schedules resume, disabled schedules stay disabled, and no duplicate execution appears. Schedule changes may wait up to 30 seconds during handoff. |
| UI and TensorBoard | The initializer creates the signing key if absent; the UI starts when it is available. Existing valid keys are preserved. | Wait for UI rollout, then reopen TensorBoard if an old URL no longer works. |

To check the standard deployments, replace `kubeflow` with your installation
namespace if needed:

```bash
kubectl -n kubeflow rollout status deployment/ml-pipeline
kubectl -n kubeflow rollout status deployment/ml-pipeline-scheduledworkflow
kubectl -n kubeflow rollout status deployment/ml-pipeline-ui
```

Use your own Deployment names if customized. Successful rollout is the first
check; also verify the workloads and integrations below.

(verify-before-reopening-access)=
## Verify after the upgrade

- Submit a representative pipeline and confirm that its terminal status appears
  in KFP. Check retained runs as well as newly submitted ones.
- If you use recurring runs, check an enabled schedule's next execution and that
  a disabled schedule stays disabled. Check retained progress and concurrency
  behavior for those schedules.
- Open the UI and test the viewers and storage paths you use: TensorBoard,
  artifact previews/downloads, and archived logs after their original Pod is gone.
  You do not need to enable unused features to verify the upgrade.
- Test applicable custom service accounts, roles, storage integrations, and
  private-pipeline lookups. The [RBAC guide](rbac-migration-2.18.md) provides
  permission checks.
- If retention is enabled, confirm the configured windows and inspect index
  preparation and collection activity. To review a backlog before collection
  resumes, configure [prepare-only mode](#automatic-retention-index-preparation)
  before rollout.

Keep original Workflows until KFP has recorded their terminal state. If reporting
or schedule adoption stalls, preserve the affected records and logs and follow
[reporting recovery](workflow-report-recovery.md) or
[scheduling diagnostics](scheduled-service-accounts.md#upgrade-and-revocation).
Do not delete history, reset counters, or edit ownership fields to force progress.

## Check applicable compatibility changes

| If you use… | Review before rollout |
| --- | --- |
| Custom execution accounts or workflow identities | Service-account and workflow-identity policies default to `enforce`. Check scoped grants for execution, pipeline references, logs, and TensorBoard. Stock roles are updated; custom roles may need changes. Temporary audit modes reduce protection and must be consistent across API replicas. See [RBAC migration](rbac-migration-2.18.md) and [scheduled service accounts](scheduled-service-accounts.md). |
| Legacy V1/raw-Argo caching | Entries with unknown ownership are no longer reused under enforcement, so affected tasks may execute again. A normal rolling update is supported; wait for all old cache-server processes to exit before treating enforcement as complete. Native V2 caching is unaffected. See [cache upgrades](https://github.com/kubeflow/pipelines/blob/release-2.18/backend/src/cache/README.md#namespace-isolation-and-upgrades). |
| Run retention | Previously configured retention can resume after missing indexes are prepared. Review [prepare-only mode](#automatic-retention-index-preparation) if you want to inspect the backlog first. |
| Custom artifact storage | Preserve credential placement and verify both artifact reads and archived logs. See [storage configuration](#custom-storage-configuration-and-rollout). |
| Large pipeline specifications | Uploads, extracted specifications, and update bodies have separate 32 MiB default limits. Overrides are bounded at 128 MiB. Existing object-store specifications used by later executions are also checked. See [size limits](server-config.md#pipeline-size-limits). |
| Legacy V1 metrics | Each metrics file has a configurable 1 MiB default limit. An oversized file can leave metrics missing from an otherwise completed task. See [metrics limits](server-config.md#legacy-metrics-file-limit). |
| Saved pagination tokens | Affected listings must restart. The updated UI retries once; SDK/custom clients must handle `PAGINATION_RESTART_REQUIRED` without appending restarted results to an old traversal. See [pagination recovery](pagination-recovery.md). |
| Direct pipeline-by-name API calls | Supply the owning namespace for private pipelines. See [lookup compatibility](#custom-api-integrations-pipeline-name-lookup-compatibility). |

## Database migration paths

KFP 2.18 retains MLMD. The index builds below run automatically and do not require
an offline database migration. Distinguish a new API Pod waiting for initialization
from database-wide unavailability: online index construction permits concurrent
reads and writes, although it consumes resources and briefly acquires metadata
locks.

### Automatic startup schema and data changes

For a 2.17-to-2.18 upgrade, each new API process checks the current schema and
prepares the recurring-run lookup index before serving requests. Existing API
replicas are not deliberately stopped for this preparation.

The recurring-run lookup index, `idx_run_details_job_uuid`, is built online
at startup. It preserves existing `JobUUID` column types and values and reuses a
valid index. MySQL uses a 191-character index prefix with online DDL; PostgreSQL
uses a concurrent full-column index. The prefix does not shorten stored values.
The helper bounds metadata-lock waits and has a 30-minute build deadline; failure
stops that API process with diagnostics rather than falling back to a blocking
index build. Retention index preparation follows a separate background path,
described below.

**The pipeline-reference backfill is a separate, one-time operation.** When the
`backfill_pipeline_refs` completion marker is absent, startup scans `run_details`
and `jobs` to populate missing pipeline and version IDs from historical references.
The updates run in one transaction and can make concurrent writes wait until it
finishes. This can delay the new API Pod's readiness; it is not an index rebuild.
The transaction also records completion, so later startups skip the scans and an
interrupted attempt retries without marking partial work complete. For a large
history, rehearse this first backfill against a representative database and observe
its duration and write latency. Do not rerun it manually or remove its marker.

Upgrades from a pre-2.15 schema also take the older column-conversion and backfill
path. That is a different starting state from the normal 2.17-to-2.18 upgrade;
review that legacy path separately before skipping intervening upgrades.

If initialization fails, inspect API startup logs and database diagnostics before
retrying. An interrupted PostgreSQL lookup-index build can require repair of an
invalid index. Do not delete migration markers or edit stored references to force
startup. See the [lookup-index reference](https://github.com/kubeflow/pipelines/blob/release-2.18/docs/agents/development.md#recurring-run-lookup-index)
for definitions, deadlines, and recovery details.

### Automatic retention index preparation

`RUNS_RETENTION_TIME` and `ARCHIVED_RUNS_RETENTION_TIME` are disabled by default.
When retention is configured, KFP prepares its required indexes in the background
before collection; this does not delay API startup. Failed preparation pauses
collection and retries. MySQL uses `ALGORITHM=INPLACE, LOCK=NONE`; PostgreSQL
uses concurrent index builds. Both permit concurrent application reads and writes.
Monitor database load while preparation runs; it does not require pausing normal
pipeline use.

**Retention that was already configured but paused because indexes were missing
can start archiving or deleting runs after this upgrade.** If you want to inspect
the backlog before resuming collection, set `RUNS_GC_INDEX_PREPARE_ONLY=true`
consistently on every API replica and installation sharing the database. Finish the rollout before relying
on that setting; an old leader does not obey another replica's configuration.
Review the backlog and database load, then clear prepare-only when ready.
Collection waits a full `RUNS_GC_INTERVAL` after preparation or a leadership change.

With both retention windows disabled and prepare-only off, no retention indexes
are built and no collection runs. Conflicting index definitions need inspection;
coordinate manual DDL with all preparers. See
[retention configuration](server-config.md#run-retention-and-index-preparation)
for timeouts, staged activation, and PostgreSQL recovery.

### Retained PostgreSQL image overlays

PostgreSQL deployments retain the
[MLMD concurrency limitation](https://github.com/kubeflow/pipelines/issues/14353).
Updated database images do not resolve it or establish production support.

For existing PostgreSQL users, the supplied images move from 18.3 to 18.6 for the
KFP database and from 14.7 to 14.24 for the legacy metadata database. Review the
[cumulative PostgreSQL 14 migration notes](https://www.postgresql.org/docs/14/release.html)
from your installed version. Conditional actions include index repairs described
in [14.10](https://www.postgresql.org/docs/release/14.10/) and extension,
logical-decoding, or encrypted-data follow-up in
[14.24](https://www.postgresql.org/docs/release/14.24/).

The legacy image also moves from Alpine 3.17 to 3.23. If you use ICU collations,
check for version mismatches and rebuild affected objects before refreshing their
recorded collation version. Refreshing the version alone does not repair indexes.
See [PostgreSQL collation guidance](https://www.postgresql.org/docs/14/sql-altercollation.html#SQL-ALTERCOLLATION-NOTES).

## SDK and custom-client migration

### Python environments installing the SDK

The SDK and companion packages support **Python 3.9 or newer**. Python 3.9 uses
Requests 2.32.5 and urllib3 2.6.3, which retain known vulnerabilities whose fixes
require a newer interpreter. Prefer newer Python where possible, refresh dependency
locks, and test compilation and submission when changing it. See the
[security tracker](https://github.com/kubeflow/pipelines/issues/14421) for accepted
limitations.

Updating the server does not update installed SDKs, interpreters, or compiled
component images. Generic container components that do not install the SDK acquire
no Python requirement from the server upgrade.

### Private pip mirrors

Newly compiled lightweight components and component-image builds no longer infer
`--trusted-host` from `pip_index_urls`. Prefer HTTPS with the mirror's CA installed
in the component image. If bypassing certificate verification is intentional, set
trusted hosts explicitly:

```python
pip_index_urls=["https://mirror.example/simple"],
pip_trusted_hosts=["mirror.example"],
```

Use bare hostnames, not URLs. An omitted value or `None` means no trusted hosts.
Test in the component environment: the notebook's CA configuration does not apply
to the container. Already compiled installation commands remain unchanged.

### Embedded artifact links

Embedded artifacts must be self-contained. A link such as
`weights.bin -> ../shared/weights.bin` can fail compilation because it escapes
the embedded tree. Copy the authorized file into the tree, replace directory
symlinks with real directories, and repair dangling links. In-tree file symlinks
remain supported and are archived as regular files.

### Download and rendering integrations

Artifact and pod-log downloads use `Content-Disposition: attachment`. Integrations
that embed raw HTML/SVG download URLs may receive a download instead of an inline
page. Use built-in previews or an appropriately isolated viewer; do not remove the
attachment protection. The former View All link is removed.

S3-compatible downloads preserve original archive bytes. Consumers that previously
received decompressed `tar.gz` content must recognize and decompress archives
explicitly. Test a known archive against its stored bytes; do not blindly decompress
every response. These changes also apply to downloads of historical artifacts.

## Automatic artifact previews and request limits

Automatic previews are enabled by default for supported Python-component Dataset
and Model outputs. Dataset previews can include up to ten rows and twenty columns;
model previews can include configuration. For sensitive outputs, choose the
appropriate control and recompile:

- `PipelineConfig(artifact_previews=False)` disables previews for a pipeline.
- `@dsl.component(artifact_previews=False)` disables them for a component.
- `artifact_preview_rows=False` produces schema-only dataset previews.

Disabling generation does not remove existing previews. The updated SDK and
launcher are both needed to upload the sibling `<artifact-uri>.kfp-preview.json`
file; older launchers do not upload it. Historical artifacts are not backfilled,
and preview-upload failure does not fail the task. See
[Dataset and Model previews](../user-guides/data-handling/artifacts.md#dataset-and-model-previews).

Previews and downloads share a **300-request-per-minute limit per user per UI
server process**, or per client IP without authenticated identity. Clients must
handle HTTP 429 and respect `Retry-After`. This is not a cluster-wide quota.
Generated namespace artifact servers disable the duplicate limiter; custom
receivers should follow the same arrangement. See
[request-limit configuration](server-config.md#artifact-request-limits).

## Artifact compatibility and retained exposure

### Ownership behavior retained from 2.17

KFP 2.18 retains the existing MLMD-based artifact ownership behavior. It does not
require moving historical artifacts, adding namespace prefixes to tracked custom
roots, or enrolling storage in a new authorization system.

The ownership limitation remains: an authenticated tenant may obtain another
namespace's objects when ownership evidence is absent or forgeable and the serving
storage identity can read those objects. Known foreign namespace evidence still
denies access; missing or unavailable evidence retains the previous permissive
behavior. The untracked-artifact prefix and `mlmd-only` settings remain, but
`mlmd-only` does not make every unavailable-metadata case fail closed.

See [artifact ownership compatibility](https://github.com/kubeflow/pipelines/blob/release-2.18/frontend/README.md#artifact-ownership-compatibility-in-218)
for the complete behavior. TaskService authorization remediation is also not
included in 2.18. Include these limitations in your deployment security assessment;
other namespace-isolation fixes do not resolve them.

### Protections that remain

Authentication, namespace permissions, path validation, trusted endpoint and
redirect restrictions, credential controls, and filesystem protections still
apply. They do not establish complete object ownership. Keep storage grants and
network access scoped to the intended tenants; different Secret names do not
necessarily mean different permissions.

### Custom-storage configuration and rollout

Preserve the integration and credential placement that currently serve each
namespace. With namespace HTTP artifact proxying, credentials stay on the namespace
receiver; do not move them to the shared UI merely to upgrade. Direct-serving
installations keep their HTTP settings on the serving UI.

Check exact trusted origins, HTTP base paths, redirects, and credentials using the
[HTTP artifact migration guide](server-config.md#http-artifact-migration-for-218).
An allowlist grants neither credentials nor bucket access. Verify that updated
receivers have rolled out before relying on their protections.

After changing endpoint or HTTP settings, restart the relevant UI/profile-controller
processes and reconcile existing namespace proxies. A controller restart alone
does not enqueue every namespace: wait for the hourly resync or annotate affected
namespaces, then check the new Pod template before rollout status. Preserve custom
Secret mounts and authentication through your own provisioning configuration.
See the [custom-storage rollout procedure](https://github.com/kubeflow/pipelines/blob/release-2.18/manifests/kustomize/README.md#upgrade-example-custom-s3-storage).

Archived logs use a separate path through the shared UI. Test an archived log after
its original Pod is gone; a successful namespace artifact read does not verify the
shared UI's network access or credentials.

## Multi-user scheduling during the upgrade

### Standard managed installations: automatic schedule adoption

The API and controller coordinate their rollout and adopt database-stored schedule
definitions automatically, preserving enabled/disabled state and retained progress.
Existing tasks need not finish first, and no history cleanup is required. Large
histories can take longer to process. Individual inconsistent records can remain
blocked while healthy schedules and ordinary API requests continue.

During writer handoff, schedule creation, enable/disable operations, and transfer
preparation can wait up to 30 seconds, subject to the client's deadline and
cancellation. If handoff completes, the same request proceeds. If the wait expires,
schedule changes return HTTP 503/gRPC `Unavailable` with the typed
`SCHEDULE_HANDOFF_IN_PROGRESS` reason before changing the schedule. The updated UI
and Python SDK retry this condition with a bounded budget.

A rejected **disable leaves the schedule enabled**. Retry the explicit rejection
and confirm the final state. A generic timeout is ambiguous: check whether a create
succeeded before resubmitting it. Transfer preparation can already have staged
earlier records; resume the same archive after resolving the handoff problem.
See [handoff and retry details](scheduled-service-accounts.md#upgrade-and-revocation).

If adoption stalls, inspect the API/controller diagnostics for failed rollouts,
permissions, incompatible writers, or inconsistent records. Do not reset counters
or delete runs, Workflows, or adoption receipts to force progress. Do not reintroduce
legacy writers after handoff; a downgrade needs a separately planned rollback.

### Customized deployments and recovery

Renaming a Deployment alone needs no extra setting. Discovery uses component labels
and standard container names. If those also change, configure the same complete
`KFP_SCHEDULE_WRITERS` inventory on every API/controller process, including old and
new writers. Separate installations, relabeled processes, or external writers
sharing a database need a plan covering every writer. See
[custom writer identities](scheduled-service-accounts.md#renamed-managed-deployments).

Custom resources must include the release RBAC, controller API routing, downward-API
Pod identity, and consistent timezones. A custom controller readiness probe must
not depend on its post-handoff listener, which would create a circular wait.
The [schedule guide](scheduled-service-accounts.md#upgrade-and-revocation) covers
these checks and the offline recovery command; that command is not a normal
upgrade prerequisite and must not run alongside uncontrolled writers.

Adoption trusts existing database definitions as the installation's baseline; it
does not prove that their historical creators were authorized. Investigate suspect
definitions before enabling them. CR-only schedules without API jobs need creation
through the API by an authorized caller. Account for existing executions before
enabling replacements: disabling a schedule does not stop its runs.

(make-pipeline-lookup-namespaces-explicit)=
## Custom API integrations: pipeline-name lookup compatibility

Direct pipeline-by-name REST/gRPC calls must specify the owning namespace for
private pipelines. Kubernetes-native multi-user lookup rejects an omitted namespace;
SQL lookup without one searches shared pipelines only. A same-named shared pipeline
can otherwise be returned instead of the intended private one.

Search custom integrations for `/apis/v2beta1/pipelines/names/`,
`pipeline_service_get_pipeline_by_name`, or `GetPipelineByName`. Correct omitted
namespaces and verify results with two profiles that have the same pipeline name.
Permission errors require reviewing access, not trying other namespaces.

The high-level SDK's `get_pipeline_id()` and CLI calls using it perform filtered
listing, not this endpoint, so they do not need this endpoint migration. Existing
pipeline/version IDs remain valid. `REQUIRE_NAMESPACE_FOR_PIPELINES=true`
independently rejects omitted namespaces rather than permitting shared lookup.

(private-pipeline-client-convenience)=
### Optional SDK/CLI convenience for private pipelines

For a client whose by-name upload helper lacks a namespace selector, resolve the
private pipeline explicitly and use its ID:

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

`get_pipeline_id()` does not inherit the client namespace; omitting its namespace
argument searches shared pipelines. Use the resolved `--pipeline-id` for CLI
commands, and an experiment in the owning namespace for a private run. New private
pipeline uploads also need their own explicit `namespace=` argument. See
[RBAC and client migration](rbac-migration-2.18.md).

(prepare-the-ui-signing-key)=
## UI signing-key behavior

### Standard manifests: automatic setup

The initializer creates the shared signing key if absent and preserves a valid
existing key. The UI waits for it automatically. First adoption briefly interrupts
the UI and invalidates old TensorBoard URLs; reopen TensorBoard after rollout.
Later restarts reuse the key, but UI updates still use `Recreate` by default unless
you configure rolling updates after adoption.

Include the signing Secret in backups. A brief `CreateContainerConfigError` while
the key is initialized can clear automatically. If startup stays blocked, follow
[signing-key troubleshooting](server-config.md#signing-key-startup-and-recovery)
rather than deleting or regenerating the key.

### Additional checks only for customized deployments

- Custom overlays must include the signing Secret, initializer Job, RBAC, and UI
  Secret reference. Retain `Recreate` for first adoption.
- GitOps must preserve generated Secret data instead of pruning or replacing it.
  See [GitOps configuration](server-config.md#custom-overlays-and-gitops).
- Externally managed keys use the [signing-key configuration](server-config.md#externally-managed-signing-keys),
  not object-store credentials.
- Rolling UI updates are optional after adoption; follow the
  [two-phase procedure](server-config.md#enabling-rolling-ui-updates).

## Optional ownership inventory and reporting recovery

The [read-only ownership scan](workflow-report-recovery.md) can help investigate
legacy records or reporting problems. It is optional, not an upgrade prerequisite
or pass/fail gate. Protect its output, which contains resource identifiers, and
investigate incomplete or unknown results rather than treating missing responses
as proof of absence.

Preserve original Workflows and reporter evidence until KFP records terminal
status. After restoring connectivity or permissions, verify that the same run IDs
finish reporting with their original experiment and schedule ownership. Recovery
of a deleted Workflow needs a retained, trustworthy terminal snapshot; a UID alone
cannot reconstruct lost results. Follow [reporting recovery](workflow-report-recovery.md)
when evidence is missing instead of editing ownership or relaxing authorization.
