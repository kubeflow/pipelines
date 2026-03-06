# KFP Protobuf Build System

Proto sources, generation pipeline (Go, Python, JavaScript/TypeScript), IR YAML mapping, and generated files inventory.

**See also:** [testing-ci.md](testing-ci.md) (proto change test coverage requirements), [backend-guide.md](backend-guide.md) (backend API workflows), [sdk-guide.md](sdk-guide.md) (SDK proto usage)

---

## Proto sources

| Proto file | Package | Key messages |
|-----------|---------|-------------|
| `api/v2alpha1/pipeline_spec.proto` | `ml_pipelines` | `PipelineJob`, `PipelineSpec`, `ComponentSpec`, `DagSpec`, `PipelineTaskSpec`, `ParameterType`, `ArtifactTypeSchema`, `PipelineDeploymentConfig` |
| `api/v2alpha1/cache_key.proto` | `ml_pipelines` | `CacheKey`, `ContainerSpec`, `ArtifactNameList` |
| `kubernetes_platform/proto/kubernetes_executor_config.proto` | `kfp_kubernetes` | `KubernetesExecutorConfig`, `SecretAsVolume`, `PvcMount`, `NodeSelector`, `Toleration` |
| `backend/api/v2beta1/*.proto` | `kubeflow.pipelines.backend.api.v2beta1` | `Pipeline`, `Run`, `Experiment`, `RecurringRun` (gRPC services) |
| `backend/api/v1beta1/*.proto` | `kubeflow.pipelines.backend.api.v1beta1` | Legacy API services |

## Generation pipeline

### Pipeline spec protos (api/)

```bash
# Both Go and Python
make -C api python && make -C api golang

# Dev mode (editable install, no Docker)
make -C api python-dev
```

- Go: `protoc --go_out` generates `api/v2alpha1/go/pipelinespec/pipeline_spec.pb.go`
- Python: `protoc --python_out` generates `api/v2alpha1/python/kfp/pipeline_spec/pipeline_spec_pb2.py`
- Uses pre-built Docker image: `ghcr.io/kubeflow/kfp-api-generator:master`
- Dependencies fetched at build time: googleapis (specific commit), protobuf v26.0

### Kubernetes platform protos

```bash
make -C kubernetes_platform python && make -C kubernetes_platform golang

# Dev mode
make -C kubernetes_platform python-dev
```

- Post-processing: `kubernetes_platform/python/generate_proto.py` rewrites `import pipeline_spec_pb2` -> `from kfp.pipeline_spec import pipeline_spec_pb2` to fix cross-package imports.

### Backend API protos (multi-phase)

```bash
make -C backend/api generate
```

Generator script (`backend/api/hack/generator.sh`) runs 6 phases:
1. `protoc --go_out --go-grpc_out` -> `*.pb.go` + `*_grpc.pb.go`
2. `protoc --grpc-gateway_out` -> `*.pb.gw.go` (REST gateway)
3. `protoc --openapiv2_out` -> `*.swagger.json` (OpenAPI specs)
4. `jq` merge -> `kfp_api_single_file.swagger.json`
5. `swagger generate client` -> Go HTTP clients per service
6. `sed` post-processing for type marshaling fixes

Dockerfile (`backend/api/Dockerfile`) bundles: protoc v31.1, protoc-gen-go v1.36.6, protoc-gen-go-grpc v1.5.1, grpc-gateway v2.27.1, go-swagger v0.32.3.

### Frontend protos

```bash
cd frontend
npm run build:protos              # MLMD protos (grpc-web)
npm run build:pipeline-spec       # Pipeline spec
npm run build:platform-spec:kubernetes-platform  # K8s platform spec
npm run apis                      # v1 Swagger API clients (requires swagger-codegen-cli.jar)
npm run apis:v2beta1              # v2beta1 Swagger API clients
```

## Pipeline spec -> compiled IR YAML mapping

```
PipelineSpec proto
+-- pipeline_info -> name, display_name, description
+-- components (map<string, ComponentSpec>)
|   +-- ComponentSpec
|       +-- input_definitions (parameters + artifacts)
|       +-- output_definitions
|       +-- dag -> DagSpec (tasks map) OR
|       +-- executor_label -> reference to deployment_spec executor
+-- root -> root ComponentSpec reference
+-- deployment_spec -> PipelineDeploymentConfig (container specs)
+-- schema_version -> "2.0.0"
+-- sdk_version -> kfp SDK version
```

## NEVER EDIT DIRECTLY (Generated files)

The following files are generated; edit their sources and regenerate:

- `api/v2alpha1/python/kfp/pipeline_spec/pipeline_spec_pb2.py`
  - Source: `api/v2alpha1/pipeline_spec.proto`
  - Generate: `make -C api python` (or `make -C api python-dev` for editable local dev)
- `api/v2alpha1/go/pipelinespec/pipeline_spec.pb.go`
  - Source: `api/v2alpha1/pipeline_spec.proto`
  - Generate: `make -C api golang`
- `api/v2alpha1/go/cachekey/cache_key.pb.go`
  - Source: `api/v2alpha1/cache_key.proto`
  - Generate: `make -C api golang`
- `kubernetes_platform/python/kfp/kubernetes/kubernetes_executor_config_pb2.py`
  - Source: `kubernetes_platform/proto/kubernetes_executor_config.proto`
  - Generate: `make -C kubernetes_platform python` (or `make -C kubernetes_platform python-dev`)
- `kubernetes_platform/go/kubernetesplatform/kubernetes_executor_config.pb.go`
  - Source: `kubernetes_platform/proto/kubernetes_executor_config.proto`
  - Generate: `make -C kubernetes_platform golang`
- `backend/api/v2beta1/go_client/*.pb.go`, `*.pb.gw.go`, `*_grpc.pb.go`
  - Sources: `backend/api/v2beta1/*.proto`
  - Generate: `make -C backend/api generate`
- `backend/api/v2beta1/go_http_client/**/*.go`
  - Sources: `backend/api/v2beta1/swagger/*.swagger.json`
  - Generate: `make -C backend/api generate`
- `backend/api/v2beta1/swagger/*.swagger.json`
  - Sources: `backend/api/v2beta1/*.proto`
  - Generate: `make -C backend/api generate`
- Frontend API clients under `frontend/src/apis` and `frontend/src/apisv2beta1`
  - Sources: Swagger specs under `backend/api/**/swagger/*.json`
  - Generate: `cd frontend && npm run apis` / `npm run apis:v2beta1` (includes postprocess for TS 4.9 delete fix)
- Frontend MLMD proto outputs under `frontend/src/third_party/mlmd/generated`
  - Sources: `third_party/ml-metadata/*.proto`
  - Generate: `cd frontend && npm run build:protos`
