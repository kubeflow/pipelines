# Agent Guide: Kubeflow Pipelines (KFP) Monorepo

## Purpose

- **Who this is for**: AI agents and developers working inside this repo.
- **What you get**: The minimum set of facts, files, and commands to navigate, modify, and run KFP locally.

### Document metadata

- Last updated: 2026-02-13
- Scope: KFP master branch (v2 engine), backend (Go), SDK (Python), frontend (React 16)

### Maintenance (agents and contributors)

- If you change commands, file paths, Make targets, environment variables, or workflows in this repo, update this guide in the relevant sections (Local development, Local testing, Local execution, Regenerate protobufs, Frontend development, CI/CD).
- When you add or change generated files, update the "🚫 NEVER EDIT DIRECTLY (Generated files)" section with sources and regeneration commands.
- When you change CI matrices (Kubernetes versions, pipeline stores, proxy/cache toggles, Argo versions) or add/remove workflows, update the CI/CD section.
- If you come across new common errors or fixes, extend "Common error patterns and quick fixes".
- Always bump the "Last updated" date above when you make substantive changes.

## Baseline architecture

- Start with inspecting the architectural diagram found here `images/kfp-cluster-wide-architecture.drawio.xml` (rendered format can be found here: `images/kfp-cluster-wide-architecture.png`).

## End-to-end flow (SDK → API Server → Driver → Launcher → Executor → completion)

- **SDK**:
  - Compiles Python DSL to the pipeline spec (IR YAML). See `sdk/python/kfp/compiler/pipeline_spec_builder.py`.
  - The pipeline spec schema is defined via Protobufs under `api/`.
  - Can execute pipelines locally via Subprocess or Docker runner modes.
- **API Server**:
  - On run creation, compiles the pipeline spec to Argo Workflows `Workflow` objects.
  - Uploads and runs pipelines remotely on a Kubernetes cluster.
- **Driver**:
  - Resolves input parameters.
  - Computes the pod spec patch based on component resource requests/limits.
  - All other Kubernetes configuration originates from the platform spec implemented by `kubernetes_platform`.
- **Launcher**:
  - Not used by Subprocess/Docker runners.
  - Downloads input artifacts, uploads outputs, invokes the Python executor, handles executor results.
- **Python executor**:
  - Entrypoint: `sdk/python/kfp/dsl/executor_main.py`.
  - Never involved during the pipeline compilation stage.
  - During task runtime, `kfp` is installed with `--no-deps` and `_KFP_RUNTIME=true` disables most SDK imports.
  - API Server mode: the Go launcher (copied via `init` container) executes the executor inside the user container
    defined by the component `base_image` (there is a default).
  - Subprocess/Docker runners: the launcher is skipped; executor runs directly.

## Packages and naming

- All Python packages are installed under the `kfp` namespace.
- KFP Python packages:
  - **kfp**: Primary SDK (DSL, client, local execution).
  - **kfp-pipeline-spec**: Protobuf-defined API contract used by SDK and backend.
  - **kfp-kubernetes**: Kubernetes Python extension layer for `kfp` located at `kubernetes_platform/python` for
    Kubernetes-specific settings and platform spec.
- The `kfp-kubernetes` package imports generated Python code from `kfp-pipeline-spec` and renames imports via
  `kubernetes_platform/python/generate_proto.py` to resolve inconsistencies.

## Local development setup

- Always use a `.venv` virtual environment.

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

- Ginkgo CLI for running Go-based test suites.

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

KFP provides Make targets for setting up local Kind clusters for development and testing:

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

## Local testing

- Python (SDK):

```bash
pip install -r sdk/python/requirements-dev.txt
pytest -v sdk/python/kfp
```

- Python (`kfp-kubernetes`):

```bash
pytest -v kubernetes_platform/python/test
```

- Go (backend) unit tests only, excluding integration/API/Compiler/E2E tests:

```bash
go test -v $(go list ./backend/... | \
  grep -v backend/test/v2/api | \
  grep -v backend/test/integration | \
  grep -v backend/test/v2/integration | \
  grep -v backend/test/initialization | \
  grep -v backend/test/v2/initialization | \
  grep -v backend/test/compiler | \
  grep -v backend/test/end2end)
```

Notes:

- API Server tests under `backend/test/v2/api` are integration tests run with Ginkgo; they require a running cluster and are not part of unit tests.
- Compiler tests live under `backend/test/compiler` and E2E tests under `backend/test/end2end`; both are Ginkgo-based and excluded from unit presubmits.

### Backend Ginkgo test suites

- Compiler tests:

```bash
# Run compiler tests
ginkgo -v ./backend/test/compiler

# Update compiled workflow goldens when intended
ginkgo -v ./backend/test/compiler -- -updateCompiledFiles=true

# Auto-create missing goldens (default true); disable with:
ginkgo -v ./backend/test/compiler -- -createGoldenFiles=false
```

- v2 API integration tests (label-filterable):

```bash
# All API tests
ginkgo -v ./backend/test/v2/api

# Example: run only Smoke-labeled tests with ginkgo
ginkgo -v --label-filter="Smoke" ./backend/test/v2/api
```

- End-to-end tests:

```bash
ginkgo -v ./backend/test/end2end -- -namespace=kubeflow -isDebugMode=true
```

Test data is centralized under:

- `test_data/pipeline_files/valid/` (inputs) with a `valid/critical/` subset for smoke lanes
- `test_data/compiled-workflows/` (expected compiled Argo Workflows)

## Local execution

- **Subprocess Runner** (no Docker required):

```python
from kfp import local
local.init(runner=local.SubprocessRunner())

# Run components directly
task = my_component(param="value")
print(task.output)
```

- **Docker Runner** (requires Docker):

```python
from kfp import local
local.init(runner=local.DockerRunner())

# Runs components in containers
task = my_component(param="value")
```

- **Pipeline execution**:

```python
# Pipelines can be executed like regular functions
run = my_pipeline(input_param="test")

# If the pipeline has a single output:
print(run.output)

# Or, for named outputs:
print(run.outputs['<output_name>'])
```

Note: Local execution outputs are stored in `./local_outputs` by default.

Notes:

- SubprocessRunner supports only Lightweight Python Components (executes the KFP Python executor directly).
- Use DockerRunner for Container Components or when task images require containerized execution.

## Regenerate protobufs after schema changes

- Pipeline spec Protobufs live under `api/`.
- Run both Python and Go generations:

```bash
make -C api python && make -C api golang
```

- Note for Linux with SELinux: protoc-related steps may fail under enforcing mode.

  - Temporarily disable before generation: `sudo setenforce 0`
  - Re-enable after: `sudo setenforce 1`

- `api/v2alpha1/python/kfp/pipeline_spec/pipeline_spec_pb2.py` is NOT committed. Any workflow or script installing
  `kfp/api` from source must generate this file beforehand.

### 🚫 NEVER EDIT DIRECTLY (Generated files)

The following files are generated; edit their sources and regenerate:

- `api/v2alpha1/python/kfp/pipeline_spec/pipeline_spec_pb2.py`
  - Source: `api/v2alpha1/pipeline_spec.proto`
  - Generate: `make -C api python` (or `make -C api python-dev` for editable local dev)
- `kubernetes_platform/python/kfp/kubernetes/kubernetes_executor_config_pb2.py`
  - Source: `kubernetes_platform/proto/kubernetes_executor_config.proto`
  - Generate: `make -C kubernetes_platform python` (or `make -C kubernetes_platform python-dev`)
- Frontend API clients under `frontend/src/apis` and `frontend/src/apisv2beta1`
  - Sources: Swagger specs under `backend/api/**/swagger/*.json`
  - Generate: `cd frontend && npm run apis` / `npm run apis:v2beta1` (includes postprocess for TS 4.9 delete fix)
- Frontend MLMD proto outputs under `frontend/src/third_party/mlmd/generated`
  - Sources: `third_party/ml-metadata/*.proto`
  - Generate: `cd frontend && npm run build:protos`

## Key paths and files

- Architecture diagram: `images/kfp-cluster-wide-architecture.png`
- SDK compiler: `sdk/python/kfp/compiler/pipeline_spec_builder.py`
- DSL core: `sdk/python/kfp/dsl/` (e.g., `component_factory.py`, `pipeline_task.py`, `pipeline_context.py`)
- Executor entrypoint: `sdk/python/kfp/dsl/executor_main.py`
- Platform integration (Python): `kubernetes_platform/python/kfp/`
- Platform spec proto: `kubernetes_platform/proto/`
- API definitions (Protobufs): `api/`
- Backend (API server, driver, launcher, etc.): `backend/`
- Backend test suites: `backend/test/compiler`, `backend/test/v2/api`, `backend/test/end2end`
- Frontend: `frontend/` (React TypeScript, see `frontend/CONTRIBUTING.md`)
- Manifests (Kustomize bases/overlays for deployments): `manifests/`
- CI manifests and overlays used by workflows: `.github/resources/manifests/{kubernetes-native,multiuser,standalone}`
- Test data (inputs/goldens): `test_data/pipeline_files/valid/`, `test_data/compiled-workflows/`

## Documentation

- SDK reference docs are auto-generated with Sphinx using autodoc from Python docstrings. Keep SDK docstrings
  user-facing and accurate, as they appear in published documentation.

## Frontend development

The KFP frontend is a React TypeScript application that provides the web UI for Kubeflow Pipelines.

### Prerequisites

- Node.js version specified in `frontend/.nvmrc` (currently v22.19.0)
- Java 8+ (required for `java -jar swagger-codegen-cli.jar` when generating API clients)
- Use [nvm](https://github.com/nvm-sh/nvm) or [fnm](https://github.com/Schniz/fnm) for Node version management:

  ```bash
  # With fnm (faster)
  fnm install 22.19.0 && fnm use 22.19.0
  # With nvm
  nvm install 22.19.0 && nvm use 22.19.0
  ```

### Setup and installation

```bash
cd frontend
npm ci  # Install exact dependencies from package-lock.json
```

### Development workflows

#### Local development with mock API

Quick start for UI development without backend dependencies:

```bash
npm run mock:api    # Start mock backend server on port 3001
npm start           # Start Vite dev server on port 3000 (hot reload)
```

#### Local development with real cluster

For full integration testing against a real KFP deployment:

1. **Single-user mode**:

   ```bash
   # Deploy KFP standalone (see Local cluster deployment section)
   make -C backend kind-cluster-agnostic

   # Scale down cluster UI
   kubectl -n kubeflow scale --replicas=0 deployment/ml-pipeline-ui

   # Start local development
   npm run start:proxy-and-server  # Proxy to cluster + hot reload
   ```

2. **Multi-user mode**:

   ```bash
   export VITE_NAMESPACE=kubeflow-user-example-com
   npm run build
   # Install mod-header Chrome extension for auth headers
   npm run start:proxy-and-server
   ```

### Key technologies and architecture

- **React 16** with TypeScript
- **Material-UI v3** for components
- **React Router v5** for navigation
- **Dagre** for graph layout visualization
- **D3** for data visualization
- **Vitest + Testing Library** for UI testing
- **Jest** for frontend server tests (UI tests migrated off Jest/Enzyme)
- **Prettier + ESLint** for code formatting/linting
- **Storybook** for component development
- **Tailwind CSS** for utility-first styling

### Essential commands (frontend)

- `npm start` - Start Vite dev server with hot reload (port 3000)
- `npm run start:proxy-and-server` - Full development with cluster proxy
- `npm run mock:api` - Start mock backend API server (port 3001)
- `npm run build` - Production build
- `npm run test` - Run Vitest UI tests (same as `test:ui`, with `LC_ALL` set)
- `npm run test:ui` - Run Vitest UI tests
- `npm run test:ui:coverage` - Run Vitest UI tests with coverage
- `npm run test:ui:coverage:loop` - Run Vitest UI coverage with a capped worker count (stability loop)
- `npm run test -u` - Update Vitest snapshots
- `npm run lint` - Run ESLint
- `npm run typecheck` - Run TypeScript typecheck (`tsc --noEmit`)
- `npm run format` - Format code with Prettier
- `npm run storybook` - Start Storybook on port 6006

### Code generation

The frontend includes several generated code components:

- **API clients**: Generated from backend Swagger specs

  ```bash
  npm run apis        # Generate v1 API clients
  npm run apis:v2beta1 # Generate v2beta1 API clients
  ```

  Note: Ensure `swagger-codegen-cli.jar` is available to `java -jar` when running from `frontend/`
  (e.g., place the JAR in `frontend/` or reference a full path).

- **Protocol Buffers**: Generated from proto definitions

  ```bash
  npm run build:protos              # MLMD protos
  npm run build:pipeline-spec       # Pipeline spec protos
  npm run build:platform-spec:kubernetes-platform # K8s platform spec
  ```

### Testing

- **UI tests**: `npm run test:ui` or `npm test` (Vitest + Testing Library)
- **Server tests**: `npm run test:server:coverage` (Jest)
- **Coverage**: `npm run test:ui:coverage` (Vitest) + `npm run test:coverage` (Vitest UI + Jest server)
- **Stability loop**: `npm run test:ui:coverage:loop` (Vitest coverage with capped workers)
- **CI pipeline**: `npm run test:ci` (format check + lint + typecheck + Vitest UI coverage + Jest coverage)
- **Snapshot tests**: Auto-update with `npm test -u` or `npm run test:ui -- -u` (Vitest)

## CI/CD (GitHub Actions)

- Workflows: `.github/workflows/` (build, test, lint, release)
- Composite actions: `.github/actions/` (e.g., `kfp-k8s`, `create-cluster`, `deploy`, `test-and-report`)
- Typical checks: Go unit tests (backend), Python SDK tests, frontend tests/lint, image builds.

### Test matrices and variants (Kubernetes, stores, proxy, cache)

- Kubernetes versions: CI runs a matrix across a low and high supported version, commonly `v1.29.2` and `v1.31.0`.
  - Examples: `e2e-test.yml`, `sdk-execution.yml`, `upgrade-test.yml`, `kfp-kubernetes-execution-tests.yml`, `kfp-webhooks.yml`, `api-server-tests.yml`, `compiler-tests.yml`, `legacy-v2-api-integration-tests.yml`, `integration-tests-v1.yml`, and frontend integration in `e2e-test-frontend.yml`.
- Pipeline store variants (v2 engine): tests run with `database` and `kubernetes` stores, and a dedicated job compiles pipelines to Kubernetes-native manifests.
  - Example: `e2e-test.yml` job "API integration tests v2 - K8s with ${pipeline_store}" and "compile pipelines with Kubernetes".
- Argo Workflows version matrix for compatibility (where relevant): e.g., `e2e-test.yml` includes an Argo job (e.g., `v3.5.14`).
- Proxy / cache toggles: dedicated jobs run with HTTP proxy enabled and with execution cache disabled to validate those modes.
- Artifacts: failing logs and test outputs are uploaded as workflow artifacts for debugging.

### CI cluster setup and helpers

- Kind-based clusters are provisioned via the `kfp-cluster` composite action, parameterized by `k8s_version`, `pipeline_store`, `proxy`, `cache_enabled`, and optional `argo_version`.
- The `create-cluster` and `deploy` actions are used by newer suites; `kfp-k8s` installs SDK components from source inside jobs that execute Python-based tests.
- The `protobuf` composite action prepares `protoc` and related dependencies when compiling Python protobufs.
- The `create-cluster` action caches Kind node images by Kubernetes version to reduce Docker Hub pulls.
- Python workflows use `actions/cache@v5` for pip cache to reduce repeated dependency installs.

### Code style and formatting

- **Prettier** config in `.prettierrc.yaml`:
  - Single quotes, trailing commas, 100 char line width
  - Format: `npm run format`
  - Check: `npm run format:check`
- **ESLint** extends `react-app` with custom rules in `.eslintrc.yaml`
- **Auto-format on save**: Configure your IDE with the Prettier extension

Notes:

- Legacy `kfp-samples.yml` and `periodic.yml` workflows were removed.

### Workflow path verification

To verify all GitHub workflow path references are valid:

1. **Iterate through all workflow files** in `.github/workflows/` (both `.yml` and `.yaml` files)
2. **Parse each YAML file** and extract path references from:
   - `working-directory` fields
   - `dockerfile` and `context` fields in Docker build steps
   - `script` or command paths (look for `./` prefixes)
   - Any string values that appear to be file/directory paths
   - Action references (e.g., `./.github/actions/...`)
3. **Clean extracted paths** by removing `./` prefixes and variable expansions
4. **Verify each extracted path exists** in the project filesystem
5. **Report missing paths** and which workflows reference them

This verification ensures workflow integrity and prevents CI failures due to missing files or incorrect path references.

### Feature flags

KFP frontend supports feature flags for development:

- Configure in `src/features.ts`
- Access via `http://localhost:3000/#/frontend_features`
- Manage locally: `localStorage.setItem('flags', "")`

### Common development tasks

- **Add new API**: Update swagger specs, run `npm run apis`
- **Update proto definitions**: Modify protos, run respective build commands
- **Add new component**: Create in `atoms/` or `components/`, add tests and stories
- **Debug server**: Use `npm run start:proxy-and-server-inspect`
- **Bundle analysis**: `npm run analyze-bundle`

### Troubleshooting

- **Port conflicts**: Frontend uses 3000 (React), 3001 (Node server), 3002 (API proxy)
- **Node version issues**: Ensure you're using the version in `.nvmrc`
- **API generation failures**: Check that swagger-codegen-cli.jar is in PATH
- **Proto generation**: Requires `protoc` and `protoc-gen-grpc-web` in PATH
- **Mock backend**: Limited API support; use real cluster for full testing

## Lint and formatting checks

- Go lint (CI uses `golangci-lint`):

```bash
golangci-lint run
```

- Python SDK import/order and unused import cleanups:

```bash
pip install -r sdk/python/requirements-dev.txt
pycln --check sdk/python
isort --check --profile google sdk/python
```

- Python SDK formatting (YAPF + string fixer):

```bash
pip install yapf pre_commit_hooks
python3 -m pre_commit_hooks.string_fixer $(find sdk/python/kfp/**/*.py -type f)
yapf --recursive --diff sdk/python/
```

- Python SDK docstring formatting:

```bash
pip install docformatter
docformatter --check --recursive sdk/python/ --exclude "compiler_test.py"
```

## Code-tree knowledge graph

A pre-built knowledge graph of the entire codebase is available under `docs/code-tree/`. Use it to quickly understand architecture, trace dependencies, and find relevant code before making changes or reviewing PRs.

### Available artifacts

| File | Contents | Use case |
|------|----------|----------|
| `docs/code-tree/summary.md` | Architecture overview, hub files, key types, inheritance, module deps | **Start here** for any exploration |
| `docs/code-tree/graph.json` | Full knowledge graph (19,632 nodes, 33,058 edges) | Programmatic queries with `jq` |
| `docs/code-tree/tags.json` | Flat symbol index (17,636 symbols with file:line locations) | Quick symbol lookup |
| `docs/code-tree/modules.json` | Directory-level dependency map (507 modules) | Module impact analysis |

### Querying the graph

Use the query tool at `tools/code-tree/query_graph.py`:

```bash
# Find where a symbol is defined
python3 tools/code-tree/query_graph.py --symbol Pipeline

# Trace what a file depends on
python3 tools/code-tree/query_graph.py --deps sdk/python/kfp/compiler/compiler.py

# Find what imports a file (reverse deps / blast radius)
python3 tools/code-tree/query_graph.py --rdeps sdk/python/kfp/dsl/pipeline_context.py

# Show inheritance hierarchy for a class
python3 tools/code-tree/query_graph.py --hierarchy Artifact

# Search symbols by keyword
python3 tools/code-tree/query_graph.py --search "compiler"

# Get module overview (files, deps, symbols)
python3 tools/code-tree/query_graph.py --module sdk/python/kfp/compiler

# Find entry points (main functions, unimported files)
python3 tools/code-tree/query_graph.py --entry-points

# Extract code chunks from a file for context
python3 tools/code-tree/query_graph.py --chunks sdk/python/kfp/compiler/compiler.py
```

Add `--json` to any query for machine-readable output.

### Codebase statistics

- **1,996 files** across 7 languages: Python (908), Go (693), TypeScript (273), Bash (57), Protobuf (54), JavaScript (10), SQL (1)
- **17,636 symbols**: 10,634 methods, 3,285 functions, 1,354 structs, 1,180 classes, 743 interfaces, 347 constants, 93 enums
- **14,938 import** relationships, **484 inheritance** relationships

### Entry points

| Entry point | Purpose |
|-------------|---------|
| `backend/src/apiserver/main.go` | API server |
| `backend/src/v2/cmd/driver/main.go` | V2 driver |
| `backend/src/v2/cmd/compiler/main.go` | Backend compiler |
| `backend/src/v2/cmd/launcher-v2/main.go` | V2 launcher |
| `backend/src/cache/main.go` | Cache server |
| `backend/src/agent/persistence/main.go` | Persistence agent |
| `backend/src/crd/controller/scheduledworkflow/main.go` | Scheduled workflow controller |
| `backend/src/crd/controller/viewer/main.go` | Viewer controller |
| `sdk/python/kfp/dsl/executor_main.py` | Python executor entrypoint |

### Hub files (most connected)

These files are imported by the most other files. Changes to them have the widest blast radius:

| File | Imported by |
|------|-------------|
| `backend/src/common/util/time.go` | 150 files |
| `components/google-cloud/.../types/artifact_types.py` | 127 files |
| `sdk/python/kfp/local/io.py` | 124 files |
| `backend/api/v1beta1/python_http_client/.../configuration.py` | 110 files |
| `backend/api/v2beta1/python_http_client/.../configuration.py` | 110 files |
| `backend/src/common/util/json.go` | 101 files |

### Key module dependencies

```
sdk/python/kfp/compiler -> sdk/python/kfp/dsl
sdk/python/kfp/client   -> sdk/python/kfp/dsl
backend/src/apiserver/*  -> backend/src/common/util
backend/src/cache/*      -> backend/src/apiserver/archive, backend/src/common/util
backend/src/v2/component -> backend/src/common/util
backend/src/v2/compiler  -> backend/src/common/util
```

### Key inheritance hierarchies

```
Artifact (sdk/python/kfp/dsl/types/artifact_types.py)
  |- ClassificationMetrics, Dataset, HTML, Markdown, Metrics, Model, SlicedClassificationMetrics

ModelBase (sdk/python)
  |- BinaryPredicate
  |    |- EqualsPredicate, GreaterThanPredicate, LessThenPredicate, ...
  |- ComponentSpec, ContainerSpec, CachingStrategySpec, ...
```

### Regenerating the graph

When significant code changes occur, regenerate:

```bash
python3 tools/code-tree/code_tree.py --repo-root .
```

### Using the graph for code development

1. **Before writing code**: Read `docs/code-tree/summary.md` for architecture context
2. **Find the right file**: Use `--search` or `--symbol` to locate definitions
3. **Understand dependencies**: Use `--deps` to see what a file needs, `--module` for directory-level view
4. **Check blast radius**: Use `--rdeps` before modifying shared files to understand impact
5. **Extract context**: Use `--chunks` to pull clean code snippets into your working context

### Using the graph for PR reviews

1. **Assess impact**: Run `--rdeps` on each changed file to gauge blast radius
2. **Verify dependency direction**: Check `--deps` to ensure changes don't introduce circular dependencies
3. **Check hub file changes**: Compare changed files against the hub files table above — changes to hub files need extra scrutiny
4. **Trace inheritance**: Use `--hierarchy` to verify that class changes don't break subclasses
5. **Module boundary check**: Use `--module` to verify changes respect module boundaries

## Common agent workflows

- **Modify pipeline spec schema**:
  1. Edit Protobufs under `api/`
  2. Regenerate: `make -C api python && make -C api golang`
  3. Update SDK/backend usages as needed
- **Adjust Kubernetes behavior for tasks**:
  - Resource requests/limits: set on component specs; the Driver converts these into pod spec patches.
  - All other Kubernetes config: handled via `kubernetes_platform` platform spec.

### Proto changes require test coverage

When any `.proto` file is modified (especially `api/v2alpha1/pipeline_spec.proto` or `kubernetes_platform/proto/kubernetes_executor_config.proto`), you **must** add or update test files that exercise the changed or new proto fields. The test system uses a three-file pipeline that must stay in sync:

```
1. Python pipeline source  (.py)  →  test_data/sdk_compiled_pipelines/valid/{critical,essential}/
2. Compiled pipeline spec  (.yaml) →  test_data/sdk_compiled_pipelines/valid/{critical,essential}/
3. Compiled Argo Workflow  (.yaml) →  test_data/compiled-workflows/
```

**Step-by-step process:**

1. **Write a Python pipeline** that uses the new or changed proto field. Place it in:
   - `test_data/sdk_compiled_pipelines/valid/critical/` — for features that are functionally important (e.g., new pipeline features, artifact handling, parameter types, pod metadata, caching behavior)
   - `test_data/sdk_compiled_pipelines/valid/essential/` — for foundational features that validate core KFP functionality (e.g., basic component composition, pip installs, pipeline nesting, conditionals, loops)

2. **Compile the Python pipeline** to a KFP pipeline spec YAML. Place the output `.yaml` in the same directory as the `.py` file:
   ```bash
   kfp dsl compile --py test_data/sdk_compiled_pipelines/valid/critical/my_pipeline.py \
                    --output test_data/sdk_compiled_pipelines/valid/critical/my_pipeline.yaml
   ```

3. **Generate the Argo Workflow golden file** by running the compiler tests with the update flag:
   ```bash
   ginkgo -v ./backend/test/compiler -- -updateCompiledFiles=true
   ```
   This writes the compiled Argo Workflow YAML to `test_data/compiled-workflows/my_pipeline.yaml`.

4. **Verify all three files are consistent** by running the compiler tests without the update flag:
   ```bash
   ginkgo -v ./backend/test/compiler
   ```

**How the test framework uses these files:**

| Test suite | Input directory | What it does |
|------------|----------------|--------------|
| Compiler tests (`backend/test/compiler`) | `test_data/sdk_compiled_pipelines/valid/**/*.yaml` → `test_data/compiled-workflows/*.yaml` | Compiles every pipeline spec to an Argo Workflow and compares against the golden YAML |
| E2E essential (`Label(E2eEssential)`) | `test_data/sdk_compiled_pipelines/valid/essential/` | Uploads and runs every pipeline in `essential/` on a live cluster |
| E2E critical (`Label(E2eCritical)`) | `test_data/sdk_compiled_pipelines/valid/critical/` | Uploads and runs every pipeline in `critical/` on a live cluster |
| Smoke tests (`Label(Smoke)`) | Hand-picked files from `essential/` and `critical/` | Runs a small subset for fast validation |

**Naming conventions:**

- The `.py` and `.yaml` files in `critical/` or `essential/` share the same base name: `my_pipeline.py` ↔ `my_pipeline.yaml`
- The Argo Workflow golden in `compiled-workflows/` also uses the same base name: `my_pipeline.yaml`
- Exception: some files use `_compiled` suffix (e.g., `iris_pipeline.py` → `iris_pipeline_compiled.yaml`)

**When NOT to add test files:**

- Pure documentation or comment changes in `.proto` files that don't alter the wire format
- Changes to deprecated v1beta1 protos (unless explicitly maintaining v1 compatibility)

## Quick reference

### Essential commands

- Compile pipeline: `kfp dsl compile --py pipeline.py --output pipeline.yaml`
- Generate protos: `make -C api python && make -C api golang`
- Deploy local cluster (standalone): `make -C backend kind-cluster-agnostic`
- Deploy local cluster (development) and run the API server in the IDE: `make -C backend dev-kind-cluster`
- Run SDK tests: `pytest -v sdk/python/kfp`
- Run backend unit tests: `go test -v $(go list ./backend/... | grep -v backend/test/)`
- Run compiler tests: `ginkgo -v ./backend/test/compiler`
- Run API tests: `ginkgo -v --label-filter="Smoke" ./backend/test/v2/api`
- Run E2E tests: `ginkgo -v ./backend/test/end2end -- -namespace=kubeflow`
- Check formatting:
  `yapf --recursive --diff sdk/python/ && pycln --check sdk/python && isort --check --profile google sdk/python`
- Frontend dev server: `cd frontend && npm start`
- Frontend with cluster: `cd frontend && npm run start:proxy-and-server`
- Frontend tests: `cd frontend && npm run test:ui` (Vitest) or `npm test` (same as `test:ui`)
- Frontend formatting: `cd frontend && npm run format`
- Generate frontend APIs: `cd frontend && npm run apis`

### Key environment variables

- `_KFP_RUNTIME=true`: Disables SDK imports during task execution
- `VITE_NAMESPACE=...`: Sets the target namespace for the frontend in multi-user mode
- `LOCAL_API_SERVER=true`: Enables local API server testing mode when running integration tests on a Kind cluster

## Troubleshooting and pitfalls

- `_KFP_RUNTIME=true` during executor runtime disables much of the SDK; avoid importing SDK-only modules from task code.
- `kfp` is installed into task containers with `--no-deps`; ensure runtime dependencies are present in `base_image`.
- SELinux enforcing can break proto generation; toggle with `setenforce` as noted above.
- Do not assume `pipeline_spec_pb2.py` exists in the repo; it must be generated.
- Frontend API generation requires `swagger-codegen-cli.jar` in PATH.
- Frontend proto generation requires `protoc` and `protoc-gen-grpc-web` binaries.
- Node version must match `.nvmrc`; use nvm/fnm to manage versions.
- Frontend port conflicts: 3000 (Vite), 3001 (Node server), 3002 (API proxy), 6006 (Storybook).

### Common error patterns and quick fixes

- Protobuf generation fails with "protoc: command not found": use the Make targets that run this in a container.
- Protobuf generation fails under SELinux enforcing: temporarily disable with `sudo setenforce 0`; re-enable after.
- API client generation fails with "Unable to access jarfile swagger-codegen-cli.jar": ensure the JAR is present and use `java -jar <path>/swagger-codegen-cli.jar` from `frontend/`.
- Frontend fails to start due to Node version mismatch: `nvm use $(cat frontend/.nvmrc)` or `fnm use`.
- Runtime component imports SDK-only modules: `_KFP_RUNTIME=true` disables many SDK imports; avoid importing SDK-only modules in task code.
