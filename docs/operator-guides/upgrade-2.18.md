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
| Legacy multi-user schedules | Jobs without API-owned scheduling state no longer execute. This includes disabled jobs when later enabled. Audit mode does not bypass the requirement. | Inventory, review, and recreate them through the API before their next execution. See [schedule migration](scheduled-service-accounts.md#upgrade-and-revocation). |
| API, controller, and RBAC | Multi-user scheduled executions go through the API under the controller's identity. Custom accounts require scoped authorization for both the creator and controller. | Apply the complete release manifests together, retain `--multiUser=true` and removal of controller `workflows/create`, and update custom roles. See [recurring-run accounts](scheduled-service-accounts.md) and [RBAC migration](rbac-migration-2.18.md). |
| Pipeline-name lookups | Multi-user Kubernetes-native storage no longer falls back to the installation namespace when a namespace is omitted. Private SDK lookups also need an explicit namespace. | Update SDK and REST callers; do not rely on `Client(namespace=...)` to scope `get_pipeline_id()`. See [lookup migration](#make-pipeline-lookup-namespaces-explicit). |
| UI signing key | The complete default manifests automatically create or reuse a persistent signing key and start the UI when it is available. First adoption briefly interrupts the UI and invalidates old TensorBoard URLs. | Apply the complete manifests, wait for the UI rollout, then reopen TensorBoard. No manual key generation is needed. Custom overlays and GitOps have separate checks. See [UI startup](#prepare-the-ui-signing-key). |

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

Use the final deployment revision, not only an earlier candidate. Check schedule
recreation and a real scheduled execution, scoped custom-account authorization,
private pipeline lookups, and UI/TensorBoard startup. Follow the API checks in the
[RBAC migration guide](rbac-migration-2.18.md#verify-before-releasing-the-upgrade-to-users).
These checks supplement installation-specific workload and upgrade testing; they
do not establish that all security remediation is complete.
