# KFP Architecture

How Kubeflow Pipelines components interact end-to-end, from SDK compilation through execution and artifact storage.

**See also:** [sdk-guide.md](sdk-guide.md) (SDK classes/compilation detail), [backend-guide.md](backend-guide.md) (Go component detail), [protobuf-build.md](protobuf-build.md) (proto schema)

---

## Baseline architecture

Start with the architectural diagram: `images/kfp-cluster-wide-architecture.drawio.xml` (rendered: `images/kfp-cluster-wide-architecture.png`).

## End-to-end flow (SDK -> API Server -> Driver -> Launcher -> Executor -> completion)

### SDK (Python)

The SDK compiles Python DSL to the pipeline spec (IR YAML) and optionally submits it to a remote cluster or runs locally. See [sdk-guide.md](sdk-guide.md#sdk-compilation-flow) for the full compilation flow.

### API Server (Go)

- Entry point: `backend/src/apiserver/main.go`
- gRPC server on port 8887 with TLS support; HTTP proxy on port 8888 via grpc-gateway.
- Registers gRPC services for both v1beta1 and v2beta1 APIs.
- On run creation, compiles the pipeline spec to Argo Workflows `Workflow` objects via `argocompiler.Compile()`.
- Uploads and runs pipelines remotely on a Kubernetes cluster.
- Uses `ResourceManager` (`backend/src/apiserver/resource/resource_manager.go`) to orchestrate pipeline/run/experiment CRUD operations.
- Uses `ClientManager` (`backend/src/apiserver/client_manager/client_manager.go`) to initialize DB, object store, and execution clients.
- Configuration via `viper` with nested env vars (e.g., `OBJECTSTORECONFIG_ACCESSKEY`).

### Backend Compiler (Go)

- Entry point: `backend/src/v2/cmd/compiler/main.go`
- Core logic: `backend/src/v2/compiler/argocompiler/argo.go` -> `Compile()` function
- Visitor pattern: `backend/src/v2/compiler/visitor.go` -> `Accept()` traverses the PipelineSpec DAG
- Converts KFP PipelineSpec IR -> Argo Workflow YAML manifest
- Configurable via `Options`: LauncherImage, DriverImage, PipelineRoot, CacheDisabled, security context defaults

### Driver (Go)

- Entry point: `backend/src/v2/cmd/driver/main.go`
- Runs as an init container before each task execution.
- Three driver types:
  - **ROOT_DAG** (`backend/src/v2/driver/dag.go:41`): Initializes the top-level pipeline execution, creates pipeline context in MLMD.
  - **DAG** (`backend/src/v2/driver/dag.go`): Executes sub-DAG nodes, manages iteration logic for ParallelFor.
  - **CONTAINER** (`backend/src/v2/driver/container.go:46`): Resolves inputs, computes pod spec patch, checks cache, writes ExecutorInput.
- Outputs written to Argo parameters: `ExecutionID`, `PodSpecPatch`, `CachedDecision`, `Condition`.
- All other Kubernetes configuration originates from the platform spec implemented by `kubernetes_platform`.

### Launcher (Go)

- Entry point: `backend/src/v2/cmd/launcher-v2/main.go`
- Core logic: `backend/src/v2/component/launcher_v2.go`
- Not used by Subprocess/Docker runners.
- Downloads input artifacts from object store, invokes the user container (or importer), uploads output artifacts, records results in MLMD.
- Two executor types: `container` (user workload) and `importer` (external data import).

### Python Executor

- Entrypoint: `sdk/python/kfp/dsl/executor_main.py`.
- Never involved during the pipeline compilation stage.
- During task runtime, `kfp` is installed with `--no-deps` and `_KFP_RUNTIME=true` disables most SDK imports.
- API Server mode: the Go launcher (copied via `init` container) executes the executor inside the user container
  defined by the component `base_image` (there is a default).
- Subprocess/Docker runners: the launcher is skipped; executor runs directly.

### Execution flows

**Remote execution (API Server mode):**

```
Client.create_run()
  -> API Server: ResourceManager.CreateRun()
    -> argocompiler.Compile(PipelineSpec) -> Argo Workflow YAML
      -> ExecClient.Submit() -> Argo Controller
        -> ROOT_DAG Driver pod -> creates pipeline context in MLMD
          -> DAG/CONTAINER Driver pods -> resolve inputs, check cache
            -> Launcher pod -> download artifacts -> invoke user container
              -> executor_main.py -> Executor.execute() -> component function
                -> Launcher -> upload outputs -> publish to MLMD
```

**Local execution (SubprocessRunner):**

```
pipeline_func()
  -> run_local_pipeline() (local/pipeline_orchestrator.py)
    -> SubprocessTaskHandler.run() (local/subprocess_task_handler.py)
      -> executor_main.py -> component function -> outputs to local_outputs/
```

## Object storage

KFP uses a multi-provider object storage layer for pipeline specs and artifacts.

### API Server object store

- Interface: `backend/src/apiserver/storage/object_store.go` -- `ObjectStore` interface with `AddFile`, `GetFile`, `DeleteFile`, etc.
- MinIO client: `backend/src/apiserver/client/minio.go` -- creates minio-go client with credential chain (static, env, IAM).
- Initialization: `backend/src/apiserver/client_manager/client_manager.go` -- builds config from env vars, creates S3 client via AWS SDK v2, supports IRSA.

### V2 artifact object store

- Config: `backend/src/v2/objectstore/config.go` -- scheme (gs://, s3://, minio://), bucket, prefix, credentials from K8s secrets.
- Operations: `backend/src/v2/objectstore/object_store.go`:
  - `OpenBucket()`: Opens bucket via gocloud.dev blob API (S3/MinIO via AWS SDK v2, GCS via gcsblob).
  - `UploadBlob()`: Recursively uploads files/directories.
  - `DownloadBlob()`: Streams downloads with prefix listing.

## Execution caching

- Cache server: `backend/src/cache/main.go` -- Kubernetes mutating admission webhook on `/mutate`.
- Cache utilities: `backend/src/v2/cacheutils/cache.go`:
  - `GenerateCacheKey()` + `GenerateFingerPrint()`: Computes cache key from inputs/outputs/container spec.
  - `GetExecutionCache()`: Queries by fingerprint, pipelineName, namespace; returns most recent match.
  - `CreateExecutionCache()`: Records new cache entry after execution.
- Cache storage: MySQL-backed via `backend/src/cache/storage/`.
- Flow: Driver(CONTAINER) generates fingerprint -> queries cache -> if hit, uses cached execution; if miss, executes and records.

## Metadata tracking (MLMD)

- Client: `backend/src/v2/metadata/client.go` -- gRPC client to ML Metadata service.
- Key operations: `GetPipeline()`, `CreateExecution()`, `PublishExecution()`, `RecordArtifact()`, `GetExecutionsInDAG()`.
- Execution types: `system.ContainerExecution`, `system.DAGExecution`, `system.ImporterExecution`.
- Connection: gRPC with retry logic, TLS support, max message size 100MB.

## CRD controllers

The repo contains CRD controllers under `backend/src/crd/controller/`:

- **ScheduledWorkflow controller** (`backend/src/crd/controller/scheduledworkflow/`): Manages recurring runs, triggers workflow creation on schedule.
- **Viewer controller** (`backend/src/crd/controller/viewer/`): Manages viewer CRDs for TensorBoard and other visualizations.

### Controller design rules

- Controllers should **not** make synchronous gRPC/HTTP calls back to the API server during reconciliation. This creates circular dependencies and fragile failure modes.
- Pass data to controllers via CRD spec fields, annotations, or labels -- not runtime API lookups.
- Controllers should read state from informer caches, not direct API calls.
- New controller dependencies (gRPC clients, additional K8s clients) require architectural justification.

### Dependency direction

```
API Server -> CRD controllers (creates CRD objects)
CRD controllers -> CRD types only (backend/src/crd/pkg)
CRD controllers should NOT import from backend/src/apiserver/
```

## Packages and naming

- All Python packages are installed under the `kfp` namespace.
- KFP Python packages:
  - **kfp**: Primary SDK (DSL, client, local execution).
  - **kfp-pipeline-spec**: Protobuf-defined API contract used by SDK and backend.
  - **kfp-kubernetes**: Kubernetes Python extension layer for `kfp` located at `kubernetes_platform/python` for
    Kubernetes-specific settings and platform spec.
- The `kfp-kubernetes` package imports generated Python code from `kfp-pipeline-spec` and renames imports via
  `kubernetes_platform/python/generate_proto.py` to resolve inconsistencies.
