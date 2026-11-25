# Troubleshoot pod failures

When the pod behind a pipeline task has trouble starting or running, the run graph tells
you why, so you don't have to dig through `kubectl describe pod` first.

## Where to look

A task with a pod problem gets a status icon on its node in the run graph:

- **Amber warning:** a known infrastructure problem that is still unresolved. The task
  may recover on its own.
- **Red error:** the task failed. If the cause was a pod problem, the reason is attached
  to the node.
- **Blue info:** a status that Kubeflow Pipelines could not classify. The raw status text is
  shown as it came from the pod.

Hover over the icon to see the category and the message. Long messages are shortened in
the tooltip. To read the full message, select the node and then select **Details** in the
notice that appears in the side panel. The amber or blue icon disappears when the problem clears
or the task finishes successfully.

## What each category means

| Category | Typical message | Common causes | What to try |
|---|---|---|---|
| Image pull problem | `ErrImagePull`, `ImagePullBackOff`, `Back-off pulling image` | Wrong image name or tag, missing registry credentials, registry unreachable | Check the image reference, add `imagePullSecrets`, confirm the cluster can reach the registry |
| Scheduling problem | `Unschedulable`, `FailedScheduling`, `Insufficient cpu`, `Insufficient memory` | The task requests more CPU, memory or accelerators than any node has, node taints, namespace quota | Lower the resource request, check node capacity and taints, check the namespace quota |
| Runtime problem | `OOMKilled`, `CrashLoopBackOff` | The container ran out of memory or keeps crashing | Raise the memory limit, read the task logs |
| Admission problem | `forbidden`, admission webhook, PodSecurity, Kyverno | A cluster policy rejected the pod | Check the policy that rejected the pod and the namespace labels it relies on |
| Pod status (blue) | Any other status text | A pod-level reason the UI does not classify, such as `CreateContainerConfigError` from a missing ConfigMap or Secret | Open the task's Details panel for the full message, or look at the pod events |

A scheduling problem is sometimes temporary. If the cluster uses an autoscaler such as
Karpenter, a pod can stay unschedulable for a few minutes while a node is created. The
amber warning clears once the pod is scheduled.

## What happens automatically

Pods stuck on an image pull failure can fail the run on their own. When image pull failure
handling is enabled on the persistence agent, a run whose pod keeps failing to pull an
image is terminated after a grace period (300 seconds by default). Set
`IMAGE_PULL_FAILURE_GRACE_PERIOD_SEC` on the persistence agent deployment to change it. The
workflow is annotated with `pipelines.kubeflow.org/termination-reason: ImagePullFailure`
and `pipelines.kubeflow.org/failed-image`. See the Kustomize manifests README for how to
enable it.

Other categories are shown on the graph but do not stop the run by themselves.

## Getting more detail

The graph shows the latest message for a task. For the full picture, use `kubectl`:

```bash
# Events and container states for the pod behind a task
kubectl describe pod <pod-name> -n <namespace>

# Logs from the previous attempt of a restarted container
kubectl logs <pod-name> -n <namespace> --previous
```
