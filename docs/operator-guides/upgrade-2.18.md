# Upgrade to KFP 2.18

Use this checklist when planning an upgrade from an earlier KFP release to 2.18.
It describes the 2.18 release branch; select a published release tag from the
[releases page](https://github.com/kubeflow/pipelines/releases) for production
deployment. This guide is not a declaration that the release or its acceptance
testing is complete.

:::{warning}
Do not upgrade only the container images. Multi-user installations need coordinated
API server, ScheduledWorkflow controller, and RBAC changes. Existing recurring
runs can stop submitting even while the UI shows them enabled, and the UI cannot
start until its signing-key Secret is populated.
:::

## Required migration actions

| Area | What changes | Operator action |
| --- | --- | --- |
| Legacy multi-user schedules | Jobs without API-owned scheduling state no longer execute. This includes disabled jobs when later enabled. Audit mode does not bypass the requirement. | Inventory, review, and recreate them through the API before their next execution. See [schedule migration](scheduled-service-accounts.md#upgrade-and-revocation). |
| API, controller, and RBAC | Multi-user scheduled executions go through the API under the controller's identity. Custom accounts require scoped authorization for both the creator and controller. | Apply the complete release manifests together, retain `--multiUser=true` and removal of controller `workflows/create`, and update custom roles. See [recurring-run accounts](scheduled-service-accounts.md) and [RBAC migration](rbac-migration-2.18.md). |
| Pipeline-name lookups | Multi-user Kubernetes-native storage no longer falls back to the installation namespace when a namespace is omitted. Private SDK lookups also need an explicit namespace. | Update SDK and REST callers; do not rely on `Client(namespace=...)` to scope `get_pipeline_id()`. See [lookup migration](#make-pipeline-lookup-namespaces-explicit). |
| UI signing key | Default manifests inject one persistent key into every UI replica through a required Secret field. First adoption briefly interrupts the UI and invalidates old TensorBoard URLs. | Deploy the initializer and its RBAC, preserve the Secret, wait for initialization and UI rollout, then reopen TensorBoard. See [UI startup](#prepare-the-ui-signing-key). |

## Plan the multi-user rollout

1. Back up the database, deployment configuration, and existing signing Secret.
   Inventory schedules, their intended inputs and accounts, and any custom roles.
   Use independently reviewed pipeline and parameter sources, not an unreviewed
   copy of a mutable ScheduledWorkflow or old database row.
2. **Block recurring-run creation and recreation for all clients before the
   rollout.** Keep that block until all old API replicas have stopped and their
   in-flight persistence reports have drained. An old API can overwrite stored
   schedule inputs during a mixed-version rollout, even for a disabled schedule.
3. Upgrade the API server and ScheduledWorkflow controller together with the
   complete release manifests and scoped RBAC. Keep their scheduling timezone
   configuration consistent. Wait for old controller pods to stop before granting
   custom-account access or enabling replacement schedules.
4. After the new API has started, inspect its `recurring_run_migration` inventory
   and perform the [database/CR cross-check](scheduled-service-accounts.md#inventory-before-completing-the-upgrade).
   An inventory error is not evidence of zero affected schedules. Also review any
   schedules created during mixed-version overlap: the missing-state inventory
   does not detect all of them.
5. Once old API replicas and reports are drained, resume API creation. Disable
   each old schedule through the API and recreate its reviewed configuration.
   Keep replacements disabled until old controllers have stopped and existing
   executions are accounted for. Disabling a schedule does not stop its runs.
   Do not seed scheduling-state rows or reset counters to bypass recreation.
6. Enable a reviewed replacement and verify execution, its selected account,
   namespace, and concurrency behavior before migrating the remaining schedules.
   Recheck the inventory and retire obsolete schedules through the API when their
   history is no longer needed.

The [detailed schedule guide](scheduled-service-accounts.md#upgrade-and-revocation)
includes inventory queries, account-specific grants, revocation behavior, and
recovery limits. KFP 2.18 retains V1 workflows where namespace policy permits;
this migration is not a requirement to convert every pipeline to V2.

## Make pipeline lookup namespaces explicit

With an SDK that provides the 2.18 `namespace` argument, pass the owning namespace
for a private pipeline even if the client already has a default:

```python
import kfp

namespace = "team-a"
client = kfp.Client(namespace=namespace)  # Configure host/credentials as usual.
pipeline_id = client.get_pipeline_id("training", namespace=namespace)
```

`get_pipeline_id()` does not inherit the client namespace. Omitting the argument
searches shared pipelines; it is not a cross-namespace private lookup. Upgrade
older clients before relying on this argument. For direct pipeline-by-name REST
requests against multi-user Kubernetes-native storage, supply the namespace query
parameter explicitly; omission now fails instead of selecting the API server's
installation namespace. Test with two profiles containing the same pipeline name
and verify that each caller resolves only its intended, authorized pipeline.

Private uploads also need their own explicit `namespace=` argument, and stored
private references must remain in the owning namespace. See the
[RBAC and client migration guide](rbac-migration-2.18.md).

## Prepare the UI signing key

The default manifests create `ml-pipeline-ui-tensorboard-proxy` and an initializer
Job with narrowly scoped Secret permissions. All UI replicas read its
`signing-secret` field through a non-optional `secretKeyRef`. Include these
resources when updating custom overlays; an image-only update is insufficient.

Wait for the rendered initializer Job to complete, then wait for the UI Deployment
rollout. Its Job name includes a generated hash, so use the name from the rendered
manifests or installed Jobs rather than assuming a fixed suffix. A temporary
`CreateContainerConfigError` while the key is absent is expected; a failed Job or
persistent error requires investigation before routing UI traffic.

Preserve the Secret and generated key data across applies, GitOps reconciliation,
and backups. Do not prune or force-recreate it, or replace the key with an object
store credential. Retain `Recreate` for first adoption; enable rolling updates only
after every UI pod uses the same persistent key. Reopen TensorBoard from the UI
after the first migration because previously issued proxy URLs are invalidated.

Follow the [signing-key guide](server-config.md#tensorboard-proxy-signing-secret)
for external Secret configuration, initialization failures, GitOps settings,
rotation, and the two-phase switch to rolling updates.

## Verify before reopening access

Use the final deployment revision, not only an earlier candidate. Check schedule
recreation and a real scheduled execution, scoped custom-account authorization,
private pipeline lookups, and UI/TensorBoard startup. Follow the API checks in the
[RBAC migration guide](rbac-migration-2.18.md#verify-before-releasing-the-upgrade-to-users).
These checks supplement installation-specific workload and upgrade testing; they
do not establish that all security remediation is complete.
