# High Availability

The Kubeflow Pipelines manifests run one replica of each component and define no
PodDisruptionBudgets (PDBs). This guide lists the components you can run with more
than one replica, shows an example overlay that adds replicas, pod anti-affinity, and
PDBs, and explains which components must stay at one replica.

:::{tip}
Replica counts depend on your load and availability targets, so this guide doesn't
recommend specific numbers. The default manifests are unchanged; apply the settings
below in your own Kustomize overlay.
:::

## Which components can run more than one replica

| Component | Deployment | More than one replica | Notes |
| --- | --- | --- | --- |
| API server | `ml-pipeline` | Yes | Serves requests statelessly. Each replica reconciles ScheduledWorkflow resources once at startup and [skips conflicting updates](https://github.com/kubeflow/pipelines/blob/ab274d0347eff91695fd74f24d68c18a6b8e22b6/backend/src/apiserver/resource/resource_manager.go#L1031). Run garbage collection, when enabled, uses [leader election](https://github.com/kubeflow/pipelines/blob/ab274d0347eff91695fd74f24d68c18a6b8e22b6/backend/src/apiserver/gc/run_gc.go#L216), so only one replica collects. See [installs and upgrades](#install-and-upgrade-the-api-server-with-one-replica). |
| UI | `ml-pipeline-ui` | Yes | Serves requests statelessly. All replicas share the [TensorBoard proxy signing secret](server-config.md#tensorboard-proxy-signing-secret). |
| Profile controller (multi-user) | `kubeflow-pipelines-profile-controller` | Yes | Stateless metacontroller webhook. Metacontroller never syncs the same namespace concurrently. Use a release that includes [kubeflow/pipelines#14809](https://github.com/kubeflow/pipelines/pull/14809) (2.18.0 and later), which adds [a health endpoint and a threaded server](https://github.com/kubeflow/pipelines/blob/ab274d0347eff91695fd74f24d68c18a6b8e22b6/manifests/kustomize/base/installs/multi-user/pipelines-profile-controller/sync.py#L619-L649). |
| Scheduled workflow controller | `ml-pipeline-scheduledworkflow` | **No** | See [components that must stay at one replica](#components-that-must-stay-at-one-replica). |
| Persistence agent | `ml-pipeline-persistenceagent` | **No** | See [components that must stay at one replica](#components-that-must-stay-at-one-replica). |
| Viewer controller | `ml-pipeline-viewer-crd` | **No** | See [components that must stay at one replica](#components-that-must-stay-at-one-replica). |

### Install and upgrade the API server with one replica

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

Upgrades that change scheduling state have additional rollout steps; see
[scheduled service accounts](scheduled-service-accounts.md).

## Example overlay

The following overlay runs two replicas of each component that supports it, spreads
them across nodes, and adds a PDB for each. Replace `<version>` with your Kubeflow
Pipelines release, and the base with the environment you install.

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
patches:
- path: spread-ml-pipeline.yaml
- path: spread-ml-pipeline-ui.yaml
- path: spread-kubeflow-pipelines-profile-controller.yaml
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

The other two patches are identical apart from the Deployment name and `app` label.
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
# Repeat for ml-pipeline and kubeflow-pipelines-profile-controller.
```

## Pod disruption budgets

A [PDB](https://kubernetes.io/docs/tasks/run-application/configure-pdb/) limits
voluntary disruptions, such as node drains, so that enough replicas keep running.

:::{warning}
Don't use `minAvailable: 1` for a component that runs one replica. Evicting its only
Pod would violate the budget, so the eviction is always refused and node drains and
upgrades never complete. Use `maxUnavailable: 1`, which allows one Pod at a time to
be evicted at any replica count, or don't define a PDB for single-replica components.
:::

## Components that must stay at one replica

The scheduled workflow controller, persistence agent, and viewer controller don't use
leader election, so every replica would process the same objects:

- **Scheduled workflow controller**
  ([`controller.Run`](https://github.com/kubeflow/pipelines/blob/ab274d0347eff91695fd74f24d68c18a6b8e22b6/backend/src/crd/controller/scheduledworkflow/main.go#L169)):
  every replica would try to start each scheduled run.
- **Persistence agent**
  ([`controller.Run`](https://github.com/kubeflow/pipelines/blob/ab274d0347eff91695fd74f24d68c18a6b8e22b6/backend/src/agent/persistence/main.go#L161)):
  every replica would report the same workflow status updates to the API server.
- **Viewer controller**
  ([`manager.New`](https://github.com/kubeflow/pipelines/blob/ab274d0347eff91695fd74f24d68c18a6b8e22b6/backend/src/crd/controller/viewer/main.go#L86)):
  every replica would reconcile the same TensorBoard viewers.

Keep these at one replica. If one of their Pods stops, Kubernetes recreates it.
Pipeline runs that are already executing are not affected while it is down, but
scheduled runs don't start and run status isn't updated until it is back.

## Dependencies

Kubeflow Pipelines availability also depends on these components:

- The database (MySQL or PostgreSQL) and the object store each need their own
  high-availability setup, such as a managed service.
- In multi-user mode, metacontroller calls the profile controller. The bundled
  metacontroller runs one replica; to run more, enable its `--leader-election`
  [flag](https://metacontroller.github.io/metacontroller/guide/configuration.html).
- Argo Workflows runs pipeline workflows. See its
  [high-availability guide](https://argo-workflows.readthedocs.io/en/latest/high-availability/).
