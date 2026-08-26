# KEP: Dynamic Resource Allocation (DRA) Support for Kubeflow Pipelines

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [User Stories](#user-stories)
- [Proposal](#proposal)
  - [SDK API](#sdk-api)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Proto Schema](#proto-schema)
  - [Python SDK](#python-sdk)
  - [Backend Driver](#backend-driver)
  - [Runtime Behavior](#runtime-behavior)
- [Frontend Considerations](#frontend-considerations)
- [Test Plan](#test-plan)
<!-- /toc -->

## Summary

Add Kubernetes Dynamic Resource Allocation (DRA) support to Kubeflow Pipelines through the `kfp-kubernetes` SDK package, enabling pipeline authors to request hardware resources (GPUs, accelerators) via the DRA claim-based model. This follows the same pattern as existing Kubernetes-specific features like tolerations, node selectors, and node affinity.

## Motivation

Kubernetes [Dynamic Resource Allocation (DRA)](https://kubernetes.io/docs/concepts/scheduling-eviction/dynamic-resource-allocation/) is the successor to the device-plugin model for accelerator scheduling. DRA enables:

- Fine-grained GPU partitioning
- Multi-instance GPU (MIG)
- Vendor-specific resource allocation strategies
- Claim-based resource management with proper lifecycle semantics

As DRA adoption grows across Kubernetes distributions, KFP pipeline authors need a way to request DRA-managed resources for their pipeline tasks. Today, KFP supports requesting accelerators via `set_accelerator_limit()` / `set_accelerator_type()`, which maps to the legacy device-plugin model. There is no way to use DRA claims.

### Goals

- Provide SDK functions in `kfp-kubernetes` for pipeline authors to attach DRA resource claims to pipeline tasks.
- Support both static claims (known at compile time) and parameterized claims (resolved at pipeline runtime).
- Propagate resource claims to the pipeline task pod spec at runtime.
- Maintain full backward compatibility — existing pipelines without DRA claims continue to work unchanged.
- Support DRA claims through `TaskConfig` passthrough for external workloads, following the existing pattern for tolerations and volumes.

### Non-Goals

- Providing DRA resource lifecycle management (creating `ResourceClaim` or `ResourceClaimTemplate` objects). These are managed outside KFP by cluster administrators or external tooling.

## User Stories

### Pipeline author: request a GPU via DRA

As a pipeline author, I want to attach a DRA resource claim to my pipeline task so that the task pod gets a GPU allocated through the DRA framework.

```python
from kfp import dsl
from kfp import kubernetes

@dsl.component
def train_model():
    import torch
    device = torch.device("cuda")
    # ... training logic ...

@dsl.pipeline
def training_pipeline():
    task = train_model()
    kubernetes.add_resource_claim(
        task,
        resource_claim_template_name="gpu-claim-template",
    )
```

The cluster administrator has pre-created a `ResourceClaimTemplate` named `gpu-claim-template` that describes the GPU allocation requirements. At runtime, the pod gets a `resourceClaims` entry and the container references it, resulting in a GPU allocated via DRA.

### Pipeline author: parameterized claims for multi-environment pipelines

As a pipeline author, I want to pass DRA claim configuration as a pipeline parameter so the same compiled pipeline can run with different resource configurations across environments.

```python
from kfp import dsl
from kfp import kubernetes

@dsl.component
def train_model():
    ...

@dsl.pipeline
def training_pipeline(resource_claim: dict):
    task = train_model()
    kubernetes.add_resource_claim_json(task, resource_claim_json=resource_claim)
```

The caller provides the claim configuration at pipeline submission time:

```python
client.create_run_from_pipeline_func(
    training_pipeline,
    arguments={
        "resource_claim": kubernetes.ResourceClaimConfig(
            resource_claim_template_name="gpu-claim-template",
        )
    },
)
```

### Cluster administrator: enable DRA for KFP workloads

As a cluster administrator, I configure `DeviceClass` and `ResourceClaimTemplate` objects on the cluster. Pipeline authors reference these templates in their pipelines via `add_resource_claim()`. No changes to the KFP installation are required — DRA support works out of the box once the cluster has DRA enabled and drivers installed.

### Component author: DRA claims for external workloads via TaskConfig passthrough

As a component author building a passthrough component that launches an external workload (e.g., a training job on a remote cluster), I want DRA resource claims to be forwarded to my component via `TaskConfig` so the external workload can be configured with the same DRA resources.

```python
from kfp import dsl
from kfp.dsl import TaskConfigField, TaskConfigPassthrough

@dsl.component(
    task_config_passthroughs=[
        TaskConfigPassthrough(
            field=TaskConfigField.KUBERNETES_RESOURCE_CLAIMS,
            apply_to_task=True,
        ),
    ],
)
def launch_training(task_config: dsl.TaskConfig):
    # task_config.resourceClaims contains the DRA claims
    # Forward them to the external workload
    ...
```

The pipeline author attaches DRA claims as usual with `kubernetes.add_resource_claim()`. The backend routes the resolved claims to both the `TaskConfig` input parameter (for the external workload) and the task pod (because `apply_to_task=True`).

## Proposal

### SDK API

Two new functions in the `kfp-kubernetes` package:

**Static variant** — claim configuration known at compile time:

```python
kubernetes.add_resource_claim(
    task: PipelineTask,
    resource_claim_template_name: Optional[str] = None,
    resource_claim_name: Optional[str] = None,
) -> PipelineTask
```

- `resource_claim_template_name`: name of a `ResourceClaimTemplate` in the same namespace. A new `ResourceClaim` is created per pod and deleted with the pod.
- `resource_claim_name`: name of a pre-existing `ResourceClaim` in the same namespace.
- Exactly one of `resource_claim_template_name` or `resource_claim_name` must be provided.
- Can be called multiple times on the same task to attach multiple claims.
- The pod-local claim name (`PodResourceClaim.Name`) is not user-configurable. The backend driver auto-generates it as `<task-name>-<index>` at runtime (e.g., `train-model-0`, `train-model-1`).

**JSON variant** — claim configuration resolved at runtime:

```python
kubernetes.add_resource_claim_json(
    task: PipelineTask,
    resource_claim_json: Union[PipelineParameterChannel, ResourceClaimConfig, list[ResourceClaimConfig]],
) -> PipelineTask
```

- Accepts a pipeline parameter, a `ResourceClaimConfig` instance, or a list of `ResourceClaimConfig` instances.
- `ResourceClaimConfig` provides compile-time validation and IDE autocomplete, preventing typos from silently passing through to the backend.
- When a list is provided, each item is treated as a separate claim.

**`ResourceClaimConfig`** — typed claim configuration:

```python
class ResourceClaimConfig:
    def __init__(
        self,
        resource_claim_template_name: Optional[str] = None,
        resource_claim_name: Optional[str] = None,
    ):
        ...
```

- `resource_claim_template_name`: name of a `ResourceClaimTemplate` in the same namespace.
- `resource_claim_name`: name of a pre-existing `ResourceClaim` in the same namespace.
- Exactly one of `resource_claim_template_name` or `resource_claim_name` must be provided. Raises `ValueError` otherwise.

**Resulting pod spec:**

```yaml
spec:
  resourceClaims:
    - name: train-model-0              # auto-generated by backend driver
      resourceClaimTemplateName: gpu-claim-template
  containers:
    - name: main
      resources:
        claims:
          - name: train-model-0        # references pod-level claim
```

### Risks and Mitigations

| Risk                                                                                                               | Mitigation                                                                                                                                                                                                                                                                                                                 |
|--------------------------------------------------------------------------------------------------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| Cluster not configured for DRA (feature gate disabled, no DRA driver, invalid template name, or Kubernetes < 1.31) | Document minimum requirements: Kubernetes 1.31+ with `DynamicResourceAllocation` feature gate enabled (GA in 1.34), appropriate DRA driver installed, and valid `ResourceClaimTemplate` or `ResourceClaim` objects in the namespace. Without these, the pod either stays `Pending` or runs without the requested resource. |
| SDK/backend version skew                                                                                           | A pipeline compiled with the new SDK and submitted to an older backend will fail at runtime — the Argo compiler uses strict `protojson.Unmarshal`, so unknown fields cause a parse error (fail-closed). Document the minimum compatible backend version in SDK release notes.                                              |

## Design Details

### Proto Schema

New messages and fields added to `kubernetes_executor_config.proto`:

```protobuf
// Pod-level resource claim for Dynamic Resource Allocation.
// Maps to corev1.PodResourceClaim.
message PodResourceClaim {
  // Name of a ResourceClaimTemplate in the same namespace.
  // Mutually exclusive with resource_claim_name.
  string resource_claim_template_name = 1;

  // Name of a pre-existing ResourceClaim in the same namespace.
  // Mutually exclusive with resource_claim_template_name.
  string resource_claim_name = 2;

  // JSON parameter for runtime-resolved claims.
  // When set, takes precedence over the static fields above.
  ml_pipelines.TaskInputsSpec.InputParameterSpec resource_claim_json = 3;
}

message KubernetesExecutorConfig {
  // ... existing fields 1-19 ...
  repeated PodResourceClaim pod_resource_claims = 20;
}
```

New enum value added to `TaskConfigPassthroughType` in `pipeline_spec.proto`:

```protobuf
message TaskConfigPassthroughType {
  enum TaskConfigPassthroughTypeEnum {
    // ... existing values 0-6 ...
    // Indicates that DRA resource claims should be passed through to the external workload.
    KUBERNETES_RESOURCE_CLAIMS = 7;
  }
}
```

### Python SDK

New module `kubernetes_platform/python/kfp/kubernetes/pod_resource_claim.py` with `add_resource_claim()` and `add_resource_claim_json()` functions, following the `add_toleration()` / `add_toleration_json()` pattern.

A new `ResourceClaimConfig` class in the same module provides typed claim configuration with compile-time validation. It serializes to the proto `PodResourceClaim` message when the pipeline is compiled.

Both functions use `common.get_existing_kubernetes_config_as_message()` to retrieve the current config and append claims to the `pod_resource_claims` repeated field.

The pod-local claim name (`PodResourceClaim.Name`) is not set by the SDK. The backend driver auto-generates it as `<task-name>-<index>` at runtime, keeping the SDK API simple and avoiding name collision logic at compile time.

A new `KUBERNETES_RESOURCE_CLAIMS` value is added to `TaskConfigField` in `sdk/python/kfp/dsl/component_task_config.py`, enabling component authors to declare DRA passthrough in their component definitions.

Container-level claim references (`resources.claims`) are automatically derived from pod-level claims by the backend driver. Each auto-generated claim name is added as a container claim reference on the `main` container so it can consume the allocated resource. KFP task pods contain a single user container (`main`), plus Argo-managed containers (init containers, wait sidecar) and optional modelcar sidecars. DRA claim references are only added to the `main` container, as it is the only container executing user workloads that require access to DRA-managed resources.

### Backend Driver

In `backend/src/v2/driver/k8s.go`, `extendPodSpecPatch()` is extended to:

1. Read `pod_resource_claims` from `KubernetesExecutorConfig`.
2. For JSON variants, resolve parameters supporting both single claim (struct) and multiple claims (list).
3. Convert to `corev1.PodResourceClaim` structs and auto-generate the `Name` field as `<task-name>-<index>` for each claim.
4. Validate all resolved claims: exactly one of `ResourceClaimName` or `ResourceClaimTemplateName` must be set. Return an error if validation fails.
5. Route resolved claims based on `TaskConfig` passthrough settings:
   - If `setOnTaskConfig[KUBERNETES_RESOURCE_CLAIMS]` is true, set `taskConfig.ResourceClaims` with the resolved `[]corev1.PodResourceClaim` list.
   - If `setOnPod[KUBERNETES_RESOURCE_CLAIMS]` is true, set `podSpec.ResourceClaims` and add corresponding `corev1.ResourceClaim` entries to `podSpec.Containers[0].Resources.Claims`.

The `TaskConfig` struct gains a new `ResourceClaims []corev1.PodResourceClaim` field. The `KUBERNETES_RESOURCE_CLAIMS` entry is added to the `setOnPod` defaults in `getTaskConfigOptions()`.

This follows the existing pattern used for tolerations and volumes.

### Runtime Behavior

1. Pipeline author calls `kubernetes.add_resource_claim()` in pipeline definition.
2. SDK compiler stores claim configuration in the platform spec under `kubernetes.deploymentSpec.executors`.
3. At runtime, the KFP driver reads the platform spec and injects `resourceClaims` into the pod spec patch.
4. For passthrough components, the driver also populates the `TaskConfig` input parameter with the resolved claims.
5. Argo Workflows applies the patch via strategic merge when creating the task pod.
6. Kubernetes scheduler allocates the DRA resource before starting the pod.
7. Container accesses the allocated resource.

## Frontend Considerations

No frontend changes are required. DRA resource claims are part of the platform spec and are handled entirely by the backend driver when creating task pods. The frontend does not render or interact with `resourceClaims` in the pod spec. The existing pipeline run detail view shows pod status (Pending/Running/Succeeded/Failed), which is sufficient for observing DRA allocation behavior.

## Test Plan

### Python SDK unit tests

**Static variant (`add_resource_claim`):**
- Single claim with `resource_claim_template_name`
- Multiple claims on one task
- Coexistence with other `kfp-kubernetes` features (tolerations, node selectors, etc.)
- Coexistence with `set_accelerator_limit()` on same task (DRA and device-plugin are alternative GPU approaches — both on same task should not break)
- Single claim with `resource_claim_name`
- Validation: error when neither `resource_claim_template_name` nor `resource_claim_name` is provided
- Validation: error when both are provided

**`ResourceClaimConfig`:**
- Valid `resource_claim_template_name` creates config successfully
- Valid `resource_claim_name` creates config successfully
- Both `resource_claim_template_name` and `resource_claim_name` provided raises `ValueError`
- Neither provided raises `ValueError`

**JSON variant (`add_resource_claim_json`):**
- Pipeline input parameter (single `ResourceClaimConfig`)
- Pipeline input parameter (list of `ResourceClaimConfig`)
- Static `ResourceClaimConfig` (single claim)
- Static list of `ResourceClaimConfig` (multiple claims)
- Upstream task output parameter

### Go backend tests

- No claims configured — pod spec unchanged
- Static variant: field mapping from proto (`resource_claim_name`, `resource_claim_template_name`) to K8s struct (`ResourceClaimName`, `ResourceClaimTemplateName`)
- Auto-generated claim `Name` follows `<task-name>-<index>` pattern
- Container-level claim references auto-generated for each pod-level claim
- Claims added when `Containers[0].Resources` has no prior limits/requests (zero-value struct)
- Multiple claims propagated to pod spec with sequential indices
- JSON parameter resolution for single claim (struct) and multiple claims (list)
- Static variant with both `resource_claim_name` and `resource_claim_template_name` empty — verify backend returns error (SDK validates, but JSON bypasses SDK)
- JSON variant with both `resourceClaimName` and `resourceClaimTemplateName` set — verify backend returns error
- JSON variant with neither `resourceClaimName` nor `resourceClaimTemplateName` set — verify backend returns error
- JSON variant with empty object `{}` — verify backend returns error
- Task name with underscores or dots — verify auto-generated `Name` is sanitized to valid DNS label
- Null/optional parameter handling (skip claim when parameter resolves to null)
- Mixed static + JSON claims on same task — verify merged list with correct naming
- Existing `Resources.Claims` on container — verify DRA claims appended, not overwritten
- TaskConfig passthrough: claims routed to `taskConfig.ResourceClaims` when passthrough enabled
- Component declares other passthroughs but NOT `KUBERNETES_RESOURCE_CLAIMS` — verify claims still applied to pod (default `setOnPod` behavior)
- TaskConfig passthrough with `apply_to_task=true`: claims on both TaskConfig and pod spec
- TaskConfig passthrough with `apply_to_task=false`: claims on TaskConfig only, pod spec unchanged

### Integration / E2E tests

On a Kind cluster (Kubernetes 1.31+) with the [dra-example-driver](https://github.com/kubernetes-sigs/dra-example-driver) installed to simulate DRA-managed resources. Similar to the existing Fake GPU Operator setup used for device-plugin GPU tests, but targeting the DRA API:
- Submit pipeline with `add_resource_claim()`, verify task pod has `spec.resourceClaims` and container `resources.claims` entries
- Submit pipeline with multiple `add_resource_claim()` calls on same task, verify all claims present in pod spec
- Submit pipeline with `add_resource_claim_json()`, verify claims resolved and applied at runtime
- Verify pod reaches `Running` state with DRA resource allocated
- Verify pipeline run completes successfully

### KFP Local

DRA fields are Kubernetes-specific. When running pipelines locally via `SubprocessRunner` or `DockerRunner`, DRA claims in the platform spec are ignored (no pod spec is generated). No changes needed in local runners — verify DRA claims in platform spec do not cause errors in local execution.
