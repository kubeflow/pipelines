# KEP-11875: Pipeline Run Workspace in Kubeflow Pipelines

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [SDK User Experience](#sdk-user-experience)
    - [1. Pipeline Configuration](#1-pipeline-configuration)
    - [2. Repository Cloning Component](#2-repository-cloning-component)
    - [3. Data Processing Component](#3-data-processing-component)
    - [4. Model Import, Training, and Model Output](#4-model-import-training-and-model-output)
    - [Key Benefits of this User Experience](#key-benefits-of-this-user-experience)
  - [User Stories](#user-stories)
    - [Story 1: Data Scientist Working with Large Datasets](#story-1-data-scientist-working-with-large-datasets)
    - [Story 2: Custom PVC Configuration for Kubernetes-Savvy Users](#story-2-custom-pvc-configuration-for-kubernetes-savvy-users)
    - [Story 3: Default PVC Configuration for Cluster Administrators](#story-3-default-pvc-configuration-for-cluster-administrators)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [SDK and API](#sdk-and-api)
    - [Workspace Configuration](#workspace-configuration)
    - [Upload Artifacts From The Workspace](#upload-artifacts-from-the-workspace)
    - [Workspace Path Access](#workspace-path-access)
    - [Downloading Artifacts](#downloading-artifacts)
    - [Accessing an Artifact in the Workspace](#accessing-an-artifact-in-the-workspace)
  - [API Server](#api-server)
    - [PVC Default Configuration](#pvc-default-configuration)
    - [Argo Workflow Compiler](#argo-workflow-compiler)
  - [Driver and Launcher Changes](#driver-and-launcher-changes)
    - [Importer](#importer)
      - [Importer Driver](#importer-driver)
    - [Importing to the Workspace](#importing-to-the-workspace)
    - [Container Driver](#container-driver)
    - [Launcher](#launcher)
  - [Test Plan](#test-plan)
    - [Unit Tests](#unit-tests)
    - [Integration tests](#integration-tests)
  - [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
<!-- /toc -->

## Summary

Many Kubeflow Pipelines (KFP) pipelines require passing large amounts of data between components. Doing so with artifact
inputs and outputs introduces overhead and requires additional S3 storage. The primary workaround is to create a
persistent volume claim (PVC) for the pipeline run but this requires Kubernetes knowledge, can be complicated, and hard
to clean up. This also doesn't work if the pipeline is run locally.

Abstracting this away with a new feature of KFP workspaces that provides shared storage between components for the
duration of a pipeline run would greatly enhance the user experience.

## Motivation

Many Kubeflow Pipelines (KFP) pipelines require passing large amounts of data between components. The current mechanism
of artifact inputs and outputs can cause unnecessary overhead as each component that requires an artifact must download
it from S3 and any intermediate artifact that needs to be passed to other components must upload the artifact to S3.
This increases pipeline run execution times and requires additional storage in S3, while also cluttering MLMD and KFP
with unnecessary artifacts. The primary workaround is to create a persistent volume claim (PVC) for the pipeline run,
mount the PVC on components that require it, and delete the PVC at the end of the pipeline run. This requires Kubernetes
knowledge as well as cluster specific knowledge of which storage classes are available on the cluster. If the pipeline
run fails, PVCs might not get cleaned up, causing unnecessary storage usage. This also doesn't work if the pipeline is
run locally.

This can be abstracted away from the user with the addition of a pipeline run workspace concept similar to
[Argo Workflows volumes](https://argo-workflows.readthedocs.io/en/latest/walk-through/volumes/#volumes) and
[Tekton workspaces](https://tekton.dev/docs/pipelines/workspaces/#workspaces-in-pipelines-and-pipelineruns) but with
more default values that are environment specific (e.g. `storageClassName`, `accessModes`, and etc.). This would greatly
enhance the user experience.

This proposal aims to simplify PVC usage in pipelines by enabling Kubernetes cluster administrators to set environment
specific defaults. While this improvement benefits all users, it particularly targets those with minimal Kubernetes
experience by making PVCs more accessible and easier to use.

### Goals

1. Define a workspace abstraction for pipeline runs defined as a pipeline configuration at the SDK level.
1. Start with adding a PVC based implementation using Argo Workflows
   [`spec.volumeClaimTemplates`](https://argo-workflows.readthedocs.io/en/latest/walk-through/volumes/#volumes) feature.
1. Only require the user to provide a workspace size for most use cases. But expose setting other PVC options for
   advanced use cases.
1. Allow the KFP administrator to set default configuration options for the PVCs created by the workspace.
1. Enable efficient data sharing between pipeline components.
1. Support artifact downloading directly to the workspace with `dsl.importer`.
1. Existing pipeline components/tasks do not need to be aware that input artifacts are in a workspace.
1. Automate the clean up of workspaces when a pipeline run succeeds or when it is cleaned up.
1. Only mount the workspace when the component uses it. This is to prevent all pipeline tasks being tied to a single
   Kubernetes node in `ReadWriteOnce` access mode.
1. The API should not be limited to just PVC as other storage backends could be added in the future.

### Non-Goals

1. Multiple workspaces per pipeline run are not an initial goal. For advanced use cases such as this, the user should
   leverage existing mechanisms to create the desired PVCs.
1. Adding support for other storage backends beyond PVC.
1. The workspace is meant to be ephemeral and is not meant as a storage of KFP artifacts. Output artifacts still must be
   uploaded to object storage.
1. It is not a goal to abstract pod affinity and/or node affinity rules to enhance scheduling for workspaces that
   leverage `ReadWriteOnce` PVCs.

## Proposal

### SDK User Experience

Below is a full example pipeline that highlights the desired SDK user experience. Below the code snippet will be a
breakdown explaining each step.

```python
from kfp import dsl
from kfp import compiler


@dsl.component()
def clone_repo(workspacePath: str, repo: str) -> str:
    import os
    import subprocess

    clone_path = os.path.join(workspacePath, "repo")
    subprocess.call(["git", "clone", repo, clone_path])

    return clone_path


@dsl.component()
def process_data(repo_path: str):
    print("Processing the data at " + repo_path)


@dsl.component()
def train(model: dsl.Model, trained_model: dsl.Output[dsl.Model]):
    with open(model.path, "r") as model_file:
        print("Training the model")

    # Upload the model to S3 and register it in MLMD
    trained_model.set_path(model.path) # or trained_model.custom_path = model.path


@dsl.pipeline(
    name="my-pipeline",
    pipeline_config=dsl.PipelineConfig(
        workspace=dsl.WorkspaceConfig(size="250GB"),
    ),
)
def pipeline(repo: str, model_uri: str):
    clone_repo_task = clone_repo(
        workspacePath=dsl.WORKSPACE_PATH_PLACEHOLDER, repo=repo,
    )
    process_data_task = process_data(repo_path=clone_repo_task.output)
    import_base_model_task = dsl.importer(
        artifact_class=dsl.Model, artifact_uri=model_uri, download_to_workspace=True,
    )
    train(model=import_base_model_task.output).after(process_data_task)
```

#### 1. Pipeline Configuration

```python
@dsl.pipeline(
    name="my-pipeline",
    pipeline_config=dsl.PipelineConfig(
        workspace=dsl.WorkspaceConfig(size="250GB"),
    ),
)
```

- The pipeline is configured with a 250GB workspace using `dsl.WorkspaceConfig`.
- This workspace is a shared storage space that persists for the duration of the pipeline run.
- For KFP remote, this is backed by a Kubernetes PersistentVolumeClaim (PVC) that's automatically created and then
  deleted after a successful run.
- For KFP local (i.e. running the pipeline locally), this would be a temporary directory instead.
- The configuration of the PVC defaults to the workspace configuration on the KFP API server, but can be overridden as
  needed.

An example of overriding the PVC configuration:

```python
@dsl.pipeline(
    name="my-pipeline",
    pipeline_config=dsl.PipelineConfig(
        workspace=dsl.WorkspaceConfig(
          size="250GB",
          kubernetes=dsl.KubernetesWorkspaceConfig(
            pvcSpecPatch={
              "storageClassName": "super-fast-storage",
              "accessModes": ["ReadWriteMany"],
            }
          ),
        ),
    ),
)
```

As a stretch goal, it'd be nice to make garbage collection of the PVC configurable. It'd default to `OnRunSuccess`. Here
are the suggested options:

- `OnRunSuccess` - when the pipeline run completes successfuly, the PVC is deleted. This maps to the Argo Workflows
  `OnWorkflowSuccess` option.
- `OnRunCompletion` - when the pipeline run completes (success or failure), the PVC is always deleted. This maps to the
  Argo Workflow `OnWorkflowCompletion` option.
- `None` - the PVC is never deleted. This would require a change in Argo Workflows to support this.

Below is an example pipeline configuration with the `workspaceDeletion` option:

```python
@dsl.pipeline(
    name="my-pipeline",
    pipeline_config=dsl.PipelineConfig(
        workspace=dsl.WorkspaceConfig(
          size="250GB",
          workspaceDeletion="OnRunCompletion",
        ),
    ),
)
```

#### 2. Repository Cloning Component

```python
@dsl.component()
def clone_repo(workspacePath: str, repo: str) -> str:
    import os
    import subprocess

    clone_path = os.path.join(workspacePath, "repo")
    subprocess.call(["git", "clone", repo, clone_path])

    return clone_path

# In the pipeline definition:
clone_repo_task = clone_repo(
    workspacePath=dsl.WORKSPACE_PATH_PLACEHOLDER, repo=repo,
)
```

- This component takes two parameters:
  - `workspacePath`: A parameter that when provided `dsl.WORKSPACE_PATH_PLACEHOLDER`, gets the path to the shared
    workspace (i.e. `/kfp-workspace`). This indicates the workspace should be mounted to this component.
  - `repo`: The Git repository URL to clone.
- It clones the repository into a subdirectory of the workspace.
- Returns the path to the cloned repository for use by subsequent components.

#### 3. Data Processing Component

```python
@dsl.component()
def process_data(repo_path: str):
    print("Processing the data at " + repo_path)

# In the pipeline definition:
process_data_task = process_data(repo_path=clone_repo_task.output)
```

- Takes the path to the cloned repository as input.
- Since the path is provided by the output of `clone_repo` and that path is in a workspace, the workspace is
  automatically mounted for this component.

#### 4. Model Import, Training, and Model Output

```python
@dsl.component()
def train(model: dsl.Model, trained_model: dsl.Output[dsl.Model]):
    with open(model.path, "r") as model_file:
        print("Training the model")

    # Upload the model to S3 and register it in MLMD
    trained_model.set_path(model.path) # or trained_model.custom_path = model.path

# In the pipeline definition:
import_base_model_task = dsl.importer(
    artifact_class=dsl.Model,
    artifact_uri=model_uri,
    download_to_workspace=True,
)
train(model=import_base_model_task.output).after(process_data_task)
```

- Uses `dsl.importer` with `download_to_workspace=True` to download a model directly to the workspace.
- The `train` component receives the model from the workspace rather than downloading it again.
- The `train` component has the `trained_model` output artifact and sets a path on the workspace of where the KFP
  Launcher should upload to S3 by calling `trained_model.set_path(model.path)`. This prevents the need for copying the
  output artifact from the workspace to `trained_model.path`, which uses the Kubernetes node's storage (i.e. `emptyDir`
  volume). By setting `set_path`, the `train_model.path` now returns the custom path.

#### Key Benefits of this User Experience

1. **Efficiency**: Artifacts are downloaded only once and shared between components.
1. **Simplicity**: Users don't need to manage PVCs or understand Kubernetes storage.
1. **Automatic Cleanup**: The workspace's PVC is automatically deleted after the pipeline completes.
1. **Flexibility**: Components only have access to the workspace when needed.

### User Stories

#### Story 1: Data Scientist Working with Large Datasets

As a data scientist, I want to process large datasets across multiple pipeline steps without worrying about data
transfer overhead and Kubernetes configuration, so that I can focus on my analysis rather than infrastructure concerns.

#### Story 2: Custom PVC Configuration for Kubernetes-Savvy Users

As a data scientist with Kubernetes experience, I want to customize the PersistentVolumeClaim (PVC) properties when
using workspaces, so that I can specify storage class, size, access modes, and other PVC parameters to match my specific
requirements, just like I can with the existing `CreatePVC` component.

#### Story 3: Default PVC Configuration for Cluster Administrators

As a Kubeflow Pipelines administrator, I want to define default PersistentVolumeClaim (PVC) configurations at the
cluster level, so that users can easily create workspaces with storage settings that match our cluster's capabilities
and policies, without users needing to specify these details in every pipeline.

### Notes/Constraints/Caveats

N/A

### Risks and Mitigations

1. **Storage Exhaustion**: Unlike S3, which is often a cloud service with virtually unlimited storage, PVCs usually have
   a finite amount of storage and too many pipeline runs could use up all the storage, causing pipelines to not run in
   parallel. This is mostly mitigated by the automatic clean up PVCs after a pipeline run succeeds and with the TTL
   feature to automatically clean up stale Argo Workflow resources.
1. **ParallelFor**: When using `dsl.ParallelFor` and the PVC has a `ReadWriteOnce` access mode, it may end up running
   serially if the pods get scheduled to different nodes. A workaround is to use pod affinity or node affinity rules but
   abstracting that is out of the scope of this KEP.

## Design Details

### SDK and API

#### Workspace Configuration

The workspace will be configured at the pipeline level using `PipelineConfig`:

```python
@dsl.pipeline(
    name="my-pipeline",
    pipeline_config=dsl.PipelineConfig(
        workspace=dsl.WorkspaceConfig(
            size="250GB",
        ),
    ),
)
def pipeline():
    # Pipeline definition
```

An example of overriding the PVC configuration defaults set on the API server:

```python
@dsl.pipeline(
    name="my-pipeline",
    pipeline_config=dsl.PipelineConfig(
        workspace=dsl.WorkspaceConfig(
          size="250GB",
          kubernetes=dsl.KubernetesWorkspaceConfig(
            pvcSpecPatch={
              "storageClassName": "super-fast-storage",
              "accessModes": ["ReadWriteMany"],
            }
          ),
        ),
    ),
)
```

This will lead to protocol buffer changes similar to the following:

```diff
diff --git a/api/v2alpha1/pipeline_spec.proto b/api/v2alpha1/pipeline_spec.proto
index d986a048b..866696419 100644
--- a/api/v2alpha1/pipeline_spec.proto
+++ b/api/v2alpha1/pipeline_spec.proto
@@ -1104,6 +1104,19 @@ message PlatformDeploymentConfig {
   map<string, google.protobuf.Struct> executors = 1;
 }

+message WorkspaceConfig {
+  // Size of the workspace
+  string size = 1;
+
+  // Kubernetes specific configuration for the workspace
+  KubernetesWorkspaceConfig kubernetes = 2;
+}
+
+message KubernetesWorkspaceConfig {
+  // Patch of a PersistentVolumeClaim (PVC) spec to override defaults set on the API server for the workspace PVC
+  google.protobuf.Struct pvc_spec_patch = 1;
+}
+
 // Spec for pipeline-level config options. See PipelineConfig DSL class.
 message PipelineConfig {
   // Name of the semaphore key to control pipeline concurrency
@@ -1115,4 +1128,7 @@ message PipelineConfig {
   // Time to live configuration after the pipeline run is completed for
   // ephemeral resources created by the pipeline run.
   int32 resource_ttl = 3;
+
+  // Configuration for the workspace
+  optional WorkspaceConfig workspace = 4;
 }
```

#### Upload Artifacts From The Workspace

Adding support for the `model.set_path()` function requires a new property named `custom_path` in the protocol buffer
type to indicate the path to upload to S3 rather than needing to copy.

```proto
message RuntimeArtifact {
  // The local path used to upload the artifact.
  optional string custom_path = 7;
}
```

It also requires setting this field to `None` on the Python `Artifact` class in the constructor. For example:

```python
  class Artifact:
    def __init__(self,
                 name: Optional[str] = None,
                 uri: Optional[str] = None,
                 metadata: Optional[Dict] = None) -> None:
        self.custom_path: str = None
```

The Python executor also needs to pass this field back to the KFP Launcher in the executor output.

#### Workspace Path Access

Components will access the workspace path using the `dsl.WORKSPACE_PATH_PLACEHOLDER`:

```python
@dsl.component()
def my_component(workspacePath: str):
    # Use workspacePath to access the workspace directory
    file_path = os.path.join(workspacePath, "my_file.txt")

# In the pipeline definition:
my_component(workspacePath=dsl.WORKSPACE_PATH_PLACEHOLDER)
```

The `dsl.WORKSPACE_PATH_PLACEHOLDER` will be defined as such:

```python
WORKSPACE_PATH_PLACEHOLDER = '{{$.workspace_path}}'
```

It is up to the KFP Driver to resolve this value, which will likely always be `/kfp-workspace`, but having the
placeholder indicates to the KFP Driver that the component requires the workspace to be mounted.

#### Downloading Artifacts

Artifacts can be downloaded directly to the workspace:

```python
import_task = dsl.importer(
    artifact_class=dsl.Model,
    artifact_uri=model_uri,
    download_to_workspace=True,
)
```

This will require a new field of `download_to_workspace` to the `ImporterSpec` protocol buffer type:

```proto
  message ImporterSpec {
    // Whether or not to download the artifact to the workspace.
    optional bool download_to_workspace = 7;
  }
```

The same change is needed in the corresponding `ImporterSpec` Python class and `importer` Python function.

#### Accessing an Artifact in the Workspace

The `Artifact` Python class' `_get_path` should check for a boolean metadata field `_kfp_workspace` on the artifact
which is set by the KFP Driver (see the [Container Driver](#container-driver) section for context). If it's `True`, then
`model.path` should return the path with the `/kfp-workspace/.artifacts/` prefix.

### API Server

#### PVC Default Configuration

The KFP API server configuration (`config.json`) should accept the following new optional fields:

```json
{
  "Workspace": {
    "VolumeClaimTemplateSpec": {
      "accessModes": ["ReadWriteMany"],
      "storageClassName": "my-storage"
    }
  }
}
```

These would be used as the default values when creating the PVC for the workspace.

#### Argo Workflow Compiler

Argo Workflows already supports PVC management with the
[`spec.volumeClaimTemplates`](https://argo-workflows.readthedocs.io/en/latest/walk-through/volumes/#volumes) field. When
a pipeline specifies a workspace, it'll be added to the Argo Workflow object such as the following:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
spec:
  volumeClaimTemplates:
    - metadata:
        name: kfp-workspace-46f1d52e-c72b-42fc-88ae-789edf7c33fd # The suffix is the run ID
      spec:
        accessModes:
          - ReadWriteMany
        storageClassName: my-storage
        resources:
          requests:
            storage: 250Gi
```

By default, the volume claims are garbage collected if the workflow succeeds or the `Workflow` object is deleted.

The base of the `volumeClaimTemplates` entry comes from the API server configuration but any overrides come from the
user's pipeline config. Ideally the API server configuration defaults only requires the user to provide the `size` which
gets set on `spec.resources.requests.storage`, but if the user supplies `storageClassName` and `accessModes` in the
`pvcSpecPatch` option, they could use a workspace without the administrator configuring the defaults.

Note that it could be possible that `pvcSpecPatch` sets the size, but that should be ignored by having `size` take
priority.

Additionally, if the artifact URI is an input parameter or has a static value starting with `oci://`, the new KFP
Importer driver should be added to the Importer DAG. See the [Importer Driver](#importer-driver) section for more
details.

### Driver and Launcher Changes

#### Importer

##### Importer Driver

To support downloading OCI artifacts in the Modelcar format to a workspace, we'll need to have a simple driver for the
KFP Importer when the artifact URI is an input parameter or has a static value starting with `oci://`. It's only goal is
to provide a spec patch with the logic from `addModelcarsToPodSpec` to set the proper sidecar containers.

#### Importing to the Workspace

The Importer now supports downloading artifacts to a workspace. To differentiate between artifacts to be downloaded by
the KFP Launcher and those already in the workspace, the importer's execution type in MLMD should be
`system.ImporterWorkspaceExecution` instead of `system.ImporterExecution`.

Additionally, just before registering the artifact in MLMD, the artifacts should be downloaded to the workspace. The
download path should be the same as the local paths returned from the `LocalPathForURI` function except that it will
have the prefix of `/kfp-workspace/.artifacts/` in the path.

#### Container Driver

When resolving input artifacts, if the artifact is from a task with an execution type of
`system.ImporterWorkspaceExecution`, `resolveUpstreamArtifacts` should set the metadata field `_kfp_workspace` to
`True`. This is not persisted to MLMD but will be present in the executor input used by the KFP Launcher and the Python
executor.

When an input artifact is from a workspace, an input parameter is `dsl.WORKSPACE_PATH_PLACEHOLDER`, or an input
parameter is a component output result that is a path to the workspace (e.g. starts with `/kfp-workspace`), the driver
should set the volume mount of the KFP workspace in the `Pod` spec patch.

Lastly, the Driver should disallow user mounted volumes in or under `/kfp-workspace` as this could lead to confusing
behavior.

#### Launcher

The KFP Launcher will skip downloading any artifacts to the local paths (`emptyDir` volume mounts) if the artifact has
the `_kfp_workspace` metadata field set to `True`, which is set the by the Container Driver.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to existing tests to make this code solid
enough prior to committing the changes necessary to implement this enhancement.

#### Unit Tests

The unit tests will be used to cover logic when possible but most coverage will come from integration tests.

#### Integration tests

- Add an E2E sample test with the example pipeline in this KEP or similar.
- Add an additional E2E test that tests overriding the default PVC configurations.
- Add an additional E2E test that tests downloading artifacts to the workspace.

### Graduation Criteria

N/A

## Implementation History

- Initial proposal: 2025-04-29

## Drawbacks

1. **Resource Overhead**: Workspaces consume additional storage resources when pipelines fail and there is no TTL set on
   cleaning up the Argo Workflow objects. This is necessary to allow pipeline run retries though.
1. **Potentially More Serialization**: Depending on the PVC's access modes (e.g. `ReadWriteOnce`), all pipeline steps
   that leverage the workspace could be serial if the pods aren't scheduled to the same node.
1. **Requires Administrator Configuration**: For the best user experience, the KFP administrator needs to configure the
   default values for the PVC spec.

## Alternatives

1. **Improve CreatePVC**: We could consider improving the existing `CreatePVC` component to allow automatic deletion at
   the end of a pipeline run and add owner references of the Argo Workflow object to the created PVC to clean up the
   PVCs when the Argo Workflow objects are deleted. The API server default PVC spec configuration proposal from this KEP
   could also be considered for improving `CreatePVC`.
