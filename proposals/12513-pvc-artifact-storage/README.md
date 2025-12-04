# KEP-12513: Filesystem-Based Artifact Storage for Kubeflow Pipelines

## Table of Contents

- [Summary](#summary)
- [Architecture at a Glance](#architecture-at-a-glance)
- [Motivation](#motivation)
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
    - [Namespace-Local Artifact Pods Architecture](#namespace-local-artifact-pods-architecture)
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

This KEP proposes adding filesystem-based storage as an alternative artifact storage backend for Kubeflow Pipelines v2. Currently, KFP requires object storage (S3-compatible storage or Google Cloud Storage) for storing pipeline artifacts, which can be a barrier for users who want to experiment with KFP in development environments. This proposal introduces filesystem storage as an additional option that eliminates the need for object storage configuration, making it easier for users to get started with KFP.

The filesystem backend will primarily use `PersistentVolumeClaim` (PVC) based storage in Kubernetes environments, providing namespace-isolated storage using Kubernetes native `PersistentVolumes`. However, the design is flexible enough to support other filesystem backends (e.g., local filesystem for development). Users can specify any access mode value which KFP will pass through to Kubernetes without validation (e.g., `ReadWriteMany` for parallel task execution across nodes, `ReadWriteOnce` for single node access, etc.), with RWO as the default if not specified. The actual behavior depends on what the underlying storage class supports. Existing pipelines will work without modification unless they contain hardcoded S3/object storage paths.

To support scalability, KFP will introduce a new artifact URI scheme (`kfp-artifacts://`) that routes all artifact requests through KFP's artifact server. Pipeline pods never mount PVCs directly - they upload and download artifacts via the artifact server API, which handles the actual filesystem operations. The artifact server can be scaled independently as an artifacts-only instance.

## Architecture at a Glance

### Quick Comparison

| Aspect               | Current (Object Storage)                   | Proposed (Filesystem Storage)         |
|----------------------|--------------------------------------------|---------------------------------------|
| **Storage Backend**  | S3, GCS, MinIO                             | Kubernetes PVC                        |
| **URI Scheme**       | `s3://`, `gs://`, `minio://`               | `kfp-artifacts://`                    |
| **Configuration**    | Credentials, endpoints, buckets (required) | Optional (uses K8s defaults)          |
| **Multi-tenancy**    | Bucket per namespace                       | PVC per namespace                     |
| **Setup Complexity** | High (external dependencies)               | Low (Kubernetes native with defaults) |

### High-Level Flow

```text
Pipeline Submission
        │
        ▼
Driver (detects "kfp-artifacts://")
        │
        ▼
Artifact Server (mounts PVC)
        │
        ├──────────────────────────────────┐
        │                                  │
        ▼                                  ▼
Pipeline Pods                         UI/API Clients
(upload/download via API)             (view artifacts)
        │                                  │
        └──────────► Artifact Server ◄─────┘
                           │
                           ▼
                    Read/Write to PVC
```

### Key Components

- **Storage**: Kubernetes `PersistentVolumeClaims` (one per namespace)
- **URI Format**: `kfp-artifacts://<namespace>/<pipeline>/<run-id>/<node-id>/<artifact-name>`
- **API Changes**: Extended `/healthz` endpoint for UI configuration discovery
- **Deployment Modes**:
  - **Central**: Single artifact server for all namespaces (default)
  - **Namespace-local**: One artifact server per namespace (better isolation)

## Motivation

Setting up object storage is often a significant hurdle for users who want to experiment with Kubeflow Pipelines. Users must:

- Configure S3-compatible storage with proper credentials
- Manage access keys and secrets
- Understand object storage concepts
- Set up and maintain additional infrastructure

For development, testing, and experimentation use cases, this overhead is unnecessary and discourages adoption. Many users simply want to try KFP without dealing with external dependencies.

Additionally, in some environments:

- Object storage may not be available or allowed
- Network policies may restrict external storage access
- Organizations may prefer to use existing Kubernetes storage infrastructure
- Development clusters may have limited resources

By providing PVC-based storage as an alternative, we can:

- Lower the barrier to entry for new users
- Enable quick experimentation and prototyping
- Leverage existing Kubernetes storage infrastructure
- Simplify development environment setup
- Enable portability between different storage backends for pipelines using KFP's artifact APIs

### Goals

Based on the user story "As a user, I want to provision Kubeflow Pipelines with just a PVC for artifact storage so that I can quickly get started", this KEP aims to:

1. **Add filesystem storage as an additional backend option** alongside S3-compatible and Google Cloud Storage, primarily using PVC but not limited to it
2. **Enable zero-configuration storage** for experimentation use cases - a KFP server can be installed with just a PVC for artifact storage
3. **Provide namespace-isolated artifact storage** with proper subject access review guards in multi-user mode
4. **Allow any Kubernetes access mode to be configured** - KFP passes through the configuration to Kubernetes (RWO default)
5. **Support existing pipelines** that use KFP's standard artifact types (Dataset, Model, etc.) - pipelines work unchanged with the new filesystem backend
6. **Match existing artifact persistence behavior** - artifacts persist indefinitely until explicitly deleted (no automatic cleanup)
7. **Enable separate scaling of artifact serving** through an artifacts-only KFP instance with `--artifacts-only` flag
8. **Ensure UI can properly handle artifact downloads** with the new `kfp-artifacts://` URI scheme

### Non-Goals

1. **Replace object storage** - filesystem storage is an additional option, not a replacement
2. **Cross-namespace artifact sharing** - Initial implementation focuses on namespace isolation
3. **Performance optimization for large-scale workloads** - filesystem storage may not match object storage performance for massive datasets or high concurrency scenarios
4. **Storage migration tools** - Migrating from S3 to PVC/filesystem is not supported
5. **Advanced storage features** - No versioning, replication, or geo-distribution initially

## Proposal

### Overview

This KEP proposes adding a new artifact storage backend that uses filesystem storage (primarily Kubernetes `PersistentVolumeClaims`) instead of object storage. The implementation will:

1. Create one PVC per namespace for artifact storage
2. Use configurable access mode with sensible defaults (RWO)
3. Organize artifacts in a filesystem hierarchy within the PVC
4. Provide transparent access through the existing KFP artifact APIs with new `kfp-artifacts://` URI scheme
5. Maintain compatibility with existing pipeline definitions that don't have hardcoded storage paths
6. Support separate scaling of artifact serving through artifacts-only instances
7. Update the UI to seamlessly handle artifact downloads from filesystem storage

### User Stories

#### Story 1: User Running Pipelines on Local Kubernetes

As a user with KFP on kind/minikube/k3s, I want my pipeline artifacts to automatically use the local cluster's default `StorageClass` via the `kfp-artifacts://` scheme, so that I can develop pipelines offline without any storage configuration.

**Acceptance Criteria:**

- KFP works out-of-the-box with filesystem storage on local clusters
- Artifacts are stored using the `kfp-artifacts://` URI scheme
- No S3/GCS credentials required
- Artifact viewing in UI works seamlessly

#### Story 2: Operator Using Default Filesystem Configuration

As an operator, I want to deploy KFP with minimal configuration using filesystem storage defaults, so that I can get KFP running quickly without cloud storage dependencies.

**Acceptance Criteria:**

- Single configuration option to enable filesystem storage
- No need to create or manage S3/GCS buckets
- No cloud IAM policies to configure
- Storage automatically provisioned via PVCs
- Backup/restore follows standard Kubernetes PVC procedures

#### Story 3: Operator Configuring Storage Class and Size

As an operator, I want to configure KFP to use a specific StorageClass and PVC size instead of defaults, so that I can match storage performance and capacity to my workload requirements.

**Acceptance Criteria:**

- Can specify `StorageClass` in KFP configuration
- Can set PVC size limits (global configuration)
- Storage quotas enforced via Kubernetes `ResourceQuotas`
- Clear error messages when storage limits are reached
- Can choose between RWO and RWX access modes based on needs

#### Story 4: User Migrating from S3 to Filesystem Storage

As a user with existing pipelines containing components that call `boto3.upload_file()` directly, I want KFP system artifacts to use `kfp-artifacts://` with PVC storage while my custom components continue accessing S3, so that I can migrate incrementally without rewriting all components at once.

**Acceptance Criteria:**

- KFP artifacts use filesystem storage via `kfp-artifacts://` URIs
- Components using boto3/gsutil for S3 continue to work
- Can mix storage backends (system artifacts on PVC, custom data on S3)
- Clear documentation on what uses filesystem vs object storage

#### Story 5: Operator Deploying Multi-Tenant KFP with Namespace Isolation

As an operator, I want to deploy KFP in namespace-local mode where each namespace annotated with `pipelines.kubeflow.org/enabled=true` gets its own artifact server pod and dedicated PVC, so that Team A's artifacts in namespace `team-a` are physically isolated from Team B's artifacts in namespace `team-b`.

**Acceptance Criteria:**

- Each namespace with `pipelines.kubeflow.org/enabled=true` annotation gets its own artifact server deployment
- Each namespace gets its own dedicated PVC (no shared storage)
- Artifact server in `team-a` namespace cannot access PVC in `team-b` namespace
- Users can only access artifacts in namespaces they have RBAC permissions for (via `SubjectAccessReview`)
- Physical isolation verified: deleting `team-a` namespace doesn't affect `team-b`'s artifacts

#### Story 6: Operator Implementing Air-Gapped ML Platform

As an operator in a regulated environment (e.g., healthcare, finance), I want to deploy KFP with filesystem storage using an encrypted StorageClass (e.g., `encrypted-gp3`) and no external network access, so that PHI/PII in artifacts never leaves our on-premises Kubernetes cluster.

**Acceptance Criteria:**

- All artifacts stored on PVCs within the cluster (no S3/GCS/external storage)
- KFP configuration uses `Filesystem.Type: "pvc"` with encrypted StorageClass
- Artifact server pods have no external network routes (verified by network policies)
- `SubjectAccessReview` validates all artifact access requests
- Encryption at rest provided by the configured StorageClass (e.g., `encrypted-gp3`)

#### Story 7: Operator Running KFP on Storage-Constrained Infrastructure

As an operator with limited storage budget (only 1TB total available), I want to deploy KFP in central mode with a single 500GB PVC shared across 10 team namespaces, so that all teams can run pipelines without each needing their own 100GB PVC (which would require 1TB total).

**Acceptance Criteria:**

- Central mode configured with: `DeploymentMode: "central"`, `Size: "500Gi"`
- Single PVC created in `kubeflow` namespace mounted by one artifact server
- All 10 teams' artifacts stored in `/artifacts/<namespace>/` directories on same PVC
- Teams can run pipelines concurrently without storage allocation failures
- No per-namespace storage limits (trade-off of central mode - shared PVC means no per-team quotas)

#### Story 8: Operator Scaling High-Throughput Model Training Platform

As an operator supporting 100+ concurrent pipeline runs with multi-GB model checkpoints, I want to deploy KFP in central mode with RWX storage (e.g., NFS/CephFS) and multiple artifact server replicas behind a load balancer, so that artifact upload/download operations can scale horizontally without bottlenecks.

**Acceptance Criteria:**

- Can deploy artifact servers in both central and namespace-local modes
- Artifact servers stream large files without loading into memory
- Can use high-performance `StorageClasses` (e.g., SSD-backed)
- Horizontal scaling possible in central mode
- Direct pod-to-pod communication in namespace-local mode reduces latency

### Notes/Constraints/Caveats

1. **Performance**: filesystem storage may have different performance characteristics than object storage, especially for large files or high concurrency
2. **Storage Classes**: Functionality depends on available `StorageClasses` - RWX preferred for parallel execution
3. **Scalability**: filesystem storage is not designed for massive scale like object storage
4. **Cost**: Cost depends on the storage class and provider - could be more or less expensive than object storage
5. **Portability**: Pipelines using filesystem storage are tied to Kubernetes infrastructure
6. **No Migration Support**: Migrating existing artifacts from S3/GCS to filesystem storage is not supported. Users must choose their storage backend at deployment time and stick with it

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

All new configurations will be added as new fields under the existing `ObjectStoreConfig`:

```text
ObjectStoreConfig/
├── [Existing Fields] (unchanged)
└── [New Fields]
```

Throughout this KEP, when new configurations are introduced, they will be referenced by their full path (e.g., `ObjectStoreConfig.Filesystem.PVC.Size`) to make the hierarchy clear. All configurations can be overridden via environment variables following KFP's naming convention where dots become underscores (e.g., `OBJECTSTORECONFIG_FILESYSTEM_PVC_SIZE`).

### Storage Backend Selection

Storage backends in KFP are determined by the `defaultPipelineRoot` URL scheme, which can be configured at multiple levels:

1. **System default** (single-user mode): Set in `pipeline-install-config` ConfigMap
2. **Namespace default** (multi-user mode): Set in each namespace's `kfp-launcher` ConfigMap  
3. **Pipeline specific**: Set via `@pipeline(pipeline_root='...')` decorator in Python SDK
4. **Runtime override**: Set via `pipeline_root` parameter when submitting a run

For filesystem storage, this KEP introduces a new URL scheme:

```text
kfp-artifacts://<namespace>/<pipeline>/<run-id>/<node-id>/<artifact-name>
```

This follows the same pattern as existing storage schemes (s3://, gs://, minio://), where the namespace maps to the bucket name and the rest maps to the prefix.

Example configurations:

```yaml
# Both modes use the same path structure for consistency
# {namespace} is substituted at runtime (e.g., "kubeflow" in single-user mode)
defaultPipelineRoot: "kfp-artifacts://{namespace}"
```

#### URI, API Endpoint, and Filesystem Path Relationship

There are three related but distinct formats:

| Concept                  | Format                                                                       | Example                                                       |
|--------------------------|------------------------------------------------------------------------------|---------------------------------------------------------------|
| **URI** (stored in MLMD) | `kfp-artifacts://<namespace>/<pipeline>/<run-id>/<node-id>/<artifact-name>`  | `kfp-artifacts://team-a/my-pipeline/abc123/step1/output`      |
| **API Endpoint**         | `/apis/v2beta1/runs/{run_id}/nodes/{node_id}/artifacts/{artifact_name}:read` | `/apis/v2beta1/runs/abc123/nodes/step1/artifacts/output:read` |
| **Filesystem Path**      | `<mount_path>/<namespace>/<pipeline>/<run-id>/<node-id>/<artifact-name>`     | `/artifacts/team-a/my-pipeline/abc123/step1/output`           |

**Why the API endpoint doesn't include namespace/pipeline:**

The API endpoint only requires `run_id`, `node_id`, and `artifact_name`. The server resolves the full filesystem path by looking up the run's metadata from the database (which contains the namespace and pipeline name). This matches the existing artifact server behavior for S3/GCS.

**How the Launcher uses the URI:**

1. Launcher receives artifact URI: `kfp-artifacts://team-a/my-pipeline/abc123/step1/output`
2. Extracts `run_id=abc123`, `node_id=step1`, `artifact_name=output`
3. In namespace-local mode, extracts `namespace=team-a` to route to correct artifact server
4. Makes API call: `GET http://ml-pipeline-artifact-server.team-a.svc:8080/apis/v2beta1/runs/abc123/nodes/step1/artifacts/output:read`

This approach provides several benefits:

1. **Abstraction**: Clients don't need to know the underlying storage backend
2. **Security**: All artifact access goes through KFP's authorization layer  
3. **Scalability**: Artifact serving can be scaled independently
4. **Flexibility**: Storage backend can be changed without client modifications

### Artifact Server Architecture

#### Backend Responsibilities

KFP will support two distinct deployment modes for artifact serving, configured at installation time via the new `ObjectStoreConfig.ArtifactServer.DeploymentMode` field. **In both modes, only the artifact server pods mount the PVC - pipeline workflow pods access artifacts exclusively through the artifact server API.**

##### Mode 1: Central Artifact Server (Default)

A single artifact server in the main KFP namespace serves all namespaces, configured via `ObjectStoreConfig.ArtifactServer.DeploymentMode: "central"`.

**Central Mode Characteristics:**

- **Single PVC** with directory structure: `/artifacts/<namespace>/<pipeline>/<run-id>/<node-id>/<artifact-name>`
- **Authorization**: Uses `SubjectAccessReview` to verify namespace access
- **Best for**: Simple deployments, single-user setups, small teams
- **Advantages**: Simple setup, single storage location, easy backup
- **Limitations**: All namespaces share same PVC and storage quota

##### Mode 2: Namespace-Local Artifact Servers

Each namespace runs its own artifact server, configured via `ObjectStoreConfig.ArtifactServer.DeploymentMode: "namespaced"`.

**Namespace-Local Mode Characteristics:**

- **PVC per namespace**: Complete storage isolation
- **Direct access**: Clients connect directly to namespace servers (no proxying)
- **Authorization**: Natural isolation (each server only accesses its namespace's PVC)
- **Best for**: Large multi-tenant deployments, strict isolation requirements
- **Advantages**: True multi-tenancy, per-namespace scaling, independent quotas
- **Deployment**: Lazy initialization when first pipeline runs in namespace

#### Request Routing

Based on the configured mode, artifact URIs are resolved differently:

**Central Mode:**

```text
Client
  │
  │ GET kfp-artifacts://<namespace>/...
  ▼
KFP API Server (central)
  │
  │ Authorization check (SubjectAccessReview)
  ▼
Serve from /artifacts/<namespace>/...
```

**Namespace-Local Mode:**

```text
Client
  │
  │ GET kfp-artifacts://<namespace>/...
  ▼
Resolve to ml-pipeline-artifact-server.<namespace>.svc
  │
  │ Direct connection to namespace server
  ▼
Serve from /artifacts/<namespace>/...
```

**Note on Path Structure**: The namespace-local mode intentionally includes `<namespace>` in the filesystem path even though each namespace has its own dedicated PVC. This design decision provides:

- **Consistent path generation logic** across both deployment modes - no conditional logic needed
- **Simplified artifact URI handling** - the same `kfp-artifacts://<namespace>/...` format works everywhere
- **Future flexibility** - allows potential migration between modes without path restructuring
- **Debugging clarity** - filesystem paths clearly indicate the namespace even in isolated PVCs

##### Architecture Diagrams

###### Central Mode Architecture

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
│        (namespace: kubeflow)         │
│                                      │
│  • SubjectAccessReview               │
│  • Mounts central PVC                │
│  • Serves all namespaces             │
└──────────────────┬───────────────────┘
                   │ mounts
                   ▼
┌──────────────────────────────────────┐
│  Central PVC (kfp-artifacts-central) │
│  /artifacts/                         │
│    ├── ns1/                          │
│    ├── ns2/                          │
│    └── ns3/                          │
└──────────────────────────────────────┘
```

###### Namespace-Local Mode Architecture

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

The UI discovers the deployment mode through the existing `/apis/v2beta1/healthz` endpoint, which will be extended to include artifact server configuration. This approach is preferred because:

- The backend already resolves the final configuration from multiple layers (defaults, config.json, environment variables)
- It maintains proper separation of concerns between UI and backend
- It provides a single source of truth for the resolved configuration
- The UI already fetches this endpoint, so no additional API calls are needed

##### Configuration Discovery via Extended Health Endpoint

```http
GET /apis/v2beta1/healthz
```

###### Extended Response Format

Current response (existing):

```json
{
  "commit_sha": "abc123...",
  "tag_name": "v2.0.0",
  "multi_user": true,
  "pipeline_store": "kubernetes"
}
```

Extended with filesystem storage config:

```json
{
  "commit_sha": "abc123...",
  "tag_name": "v2.0.0",
  "multi_user": true,
  "pipeline_store": "kubernetes",
  "artifact_server": {
    "deployment_mode": "central"  // or "namespaced"
  }
}
```

###### Implementation Details

- **Security**: Never exposes sensitive configuration (credentials, secrets, internal paths)
- **On-Demand**: Fields are only added when needed (currently just `deploymentMode`)
- **Hierarchy**: Nested under `objectStoreConfig` to mirror backend configuration
- **Caching**: Leverages existing `BuildInfoProvider` caching in UI
- **Authentication**: Follows existing `/healthz` endpoint authentication policy

###### UI Usage

```javascript
// The BuildInfo type will be extended to include objectStoreConfig
interface BuildInfo {
  // ... existing fields
  objectStoreConfig?: {
    artifactServer?: {
      deploymentMode?: 'central' | 'namespaced';
    };
  };
}

// UI components access it via the existing BuildInfoContext
const MyComponent = () => {
  const buildInfo = useContext(BuildInfoContext);
  const deploymentMode = buildInfo?.objectStoreConfig?.artifactServer?.deploymentMode || 'central';
  
  // Route artifacts based on deployment mode
  if (deploymentMode === 'namespaced') {
    // Route to namespace-local server
  } else {
    // Route to central server
  }
};
```

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
  "uri": "kfp-artifacts://<namespace>/<pipeline>/<run_id>/<node_id>/<artifact_name>"
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

For filesystem storage, the UI's `/artifacts/get` endpoint cannot directly access PVCs. Instead, it must proxy through the artifact server. Following the existing frontend code pattern of handler functions:

The UI's `/artifacts/get` endpoint needs to proxy filesystem artifacts through the artifact server API:

```typescript
// frontend/server/handlers/artifacts.ts
// Add new case to existing switch statement:

case 'filesystem':
  // Proxy to artifact server API since UI can't access PVC directly
  // Calls: GET /apis/v2beta1/runs/{runId}/nodes/{nodeId}/artifacts/{artifactName}:read
  // Returns decoded tar.gz content to browser
  break;

// frontend/src/components/ArtifactPreview.tsx
// Convert kfp-artifacts:// URIs to UI proxy calls:

if (uri.startsWith('kfp-artifacts://')) {
  // Parse URI and return: /artifacts/get?source=filesystem&runId=...&nodeId=...&artifactName=...
}
```

From the user's perspective, artifact handling remains unchanged:

- Click on artifact links in the UI to view/download
- Artifact preview works for common formats (text, images, etc.)
- Download button saves artifacts to local machine
- No visible difference between storage backends

The only change is that artifact URIs in the UI will show `kfp-artifacts://` instead of `s3://` or `gs://` when using filesystem storage, providing a clear indication of the storage backend being used.

### PVC Management

The PVC lifecycle is controlled by the `ObjectStoreConfig.Filesystem.PVC` configuration fields.

#### PVC Creation and Configuration

`PersistentVolumeClaims` are created automatically when the artifact server is deployed in a namespace. The PVC uses a standard naming convention (`kfp-artifacts-{namespace}`) and the configured storage size, access mode, and storage class. Pipeline pods interact with artifacts exclusively through the artifact server API, ensuring proper isolation and access control.

##### StorageClass Selection (`ObjectStoreConfig.Filesystem.PVC.StorageClassName`)

- Empty string (default): Uses cluster's default StorageClass
- Explicit name: Uses the specified `StorageClass`
- `"manual"`: Static PV binding (for pre-provisioned volumes)

##### Access Mode Configuration (`ObjectStoreConfig.Filesystem.PVC.AccessMode`)

- Passed directly to Kubernetes without validation
- Defaults to `ReadWriteOnce` if not specified
- Actual behavior depends on what your `StorageClass` supports

See [Kubernetes StorageClass documentation](https://kubernetes.io/docs/concepts/storage/storage-classes/) and [Access Modes documentation](https://kubernetes.io/docs/concepts/storage/persistent-volumes/#access-modes) for details.

##### Known Limitation: RWO in Multi-Node Clusters

The default `ReadWriteOnce` (RWO) access mode constrains PVC mounting to a single node at a time. This creates architectural implications for the artifact server deployment that must be considered in the design.

**Impact on Central Mode:**

- The artifact server pod(s) are constrained to run on a single node (this KEP does not implement scheduling constraints to ensure this)
- Horizontal scaling is possible but all replicas must run on the same node (limited benefit)
- Node failures result in service interruption until PVC reattachment completes

**Impact on Namespace-Local Mode:**

- Each namespace's artifact server pod(s) must run on the same node as their PVC (this KEP does not implement pod affinity rules)
- Horizontal scaling within a namespace is possible but confined to a single node
- Resource scheduling becomes less flexible due to node affinity requirements

**Design Considerations:**

This limitation is acceptable for the initial implementation because:

1. Single-node development clusters (primary target use case) are unaffected
2. RWX-capable `StorageClasses` can be configured where high availability is required
3. Pipeline pods remain unconstrained, maintaining scheduling flexibility for compute workloads

**Scope Clarification:**

This KEP accepts the RWO limitation as a reasonable trade-off for the initial implementation. The design explicitly does not attempt to work around RWO limitations through complex scheduling or node affinity rules, as this would add complexity without addressing the fundamental constraint.

**Documentation Requirements:**

- Installation guide must clearly state this limitation
- Troubleshooting guide must include "volume node affinity conflict" errors
- Configuration examples should default to RWX for multi-node clusters

**Scope of this KEP**: This initial implementation targets single-node clusters and multi-node clusters with RWX storage. Full RWO support in multi-node clusters is deferred to future enhancements.

#### Storage Quota Enforcement

Storage limits are enforced by Kubernetes, not KFP:

1. **Kubernetes `ResourceQuotas`**: Cluster admins configure quotas per namespace
2. **PVC Creation**: Fails if requested size exceeds namespace quota
3. **Runtime Enforcement**: Kubernetes prevents writes when PVC is full
4. **KFP Behavior**: Respects these limits but doesn't manage them

### Component Modifications

#### Object Store Interface (`backend/src/v2/objectstore/`)

The object store interface remains unchanged for S3/GCS. For filesystem storage, artifact operations are redirected to the artifact server API:

```go
func OpenBucket(ctx context.Context, k8sClient kubernetes.Interface, namespace string, config *Config) (*blob.Bucket, error) {
    // Check if filesystem storage is configured (based on pipeline root URL scheme)
    if strings.HasPrefix(config.Scheme, "kfp-artifacts://") {
        // Return nil - filesystem storage uses artifact server API, not direct access
        // The launcher will detect this and use the artifact client instead
        return nil, &FilesystemAPIMode{
            Namespace: namespace,
            Endpoint:  getArtifactServerEndpoint(namespace),
        }
    }
    // Existing logic for s3, minio, gs...
}

func ParseBucketConfig(path string) (*Config, error) {
    if strings.HasPrefix(path, "kfp-artifacts://") {
        // Parse kfp-artifacts path: kfp-artifacts://namespace/artifacts/...
        parts := strings.SplitN(strings.TrimPrefix(path, "kfp-artifacts://"), "/", 2)
        return &Config{
            Scheme:      "kfp-artifacts://",
            BucketName:  parts[0], // namespace
            Prefix:      parts[1] if len(parts) > 1 else "",
        }, nil
    }
    // Existing parsing logic...
}
```

#### KFP API Server Artifact Management

The KFP API server manages artifact infrastructure lifecycle for both deployment modes:

##### Namespace Annotation-Based Provisioning

The KFP API server watches for namespaces with a specific annotation to provision artifact infrastructure. When Kubeflow Profiles are used, the Profile controller creates namespaces with the `pipelines.kubeflow.org/enabled: "true"` annotation. For standalone KFP, namespaces can be annotated directly.

```go
// Watch namespaces and provision based on annotations
func (m *ArtifactServerManager) WatchNamespaces() error {
    informer := informers.NewSharedInformerFactory(m.k8sClient, 0)
    informer.Core().V1().Namespaces().Informer().AddEventHandler(
        cache.ResourceEventHandlerFuncs{
            AddFunc: func(obj interface{}) {
                m.handleNamespace(obj.(*v1.Namespace))
            },
            UpdateFunc: func(oldObj, newObj interface{}) {
                oldNs := oldObj.(*v1.Namespace)
                newNs := newObj.(*v1.Namespace)
                // Only process if annotations changed
                if !reflect.DeepEqual(oldNs.Annotations, newNs.Annotations) {
                    m.handleNamespace(newNs)
                }
            },
        },
    )
    
    go informer.Start(wait.NeverStop)
    return nil
}

func (m *ArtifactServerManager) handleNamespace(ns *v1.Namespace) {
    // Check if namespace should have KFP artifact infrastructure
    if enabled, ok := ns.Annotations["pipelines.kubeflow.org/enabled"]; ok && enabled == "true" {
        // Provision artifact infrastructure based on deployment mode
        if err := m.EnsureArtifactInfrastructure(ns.Name); err != nil {
            log.Errorf("Failed to provision artifact infrastructure for namespace %s: %v", 
                ns.Name, err)
        }
    }
}
```

**Benefits of annotation-only approach:**

- **Simple and predictable**: Explicit opt-in required, no surprises
- **Clear boundaries**: Either a namespace has artifact infrastructure or it doesn't  
- **Works everywhere**: Same behavior in Kubeflow and standalone

Namespaces without the annotation cannot use filesystem storage - they must use S3/GCS/Minio or add the annotation first.

##### API Server Implementation

```go
// backend/src/apiserver/server/artifact_manager.go

type ArtifactServerManager struct {
    k8sClient     kubernetes.Interface
    config        *ObjectStoreConfig
    informerCache cache.SharedIndexInformer
    workqueue     workqueue.RateLimitingInterface
}

// EnsureArtifactInfrastructure ensures artifact infrastructure exists for a namespace
// Handles both central and namespace-local deployment modes
func (m *ArtifactServerManager) EnsureArtifactInfrastructure(namespace string) error {
    switch m.config.DeploymentMode {
    case "central":
        // For central mode, ensure the shared artifact server exists in kubeflow namespace
        return m.ensureCentralArtifactServer()
    case "namespaced":
        // For namespace-local mode, create per-namespace resources
        return m.ensureNamespaceLocalResources(namespace)
    default:
        return fmt.Errorf("unknown deployment mode: %s", m.config.DeploymentMode)
    }
}

func (m *ArtifactServerManager) ensureCentralArtifactServer() error {
    kubeflowNs := "kubeflow" // Or from config
    
    // Check if central artifact server exists
    _, err := m.k8sClient.AppsV1().Deployments(kubeflowNs).
        Get(context.Background(), "ml-pipeline-artifact-server", metav1.GetOptions{})
    if errors.IsNotFound(err) {
        // Create central artifact server, service, and PVC
        if err := m.createCentralInfrastructure(kubeflowNs); err != nil {
            return fmt.Errorf("failed to create central artifact infrastructure: %w", err)
        }
    }
    
    return nil
}

func (m *ArtifactServerManager) ensureNamespaceLocalResources(namespace string) error {
    // Check and create PVC if needed
    pvcName := "ml-pipeline-artifacts"
    _, err := m.k8sClient.CoreV1().PersistentVolumeClaims(namespace).
        Get(context.Background(), pvcName, metav1.GetOptions{})
    if errors.IsNotFound(err) {
        pvc := m.buildArtifactPVC(namespace, pvcName)
        _, err = m.k8sClient.CoreV1().PersistentVolumeClaims(namespace).
            Create(context.Background(), pvc, metav1.CreateOptions{})
        if err != nil && !errors.IsAlreadyExists(err) {
            return fmt.Errorf("failed to create PVC: %w", err)
        }
    }
    
    // Check and create Deployment if needed
    deploymentName := "ml-pipeline-artifact-server"
    deployment, err := m.k8sClient.AppsV1().Deployments(namespace).
        Get(context.Background(), deploymentName, metav1.GetOptions{})
    if errors.IsNotFound(err) {
        deployment = m.buildArtifactServerDeployment(namespace, deploymentName, pvcName)
        _, err = m.k8sClient.AppsV1().Deployments(namespace).
            Create(context.Background(), deployment, metav1.CreateOptions{})
        if err != nil && !errors.IsAlreadyExists(err) {
            return fmt.Errorf("failed to create deployment: %w", err)
        }
    } else if err == nil {
        // Update deployment if configuration changed
        if m.deploymentNeedsUpdate(deployment) {
            updated := m.updateDeploymentSpec(deployment)
            _, err = m.k8sClient.AppsV1().Deployments(namespace).
                Update(context.Background(), updated, metav1.UpdateOptions{})
            if err != nil {
                return fmt.Errorf("failed to update deployment: %w", err)
            }
        }
    }
    
    // Check and create Service if needed
    _, err = m.k8sClient.CoreV1().Services(namespace).
        Get(context.Background(), deploymentName, metav1.GetOptions{})
    if errors.IsNotFound(err) {
        service := m.buildArtifactServerService(namespace, deploymentName)
        _, err = m.k8sClient.CoreV1().Services(namespace).
            Create(context.Background(), service, metav1.CreateOptions{})
        if err != nil && !errors.IsAlreadyExists(err) {
            return fmt.Errorf("failed to create service: %w", err)
        }
    }
    
    return nil
}

// Called during API server startup to initialize artifact management
func (m *ArtifactServerManager) Start(ctx context.Context) error {
    // For central mode, ensure the central server exists on startup
    if m.config.DeploymentMode == "central" {
        if err := m.ensureCentralArtifactServer(); err != nil {
            return fmt.Errorf("failed to ensure central artifact server: %w", err)
        }
        log.Info("Central artifact server verified/created")
    }
    
    // Watch namespaces for both modes
    // - Central: to set up namespace directories in shared PVC
    // - Namespace-local: to create per-namespace infrastructure
    if err := m.WatchNamespaces(); err != nil {
        return fmt.Errorf("failed to start namespace watcher: %w", err)
    }
    
    log.Infof("Artifact manager started in %s mode", m.config.DeploymentMode)
    return nil
}

// Integration with pipeline run creation (fallback/verification only)
func (s *PipelineRunServer) CreatePipelineRun(ctx context.Context, req *api.CreateRunRequest) (*api.Run, error) {
    // Existing validation...
    
    // Verify artifact infrastructure exists for filesystem storage
    if s.isFilesystemStorageEnabled() && s.artifactManager != nil {
        if err := s.artifactManager.VerifyArtifactInfrastructure(req.Run.Namespace); err != nil {
            return nil, fmt.Errorf("artifact infrastructure not available for namespace %s. "+
                "Please ensure the namespace has the 'pipelines.kubeflow.org/enabled=true' annotation", 
                req.Run.Namespace)
        }
    }
    
    // Continue with pipeline run creation...
}

// Quick check without creation - used during pipeline run creation
func (m *ArtifactServerManager) VerifyArtifactInfrastructure(namespace string) error {
    if m.config.DeploymentMode != "namespaced" {
        return nil
    }
    
    // Just verify resources exist, don't create
    _, err := m.k8sClient.AppsV1().Deployments(namespace).
        Get(context.Background(), "ml-pipeline-artifact-server", metav1.GetOptions{})
    if err != nil {
        return fmt.Errorf("artifact server not found: %w", err)
    }
    return nil
}
```

**Benefits of annotation-based provisioning:**

- **Zero-delay pipeline starts**: Infrastructure ready before first pipeline run
- **Simple and consistent**: Same mechanism works everywhere

#### Launcher (`backend/src/v2/component/launcher_v2.go`)

The launcher will use the artifact server API for all artifact operations when filesystem storage is configured:

The launcher will be modified to:

- Detect `kfp-artifacts://` URIs
- Use the artifact server API for filesystem artifacts instead of direct storage access
- Call the read/write endpoints with proper authentication
- Continue using existing `objectstore` package for S3/GCS/MinIO

#### Compiler (`backend/src/v2/compiler/argocompiler/`)

Configure launcher to use artifact API when filesystem storage is detected:

```go
func (c *workflowCompiler) configureArtifactAPI(template *wfapi.Template) {
    if c.pipelineRoot != nil && strings.HasPrefix(*c.pipelineRoot, "kfp-artifacts://") {
        template.Container.Env = append(template.Container.Env, 
            corev1.EnvVar{
                Name:  "ARTIFACT_API_ENABLED",
                Value: "true",
            },
            corev1.EnvVar{
                Name:  "ARTIFACT_SERVER_ENDPOINT",
                Value: c.getArtifactServerEndpoint(),
            },
        )
        
        // No PVC mounts needed - all artifact operations go through API
    }
}
```

#### Driver (`backend/src/v2/driver/`)

The driver validates that required artifact infrastructure exists before pipeline execution:

```go
func (d *Driver) validateArtifactInfrastructure(ctx context.Context, pipelineRoot string) error {
    if !strings.HasPrefix(pipelineRoot, "kfp-artifacts://") {
        return nil // Not using filesystem storage
    }
    
    objectStoreConfig := viper.Sub("ObjectStoreConfig")
    deploymentMode := objectStoreConfig.GetString("ArtifactServer.DeploymentMode")
    
    if deploymentMode == "namespaced" {
    namespace := d.options.Namespace
        deploymentName := "ml-pipeline-artifact-server"
        
        // Verify artifact server exists
        _, err := d.k8sClient.AppsV1().Deployments(namespace).
            Get(ctx, deploymentName, metav1.GetOptions{})
    if err != nil {
            if errors.IsNotFound(err) {
                return fmt.Errorf("artifact server not found in namespace %s. "+
                    "Please ensure the KFP backend has provisioned artifact infrastructure "+
                    "for this namespace", namespace)
            }
            return fmt.Errorf("failed to check artifact server: %w", err)
        }
        
        // Verify service exists
        _, err = d.k8sClient.CoreV1().Services(namespace).
            Get(ctx, deploymentName, metav1.GetOptions{})
            if err != nil {
            return fmt.Errorf("artifact service not found or inaccessible: %w", err)
            }
        }
    // For central mode, artifact server is pre-deployed in kubeflow namespace
    
    return nil
}
```

### Security and RBAC

#### Permission Model

The filesystem storage implementation should follow the principle of least privilege:

**KFP API Server Permissions:**

For both deployment modes, the API server needs:

```yaml
# Watch namespaces for annotations
- apiGroups: [""]
  resources: ["namespaces"]
  verbs: ["get", "list", "watch"]

# For namespace-local mode: manage per-namespace resources
- apiGroups: [""]
  resources: ["persistentvolumeclaims"]
  verbs: ["get", "create"]
- apiGroups: ["apps"]
  resources: ["deployments"]
  verbs: ["get", "create", "update"]
- apiGroups: [""]
  resources: ["services"]
  verbs: ["get", "create"]

# For central mode: manage resources in kubeflow namespace
# (same permissions but scoped to kubeflow namespace)
```

**Driver/Workflow Pod Permissions:**

```yaml
# Only verify that resources exist
- apiGroups: ["apps"]
  resources: ["deployments"]
  verbs: ["get"]
- apiGroups: [""]
  resources: ["services"]
  verbs: ["get"]
```

**Artifact Server Pod Permissions:**

```yaml
# Authorization checks via SubjectAccessReview
- apiGroups: ["authorization.k8s.io"]
  resources: ["subjectaccessreviews"]
  verbs: ["create"]
```

This separation ensures that:

- Only the KFP API server can create infrastructure (not workflow pods)
- Drivers can only verify resources exist
- Artifact servers can perform authorization checks but have no other Kubernetes API access

### Multi-User Isolation and Authorization

In multi-user mode, KFP enforces strict isolation and authorization using Kubernetes native mechanisms:

#### Namespace Isolation

Each namespace gets its own PVC with strict isolation:

1. **PVC per Namespace**: Each namespace has a dedicated PVC named `kfp-artifacts-<namespace>`
2. **Directory Organization**: Artifacts organized by pipeline/run-id for clarity (not security)

#### Multi-User Isolation Strategy

The approach to multi-user isolation depends on the deployment mode:

##### Central Mode Isolation

- Directory-based separation: Artifacts stored in `/artifacts/<namespace>/...` structure
- Authorization checks: Every request validated using `SubjectAccessReview`
- Single PVC: Shared storage with logical separation

##### Namespace-Local Mode Isolation

- Physical separation: Each namespace has its own PVC
- Network accessibility: Artifact servers can be accessed from other namespaces
- RBAC authorization: All requests validated via `SubjectAccessReview`

#### Namespace-Local Artifact Pods Architecture

For namespace-local mode, each namespace runs its own lightweight artifact server:

##### Service Discovery

Each namespace's artifact server is exposed via a Service:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: ml-pipeline-artifact-server
  namespace: <user-namespace>
spec:
  selector:
    app: ml-pipeline-artifact-server
  ports:
  - port: 8080
    protocol: TCP
```

##### Cross-Namespace Access Pattern

Control plane components (UI, API server) in the `kubeflow` namespace need to access artifact servers in user namespaces:

**Access Flow:**

1. **UI determines target namespace**: Reads run metadata to identify which namespace contains the artifact
2. **Service call**: UI calls `http://ml-pipeline-artifact-server.<namespace>.svc:8080/apis/v2beta1/...`
3. **Authorization check**: Target namespace's artifact server performs `SubjectAccessReview`
4. **Artifact retrieval**: If authorized, serves artifact from namespace's PVC

**Example: UI Accessing User Namespace Artifacts**

```go
// UI backend code running in kubeflow namespace
func (s *UIServer) GetArtifact(namespace, runID, nodeID, artifactName string) (*Artifact, error) {
    // Construct cross-namespace URL
    artifactURL := fmt.Sprintf(
        "http://ml-pipeline-artifact-server.%s.svc:8080/apis/v2beta1/runs/%s/nodes/%s/artifacts/%s:read",
        namespace, runID, nodeID, artifactName,
    )
    
    // Request includes user's credentials for `SubjectAccessReview`
    req, _ := http.NewRequest("GET", artifactURL, nil)
    req.Header.Set("Authorization", userToken)
    
    // Artifact server in target namespace will:
    // 1. Extract user identity from token
    // 2. Perform `SubjectAccessReview` against pipelines.kubeflow.org/runs resource
    // 3. Return artifact if user has access
    resp, err := http.DefaultClient.Do(req)
    // ...
}
```

**Why Both Network Access + RBAC?**

- **Network access** (`ml-pipeline-artifact-server.<namespace>.svc`): Enables cross-namespace communication via standard Kubernetes DNS
- **RBAC authorization** (`SubjectAccessReview`): Ensures users can only access artifacts they have permissions for
- **Defense in depth**: Network reachability doesn't imply authorization - both layers provide security

This design allows the control plane to serve artifacts from any namespace while maintaining strict user-level access control.

##### Direct Client Access

In namespace-local mode, workflow pods connect directly to their namespace's artifact server:

- Artifact URI: `kfp-artifacts://<namespace>/<pipeline>/<run-id>/...`
- Resolved to: `http://ml-pipeline-artifact-server.<namespace>.svc.cluster.local:8080/...`
- UI handles the resolution based on deployment mode

##### Deployment Strategy

1. **Proactive Provisioning**: The KFP API server watches for namespaces with `pipelines.kubeflow.org/enabled=true` annotation and creates:
   - The artifact server Deployment
   - The artifact server Service
   - The namespace PVC (which only the artifact server will mount)
2. **Lifecycle Management**: Artifact servers persist as long as the namespace exists
3. **Resource Efficiency**: Minimal resource usage when idle (can scale to zero if configured)

#### Subject Access Review Integration

Both deployment modes use KFP's existing authorization layer with Kubernetes `SubjectAccessReview`. In namespace-local mode, this is especially important as the artifact servers perform authorization checks for cross-namespace requests:

```go
// Artifact access uses the same authorization regardless of storage backend
func (s *RunServer) ReadArtifact(ctx context.Context, request *apiv2beta1.ReadArtifactRequest) (*apiv2beta1.ReadArtifactResponse, error) {
    // Authorization check using `SubjectAccessReview`
    err := s.canAccessRun(ctx, request.GetRunId(), &authorizationv1.ResourceAttributes{
        Verb: common.RbacResourceVerbReadArtifact,
    })
    if err != nil {
        return nil, util.Wrap(err, "Failed to authorize the request")
    }
    
    // Use ObjectStore interface - works for any backend (S3, GCS, PVC, etc.)
    reader, err := s.resourceManager.ObjectStore().GetFileReader(ctx, artifactPath)
    if err != nil {
        return nil, err
    }
    defer reader.Close()
    
    // Stream and encode response (same for all backends)
    // ...
}
```

The authorization flow for filesystem storage:

1. **Extract User Identity**: From request context (existing KFP authentication)
2. **Create `SubjectAccessReview`**: Check if user can access the run's artifacts
3. **Verify Namespace**: Ensure user has access to the namespace
4. **Access Filesystem**: Read artifact from PVC mount if authorized

#### Single-User vs Multi-User Deployments

The namespace-isolated design works for both deployment modes:

##### Multi-User Mode

- Each namespace (team/user) gets its own PVC
- Kubernetes enforces isolation between namespaces
- Natural fit for multi-tenant environments

##### Single-User Mode

- Typically uses one namespace (e.g., `kubeflow`)
- One PVC for all pipelines: `kfp-artifacts-kubeflow`
- If multiple namespaces are used, each still gets its own PVC (same model applies)
- Simpler than S3 (no credentials needed) while maintaining security best practices

This unified approach avoids code complexity while providing secure defaults for all deployment types.

### Artifact Lifecycle Management

The filesystem storage backend provides basic artifact persistence similar to S3, but without advanced lifecycle features.

#### Artifact Persistence

##### S3 Behavior (current)

- Artifacts uploaded after task completion
- Persist indefinitely in object storage
- No automatic deletion after pipeline completion
- Available through KFP UI/API until explicitly deleted

##### PVC Behavior (matching)

- Artifacts saved to PVC after task completion
- Persist indefinitely in PVC
- No automatic deletion after pipeline completion
- Available through KFP UI/API until explicitly deleted

#### Cleanup Mechanisms

- **Metadata deletion**: Deleting runs/experiments through KFP UI/API removes metadata but typically leaves artifacts in storage (same as S3 behavior)
- **Manual artifact cleanup**:
  - S3: Can use AWS CLI, S3 console, or other S3 tools
  - Filesystem: Requires kubectl access to exec into a pod with PVC mounted, or deletion of the entire PVC

- **No automatic cleanup**: Neither S3 nor filesystem storage automatically delete artifacts when runs are deleted

##### PVC Lifecycle

- Created on first pipeline run in namespace
- Persists even when empty (avoids recreation overhead)
- Deleted only when namespace is deleted or manually removed

#### Caching Support

Artifact caching works identically:

- Cached artifacts identified by hash/fingerprint
- Stored in same PVC with cache metadata
- Reused across pipeline runs within namespace

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

```python
# ✅ These patterns work with S3, GCS, and PVC
@component
def process_data(
    input_data: Input[Dataset],
    output_data: Output[Dataset],
    model_path: OutputPath('Model')
):
    # KFP handles storage operations transparently
    df = pd.read_csv(input_data.path)
    # ... process ...
    df.to_csv(output_data.path)
```

#### What Doesn't Work

Pipelines with storage-specific code may fail depending on service availability:

```python
# ❌ This only works if S3 is accessible, regardless of KFP's storage backend
@component
def s3_specific_component():
    s3 = boto3.client('s3')
    s3.download_file('my-bucket', 'my-key', '/tmp/data')
```

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

- Watches namespaces with `pipelines.kubeflow.org/enabled=true` annotation
- Creates PVC, Deployment, Service when annotation added
- No action for namespaces without annotation
- Skips namespaces where infrastructure already exists
- Idempotent: Re-processing same namespace doesn't create duplicate resources

#### Configuration Parsing

- `ObjectStoreConfig.Filesystem.Type: "pvc"` enables filesystem storage
- `ObjectStoreConfig.Filesystem.PVC.StorageClassName` sets PVC storage class
- `ObjectStoreConfig.Filesystem.PVC.Size` sets PVC capacity
- `ObjectStoreConfig.Filesystem.PVC.AccessMode` defaults to `ReadWriteOnce`
- `ObjectStoreConfig.ArtifactServer.DeploymentMode` accepts `central` or `namespaced`

### Integration Tests

#### Basic Filesystem Storage

- Deploy KFP with `defaultPipelineRoot: "kfp-artifacts://"`
- Run pipeline with two components passing artifacts
- Verify artifacts stored at correct filesystem paths
- Verify artifact server API returns correct content

#### Central Mode

- Deploy with `DeploymentMode: "central"`
- Run pipelines in multiple namespaces
- Verify single PVC in `kubeflow` namespace contains all artifacts
- Verify path structure: `/artifacts/<namespace>/<pipeline>/<run-id>/...`
- Verify `SubjectAccessReview` prevents cross-namespace access

#### Namespace-Local Mode

- Deploy with `DeploymentMode: "namespaced"`
- Create namespace with `pipelines.kubeflow.org/enabled=true` annotation
- Verify API server creates PVC, Deployment, Service in that namespace
- Run pipeline, verify artifacts stored in namespace-specific PVC
- Verify namespace isolation: artifacts in `team-a` not accessible from `team-b`

#### Authorization

- User with `readArtifact` permission can download artifacts
- User without permission receives 403 Forbidden
- `SubjectAccessReview` correctly validates namespace-scoped access

#### Sample Pipeline Compatibility

- Run official KFP sample pipelines (e.g., iris classification)
- Verify pipelines work without modification
- Verify artifact URIs in MLMD use `kfp-artifacts://` scheme

#### Error Handling

- PVC full: Clear error message when storage exhausted
- Missing namespace annotation: Pipeline fails with actionable error
- Artifact server unavailable: Pipeline step fails with clear error message
  
## Configuration Reference

### Complete Configuration Example

Here's a complete example showing all new configuration fields for filesystem storage:

```json
{
  "ObjectStoreConfig": {
    "Filesystem": {
      "Type": "pvc",
      "MountPath": "/artifacts",
      "PVC": {
        "StorageClassName": "standard",
        "Size": "100Gi",
        "AccessMode": "ReadWriteMany",
        "CreateIfNotExists": true
      }
    },
    
    "ArtifactServer": {
      "DeploymentMode": "central"
    }
  }
}
```

### Configuration Field Reference

| Field                                                | Description                       | Default           | Valid Values                            |
|------------------------------------------------------|-----------------------------------|-------------------|-----------------------------------------|
| `ObjectStoreConfig.Filesystem.Type`                  | Storage backend type              | -                 | `"pvc"`, `"local"` (testing only)       |
| `ObjectStoreConfig.Filesystem.MountPath`             | Path where PVC is mounted         | `"/artifacts"`    | Any valid path                          |
| `ObjectStoreConfig.Filesystem.PVC.StorageClassName`  | K8s StorageClass to use           | Cluster default   | Any available StorageClass              |
| `ObjectStoreConfig.Filesystem.PVC.Size`              | Size of PVC to create             | `"10Gi"`          | K8s quantity (e.g., `"100Gi"`, `"1Ti"`) |
| `ObjectStoreConfig.Filesystem.PVC.AccessMode`        | PVC access mode                   | `"ReadWriteOnce"` | `"ReadWriteOnce"`, `"ReadWriteMany"`    |
| `ObjectStoreConfig.Filesystem.PVC.CreateIfNotExists` | Auto-create PVC if missing        | `true`            | `true`, `false`                         |
| `ObjectStoreConfig.ArtifactServer.DeploymentMode`    | How artifact servers are deployed | `"central"`       | `"central"`, `"namespaced"`             |

**Notes:**

- `DeploymentMode: "central"`: Single shared artifact server in kubeflow namespace
- `DeploymentMode: "namespaced"`: Per-namespace artifact servers for isolation

### Environment Variable Overrides

All configuration fields can be overridden via environment variables:

```yaml
# Filesystem configuration
OBJECTSTORECONFIG_FILESYSTEM_TYPE=pvc
OBJECTSTORECONFIG_FILESYSTEM_MOUNTPATH=/artifacts

# PVC configuration
OBJECTSTORECONFIG_FILESYSTEM_PVC_STORAGECLASSNAME=standard
OBJECTSTORECONFIG_FILESYSTEM_PVC_SIZE=100Gi
OBJECTSTORECONFIG_FILESYSTEM_PVC_ACCESSMODE=ReadWriteMany
OBJECTSTORECONFIG_FILESYSTEM_PVC_CREATEIFNOTEXISTS=true

# Artifact server configuration
OBJECTSTORECONFIG_ARTIFACTSERVER_DEPLOYMENTMODE=central
```

## Migration and Compatibility

### Migration Path

To migrate from object storage to filesystem storage:

1. Update `defaultPipelineRoot` to use `kfp-artifacts://` scheme
2. Add `ObjectStoreConfig.Filesystem` configuration
3. Configure `ObjectStoreConfig.ArtifactServer` based on deployment needs
4. Restart KFP components to apply changes

**Note**: Migrating existing artifacts from S3/GCS to filesystem storage is not supported. Users must choose their storage backend at deployment time and stick with it.

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

2. **Access Mode Validation**: Should KFP validate the `PVC_ACCESSMODE` value or just pass it through to Kubernetes?
   - Current design: Pass through any value to Kubernetes (simpler, lets Kubernetes handle validation)
   - Alternative: Validate and restrict to `ReadWriteOnce`/`ReadWriteMany` only
   - Consideration: `ReadOnlyMany` won't work for output artifacts but might be useful for shared input data
   - Decision needed: Let Kubernetes handle all validation or add KFP-level restrictions?

3. **External Client Support**: Should the artifact server API be considered a public API for external clients, or is it an internal KFP API only?

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
- RBAC permissions for PVC management
- Sufficient storage capacity for artifacts
