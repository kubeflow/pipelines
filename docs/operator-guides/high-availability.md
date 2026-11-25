# High availability

The Kubeflow Pipelines manifests deploy one replica of each component and define no
PodDisruptionBudgets (PDBs). This guide lists the components you can run with more
than one replica, shows an example overlay that adds replicas, pod anti-affinity, and
PDBs, and explains which components must stay at one replica.

:::{tip}
Replica counts depend on your load and availability targets, so this guide doesn't
recommend specific numbers. Apply the settings below in your own Kustomize overlay;
this guide doesn't change the default manifests.
:::

## Which components can run more than one replica

| Component | Deployment | More than one replica | Notes |
| --- | --- | --- | --- |
| API server | `ml-pipeline` | Yes | Serves requests statelessly. Each replica reconciles ScheduledWorkflow resources once at startup and [skips conflicting updates](https://github.com/kubeflow/pipelines/blob/ab274d0347eff91695fd74f24d68c18a6b8e22b6/backend/src/apiserver/resource/resource_manager.go#L1031). Garbage collection of expired runs, when enabled, uses [leader election](https://github.com/kubeflow/pipelines/blob/ab274d0347eff91695fd74f24d68c18a6b8e22b6/backend/src/apiserver/gc/run_gc.go#L216), so only one replica collects. See [API server installs and upgrades](#api-server-installs-and-upgrades). |
| UI | `ml-pipeline-ui` | Yes | Serves requests statelessly. All replicas share the [TensorBoard proxy signing secret](server-config.md#tensorboard-proxy-signing-secret). The Deployment uses `RollingUpdate` with zero unavailable replicas and one surge Pod. Existing installations must first complete the [2.18 upgrade](installation.md#upgrading-existing-installations) to adopt the shared key; preserve it across upgrades and provide capacity for the surge Pod. |
| Profile controller (multi-user) | `kubeflow-pipelines-profile-controller` | Yes | Stateless metacontroller webhook. Metacontroller never syncs the same namespace concurrently. Use a release that includes [kubeflow/pipelines#14809](https://github.com/kubeflow/pipelines/pull/14809) (2.18.0 or later), which adds [a health endpoint and a threaded server](https://github.com/kubeflow/pipelines/blob/ab274d0347eff91695fd74f24d68c18a6b8e22b6/manifests/kustomize/base/installs/multi-user/pipelines-profile-controller/sync.py#L619-L649). |
| Scheduled workflow controller | `ml-pipeline-scheduledworkflow` | Yes | Uses leader election, which the default manifests enable: only the replica holding a Lease starts scheduled runs. See [scheduled workflow controller leader election](#scheduled-workflow-controller-leader-election). |
| Persistence agent | `ml-pipeline-persistenceagent` | **No** | See [components that must stay at one replica](#components-that-must-stay-at-one-replica). |
| Viewer controller | `ml-pipeline-viewer-crd` | **No** | See [components that must stay at one replica](#components-that-must-stay-at-one-replica). |

### API server installs and upgrades

Every API server replica checks and migrates the database schema when it starts.
These startup migrations are [not coordinated across replicas](https://github.com/kubeflow/pipelines/blob/ab274d0347eff91695fd74f24d68c18a6b8e22b6/backend/src/apiserver/client_manager/client_manager.go#L1358-L1364),
so avoid starting several API server replicas at the same time against a schema
that needs changes:

- For a fresh install or an upgrade that changes the schema, run the API server with
  one replica until it is ready, then scale it up.
- If you keep several replicas during an upgrade, use a rollout that starts one new
  Pod at a time, such as a `RollingUpdate` strategy with `maxSurge: 1` and
  `maxUnavailable: 0`. The default strategy (`maxSurge` and `maxUnavailable` of 25%)
  only does this for up to three replicas.

During a rolling upgrade, old and new API server replicas run at the same time. For
the steps that keep recurring runs consistent, see
[Upgrade and revocation](scheduled-service-accounts.md#upgrade-and-revocation).

#### Pipeline-reference backfill during a rollout

The updated API can become Ready while one replica repairs historical pipeline
references in the background. Already-running source replicas can continue to
create runs and report workflow status. A source binary with the older startup
backfill checks the completion marker at startup and ignores the worker's
advisory lock. Restarting that binary while the marker is absent can run a
whole-table update and block application writers.

Identify the deployed API images and database version, check the size of
`run_details` and `jobs`, and verify backup readiness. Watch batch progress in
the API logs and committed updates in `pipeline_ref_backfill_updates_total`.
For an already-blocked original 2.18.0 MySQL deployment, see the
[recovery procedure](mysql-2.18-upgrade.md).

Before mixing replicas, check the marker with a read-only query.

```sql
-- MySQL
SELECT 1 FROM migration_statuses WHERE Name = 'backfill_pipeline_refs';
-- PostgreSQL
SELECT 1 FROM migration_statuses WHERE "Name" = 'backfill_pipeline_refs';
```

If the row exists, the older startup backfill skips its sweep. If the row is
absent and the source image contains that startup backfill, keep old binaries
from starting until the worker commits the marker. This includes crash
replacements, scale-ups and rollbacks. If your deployment cannot enforce that
condition, stop all old API replicas before starting the updated image and plan
for an API interruption. After the marker appears, the restriction for this
backfill ends. Leave marker creation to the worker so interrupted repairs resume.

The worker repairs jobs first and preserves existing IDs. A restarted worker
resumes the remaining repairs. An old reporter can still insert a version-only
run after the final pass. The API resolves missing pipeline IDs for retrieval,
filtering, counts and pagination from direct references, the version's parent or
a repaired source job that still has the same version. If all those sources are
gone, the original parent cannot be recovered.

The worker reads up to 500 candidate keys per page. With MySQL binary logging
active, FULL row images can include both copies of a large manifest, so the
worker commits one row at a time. With statement-based logging, it uses one
primary key per transaction at REPEATABLE READ because InnoDB rejects writes at
READ COMMITTED in that mode. The same single-row policy applies when the
worker cannot read the logging settings. MySQL row-lock and metadata-lock waits on the
worker connection are limited to two seconds. After a hard kill or node loss,
MySQL can finish the active statement and retain the advisory lock until it
detects the disconnect.

### Scheduled workflow controller leader election

Only the replica holding the `kfp-scheduledworkflow` Lease starts scheduled runs. If
it stops renewing the Lease, another replica takes over once the Lease expires, 15
seconds after the last renewal. After a restart, the new Pod can wait up to 15
seconds for the previous Lease to expire before it starts scheduled runs.

Leader election does not guarantee that only one replica acts at a time: a replica
that pauses and then resumes can still send a request it had in progress. Duplicate
runs are rejected: in multi-user mode, the API server refuses a second claim of the
same scheduled run, and in single-user mode, a second `Create` of the same Workflow
fails with `AlreadyExists`. These protections also cover the brief overlap when you
upgrade from a release without leader election.

The default manifests set the `LEADER_ELECT` environment variable to `"true"`. To turn
leader election off, set the Deployment's `replicas` to 0, set `LEADER_ELECT` to
`"false"`, then set `replicas` to 1.

If your overlay replaces the controller's Role, add the Lease rules from the base Role,
or set `LEADER_ELECT` to `"false"`. If it replaces the controller's environment
variables, leader election stays off unless you add `LEADER_ELECT` and `POD_NAMESPACE`.

## Example overlay

The following overlay sets two replicas for each component that supports it, spreads
them across nodes, and adds a PDB for each. Replace `<version>` with your Kubeflow
Pipelines release (2.18.0 or later for the profile controller, and a release that
includes [kubeflow/pipelines#15043](https://github.com/kubeflow/pipelines/pull/15043)
for the scheduled workflow controller), and the base with the environment you install.

```yaml
# kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- github.com/kubeflow/pipelines/manifests/kustomize/env/platform-agnostic-multi-user?ref=<version>
- pdb.yaml
replicas:
- name: ml-pipeline
  count: 2
- name: ml-pipeline-ui
  count: 2
- name: kubeflow-pipelines-profile-controller
  count: 2
- name: ml-pipeline-scheduledworkflow
  count: 2
patches:
- path: spread-ml-pipeline.yaml
- path: spread-ml-pipeline-ui.yaml
- path: spread-kubeflow-pipelines-profile-controller.yaml
- path: spread-ml-pipeline-scheduledworkflow.yaml
```

Each `spread-<deployment>.yaml` patch adds preferred pod anti-affinity, so the
scheduler places replicas on different nodes when it can, but still schedules them
on a single-node cluster. For example, `spread-ml-pipeline-ui.yaml`:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ml-pipeline-ui
  namespace: kubeflow
spec:
  template:
    spec:
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - weight: 100
            podAffinityTerm:
              topologyKey: kubernetes.io/hostname
              labelSelector:
                matchLabels:
                  app: ml-pipeline-ui
```

The other spread patches are identical apart from the Deployment name and `app` label.
To spread replicas across zones instead of nodes, use
`topologyKey: topology.kubernetes.io/zone` or
[topology spread constraints](https://kubernetes.io/docs/concepts/scheduling-eviction/topology-spread-constraints/).

`pdb.yaml` defines one PDB per component:

```yaml
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: ml-pipeline-ui
  namespace: kubeflow
spec:
  maxUnavailable: 1
  selector:
    matchLabels:
      app: ml-pipeline-ui
---
# Repeat for ml-pipeline, kubeflow-pipelines-profile-controller, and
# ml-pipeline-scheduledworkflow.
```

## Pod disruption budgets

A [PDB](https://kubernetes.io/docs/tasks/run-application/configure-pdb/) limits
voluntary disruptions, such as node drains, so that enough replicas keep running.

:::{warning}
Don't use `minAvailable: 1` for a component that runs one replica. Evicting its only
Pod would violate the budget, so the eviction is always refused, and node drains,
including those during cluster or node upgrades, never complete. Use `maxUnavailable: 1`, which allows one Pod at a time to
be evicted at any replica count, or don't define a PDB for single-replica components.
:::

## Components that must stay at one replica

The persistence agent and viewer controller do not use leader election, so every
replica would process the same objects:

- **Persistence agent**
  ([`controller.Run`](https://github.com/kubeflow/pipelines/blob/ab274d0347eff91695fd74f24d68c18a6b8e22b6/backend/src/agent/persistence/main.go#L161)):
  every replica would report the same workflow status updates to the API server.
- **Viewer controller**
  ([`manager.New`](https://github.com/kubeflow/pipelines/blob/ab274d0347eff91695fd74f24d68c18a6b8e22b6/backend/src/crd/controller/viewer/main.go#L86)):
  every replica would reconcile the same TensorBoard viewers.

Keep these at one replica. If one of their Pods stops, Kubernetes recreates it.
Pipeline runs that are already executing continue in the meantime, but run status
is not updated and new TensorBoard viewers are not created until the corresponding Pod
is back.

## Dependencies

Kubeflow Pipelines availability also depends on these components:

- The database (MySQL or PostgreSQL) and the object store each need their own
  high-availability setup, such as a managed service.
- In multi-user mode, metacontroller calls the profile controller. The bundled
  metacontroller runs one replica without leader election, and its ClusterRole doesn't
  grant access to Leases. To run more replicas, enable its `--leader-election` flag and
  grant that access; see the
  [metacontroller configuration guide](https://metacontroller.github.io/metacontroller/guide/configuration.html).
- Argo Workflows runs pipeline workflows. See its
  [high-availability guide](https://argo-workflows.readthedocs.io/en/latest/high-availability/).
