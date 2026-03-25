# KFP Backend Guide (Go)

Backend component details, key Go files, local development setup, cluster deployment, and common agent workflows for working on `backend/`.

**See also:** [architecture.md](architecture.md) (E2E flow, object storage, caching, MLMD), [protobuf-build.md](protobuf-build.md) (backend API proto generation), [testing-ci.md](testing-ci.md) (Go tests, Ginkgo suites)

---

## Key backend files

| Component | File | Key symbols |
|-----------|------|-------------|
| API Server | `backend/src/apiserver/main.go` | `main()`, RPC/HTTP server init |
| Resource Manager | `backend/src/apiserver/resource/resource_manager.go` | `ResourceManager` struct |
| Client Manager | `backend/src/apiserver/client_manager/client_manager.go` | `ClientManager` -- DB, object store, exec clients |
| Pipeline Upload | `backend/src/apiserver/server/pipeline_upload_server.go` | HTTP multipart upload handler |
| Run Server | `backend/src/apiserver/server/run_server.go` | gRPC CreateRun, GetRun, TerminateRun |
| Argo Compiler | `backend/src/v2/compiler/argocompiler/argo.go` | `Compile()` -- PipelineSpec -> Argo Workflow |
| Visitor | `backend/src/v2/compiler/visitor.go` | `Accept()` -- DAG traversal |
| Container Driver | `backend/src/v2/driver/container.go` | `Container()` -- input resolution, cache check |
| DAG Driver | `backend/src/v2/driver/dag.go` | `DAG()`, `ROOT_DAG` -- execution context setup |
| Launcher V2 | `backend/src/v2/component/launcher_v2.go` | Artifact download/upload, executor invocation |
| MLMD Client | `backend/src/v2/metadata/client.go` | gRPC client for ML Metadata |
| Cache Utils | `backend/src/v2/cacheutils/cache.go` | Fingerprint generation, cache lookup |
| Cache Server | `backend/src/cache/main.go` | Mutating webhook for caching |
| Object Store | `backend/src/v2/objectstore/object_store.go` | `OpenBucket()`, `UploadBlob()`, `DownloadBlob()` |
| Object Store Config | `backend/src/v2/objectstore/config.go` | Scheme/bucket/prefix/credentials |
| MinIO Client | `backend/src/apiserver/client/minio.go` | MinIO/S3 client with credential chain |
| Store Interfaces | `backend/src/apiserver/storage/` | `PipelineStoreInterface`, `RunStoreInterface`, etc. |
| Webhook | `backend/src/apiserver/webhook/` | PipelineVersion validation webhook |

## Other key paths

- Architecture diagram: `images/kfp-cluster-wide-architecture.png`
- Platform integration (Python): `kubernetes_platform/python/kfp/`
- Platform spec proto: `kubernetes_platform/proto/`
- API definitions (Protobufs): `api/`
- Backend test suites: `backend/test/compiler`, `backend/test/v2/api`, `backend/test/end2end`
- Frontend: `frontend/` (React TypeScript, see `frontend/CONTRIBUTING.md`)
- Manifests (Kustomize bases/overlays for deployments): `manifests/`
- CI manifests and overlays used by workflows: `.github/resources/manifests/{kubernetes-native,multiuser,standalone}`
- Test data (inputs/goldens): `test_data/sdk_compiled_pipelines/valid/`, `test_data/compiled-workflows/`

## Local development setup

Always use a `.venv` virtual environment.

```bash
python3 -m venv .venv
source .venv/bin/activate
python -m pip install -U pip setuptools wheel

make -C api python-dev
make -C kubernetes_platform python-dev

pip install -e api/v2alpha1/python --config-settings editable_mode=strict
pip install -e sdk/python --config-settings editable_mode=strict
pip install -e kubernetes_platform/python --config-settings editable_mode=strict
```

### Required CLI tools

Ginkgo CLI for running Go-based test suites.

Install locally into `./bin`:

```bash
make ginkgo
export PATH="$PWD/bin:$PATH"  # ensure the ginkgo binary is on PATH
```

Or install directly with `go install` into a project-local `./bin`:

```bash
GOBIN=$PWD/bin go install github.com/onsi/ginkgo/v2/ginkgo@latest
export PATH="$PWD/bin:$PATH"
```

## Local cluster deployment

KFP provides Make targets for setting up local Kind clusters for development and testing.

### Standalone mode deployment

For deploying the latest master branch in standalone mode (single-user, no authentication):

```bash
make -C backend kind-cluster-agnostic
```

This target:

- Creates a Kind cluster named `dev-pipelines-api`
- Deploys KFP in standalone mode using `manifests/kustomize/env/platform-agnostic`
- Sets up MySQL database and metadata services
- Switches kubectl context to the `kubeflow` namespace

### Development mode deployment

For local API server development with additional debugging capabilities:

```bash
make -C backend dev-kind-cluster
```

This target:

- Creates a Kind cluster with webhook proxy support
- Installs cert-manager for certificate management
- Deploys KFP using `manifests/kustomize/env/dev-kind`
- Includes webhook proxy for advanced debugging scenarios

### Deployment modes

KFP supports two main deployment modes:

**Standalone Mode:**

- Single-user deployment without authentication
- Simpler setup, ideal for development and testing
- Uses manifests from `manifests/kustomize/env/platform-agnostic` or
  `manifests/kustomize/env/cert-manager/platform-agnostic-k8s-native`
- All users have full access to all pipelines and experiments

**Multi-user Mode:**

- Multi-tenant deployment with authentication and authorization
- Requires integration with identity providers (e.g., Dex, OIDC)
- Uses manifests from `manifests/kustomize/env/cert-manager/platform-agnostic-multi-user` or
  `manifests/kustomize/env/cert-manager/platform-agnostic-multi-user-k8s-native`
- Includes user isolation, namespace-based access control, and Istio integration
- Suitable for production environments with multiple users/teams

## Go coding standards

Follow [language-checklists.md](../../.github/review-guides/language-checklists.md#go-backend) for error handling, concurrency, and code quality rules. Key patterns: error wrapping with `%w`, no nested goroutines, reuse K8s clients, cross-reference comments for Go/Python struct alignment.

For RBAC and manifest guidelines, see [security-review.md](../../.github/review-guides/security-review.md#rbac-manifest-review).

## Common agent workflows

### Modify pipeline spec schema

1. Edit Protobufs under `api/`
2. Regenerate: `make -C api python && make -C api golang`
3. Update SDK/backend usages as needed

### Adjust Kubernetes behavior for tasks

- Resource requests/limits: set on component specs; the Driver converts these into pod spec patches.
- All other Kubernetes config: handled via `kubernetes_platform` platform spec.

### Add a new backend API endpoint

1. Edit proto files under `backend/api/v2beta1/`
2. Regenerate: `make -C backend/api generate`
3. Implement the gRPC service method in `backend/src/apiserver/server/`
4. Add ResourceManager method in `backend/src/apiserver/resource/`
5. Add storage interface method if persistence is needed

### Backend endpoint and query review reference

Whenever a new backend API endpoint and/or SQL/query logic is added or edited, use the
[PR #12999](https://github.com/kubeflow/pipelines/pull/12999) review comments and test coverage level
as a reference for review depth and expected test quality.

For endpoint additions/changes, verify:

1. **API completeness across entry points**: if behavior exists in gRPC/REST create flows, ensure upload and related flows are intentionally aligned (or explicitly documented as different).
2. **Request/response parity**: thread fields through proto -> server -> resource manager -> store -> client models; no partial plumbing.
3. **Validation coverage**: every new validation path has negative tests (bad input, malformed query params, invalid limits).
4. **Integration coverage**: add/update tests for create/get/list/update/delete paths and endpoint-specific behavior.

For `pipeline_store.go` and `pipeline_store_kubernetes.go` changes, verify:

1. **Query semantics are preserved**: filtering (including tags predicates), pagination, and sorting remain correct under combined predicates.
2. **SQL/query building safety**: dynamic filters stay parameterized and constrained; avoid brittle string concatenation.
3. **Read/write path split is deliberate**: mutating paths should avoid stale/cache-shared objects; read-heavy paths should avoid unnecessary API-server load regressions.
4. **Race-sensitive behavior is tested**: update/delete followed by immediate get/list should be covered to catch cache-latency and consistency issues.

### Modify object storage behavior

- API server storage: `backend/src/apiserver/storage/object_store.go` (interface), `backend/src/apiserver/client/minio.go` (MinIO/S3 client)
- V2 artifact storage: `backend/src/v2/objectstore/` (OpenBucket/UploadBlob/DownloadBlob)
- Config: `backend/src/v2/objectstore/config.go` (scheme/bucket/prefix/credentials)
