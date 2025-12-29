# KEP-12513: Filesystem-Based Artifact Storage for Kubeflow Pipelines

## Table of Contents

- [Summary](#summary)
- [Architecture at a Glance](#architecture-at-a-glance)
- [Motivation](#motivation)
  - [Enterprise Considerations](#enterprise-considerations)
  - [Per-Namespace Isolation and Scaling](#per-namespace-isolation-and-scaling)
  - [Path to Local Development](#path-to-local-development)
  - [Additional Use Cases](#additional-use-cases)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Overview](#overview)
  - [User Stories](#user-stories)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Configuration Approach](#configuration-approach)
  - [Storage Backend Selection](#storage-backend-selection)
    - [URI, API Endpoint, and Filesystem Path Relationship](#uri-api-endpoint-and-filesystem-path-relationship)
  - [Artifact Server Architecture](#artifact-server-architecture)
    - [Backend Responsibilities](#backend-responsibilities)
    - [Request Routing](#request-routing)
    - [UI Integration](#ui-integration)
  - [PVC Management](#pvc-management)
    - [PVC Creation and Configuration](#pvc-creation-and-configuration)
    - [Storage Quota Enforcement](#storage-quota-enforcement)
  - [Component Modifications](#component-modifications)
  - [Multi-User Isolation and Authorization](#multi-user-isolation-and-authorization)
    - [Namespace Isolation](#namespace-isolation)
    - [Multi-User Isolation Strategy](#multi-user-isolation-strategy)
    - [Dedicated Per-Namespace Artifact Server Architecture](#dedicated-per-namespace-artifact-server-architecture)
    - [Subject Access Review Integration](#subject-access-review-integration)
    - [Single-User vs Multi-User Deployments](#single-user-vs-multi-user-deployments)
  - [Artifact Lifecycle Management](#artifact-lifecycle-management)
    - [Artifact Persistence](#artifact-persistence)
    - [Cleanup Mechanisms](#cleanup-mechanisms)
    - [Caching Support](#caching-support)
    - [Features Not Replicated](#features-not-replicated)
  - [Pipeline Compatibility](#pipeline-compatibility)
- [Test Plan](#test-plan)
  - [Unit Tests](#unit-tests)
  - [Integration Tests](#integration-tests)
- [Configuration Reference](#configuration-reference)
  - [Complete Configuration Example](#complete-configuration-example)
  - [Environment Variable Overrides](#environment-variable-overrides)
- [Migration and Compatibility](#migration-and-compatibility)
  - [Migration Path](#migration-path)
  - [Backward Compatibility](#backward-compatibility)
- [Open Questions](#open-questions)
- [Drawbacks](#drawbacks)
- [Infrastructure Needed](#infrastructure-needed)

## Summary

This KEP proposes adding filesystem-based storage as an alternative artifact storage backend for Kubeflow Pipelines v2. While KFP currently ships with S3-compatible storage by default, some deployments prefer not to depend on a separate object storage system. This proposal introduces filesystem storage as an additional option where artifact handling is integrated into KFP itself, eliminating the need for an external object storage component.

The filesystem backend uses Kubernetes `PersistentVolumeClaims` (PVCs) for storage. KFP creates a shared PVC for the default shared artifact server, and can also create per-namespace PVCs when a namespace is configured to use a dedicated artifact server. **Only the artifact server mounts these PVCs**; pipeline pods never mount them directly. When KFP creates a PVC, it forwards the configured access mode string into the PVC spec (without validation); if not specified, it defaults to `ReadWriteOnce`. The actual behavior depends on what the underlying `StorageClass` supports. Existing pipelines will work without modification unless they contain hardcoded S3/object storage paths.

To support scalability, KFP will introduce a new artifact URI scheme (`kfp-artifacts://`) that routes all artifact requests through KFP's artifact server. Pipeline pods never mount PVCs directly - they upload and download artifacts via the artifact server API, which handles the actual filesystem operations. The artifact server can be scaled independently as an artifacts-only instance.

## Architecture at a Glance

### Quick Comparison

| Aspect                  | Current (S3-compatible Storage)                  | Proposed (Filesystem Storage)                                |
|-------------------------|--------------------------------------------------|--------------------------------------------------------------|
| **Storage Backend**     | S3-compatible (SeaweedFS, MinIO, S3, GCS, etc.)  | Kubernetes PVC                                               |
| **URI Scheme**          | `s3://`, `gs://`, `minio://`                     | `kfp-artifacts://`                                           |
| **Architecture**        | Separate object storage service                  | KFP-native artifact server                                   |
| **Required Knowledge**  | S3 concepts (buckets, endpoints, regions)        | Kubernetes concepts (PVCs, StorageClasses)                   |
| **Multi-tenancy**       | Shared storage (single instance)                 | Shared by default; per-namespace override supported          |
| **Scaling**             | Single object storage instance                   | Shared by default; per-namespace scaling/quotas optional     |

Note: per-namespace overrides can choose filesystem storage (dedicated artifact server + PVC) or keep using S3-compatible storage.

### High-Level Flow

This proposal introduces a filesystem artifact backend **shared-by-default**, with **optional per-namespace overrides**.

#### Inputs → Behavior

| Input                                             | Outcome                                                                                |
|---------------------------------------------------|----------------------------------------------------------------------------------------|
| `defaultPipelineRoot` scheme                      | `kfp-artifacts://` → filesystem via artifact APIs; other schemes unchanged             |
| Namespace `artifactServer.dedicated` (filesystem) | `false` → shared; `true` → dedicated + PVC                                             |
| `ObjectStoreConfig.ArtifactServer.WorkloadKind`   | `deployment` or `daemonset` (`daemonset` → `internalTrafficPolicy: Local`)             |

#### Component Responsibilities

| Component       | Responsibility (filesystem backend only)                                              |
|-----------------|---------------------------------------------------------------------------------------|
| Compiler        | Enable artifact API mode for `kfp-artifacts://`                                       |
| Driver          | Validate artifact infrastructure exists for the namespace                             |
| Launcher        | Upload/download artifacts via artifact server APIs                                    |
| Artifact server | Mount PVC, perform filesystem I/O, authorize via `SubjectAccessReview`                |
| UI/API          | Discover routing via `/apis/v2beta1/filesystem-storage/config` and fetch artifacts    |

## Motivation

**This proposal does not aim to replace existing object storage solutions.** S3-compatible storage remains fully supported and recommended for most production workloads. Instead, this KEP provides an additional option for deployments where a simpler, KFP-native artifact storage solution is preferred. This also enables future work toward running KFP locally (outside Kubernetes) by pointing artifact APIs to local filesystem storage instead of requiring an object store deployment.

While KFP currently ships with S3-compatible storage by default, this still requires deploying and maintaining a separate object storage service. For some deployment scenarios, this additional component may not be desired.

Filesystem storage replaces the object storage system with a KFP-native artifact server and PVCs. This reduces reliance on a separate storage system and shifts operational knowledge from S3 concepts (buckets, endpoints, regions) to Kubernetes concepts (PVCs, `StorageClasses`).

### Enterprise Considerations

Many enterprises and Kubeflow distributions prefer not to have additional external dependencies. With filesystem storage:

- No separate object storage project to productize and support
- Artifact handling is part of the KFP codebase (same team, same release cycle)
- Troubleshooting stays within the KFP domain
- Access control reuses KFP's existing authorization mechanisms, avoiding object-store-specific credential/policy management
- In the default shared setup, artifacts are stored under namespace-aware paths (e.g., `/artifacts/<namespace>/...`) and access is enforced via KFP authorization, providing logical namespace isolation even with shared storage

### Per-Namespace Isolation and Scaling

When configured for per-namespace isolation, a namespace gets its own dedicated artifact server and PVC. This provides:

- **Storage isolation**: artifacts are physically separated per namespace
- **Independent scaling**: teams can size/scale dedicated artifact servers independently
- **Per-namespace quotas**: enforceable outside of KFP (for example via Kubernetes `ResourceQuotas`)

Scaling options for the artifact server include running it as a standard Kubernetes Deployment (default, scale by replicas) or as a DaemonSet via `ObjectStoreConfig.ArtifactServer.WorkloadKind: "daemonset"` (one pod per node, intended for `ReadWriteMany` (RWX) or node-local storage; see [Artifact Server Architecture](#artifact-server-architecture)).

### Path to Local Development

This design enables future work on running full KFP locally without a Kubernetes cluster, as the artifact APIs can point directly to local filesystem storage. This improves developer experience and enables offline development workflows.

### Additional Use Cases

PVC-based storage provides a KFP-native artifact backend option (no separate object storage system) and keeps pipeline definitions portable as long as they rely on KFP-managed artifact passing.

**When to use filesystem storage:**

- Deployments where eliminating object storage dependency is preferred
- Environments where Kubeflow distributions or platform providers do not offer a fully supported object store solution
- Development and experimentation scenarios

**When NOT to use filesystem storage:**

- High-throughput production workloads with large artifacts
- Scenarios requiring object storage features (versioning, lifecycle policies, geo-replication)
- When existing object storage infrastructure is already available and preferred

### Goals

Based on the user story "As a user, I want to provision Kubeflow Pipelines with just a PVC for artifact storage so that I can quickly get started", this KEP aims to:

1. **Add filesystem storage as an additional backend option** alongside S3-compatible and Google Cloud Storage, primarily using PVC but not limited to it
2. **Enable zero-configuration storage** for experimentation use cases - a KFP server can be installed with just a PVC for artifact storage
3. **Provide namespace-isolated artifact storage** with proper subject access review guards in multi-user mode
4. **Allow any Kubernetes access mode to be configured** - KFP passes through the configuration to Kubernetes (defaults to `ReadWriteOnce` (RWO))
5. **Pipeline compatibility (no pipeline code changes)** - pipelines that use KFP-managed artifact passing (e.g., `Input/Output[Dataset|Model]` and `.path`) work unchanged with the filesystem backend (unless they contain hardcoded `s3://`/`gs://`/`minio://` URIs or call object-store SDKs directly)
6. **Match existing artifact persistence behavior** - artifacts persist indefinitely until explicitly deleted (no automatic cleanup)
7. **Enable separate scaling of artifact serving** through an artifacts-only KFP instance with `--artifacts-only` flag
8. **Ensure UI can properly handle artifact downloads** with the new `kfp-artifacts://` URI scheme

### Non-Goals

1. **Replace object storage** - filesystem storage is an additional option, not a replacement
2. **Cross-namespace artifact sharing** - Initial implementation focuses on namespace isolation
3. **Performance optimization for large-scale workloads** - filesystem storage may not match object storage performance for massive datasets or high concurrency scenarios
4. **Migration of existing installations / artifacts** - switching an existing KFP installation from S3-compatible storage to filesystem storage does not migrate previously produced artifacts or rewrite stored artifact URIs (no migration tooling in this KEP)
5. **Advanced storage features** - No versioning, replication, or geo-distribution initially

## Proposal

### Overview

This KEP proposes adding a new artifact storage backend that uses filesystem storage (primarily Kubernetes `PersistentVolumeClaims`) instead of object storage. The implementation will:

1. Create PVC-backed storage for artifact serving (a shared PVC by default, with optional per-namespace PVCs for dedicated artifact servers)
2. Use configurable PVC access mode (defaults to RWO if not specified; RWX is recommended for multi-node clusters)
3. Organize artifacts in a filesystem hierarchy within the PVC that is namespace aware
4. Provide artifact access via KFP's artifact endpoints (`.../artifacts/...:read` and a new `...:write`) using the `kfp-artifacts://` URI scheme
5. Maintain pipeline compatibility for new runs: existing pipeline definitions that use KFP-managed artifact passing (e.g., `Input/Output[Dataset|Model]`, `OutputPath`, and `.path`, and do not hardcode storage-specific URIs/paths) work unchanged when switching the pipeline root to `kfp-artifacts://` (this does not migrate previously stored artifacts/URIs)
6. Support separate scaling of artifact serving through artifacts-only instances
7. Update the UI to seamlessly handle artifact downloads from filesystem storage

### User Stories

#### Story 1: Admin Needing KFP-Native Artifact Storage (On-Prem / Compliance)

As an admin with a requirement to have storage on-premise, I want a simple artifact storage solution for KFP artifacts without having to maintain a separate service for artifacts due to administrative overhead or enterprise licensing costs.

**Acceptance Criteria:**

- KFP can be configured to use filesystem storage (PVC-backed) as an artifact backend without requiring an external object store deployment.
- Artifact upload/download is handled via KFP's artifact server API (pipeline pods do not mount PVCs directly).

#### Story 2: Admin Overriding a Namespace's Artifact Storage

As an admin using the KFP artifact server, I want the ability to override a namespace's artifact configuration to use alternative storage such as S3 or a dedicated artifact server.

**Acceptance Criteria:**

- A namespace can override the default shared artifact server behavior without affecting other namespaces.
- Per-namespace override can either:
  - keep using S3-compatible storage (`s3://`, `minio://`, etc.), or
  - opt into filesystem storage with a dedicated artifact server + PVC (`artifactServer.dedicated: true`).

### Notes/Constraints/Caveats

1. **Performance**: filesystem storage may have different performance characteristics than object storage, especially for large files or high concurrency
2. **Storage Classes**: Functionality depends on available `StorageClasses` - RWX preferred for parallel execution
3. **Scalability**: filesystem storage is not designed for massive scale like object storage
4. **Cost**: Cost depends on the storage class and provider - could be more or less expensive than object storage
5. **Portability**: Pipelines using filesystem storage are tied to Kubernetes infrastructure
6. **No Migration Support**: Migrating existing artifacts from S3-compatible storage to filesystem storage is not supported in this KEP (no artifact copy or URI rewrite tooling)

### Risks and Mitigations

| Risk                                               | Impact                                                                             | Mitigation                                                                                       |
|----------------------------------------------------|------------------------------------------------------------------------------------|--------------------------------------------------------------------------------------------------|
| Breaking changes to existing pipelines             | Pipelines with hardcoded S3 paths fail if S3 becomes inaccessible                  | Document clearly that storage-specific code depends on service availability                      |
| RWO default in multi-node clusters                 | Pods scheduled on different nodes cannot mount the same RWO PVC                    | See [Known Limitation: RWO in Multi-Node Clusters](#known-limitation-rwo-in-multi-node-clusters) |
| Performance regression for parallel tasks with RWO | Tasks that could run in parallel may be forced to run sequentially or on same node | Require explicit scheduling configuration for RWO usage                                          |
| Confusion about storage backend capabilities       | Users may expect S3-specific features not exposed by KFP API                       | Clear documentation that filesystem storage provides basic file storage only                     |

## Design Details

### Configuration Approach

This KEP extends KFP's existing configuration structure rather than creating new top-level configurations. All new filesystem storage configurations will be added under the existing `ObjectStoreConfig` structure in the `config.json` file. This approach ensures:

1. **Backward Compatibility**: Existing deployments continue working without any changes
2. **Logical Organization**: filesystem storage is conceptually an alternative to object storage, so it belongs in the same configuration section
3. **Minimal Code Changes**: Reuses existing configuration loading and validation logic
4. **Familiar Pattern**: Follows the same pattern as database configuration where MySQL and PostgreSQL share the `DBConfig` section

All new configurations will be added as new fields under the existing `ObjectStoreConfig`.

Throughout this KEP, when new configurations are introduced, they will be referenced by their full path (e.g., `ObjectStoreConfig.Filesystem.PVC.Size`) to make the hierarchy clear. All configurations can be overridden via environment variables following KFP's naming convention where dots become underscores (e.g., `OBJECTSTORECONFIG_FILESYSTEM_PVC_SIZE`).

### Storage Backend Selection

Storage backends in KFP are determined by the `defaultPipelineRoot` URL scheme, which can be configured at multiple levels:

1. **System default** (single-user mode): Set in `pipeline-install-config` ConfigMap
2. **Namespace default** (multi-user mode): Set in each namespace's `kfp-launcher` ConfigMap  
3. **Pipeline specific**: Set via `@pipeline(pipeline_root='...')` decorator in Python SDK
4. **Runtime override**: Set via `pipeline_root` parameter when submitting a run

For filesystem storage, this KEP introduces a new URL scheme:

```text
kfp-artifacts://<namespace>/<pipeline-name>/<run-id>/<node-id>/<artifact-name>
```

This follows the same pattern as existing storage schemes (s3://, gs://, minio://), where the namespace maps to the bucket name and the rest maps to the prefix.

Example configurations:

```yaml
# Both modes use the same path structure for consistency
# {namespace} is substituted at runtime (single-user mode uses the KFP install namespace)
defaultPipelineRoot: "kfp-artifacts://{namespace}"
```

#### URI, API Endpoint, and Filesystem Path Relationship

There are three related but distinct formats:

| Concept                  | Format                                                                           | Example                                                       |
|--------------------------|----------------------------------------------------------------------------------|---------------------------------------------------------------|
| **URI** (stored in MLMD) | `kfp-artifacts://<namespace>/<pipeline-name>/<run-id>/<node-id>/<artifact-name>` | `kfp-artifacts://team-a/my-pipeline/abc123/step1/output`      |
| **API Endpoint**         | `/apis/v2beta1/runs/{run_id}/nodes/{node_id}/artifacts/{artifact_name}:read`     | `/apis/v2beta1/runs/abc123/nodes/step1/artifacts/output:read` |
| **Filesystem Path**      | `<mount_path>/<namespace>/<pipeline-name>/<run-id>/<node-id>/<artifact-name>`    | `/artifacts/team-a/my-pipeline/abc123/step1/output`           |

The API endpoint only needs `run_id`, `node_id`, and `artifact_name`; the server derives the filesystem path from run metadata (namespace + pipeline name), consistent with existing behavior for object storage backends. The namespace segment in the URI is also used for routing when a namespace opts into a dedicated artifact server.

### Artifact Server Architecture

#### Backend Responsibilities

KFP uses a shared artifact server by default, and namespaces can optionally be configured to use a dedicated artifact server and PVC for physical isolation. **Only artifact server pods mount PVCs - pipeline workflow pods access artifacts exclusively through the artifact server API.**

##### Artifact Server Workload Type (Deployment vs DaemonSet)

In addition to whether a namespace uses the shared artifact server or a dedicated per-namespace artifact server, the artifact server can be deployed using different Kubernetes workload types:

- **Deployment (default)**: Runs a configurable number of replicas. This is the recommended default for most clusters.
- **DaemonSet (optional)**: Runs one artifact server pod per node. When using `WorkloadKind: "daemonset"`, KFP configures the artifact server Service with `internalTrafficPolicy: Local` so artifact traffic stays node-local (requests only route to endpoints on the same node).

**Important**: DaemonSet + `internalTrafficPolicy: Local` only works when the underlying storage can be accessed by an artifact server pod on every node (for example, RWX storage such as NFS/CephFS, or node-local volumes). With RWO PVCs, most DaemonSet pods will be unable to mount the PVC and local-only routing would fail for clients on other nodes.

##### Default: Shared Artifact Server

A single artifact server in the main KFP namespace serves all namespaces by default.

**Characteristics:**

- **Single PVC** with directory structure: `/artifacts/<namespace>/<pipeline>/<run-id>/<node-id>/<artifact-name>`
- **Authorization**: Uses `SubjectAccessReview` to verify namespace access
- **Best for**: Simple deployments, single-user setups, small teams
- **Advantages**: Simple setup, single storage location, easy backup
- **Limitations**: All namespaces share same PVC and storage quota

##### Optional: Dedicated Artifact Server per Namespace

Each namespace can run its own artifact server when configured via its `kfp-launcher` ConfigMap (`artifactServer.dedicated: true`).

**Characteristics:**

- **PVC per namespace**: Complete storage isolation
- **Direct access**: Clients connect directly to namespace servers (no proxying)
- **Authorization**: Natural isolation (each server only accesses its namespace's PVC)
- **Best for**: Large multi-tenant deployments, strict isolation requirements
- **Advantages**: True multi-tenancy, per-namespace scaling, independent quotas
- **Deployment**: Proactive initialization when namespace has `pipelines.kubeflow.org/enabled=true` annotation

##### Per-Namespace Configuration Support

For multi-tenant deployments with varying isolation requirements, administrators can configure namespace-local artifact servers per namespace using the `kfp-launcher` ConfigMap:

Configure the namespace `kfp-launcher` ConfigMap to:

- set `defaultPipelineRoot` to `kfp-artifacts://<namespace>`, and
- set `artifactServer.dedicated: true` to opt the namespace into a dedicated artifact server + PVC.

**Notation note**: In this document, `artifactServer.dedicated` is a key-path notation. In the `kfp-launcher` ConfigMap it is expressed as the `dedicated` field inside the YAML document stored under the `artifactServer` data key (see the example in the Configuration Reference).

The per-namespace `artifactServer` YAML mirrors the global `ObjectStoreConfig.ArtifactServer` structure.

#### Request Routing

Artifact URIs are resolved based on whether a namespace has opted into a dedicated artifact server via its `kfp-launcher` ConfigMap:

Routing rules:

- **Shared default**: serve via shared artifact server.
- **Dedicated override**: for namespaces with `artifactServer.dedicated: true`, route to the namespace-local artifact server service.

**Note on Path Structure**: The dedicated per-namespace setup still includes `<namespace>` in the filesystem path to keep path/URI handling consistent across shared and dedicated setups.

##### Architecture Diagrams

###### Shared Artifact Server Architecture

```text
┌──────────────────────────────────────┐
│              KFP SDK                 │
└──────────────────┬───────────────────┘
                   │ pipeline_root: "kfp-artifacts://..."
                   ▼
┌──────────────────────────────────────┐
│           Pipeline Spec              │
└──────────────────┬───────────────────┘
                   │
                   ▼
┌──────────────────────────────────────┐
│             Compiler                 │
│     (generates artifact API calls)   │
└──────────────────┬───────────────────┘
                   │
                   ▼
┌──────────────────────────────────────┐
│              Driver                  │
│   (validates artifact server exists) │
└──────────────────┬───────────────────┘
                   │
                   ▼
┌──────────────────────────────────────┐
│             Launcher                 │
│     (uploads/downloads via API)      │
└──────────────────┬───────────────────┘
                   │ API calls
                   ▼
┌──────────────────────────────────────┐
│      Central Artifact Server         │
│   (namespace: KFP install namespace) │
│                                      │
│  • SubjectAccessReview               │
│  • Mounts shared PVC                 │
│  • Serves all namespaces             │
└──────────────────┬───────────────────┘
                   │ mounts
                   ▼
┌──────────────────────────────────────┐
│  Shared PVC (kfp-artifacts-central)  │
│  /artifacts/                         │
│    ├── ns1/                          │
│    ├── ns2/                          │
│    └── ns3/                          │
└──────────────────────────────────────┘
```

###### Dedicated Per-Namespace Architecture

```text
┌──────────────────────────────────────┐
│              KFP SDK                 │
└──────────────────┬───────────────────┘
                   │ pipeline_root: "kfp-artifacts://..."
                   ▼
┌──────────────────────────────────────┐
│           Pipeline Spec              │
└──────────────────┬───────────────────┘
                   │
                   ▼
┌──────────────────────────────────────┐
│             Compiler                 │
│     (generates artifact API calls)   │
└──────────────────┬───────────────────┘
                   │
                   ▼
┌──────────────────────────────────────┐
│              Driver                  │
│   (validates artifact server exists) │
└──────────────────┬───────────────────┘
                   │
                   ▼
┌──────────────────────────────────────┐
│             Launcher                 │
│   (uploads/downloads via namespace   │
│    artifact server API)              │
└──────────────────┬───────────────────┘
                   │ API calls to namespace servers
                   │
       ┌───────────┼───────────┬───────────┐
       ▼           ▼           ▼           ▼
┌────────────┐ ┌────────────┐ ┌────────────┐ ┌─────┐
│  Artifact  │ │  Artifact  │ │  Artifact  │ │     │
│  Server    │ │  Server    │ │  Server    │ │ ... │
│  (ns1)     │ │  (ns2)     │ │  (ns3)     │ │     │
└─────┬──────┘ └─────┬──────┘ └─────┬──────┘ └──┬──┘
      │ mounts       │ mounts       │ mounts    │
      ▼              ▼              ▼           ▼
┌────────────┐ ┌────────────┐ ┌────────────┐ ┌─────┐
│  PVC (ns1) │ │  PVC (ns2) │ │  PVC (ns3) │ │ ... │
│ /artifacts │ │ /artifacts │ │ /artifacts │ │     │
└────────────┘ └────────────┘ └────────────┘ └─────┘

Direct client access to namespace servers (no proxying)
```

#### UI Integration

The UI discovers artifact server routing through the KFP API server, which acts as a registry. The API server already watches namespaces and reads their `kfp-launcher` ConfigMaps to determine whether a namespace has opted into a dedicated artifact server, so it can derive the routing information without additional infrastructure.

##### Filesystem Storage Configuration Endpoint

A dedicated endpoint provides filesystem storage configuration and routing:

```http
GET /apis/v2beta1/filesystem-storage/config
```

```json
{
  "enabled": true,
  "dedicated_namespaces": ["team-a", "team-b"]
}
```

When filesystem storage is not enabled (using S3/GCS/MinIO):

```json
{
  "enabled": false
}
```

##### Override Rules and Rationale

Namespaces can opt in to dedicated per-namespace artifact servers via their `kfp-launcher` ConfigMap. If `artifactServer.dedicated` is omitted (or set to `false`), the namespace uses the shared artifact server.

| Namespace Setting (`kfp-launcher`) | Result                                  | Shared Server Deployed? |
|------------------------------------|-----------------------------------------|-------------------------|
| (none) or `dedicated: false`       | Shared artifact server                  | Yes                     |
| `dedicated: true`                  | Dedicated per-namespace artifact server | Yes                     |

**Rationale:**

1. **Common use case is isolation opt-in**: Organizations typically start with the shared artifact server (default) for simplicity, then specific teams with strict requirements (security, compliance) opt in to dedicated per-namespace servers for physical isolation.

This keeps the configuration model simple while covering realistic use cases.

##### How the API Server Builds the Response

The API server combines global configuration with per-namespace `kfp-launcher` ConfigMaps by listing namespaces that have opted into dedicated artifact servers.

##### UI Usage

The UI fetches `/apis/v2beta1/filesystem-storage/config`, caches the response, and routes requests to either the shared service or the namespace-local service based on whether the namespace is listed in `dedicated_namespaces`.

##### Implementation Details

- **Dedicated endpoint**: `/filesystem-storage/config` returns configuration and routing
- **Single source of truth**: API server derives routing from per-namespace ConfigMaps (shared is default; namespaces can opt in to dedicated)
- **Cacheable**: UI can cache the response with 1 minute TTL
- **Works for shared + per-namespace**: Shared default with optional dedicated per-namespace overrides

##### Artifact Server API Specification

The artifact server must implement the following REST API endpoints. Note that:

- The existing read endpoint is currently **only used by the persistence agent** for metrics extraction
- Pipeline pods (both v1 and v2) currently write directly to object storage, not through the API
- The UI uses its own proxy endpoint (`/artifacts/get`) that directly accesses object storage, not the artifact API
- For filesystem storage, we need both read and write endpoints since pipeline pods cannot directly access the PVC

###### Read Artifact Endpoint

```http
GET /apis/v2beta1/runs/{run_id}/nodes/{node_id}/artifacts/{artifact_name}:read
GET /apis/v1beta1/runs/{run_id}/nodes/{node_id}/artifacts/{artifact_name}:read
```

Note: Both v1beta1 and v2beta1 endpoints are supported because both V1 and V2 pipelines will need to read artifacts via this API when using filesystem storage. Both endpoints use the same implementation. There is an open question about whether this API should be considered public or internal. See [Open Questions: External Client Support](#open-questions).

**Request Parameters:**

- `run_id`: The pipeline run ID
- `node_id`: The node/task ID within the pipeline
- `artifact_name`: Name of the artifact to retrieve

**Response:**

- Content-Type: `application/json`
- Body: Single JSON object with base64-encoded tar.gz content

```json
{
  "data": "<base64-encoded-tar.gz-content>"
}
```

**Compatibility Rationale:**

The JSON wrapper format with base64-encoded tar.gz content is required for backward compatibility with existing KFP components:

- **Launcher**: Expects this format when downloading input artifacts (see `backend/src/v2/component/launcher_v2.go`)
- **Driver**: Uses this format to pass artifacts between pipeline steps
- **Persistence Agent**: Stores artifact metadata expecting this structure
- **UI**: Frontend artifact preview expects to decode base64 JSON responses

This matches the existing MinIO/S3 artifact server API which returns artifacts in the same JSON format. Changing to raw file downloads would break these components and require extensive refactoring across the codebase.

**Implementation Notes:**

- The server uses HTTP chunked transfer encoding to stream the response
- Builds a single JSON document by writing `{"data":"`, streaming base64-encoded content in chunks, then writing `"}`
- This approach avoids loading large files into memory while producing valid JSON
- HTTP clients automatically handle the chunked transfer encoding
- Artifacts are stored as tar.gz (gzip-compressed tar archives) on the filesystem
- The response format matches the existing KFP artifact server implementation

**Authorization:**

- Validates access using `SubjectAccessReview` with verb `readArtifact`
- Returns 403 Forbidden if unauthorized

###### Write Artifact Endpoint

```http
POST /apis/v2beta1/runs/{run_id}/nodes/{node_id}/artifacts/{artifact_name}:write
POST /apis/v1beta1/runs/{run_id}/nodes/{node_id}/artifacts/{artifact_name}:write
```

Note: This is a new endpoint that doesn't exist in the current KFP artifact server. Both v1beta1 and v2beta1 versions are needed because both V1 and V2 pipelines will use this API for filesystem storage (they currently access object storage directly).

**Request:**

- Content-Type: `application/octet-stream`
- Body: Raw tar.gz binary content (streamed)

**Response:**

- 200 OK on success
- Returns the artifact URI in response

```json
{
  "uri": "kfp-artifacts://<namespace>/<pipeline-name>/<run_id>/<node_id>/<artifact_name>"
}
```

**Implementation Notes:**

- The write endpoint uses raw binary streaming (not base64-encoded JSON) to efficiently handle large artifacts like multi-GB model checkpoints.
- This differs from the read endpoint (which uses base64 JSON for backward compatibility with existing clients).
- The server streams the request body directly to disk without buffering the entire artifact in memory.
- Uploaded artifacts should already be in tar.gz format (matching existing KFP behavior).

**Authorization:**

- Validates access using `SubjectAccessReview` with verb `writeArtifact`
- Returns 403 Forbidden if unauthorized

##### Backend Implementation

The artifact server implements streaming for efficient handling of large artifacts:

The artifact server uses the existing `ObjectStore` interface pattern. For filesystem storage, we add a new implementation:

A new `PVCObjectStore` implementation will be added that:

- Implements the existing `ObjectStore` interface
- Reads/writes artifacts directly to the mounted PVC path
- Handles directory creation and file operations
- Works alongside existing `BlobObjectStore` (S3/GCS/MinIO)

The artifact server remains unchanged for reads and adds a new write handler.

##### Artifact Display Handling

For filesystem storage, the UI proxies artifact reads through the artifact server read endpoint (PVCs are not directly accessible to the UI).

### PVC Management

The PVC lifecycle is controlled by the `ObjectStoreConfig.Filesystem.PVC` configuration fields.

#### PVC Creation and Configuration

`PersistentVolumeClaims` are created automatically when the artifact server is deployed in a namespace. The PVC uses a standard naming convention (`kfp-artifacts-{namespace}`) and the configured storage size, access mode, and storage class. Pipeline pods interact with artifacts exclusively through the artifact server API, ensuring proper isolation and access control.

##### Create-on-Missing Behavior (`ObjectStoreConfig.Filesystem.PVC.CreateIfNotExists`)

Controls whether KFP creates the expected PVC when provisioning filesystem storage:

- `true` (default): the control plane creates the PVC for the shared artifact server and for any namespace that opts into a dedicated artifact server.
- `false`: KFP does not create PVCs. The PVC must already exist (same naming convention); otherwise, filesystem storage setup fails during prerequisite validation.

##### StorageClass Selection (`ObjectStoreConfig.Filesystem.PVC.StorageClassName`)

Selects which `StorageClass` to use for PVC provisioning (empty string uses the cluster default).

##### Access Mode Configuration (`ObjectStoreConfig.Filesystem.PVC.AccessMode`)

- Passed directly to Kubernetes without validation
- Defaults to `ReadWriteOnce` if not specified
- Actual behavior depends on what your `StorageClass` supports

##### Known Limitation: RWO in Multi-Node Clusters

The default `ReadWriteOnce` (RWO) access mode constrains PVC mounting to a single node at a time.

Key implications:

- With `WorkloadKind: "daemonset"`, RWO is not compatible in multi-node clusters (DaemonSet requires RWX or node-local storage).
- With `WorkloadKind: "deployment"`, horizontal scaling is constrained by where the PVC can be mounted.

#### Storage Quota Enforcement

Storage quotas are configured outside of KFP (e.g., via Kubernetes `ResourceQuotas`); KFP does not implement quota management.

### Component Modifications

#### Object Store Interface (`backend/src/v2/objectstore/`)

The object store interface remains unchanged for S3/GCS. For filesystem storage, artifact operations are redirected to the artifact server API:

- `kfp-artifacts://` is treated as filesystem artifact mode: artifacts are read/written via the artifact server API (not via direct bucket/object store access).
- The namespace is derived from the first path segment after `kfp-artifacts://` (similar to how KFP maps namespace → bucket for S3-compatible storage).

#### KFP API Server Artifact Management

The KFP API server manages artifact infrastructure lifecycle for the shared artifact server and for namespaces configured to use dedicated artifact servers:

##### Namespace Annotation-Based Provisioning

The shared artifact server (default) is managed in the KFP install namespace and does not depend on namespace annotations (including in single-user mode).

Namespace annotations are only used to drive optional per-namespace provisioning in multi-tenant setups. In multi-user Kubeflow, the Profile controller creates user namespaces (Profiles) and can apply the `pipelines.kubeflow.org/enabled: "true"` annotation to those namespaces. The `kubeflow` namespace is the control-plane namespace, not a Profile. For standalone KFP, namespaces can be annotated directly.

For annotated namespaces, the API server reads the namespace `kfp-launcher` ConfigMap to determine whether `artifactServer.dedicated: true` is set; if so, it provisions dedicated per-namespace infrastructure.

##### API Server Implementation

Implementation notes:

- Shared artifact server is the default, created/verified during API server startup.
- Namespaces opt into dedicated artifact servers via `artifactServer.dedicated: true` in their `kfp-launcher` ConfigMap.
- The manager reconciles PVC/Service/workload resources idempotently for opted-in namespaces.

#### Launcher (`backend/src/v2/component/launcher_v2.go`)

The launcher will use the artifact server API for all artifact operations when filesystem storage is configured:

The launcher will be modified to:

- Detect `kfp-artifacts://` URIs
- Use the artifact server API for filesystem artifacts instead of direct storage access
- Call the read/write endpoints with proper authentication
- Continue using existing `objectstore` package for S3/GCS/MinIO

#### Compiler (`backend/src/v2/compiler/argocompiler/`)

Configure launcher to use artifact API when filesystem storage is detected:
When `pipeline_root` uses `kfp-artifacts://`, the compiler configures the launcher to enable artifact API mode and to use the resolved artifact server endpoint (no PVC mounts in pipeline pods).

#### Driver (`backend/src/v2/driver/`)

The driver validates that required artifact infrastructure exists before pipeline execution:

- If not using `kfp-artifacts://`, no filesystem checks are needed.
- If the namespace opted into a dedicated artifact server, verify the namespace-local Service/workload exist.
- Otherwise, the shared artifact server is used.

### Multi-User Isolation and Authorization

In multi-user mode, KFP enforces isolation and authorization via `SubjectAccessReview` and the chosen artifact server topology (shared vs dedicated).

#### Permission Model

The filesystem storage implementation should follow the principle of least privilege:

**KFP API Server Permissions:**

For both the shared and dedicated setups, the API server needs:

- Watch `namespaces` for the enabling annotation.
- Create/read PVCs, Services, and the artifact server workload (Deployment or DaemonSet), scoped appropriately (shared infra in the KFP namespace; dedicated infra in user namespaces).

**Driver/Workflow Pod Permissions:**

- Read-only access to verify the artifact server workload and Service exist when validating filesystem storage prerequisites.

**Artifact Server Pod Permissions:**

- Create `SubjectAccessReview` resources for authorization checks.

This keeps infrastructure management in the control plane, while workflow pods only perform read-only verification.

#### Namespace Isolation

Namespace isolation depends on whether the namespace uses the shared default artifact server or a dedicated per-namespace artifact server:

1. **Shared default**: a single shared PVC is used, with namespace-aware paths (`/artifacts/<namespace>/...`) and `SubjectAccessReview` enforcing access.
2. **Dedicated per-namespace**: the namespace has its own PVC (physical isolation), and the namespace-local artifact server only mounts that PVC.

#### Multi-User Isolation Strategy

The approach to multi-user isolation depends on whether a namespace uses the shared artifact server or a dedicated per-namespace artifact server:

##### Shared Artifact Server Isolation

- Directory-based separation: Artifacts stored in `/artifacts/<namespace>/...` structure
- Authorization checks: Every request validated using `SubjectAccessReview`
- Single PVC: Shared storage with logical separation

##### Dedicated Per-Namespace Isolation

- Physical separation: Each namespace has its own PVC
- Network accessibility: Artifact servers can be accessed from other namespaces
- Authorization: All requests validated via `SubjectAccessReview`

#### Dedicated Per-Namespace Artifact Server Architecture

With the dedicated per-namespace setup, each namespace runs its own artifact server and mounts its own PVC; control-plane components route to the namespace-local service and requests are authorized via `SubjectAccessReview`.

#### Subject Access Review Integration

Both the shared and dedicated setups use KFP's existing authorization layer with Kubernetes `SubjectAccessReview` before reading/writing any artifact content.

#### Single-User vs Multi-User Deployments

The namespace-isolated design works for both the shared and dedicated setups:

##### Multi-User Mode

- Shared default uses a single PVC with namespace-aware paths plus `SubjectAccessReview` enforcement.
- Dedicated per-namespace mode provides a PVC per opted-in namespace (physical isolation).

##### Single-User Mode

- Typically uses one namespace (e.g., `kubeflow`)
- One PVC for all pipelines: `kfp-artifacts-kubeflow`
- If multiple namespaces are used, each still gets its own PVC (same model applies)
- Simpler than S3 (no credentials needed) while maintaining security best practices

This unified approach avoids code complexity while providing secure defaults for all deployment types.

### Artifact Lifecycle Management

Filesystem storage matches current persistence behavior (artifacts persist until explicitly deleted) but does not replicate object-storage-specific lifecycle features.

#### Artifact Persistence

Behavior matches the current object-storage backend: artifacts are persisted and not automatically deleted on run deletion.

#### Cleanup Mechanisms

- No automatic cleanup is introduced in this KEP (consistent with current behavior).

#### Caching Support

Caching behavior is unchanged.

#### Features Not Replicated

Filesystem storage does not replicate S3-specific features such as:

- Object versioning
- Lifecycle policies (e.g., auto-delete after 30 days)
- Cross-region replication
- Object tagging and metadata
- S3 Select, Glacier transitions, etc.

These features are considered out of scope for the initial implementation as they're not essential for the experimentation use case.

### Pipeline Compatibility

#### What Works Across All Storage Backends

Pipelines that use KFP's standard artifact mechanisms work with any storage backend without modification:
For example, using KFP-managed artifact passing (e.g., `Input/Output[Dataset|Model]`, `OutputPath`, and `.path`) remains portable across storage backends because KFP manages the artifact I/O behind the scenes.

#### What Doesn't Work

Pipelines with storage-specific code may fail depending on service availability:
For example, components that directly call an object-store SDK (e.g., `boto3`) still require that service to be reachable, regardless of which backend KFP uses for its own managed artifacts.

**This is expected behavior** - KFP doesn't validate or prevent storage-specific code. Components with storage-specific operations will behave according to their implementation - they may fail if the required service isn't available, or they may succeed if they can still access that service. The artifacts managed by KFP will be stored in the configured backend (PVC in this case), but any direct storage operations in the component code are independent of KFP's configuration.

## Test Plan

### Unit Tests

#### `PVCObjectStore` Implementation

- `GetFileReader()` returns valid reader for existing artifact path
- `AddFile()` writes content to correct filesystem path
- `DeleteFile()` removes artifact from filesystem
- Path construction: `<namespace>/<pipeline>/<run-id>/<node-id>/<artifact-name>` maps to `<mount_path>/<namespace>/<pipeline>/<run-id>/<node-id>/<artifact-name>`
- Error handling for non-existent paths, permission errors, disk full

#### Launcher Artifact Client

- Parses `kfp-artifacts://` URIs and extracts path components
- Download: Calls artifact server API, decodes base64, extracts tar.gz
- Upload: Creates tar.gz, streams raw binary to artifact server API
- Existing blob storage (`s3://`, `gs://`, `minio://`) continues to work unchanged

#### Artifact Server API Endpoints

- `ReadArtifact`: Returns base64-encoded tar.gz content in JSON response
- `WriteArtifact`: Accepts raw tar.gz stream, writes to correct path, returns URI
- `SubjectAccessReview` authorization validates user permissions before read/write
- Streaming: Large artifacts streamed without loading entirely into memory

#### `ArtifactServerManager` (API Server)

- Ensures the shared artifact server (default) exists in the KFP install namespace when filesystem storage is enabled
- Watches KFP-enabled namespaces (e.g., `pipelines.kubeflow.org/enabled: "true"`) for optional per-namespace provisioning
- Creates per-namespace PVC and artifact server workload only when the namespace opts in (`artifactServer.dedicated: true` in `kfp-launcher`)
- Skips namespaces where infrastructure already exists
- Idempotent: Re-processing same namespace doesn't create duplicate resources
- Different namespaces can use different settings (shared default or dedicated per-namespace)

#### Configuration Parsing

- `ObjectStoreConfig.Filesystem.PVC.StorageClassName` sets PVC storage class
- `ObjectStoreConfig.Filesystem.PVC.Size` sets PVC capacity
- `ObjectStoreConfig.Filesystem.PVC.AccessMode` defaults to `ReadWriteOnce`
- `ObjectStoreConfig.Filesystem.PVC.CreateIfNotExists` controls whether KFP auto-creates the PVC (default `true`); when `false`, the PVC must already exist
- Per-namespace `kfp-launcher` ConfigMap `artifactServer.dedicated: true` opts a namespace into a dedicated per-namespace server (default is shared)

### Integration Tests

#### Story 1: KFP-Native Filesystem Artifacts (No External Object Store)

- Deploy KFP with filesystem storage enabled and `defaultPipelineRoot: "kfp-artifacts://"` (shared artifact server by default)
- Run a pipeline with two components passing artifacts via KFP-managed mechanisms (e.g., `Output[Dataset]` → `Input[Dataset]`)
- Verify:
  - artifacts are stored under the expected namespace-aware path (e.g., `/artifacts/<namespace>/<pipeline-name>/<run-id>/...`)
  - artifact server read/write endpoints return the correct content
  - cross-namespace access is denied via `SubjectAccessReview` (multi-user mode)

#### Story 2: Per-Namespace Overrides (S3-Compatible or Dedicated Server+PVC)

- Deploy KFP with filesystem storage enabled (shared default)
- Configure namespaces to exercise overrides:
  - `team-a`: no override (uses shared artifact server)
  - `team-b`: `artifactServer.dedicated: true` (dedicated artifact server + PVC)
  - `team-c`: override `defaultPipelineRoot` to an S3-compatible scheme (e.g., `s3://...` or `minio://...`)
- Verify:
  - KFP routes artifact operations correctly per namespace (shared vs dedicated vs S3-compatible)
  - enabling/disabling a namespace override does not require a cluster-wide restart
  - when `WorkloadKind: "daemonset"` is configured, the expected Kubernetes resources are deployed (DaemonSet and Service with `internalTrafficPolicy: Local`)

## Configuration Reference

### Complete Configuration Example

Here's a complete example showing all new configuration fields for filesystem storage:

```json
{
  "ObjectStoreConfig": {
    "Filesystem": {
      "MountPath": "/artifacts",
      "PVC": {
        "StorageClassName": "standard",
        "Size": "100Gi",
        "AccessMode": "ReadWriteMany",
        "CreateIfNotExists": true
      }
    },
    
    "ArtifactServer": {
      "WorkloadKind": "deployment"
    }
  }
}
```

### Configuration Field Reference
| Field                                                | Description                       | Default                         | Valid Values                            |
|------------------------------------------------------|-----------------------------------|---------------------------------|-----------------------------------------|
| `ObjectStoreConfig.Filesystem.Type`                  | Storage backend type              | -                               | `"pvc"`, `"local"` (testing only)       |
| `ObjectStoreConfig.Filesystem.MountPath`             | Path where PVC is mounted         | `"/artifacts"`                  | Any valid path                          |
| `ObjectStoreConfig.Filesystem.PVC.StorageClassName`  | K8s StorageClass to use           | empty string (uses cluster default) | Any available StorageClass              |
| `ObjectStoreConfig.Filesystem.PVC.Size`              | Size of PVC to create             | `"10Gi"`                        | K8s quantity (e.g., `"100Gi"`, `"1Ti"`) |
| `ObjectStoreConfig.Filesystem.PVC.AccessMode`        | PVC access mode                   | `"ReadWriteOnce"`               | `"ReadWriteOnce"`, `"ReadWriteMany"`    |
| `ObjectStoreConfig.Filesystem.PVC.CreateIfNotExists` | Auto-create PVC if missing        | `true`                          | `true`, `false`                         |
| `ObjectStoreConfig.ArtifactServer.WorkloadKind`      | Artifact server workload kind     | `"deployment"`                  | `"deployment"`, `"daemonset"`           |

**Notes:**

- `WorkloadKind: "daemonset"` is intended to be used with RWX or node-local storage. When set, KFP configures the artifact server Service with `internalTrafficPolicy: Local` so requests only route to endpoints on the same node.

#### Per-Namespace Configuration (kfp-launcher ConfigMap)

Namespaces can opt in to dedicated per-namespace artifact servers by setting `artifactServer.dedicated: true` in their `kfp-launcher` ConfigMap. If `artifactServer` is omitted (or `dedicated` is omitted / `false`), the namespace uses the shared artifact server by default.

| Key                        | Description                                                               | Default                  | Valid Values                                                         |
|----------------------------|---------------------------------------------------------------------------|--------------------------|----------------------------------------------------------------------|
| `defaultPipelineRoot`      | Artifact storage root URI                                                 | `minio://mlpipeline/...` | `minio://...`, `s3://...`, `gs://...`, `kfp-artifacts://<namespace>` |
| `artifactServer.dedicated` | (Key path) `dedicated` inside the `artifactServer` YAML value             | `false`                  | `true`, `false`                                                      |

**Example ConfigMap for dedicated per-namespace artifact server:**

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: kfp-launcher
  namespace: team-a
data:
  defaultPipelineRoot: "kfp-artifacts://team-a"
  artifactServer: |
    dedicated: true
```

**Dedicated Artifact Server Opt-In Scenarios:**

| Namespace Setting (`kfp-launcher`) | Result                                       |
|------------------------------------|----------------------------------------------|
| (none) or `dedicated: false`       | Uses shared artifact server                  |
| `dedicated: true`                  | Uses dedicated per-namespace artifact server |

### Environment Variable Overrides

All configuration fields can be overridden via environment variables:
This follows KFP's existing convention where the full configuration path is uppercased and all dots are replaced with underscores (for example, `ObjectStoreConfig.Filesystem.PVC.Size` → `OBJECTSTORECONFIG_FILESYSTEM_PVC_SIZE`).

## Migration and Compatibility

### Migration Path

To switch an installation from S3-compatible storage to filesystem storage for **new runs**:

1. Update `defaultPipelineRoot` to use `kfp-artifacts://` scheme
2. Add `ObjectStoreConfig.Filesystem` configuration
3. Configure `ObjectStoreConfig.ArtifactServer` based on deployment needs
4. Restart KFP components to apply changes

**Note**: Existing runs/artifacts are not migrated. Artifact URIs already recorded in MLMD remain `s3://`/`minio://`/`gs://` (etc.) and continue to refer to the original object store location. This KEP does not include tooling to copy historical artifacts or rewrite stored URIs.

### Backward Compatibility

- Existing configurations continue to work unchanged
- Storage backend is auto-detected from the URL scheme in `defaultPipelineRoot`
- Object storage fields are ignored when using `kfp-artifacts://` scheme
- Filesystem fields are ignored when using `s3://`, `gs://`, or `minio://` schemes
  
## Open Questions

1. **Mixed Storage Backend Testing**: Should we explicitly test and document mixed storage backend scenarios?
   - KFP already supports mixing S3 and GCS backends via per-pipeline configuration
   - Filesystem backend will work the same way (via `kfp-artifacts://` URLs)
   - Question: Should we add integration tests for mixed S3+PVC or GCS+PVC deployments?
   - Consideration: Added test complexity vs. validation of real-world usage patterns

2. **Profile Controller Integration for Namespaced Opt-In**: How should the Profile Controller support per-namespace dedicated artifact server opt-in configuration?
   - Option A: Profile Controller always creates ConfigMap with explicit `artifactServer` block from a template
   - Option B: Profile Controller omits `artifactServer` block, letting namespaces use shared by default unless manually overridden
   - Recommendation: Option B (simplest, aligns with current design where ConfigMap override is optional)

3. **Access Mode Validation**: Should KFP validate the `PVC_ACCESSMODE` value or just pass it through to Kubernetes?
   - Current design: Pass through any value to Kubernetes (simpler, lets Kubernetes handle validation)
   - Alternative: Validate and restrict to `ReadWriteOnce`/`ReadWriteMany` only
   - Consideration: `ReadOnlyMany` won't work for output artifacts but might be useful for shared input data
   - Decision needed: Let Kubernetes handle all validation or add KFP-level restrictions?

4. **External Client Support**: Should the artifact server API be considered a public API for external clients, or is it an internal KFP API only?

   Options to consider:

   a. **Internal API only** - Document that this API is internal to KFP and not supported for external use
      - Pros: Freedom to change the API as needed, no backward compatibility burden
      - Cons: May break community integrations
      - Implementation: Could use internal paths like `/internal/artifacts/...` to make this clear

   b. **Public API with versioning** - Treat as a public API with proper versioning and compatibility guarantees
      - Pros: Enables ecosystem integrations, follows API best practices
      - Cons: Must maintain backward compatibility, slower evolution
      - Implementation: Keep versioned endpoints (`/apis/v2beta1/...`), follow deprecation policies

   c. **Hybrid approach** - Public read API, internal write API
      - Pros: Allows external tools to read artifacts while keeping write operations controlled
      - Cons: Inconsistent API surface, confusion about what's supported

## Drawbacks

1. **Performance considerations**: filesystem storage may have different performance characteristics than object storage, especially for very large datasets or high concurrency scenarios
2. **Storage Class Dependencies**: Requires a Kubernetes `StorageClass` to be available
3. **Limited features**: Lacks advanced object storage capabilities (versioning, lifecycle policies, cross-region replication, etc.)

## Infrastructure Needed

No special infrastructure needed beyond standard Kubernetes:

- Kubernetes cluster with PVC support
- `StorageClass` (preferably with RWX support)
- Permissions to manage PVC-backed storage for artifact servers (as described in the Permission Model above)
- Sufficient storage capacity for artifacts
