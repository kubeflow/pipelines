# KEP-13323: Import Models from Model Registry by Name

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Current State and Limitations](#current-state-and-limitations)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Dependencies](#dependencies)
- [Proposal](#proposal)
  - [SDK User Experience](#sdk-user-experience)
  - [User Stories](#user-stories)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [How It Works](#how-it-works)
  - [Configuration](#configuration)
  - [Implementation Pointers](#implementation-pointers)
- [Frontend Considerations](#frontend-considerations)
- [KFP Local Considerations](#kfp-local-considerations)
- [Test Plan](#test-plan)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
<!-- /toc -->

## Summary

This proposal adds a `model-registry://` URI protocol to `dsl.importer`. It lets pipelines import models from
Kubeflow Model Registry by logical name and version, instead of requiring the raw storage path. At runtime, the KFP
backend resolves the logical URI to the model's actual storage location by querying the Model Registry API, then
downloads the artifact through the standard import flow. This gives pipelines the same model-by-name access that KServe
already provides for online inference.

## Motivation

### Current State and Limitations

KServe lets users serve models by referencing them as `model-registry://my-model/v3` in an `InferenceService`. The
serving infrastructure resolves that to the real storage path automatically. KFP has no equivalent for batch or offline
inference.

Today, a pipeline author who wants to use a model from Model Registry must:

1. Look up the model version's storage URI manually (e.g., `s3://bucket/models/my-model/v3/`)
2. Hardcode that path in their pipeline code
3. Update the path every time the model is retrained or moved to a new location

This defeats the purpose of having a Model Registry. The registry should be the single source of truth for where a
model lives, regardless of whether the consumer is an online serving system or a batch pipeline.

### Goals

1. Allow `dsl.importer` to accept URIs in the form `model-registry://{model_name}/{model_version}`
2. Resolve the logical URI to the real storage path at runtime, inside the importer launcher
3. Fail clearly when Model Registry is not configured or the model cannot be found
4. Share configuration infrastructure with KEP-12020 so both registration and import use the same setup

### Non-Goals

1. This proposal does not require the user's machine to have network access to Model Registry. Resolution happens on
   the cluster at runtime.
2. Model Registry is only used for looking up where the model is stored. The actual model download goes directly to the
   storage backend (S3, GCS, etc.). Model Registry never proxies or caches artifact content.
3. This protocol is scoped to `dsl.importer`. It does not apply to pipeline inputs, outputs, or component artifact
   parameters.

## Dependencies

This proposal depends on [KEP-12020: Model Registry Integration](../12020-model-registry-integration/README.md), which
is merged and approved but not yet implemented.

KEP-12020 adds the ability to register pipeline-produced models to Model Registry via `model.register()`. This
proposal adds the ability to import existing models from Model Registry into pipelines via `dsl.importer`.

Both proposals share the same `modelRegistry` configuration block in the `kfp-launcher` ConfigMap (see
[Configuration](#configuration)). They can be implemented in either order. Whichever lands first introduces the shared
configuration; the other builds on it.

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
   models by name (e.g., `model-registry://fraud-detector/v2` or `model-registry://fraud-detector/production`), so I
   never need to track or update the underlying storage paths when models are retrained.

3. **As an ML/DS engineer**, I want model consumption in my pipelines to go through Model Registry, so I have a single
   place to audit which models are being used in production batch jobs.

### Risks and Mitigations

- **Model Registry unavailable at runtime**: The launcher retries with a configurable timeout. On failure, it surfaces
  a clear error with the registry URL and the HTTP status.
- **Model or version not found**: The error message includes the exact model name and version that were requested.
- **Resolved URI uses an unsupported storage scheme**: The launcher validates the resolved URI against supported
  schemes (`gs://`, `s3://`, `minio://`) before attempting to download.

## Design Details

### How It Works

The resolution happens inside the importer pod at runtime, following these steps:

1. **SDK compiles the pipeline.** The `model-registry://name/version` string is stored as-is in the pipeline spec's
   `ImporterSpec.artifact_uri` field. No resolution happens at compile time.

2. **Importer pod starts.** The launcher binary detects the `model-registry://` prefix on the artifact URI.

3. **Launcher calls Model Registry.** It parses the model name and version from the URI, then queries the Model
   Registry REST API to look up the model version's `storageUri` field. This requires two API calls: one to find the
   registered model by name, and one to get the version's storage URI.

4. **Launcher downloads from storage.** The resolved URI (e.g., `s3://bucket/models/my-model/v3/`) is used to
   download the artifact through the standard object store path.

5. **Artifact is stored in MLMD (ML Metadata).** The artifact is recorded with the resolved storage URI, not the
   logical `model-registry://` URI. This means downstream tasks and the UI see a normal storage path.

When `reimport=False` (the default), the cache lookup happens against the resolved storage URI. This ensures that if a
model version's storage location changes, the next pipeline run imports the updated artifact.

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
    tokenSecretRef:
      secretName: model-registry-auth
      secretNamespace: model-registry-system
      tokenKey: token
    caConfigMapRef:
      configMapName: model-registry-ca-bundle
      configMapNamespace: model-registry-system
      key: ca-bundle.crt
    timeout: 30s
    retryAttempts: 3
```

The exact schema will be finalized during implementation. If this configuration is absent and a pipeline uses a
`model-registry://` URI, the importer fails immediately with an error explaining that Model Registry is not configured.

### Implementation Pointers

- **Importer launcher** (`backend/src/v2/component/importer_launcher.go`): Add a `model-registry://` branch in
  `ImportSpecToMLMDArtifact`, after the URI is resolved from the pipeline spec. This follows the same pattern as the
  existing `oci://` check in that function.
- **Config parsing** (`backend/src/v2/config/env.go`): Add a `modelRegistry` key and parser method on `Config`,
  following the same pattern as `getBucketProviders()` which parses structured YAML from the `kfp-launcher` ConfigMap.
- **Auth**: The importer pod reads the token from the referenced Kubernetes Secret. Only read access to Model Registry
  is needed.

## Frontend Considerations

No frontend changes are required. The artifact is stored in MLMD with the resolved storage URI, so existing artifact
viewers work as-is.

## KFP Local Considerations

`model-registry://` URIs are not supported in local execution (SubprocessRunner and DockerRunner). Local mode has no
access to the in-cluster Model Registry service. If attempted, the executor raises a clear error directing the user to
use the resolved storage URI directly.

## Test Plan

**Unit tests:**
- URI parsing: valid and invalid `model-registry://` URIs, edge cases (special characters, missing version)
- Resolution logic: mock Model Registry HTTP responses for success, model not found, version not found, network errors
- Config parsing: valid config, missing config, partial config
- Cache key: verify the cache lookup uses the resolved URI, not the logical URI
- Local execution: verify that `model-registry://` URIs produce a clear unsupported-protocol error in
  SubprocessRunner/DockerRunner

**Integration tests:**
- Deploy KFP and Model Registry on a Kind cluster
- Register a model with a known storage URI
- Run a pipeline that imports via `model-registry://`, verify the artifact is available to downstream tasks
- Verify error behavior when the model or version does not exist

## Implementation History

- 2026-07-19: Initial proposal

## Drawbacks

- Adds a runtime dependency on Model Registry availability for pipelines that use this protocol
- Resolution requires two sequential HTTP calls to Model Registry, adding a small amount of latency per import
- Requires Model Registry to be deployed and configured before the feature works

## Alternatives

**SDK-side resolution:** Resolve the URI at compile time or submission time in the SDK client. This was rejected because
it requires the SDK machine to have network access to Model Registry, and the resolved URI becomes stale if the model
moves between compilation and execution.

**Custom container component:** Users can write a component that calls the Model Registry API and outputs the resolved
URI as a string. This works today but forces every pipeline author to manage connection details, authentication, and
client code manually. This is exactly the boilerplate this proposal eliminates.
