# KEP-13323: Import Models from Model Registry by Name

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Current State and Limitations](#current-state-and-limitations)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Backward Compatibility](#backward-compatibility)
- [Proposal](#proposal)
  - [SDK User Experience](#sdk-user-experience)
  - [User Stories](#user-stories)
  - [End-to-End Workflow](#end-to-end-workflow)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [How It Works](#how-it-works)
  - [Configuration](#configuration)
- [Frontend Considerations](#frontend-considerations)
- [KFP Local Considerations](#kfp-local-considerations)
- [Test Plan](#test-plan)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
<!-- /toc -->

## Summary

This proposal adds a `model-registry://` URI protocol to `dsl.importer`. It lets pipelines import models from
Kubeflow Model Registry by logical name and version, instead of the raw storage path. At run time the importer
resolves that URI through the Model Registry API. It then proceeds with the standard import flow. This gives
pipelines the model-by-name access that KServe already provides for online inference.

## Motivation

### Current State and Limitations

KServe lets users serve models by referencing them as `model-registry://my-model/v3` in an `InferenceService`. The
serving infrastructure resolves that to the real storage path automatically. KFP has no equivalent for batch or
offline inference.

Today, a pipeline author who wants to use a model from Model Registry must:

1. Look up the model version's storage URI manually, for example `s3://bucket/models/my-model/v3/`
2. Hardcode that path in their pipeline code
3. Update the path every time the model is retrained or moved to a new location

This defeats the purpose of having a Model Registry. The registry is the right place to record where a model lives,
whether the consumer is an online serving system or a batch pipeline.

An earlier proposal, KEP-12020, covered the opposite direction, which is registering pipeline outputs into the
registry. It is not being pursued, and this proposal stands alone.

### Goals

1. Allow `dsl.importer` to accept URIs in the form `model-registry://{model_name}/{model_version}`.
   A name segment and a version segment hold no forward slash. KFP decodes each segment after it parses the URI.
2. Resolve the logical URI to the real storage path at runtime, inside the importer launcher
3. Fail clearly in two cases: Model Registry is not configured, or the model cannot be found

### Non-Goals

1. This proposal does not require the user's machine to have network access to Model Registry. Resolution happens on
   the cluster at runtime.
2. Model Registry is only used for looking up where the model is stored. The actual model download goes directly to
   the storage backend, such as S3 or GCS. Model Registry never proxies or caches artifact content.
3. This protocol is scoped to `dsl.importer`. It does not apply to pipeline inputs, outputs, or component artifact
   parameters.

## Backward Compatibility

This proposal is backward compatible. No SDK changes and no proto changes are required. The `modelRegistry` key in the
`kfp-launcher` ConfigMap is optional, and the launcher already tolerates a key it does not recognize. If the key is
absent, KFP behaves exactly as it does today, and pipelines that do not use `model-registry://` URIs have zero new
dependencies. If Model Registry is not deployed or configured, only pipelines that reference `model-registry://` URIs
are affected. They fail immediately with a clear error explaining that Model Registry is not configured.

## Proposal

### SDK User Experience

```python
from kfp import dsl

@dsl.pipeline
def batch_inference_pipeline(model_name: str, model_version: str):
    model_task = dsl.importer(
        artifact_uri=f"model-registry://{model_name}/{model_version}",
        artifact_class=dsl.Model,
    )
    inference_task = run_inference(model=model_task.output)
```

No new SDK APIs are needed. The existing `dsl.importer` already accepts arbitrary URI strings and passes them through
to the backend unchanged. The `model-registry://` prefix is only interpreted at runtime by the importer launcher.

### User Stories

1. **As an ML/DS engineer**, I serve models online with KServe using `model-registry://my-model/v3`. I want to use the
   same reference in a KFP pipeline for batch inference, so I have one model identifier for both online and offline
   workloads.

2. **As an ML/DS engineer**, I register models in Model Registry with version names. I want my pipelines to reference
   models by name, such as `model-registry://fraud-detector/v2`. I then never need to track or update the underlying
   storage paths across retrains.

3. **As an ML/DS engineer**, I want model consumption in my pipelines to go through Model Registry, so I have a single
   place to audit which models are being used in production batch jobs.

### End-to-End Workflow

**Platform admin setup:**

1. Deploy Model Registry on the cluster
2. Add the `modelRegistry` configuration to the `kfp-launcher` ConfigMap of each namespace that needs it:
   - Standalone mode: edit the ConfigMap directly, with `kubectl edit configmap kfp-launcher -n kubeflow`.
   - Multi-user mode: edit the namespace-local ConfigMap. The profile controller does not overwrite it.
     See [Configuration](#configuration).
3. If the registry requires a static token, create a Kubernetes Secret for it in the same namespace

**ML engineer usage:**

1. Register a model in Model Registry (via the Model Registry UI or API), which records the model name,
   version, and storage URI
2. Use `model-registry://model-name/version` as the `artifact_uri` in `dsl.importer`
3. Run the pipeline. The importer resolves the name to the storage path, records it on the artifact, and
   makes it available to downstream tasks

### Risks and Mitigations

- **Model Registry unavailable at runtime**: The importer retries with the configured timeout. On failure, it
  surfaces a clear error with the registry URL and the HTTP status.
- **Model or version not found**: The error message includes the exact model name and version that were requested.
- **Resolved URI uses an unsupported storage scheme**: The importer rejects the resolved URI before it writes the
  artifact, so a registry import fails in the same way as a direct one. Today an unsupported scheme surfaces later,
  in the task that reads the artifact.
- **Resolved URI points at storage the namespace cannot read**: The run fails with an explicit error instead of a
  confusing permission denial. See [Credentials](#credentials).

## Design Details

### How It Works

The resolution happens inside the importer pod at runtime, following these steps:

1. **SDK compiles the pipeline.** The `model-registry://name/version` string is stored as-is in the pipeline spec's
   `ImporterSpec.artifact_uri` field. No resolution happens at compile time.

2. **Importer pod starts.** The launcher binary detects the `model-registry://` prefix on the artifact URI.

3. **Importer calls Model Registry.** It parses the model name and version from the URI. It then queries the Model
   Registry REST API to find the registered model, find the version by name, and list that version's artifacts. It
   asks for model artifacts only, newest first by create time, and takes the first result. This matches the Model
   Registry storage initializer in
   [kubeflow/hub](https://github.com/kubeflow/hub/blob/main/cmd/csi/internal/storage/modelregistry_provider.go). The
   type filter matters, because a version can also hold newer metrics or documentation.

4. **Importer records the resolved storage location.** The resolved URI, for example
   `s3://bucket/models/my-model/v3/`, is stored on the artifact for the standard downstream artifact-consumption
   flow. If the author sets `download_to_workspace=True`, the importer also downloads it into the workspace.

5. **Artifact is recorded in the KFP backend database.** The artifact URI is the resolved storage path. The importer
   also records the logical `model-registry://` URI, the model name, the version name, and the registry artifact id
   in the `metadata` map of the artifact. That preserves auditability, because a user can trace which pipeline runs
   consumed which registered models. KFP prefixes the keys it sets itself, following the existing schema keys.

The URI carries a model name and a version name, and both are required. A registry hostname is deliberately not part
of it. The namespace configuration already selects the instance, and a hostname segment does not select a server in
the Model Registry storage initializer either.

### Configuration

The `kfp-launcher` ConfigMap gains a `modelRegistry` key:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: kfp-launcher
  namespace: kubeflow
data:
  modelRegistry: |
    url: https://model-registry.example.com:8443
    tokenSecretRef:          # optional
      secretName: model-registry-auth
      tokenKey: token
    timeout: 30s             # optional
    retryAttempts: 3         # optional
```

The `url` field is required. `tokenSecretRef` applies to a registry that needs a static token, and a registry without
authentication needs no token. The ConfigMap holds a reference to a Secret and never holds a token value. `timeout`
and `retryAttempts` are optional with sensible defaults. There is no certificate authority field, because KFP already
mounts the configured cluster-wide CA bundle into the importer template. If this configuration is absent, a pipeline
that uses a `model-registry://` URI fails immediately. The error explains that Model Registry is not configured.

#### Standalone mode

The admin adds the `modelRegistry` block to the `kfp-launcher` ConfigMap directly, with
`kubectl edit configmap kfp-launcher -n kubeflow`. No reconciler runs in standalone mode, so manual edits persist.

#### Multi-user mode

In multi-user deployments, the profile controller creates a namespace-local `kfp-launcher` ConfigMap for each user
namespace, with `defaultPipelineRoot` and `clusterDomain`
(`manifests/kustomize/base/installs/multi-user/pipelines-profile-controller/sync.py`). It does not overwrite an
existing ConfigMap, because the controller attaches ConfigMaps with `updateStrategy: OnDelete`. A key that an
operator adds therefore persists, and the object store guide already directs operators to edit `kfp-launcher` in a
user namespace. This proposal needs no change to the profile controller.

Since Model Registry does not support multi-tenant isolation out of the box, companies can deploy separate Model
Registry instances per team. Because the `kfp-launcher` ConfigMap is per namespace, each namespace can point to a
different Model Registry URL without any additional code changes.

#### Credentials

If `tokenSecretRef` is set, that Secret must exist in the same namespace as the pipeline run. The service account of
the importer pod is `pipeline-runner`. Its namespace-scoped Role permits reading secrets and ConfigMaps in its own
namespace alone (`manifests/kustomize/base/pipeline/pipeline-runner-role.yaml`).

The importer pod also carries a projected service account token, which it uses for the KFP API server. Reusing that
identity for Model Registry avoids a provisioned Secret, and this proposal prefers it. The audience of that
token is bound to the run today, so the path needs a second projection with a registry audience.

A registry record can also name a bucket that the pipeline namespace holds no credentials for. KFP already defines
that case. A URI outside the default pipeline root needs its own connection parameters. A matching provider override
in the launcher configuration of the namespace also works. Otherwise the launcher rejects the URI with
an explicit error (`backend/src/v2/component/interfaces.go`). It never falls back to the default credentials in
silence. Importing a model that sits outside the pipeline root therefore needs a matching provider entry in that
namespace.

## Frontend Considerations

No frontend changes are required. The artifact is recorded in the KFP backend database with the resolved storage URI,
so existing artifact viewers work as-is.

## KFP Local Considerations

`model-registry://` URIs are not supported in local execution (SubprocessRunner and DockerRunner). An importer task
resolves inside the process of the author, which has neither a registry client nor registry credentials. If
attempted, the executor raises a clear error directing the user to use the resolved storage URI directly.

## Test Plan

**Unit tests:**
- URI parsing: valid and invalid `model-registry://` URIs, edge cases (special characters, missing version)
- Resolution logic: mock Model Registry HTTP responses for success, model not found, version not found, network errors
- Configuration parsing: valid, missing, and partial configuration
- Local execution: make sure that `model-registry://` URIs produce a clear unsupported-protocol error in
  SubprocessRunner/DockerRunner

**Integration tests:**
- Deploy KFP and Model Registry on a Kind cluster
- Register a model with a known storage URI
- Run a pipeline that imports via `model-registry://`, and make sure that the artifact reaches downstream tasks
- Make sure that a missing model and a missing version each fail with the expected error
- Make sure that a model stored outside the pipeline root, with no provider override, fails with the expected error
- Make sure that the artifact metadata holds the original `model-registry://` URI, plus the registry model and
  version identifiers

## Implementation History

- 2026-07-19: Initial proposal
- 2026-08-09: Revised to address maintainer feedback (backward compatibility, end-to-end workflow, multi-user
  reconciler behavior)
- 2026-11-08: Revised for the current backend. MLMD references replaced with the native artifact APIs, the dependency
  on KEP-12020 dropped, and `tokenSecretRef` made optional. Three earlier statements were wrong and are withdrawn.
  The profile controller does not overwrite `kfp-launcher`. The importer does not reject an unsupported scheme in a
  resolved URI today. Task caching does not key on the artifact URI.

## Drawbacks

- Adds a runtime dependency on Model Registry availability for pipelines that use this protocol
- Resolution requires three sequential HTTP calls to Model Registry, adding a small amount of latency per import
- Requires Model Registry to be deployed and configured before the feature works

## Alternatives

**SDK-side resolution:** Resolve the URI at compile time or submission time in the SDK client. This was rejected for
two reasons. It requires the SDK machine to reach Model Registry over the network. A model that moves between
compilation and execution also leaves the resolved URI stale.

**Custom container component:** Users can write a component that calls the Model Registry API and outputs the
resolved URI as a string. This works today but forces every pipeline author to manage connection details,
authentication, and client code manually. This is exactly the boilerplate this proposal eliminates.
