# KEP-12020: Model Registry Integration for Kubeflow Pipelines

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Current State and Limitations](#current-state-and-limitations)
  - [Integration Benefits](#integration-benefits)
  - [Current Workarounds](#current-workarounds)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [SDK User Experience](#sdk-user-experience)
    - [API Reference](#api-reference)
    - [Backend Translation](#backend-translation)
    - [Cross-Reference Metadata](#cross-reference-metadata)
  - [Configuration Management](#configuration-management)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Architecture Overview](#architecture-overview)
  - [Implementation Notes](#implementation-notes)
    - [Metadata Handling](#metadata-handling)
    - [Launcher Processing](#launcher-processing)
  - [API Design](#api-design)
    - [Model Registry Request Structure](#model-registry-request-structure)
  - [Security Considerations](#security-considerations)
- [Test Plan](#test-plan)
  - [Unit Tests](#unit-tests)
    - [Configuration Tests](#configuration-tests)
    - [SDK Tests](#sdk-tests)
  - [Integration Tests](#integration-tests)
    - [Successful Scenarios](#successful-scenarios)
    - [Error Scenarios](#error-scenarios)
  - [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
  - [Alternative 1: Direct SDK Integration](#alternative-1-direct-sdk-integration)
<!-- /toc -->

## Summary

The [Kubeflow Model Registry](https://www.kubeflow.org/docs/components/model-registry/) serves as the centralized
metadata store for the Kubeflow ecosystem, providing comprehensive model cataloging, versioning, and discovery
capabilities. This proposal introduces seamless integration between Kubeflow Pipelines (KFP) and Model Registry,
enabling automated model registration as part of pipeline execution workflows.

The integration abstracts away connection details, authentication mechanisms, and Model Registry-specific APIs,
providing data scientists with a simple interface for model registration through the KFP SDK. This enhancement bridges
the gap between KFP's artifact storage and Model Registry's cataloging capabilities, creating a unified model management
experience.

## Motivation

### Current State and Limitations

Kubeflow Pipelines currently maintains its own artifact store for pipeline outputs, which provides basic storage
functionality but lacks advanced cataloging and versioning features. Models can only be added to this store through:

1. **Pipeline execution outputs** (e.g., `dsl.Output[dsl.Model]`)
2. **Importer components** for external model registration

This approach has several limitations:

- **No centralized cataloging**: Models are scattered across pipeline runs without unified discovery
- **Limited versioning**: No structured version management or lineage tracking
- **Tool-specific isolation**: Models created in KFP are not discoverable by other Kubeflow components
- **Manual registration overhead**: Users must manually register models outside of pipeline workflows

### Integration Benefits

By enabling seamless Model Registry integration, users will experience:

- **Centralized model catalog**: Single source of truth for all models across the Kubeflow ecosystem
- **Versioning**: Structured version management with lineage tracking
- **Cross-tool discovery**: Models become discoverable by other Kubeflow components (e.g. KServe)
- **Simplified workflows**: One-line model registration within pipeline components
- **Enhanced governance**: Better model lifecycle management and compliance tracking

### Current Workarounds

Without this integration, users must implement complex workarounds:

1. Add Model Registry infrastructure details in the pipeline (e.g. URLs) through hardcoded values, pipeline input
   parameters, or mounting a Kubernetes `ConfigMap`/`Secret`.
1. Mount a Kubernetes `Secret` to access the token.
1. Know the KFP standards for registering models in Model Registry (e.g. `model_source_kind="kfp"`).

### Goals

1. **Seamless SDK Integration**: Provide a simple, API for model registration within KFP components
2. **Infrastructure Abstraction**: Hide Model Registry connection details and authentication from user code
3. **Standardized Metadata**: Automatically populate KFP-specific metadata for proper model lineage tracking
4. **Error Resilience**: Provide configurable error handling to prevent pipeline failures due to registration issues
5. **Multi-tenancy Support**: Enable namespace-specific configuration for isolated deployments

### Non-Goals

1. Implement KFP-specific RBAC controls over Model Registry APIs, such as model allowlists for version registration.

## Proposal

### SDK User Experience

The proposed integration introduces a `register()` method on KFP Model artifacts, providing a clean interface for model
registration:

```python
@dsl.component()
def train_model(
    model: dsl.Output[dsl.Model],
):
    # Training logic
    with open(model.path, "r") as model_file:
        print("Training the model...")

    # Set model metadata
    model.name = "my-model-v1.0.0"
    model.metadata["training_epochs"] = 100
    model.metadata["accuracy"] = 0.95

    # Register model with Model Registry
    model.register(
        model_name="sentiment-classifier",
        description="BERT-based sentiment classification model",
        model_format_name="vLLM",
        model_format_version=None,
        owner="ml-team",
        author="data-scientist@company.com",
        continue_on_error=True,  # Default: True
    )
```

#### API Reference

The `model.register()` method accepts the following parameters:

| Parameter              | Type | Required | Default              | Description                                          |
| ---------------------- | ---- | -------- | -------------------- | ---------------------------------------------------- |
| `model_name`           | str  | Yes      | -                    | Name of the model in Model Registry                  |
| `description`          | str  | No       | ""                   | Human-readable model description                     |
| `model_format_name`    | str  | No       | None                 | Model format (e.g., "PyTorch", "TensorFlow", "ONNX") |
| `model_format_version` | str  | No       | None                 | Version of the model format                          |
| `owner`                | str  | No       | "Kubeflow Pipelines" | Model owner/team                                     |
| `author`               | str  | No       | "Kubeflow Pipelines" | Model author                                         |
| `continue_on_error`    | bool | No       | True                 | Whether to fail pipeline on registration error       |

#### Backend Translation

The SDK call translates to the following Model Registry API invocation:

```python
# Equivalent Model Registry client call
registered_model = registry.register_model(
    name="sentiment-classifier",
    description="BERT-based sentiment classification model",
    owner="ml-team",
    author="data-scientist@company.com",
    version="my-model-v1.0.0",  # model.name
    uri="s3://kfp-artifacts/run-123/model",  # model.uri
    metadata={
        "training_epochs": 100,
        "accuracy": 0.95
    },  # model.metadata
    model_format_name="vLLM",
    model_format_version=None,
    model_source_id="b6a9dde3-1647-463f-aeb8-5800089c84e8",  # Pipeline run ID
    model_source_name="sentiment-training-pipeline",  # Pipeline run name
    model_source_class="pipelinerun",  # KFP-specific identifier
    model_source_kind="kfp",  # KFP-specific identifier
    model_source_group="ml-team",  # Pipeline namespace
)
```

#### Cross-Reference Metadata

After successful registration, KFP adds metadata to the model artifact for UI cross-referencing. This is a list/array
since multiple pipeline runs could register the same model.

```python
model.metadata["registered_models"] = [{
    "modelName": "sentiment-classifier",
    "versionName": "my-model-v1.0.0",
    "versionID": 42,
    "modelID": 15,
    "modelRegistryURL": "https://model-registry.example.com:8443/models/15/versions/42",
}]
```

The KFP UI should prominently display the model versions and link to the Model Registry UI when viewing the model's
details.

### Configuration Management

Extend the existing `kfp-launcher` ConfigMap to include Model Registry configuration:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: kfp-launcher
  namespace: kubeflow
data:
  # Existing configuration...
  pipelineRoot: "s3://kfp-artifacts"

  # New Model Registry configuration
  modelRegistry: |
    url: https://model-registry.example.com:8443
    tokenSecretRef: # Follows the same key names for existing configurations
      secretName: model-registry-auth
      secretNamespace: model-registry-system  # Defaults to current namespace
      tokenKey: token
    caConfigMapRef:  # Optional TLS certificate
      configMapName: model-registry-ca-bundle
      configMapNamespace: model-registry-system
      key: ca-bundle.crt
    timeout: 30s  # HTTP timeout for registration requests
    retryAttempts: 3  # Number of retry attempts on failure
```

### User Stories

#### Story 1

As a data scientist, I would like to register my model to Model Registry in a pipeline without knowing the underlying
Model Registry infrastructure and APIs, so that I can easily track model versions, share models with my team, and
maintain a centralized model catalog as part of my automated ML workflows.

#### Story 2

As a data scientist, I would like to access a centralized model catalog that is independent of the specific tool used to
create the models, enabling easy discovery and simplified model management across different ML workflows.

### Risks and Mitigations

The Model Registry API doesn't have granular RBAC so the `pipeline-runner` service account has full access to the Model
Registry API.

## Design Details

### Architecture Overview

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│   KFP Pipeline  │     │   KFP Launcher   │     │ Model Registry  │
│                 │     │                  │     │                 │
│ model.register()│───▶│ Extract metadata │───▶│   API Server    │
│                 │     │ Register model   │     │                 │
└─────────────────┘     └──────────────────┘     └─────────────────┘
                                ▲
                                │
                       ┌───────────────────┐
                       │    kfp-launcher   │
                       │     ConfigMap     │
                       │                   │
                       └───────────────────┘
```

### Implementation Notes

#### Metadata Handling

The `model.register()` call sets a special metadata field that the launcher processes:

```python
# SDK sets this metadata field
model.metadata["_kfp_model_registry_request"] = json.dumps({
    "model_name": "sentiment-classifier",
    "description": "BERT-based sentiment classification model",
    "model_format_name": "vLLM",
    "model_format_version": None,
    "owner": "ml-team",
    "author": "data-scientist@company.com",
    "continue_on_error": True
})
```

#### Launcher Processing

In `backend/src/v2/component/launcher_v2.go`, here is some sample code to illustrate the potential flow:

```go
// Before uploadOutputArtifacts
var modelRegistryRequests []ModelRegistryRequest
for _, artifact := range outputArtifacts {
    if request, exists := artifact.Metadata["_kfp_model_registry_request"]; exists {
        var req ModelRegistryRequest
        if err := json.Unmarshal([]byte(request), &req); err == nil {
            modelRegistryRequests = append(modelRegistryRequests, req)
        }
        // Remove from metadata to avoid MLMD storage
        delete(artifact.Metadata, "_kfp_model_registry_request")
    }
}

// After uploadOutputArtifacts
for _, req := range modelRegistryRequests {
    if err := registerModelInRegistry(req, artifact); err != nil {
        if !req.ContinueOnError {
            return fmt.Errorf("model registration failed: %w", err)
        }
        log.Warnf("Model registration failed (continuing): %v", err)
    }
}
```

### API Design

#### Model Registry Request Structure

```go
type ModelRegistryRequest struct {
    ModelName         string            `json:"model_name"`
    Description       string            `json:"description,omitempty"`
    ModelFormatName   string            `json:"model_format_name,omitempty"`
    ModelFormatVersion string           `json:"model_format_version,omitempty"`
    Owner             string            `json:"owner,omitempty"`
    Author            string            `json:"author,omitempty"`
    ContinueOnError   bool              `json:"continue_on_error,omitempty"`
}

type ModelRegistryConfig struct {
    URL              string                   `json:"url"`
    TokenSecretRef   TokenSecretReference     `json:"tokenSecretRef"`
    CAConfigMapRef   *CAConfigMapReference    `json:"caConfigMapRef,omitempty"`
    Timeout          string                   `json:"timeout,omitempty"`
    RetryAttempts    int                      `json:"retryAttempts,omitempty"`
    TLSVerify        *bool                    `json:"tlsVerify,omitempty"`
}

type TokenSecretReference struct {
    SecretName      string `json:"secretName"`
    SecretNamespace string `json:"secretNamespace,omitempty"`
    TokenKey        string `json:"tokenKey"`
}
```

### Security Considerations

1. **Token Management**: Authentication tokens stored in Kubernetes secrets
2. **Network Security**: TLS certificate validation for Model Registry connections
3. **Namespace Isolation**: Configuration scoped to individual namespaces
4. **Audit Logging**: All registration attempts are logged
5. **Input Validation**: Limits the fields that can be set by a user

## Test Plan

### Unit Tests

#### Configuration Tests

- Valid configuration parsing
- Invalid configuration error handling
- Default value application

#### SDK Tests

- `model.register()` method validation
- Parameter validation and defaults
- Metadata serialization
- Error handling in SDK

### Integration Tests

#### Successful Scenarios

- Model registration with minimal configuration
- Model registration with full metadata

#### Error Scenarios

- Model Registry API unavailable
- Invalid authentication token
- Invalid model metadata
- Duplicate model version registration

### Graduation Criteria

N/A

## Implementation History

- Initial proposal: 2025-06-27

## Drawbacks

1. **Configuration Overhead**: Requires per-namespace configuration, though this enables proper multi-tenancy
2. **Dependency on Model Registry**: Creates dependency on external Model Registry service availability
3. **API Coupling**: Tight coupling to Model Registry API version and structure

## Alternatives

### Alternative 1: Direct SDK Integration

Instead of launcher-based registration, implement direct Model Registry client integration in the SDK. This would expose
infrastructure details to user code and require authentication handling in components.
