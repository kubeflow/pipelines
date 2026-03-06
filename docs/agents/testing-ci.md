# KFP Testing and CI/CD

Local testing commands, Ginkgo suites, test data iteration mechanics, proto change test coverage requirements, CI/CD matrices, cluster setup, and linting/code style.

**See also:** [protobuf-build.md](protobuf-build.md) (proto generation commands), [backend-guide.md](backend-guide.md) (local dev setup, cluster deployment), [sdk-guide.md](sdk-guide.md) (SDK details)

---

## Local testing

### Python (SDK)

```bash
pip install -r sdk/python/requirements-dev.txt
pytest -v sdk/python/kfp
```

### Python (kfp-kubernetes)

```bash
pytest -v kubernetes_platform/python/test
```

### Go (backend) unit tests

Excludes integration/API/Compiler/E2E tests:

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

## Backend Ginkgo test suites

### Compiler tests

```bash
# Run compiler tests
ginkgo -v ./backend/test/compiler

# Update compiled workflow goldens when intended
ginkgo -v ./backend/test/compiler -- -updateCompiledFiles=true

# Auto-create missing goldens (default true); disable with:
ginkgo -v ./backend/test/compiler -- -createGoldenFiles=false
```

### v2 API integration tests (label-filterable)

```bash
# All API tests
ginkgo -v ./backend/test/v2/api

# Example: run only Smoke-labeled tests with ginkgo
ginkgo -v --label-filter="Smoke" ./backend/test/v2/api
```

### End-to-end tests

```bash
ginkgo -v ./backend/test/end2end -- -namespace=kubeflow -isDebugMode=true
```

### Test data locations

- `test_data/sdk_compiled_pipelines/valid/` (inputs) with a `valid/critical/` subset for smoke lanes
- `test_data/compiled-workflows/` (expected compiled Argo Workflows)

## Test data iteration mechanics

Each test suite discovers and iterates over test files differently:

### Compiler tests

- **Discovery function**: `GetListOfAllFilesInDir()` (recursive)
- **Input directory**: `test_data/sdk_compiled_pipelines/valid/`
- **Process**: Compiles each `.yaml` pipeline spec -> compares against `test_data/compiled-workflows/{name}.yaml`
- **File filters**: Discovery functions exclude `.py`, `Dockerfile`, `.md`, `.ipynb` -- only `.yaml` and `.zip` are tested

### E2E tests

- **Discovery function**: `GetListOfFilesInADir()` (non-recursive)
- **Directories and labels**:
  - `valid/critical/` -> `Label: E2eCritical`
  - `valid/essential/` -> `Label: E2eEssential`
  - `valid/failing/` -> `Label: E2eFailed`
  - `valid/integration/` -> `Label: Integration`

### Smoke/Sanity tests

- Hardcoded subset of files from `essential/` and `critical/`

### API tests

- **Upload tests**: `GetListOfAllFilesInDir()` (recursive) on `valid/`
- **Negative tests**: `GetListOfFilesInADir()` on `invalid/`

### SDK Python tests

- Explicit test functions referencing specific `.py` files and comparing to `.yaml` goldens

### Skip logic

- Files with `_GH-{issue}` in name are auto-skipped

## Proto changes require test coverage

When any `.proto` file is modified (especially `api/v2alpha1/pipeline_spec.proto` or `kubernetes_platform/proto/kubernetes_executor_config.proto`), you **must** add or update test files that exercise the changed or new proto fields. The test system uses a three-file pipeline that must stay in sync:

```
1. Python pipeline source  (.py)  ->  test_data/sdk_compiled_pipelines/valid/{critical,essential}/
2. Compiled pipeline spec  (.yaml) ->  test_data/sdk_compiled_pipelines/valid/{critical,essential}/
3. Compiled Argo Workflow  (.yaml) ->  test_data/compiled-workflows/
```

### Step-by-step process

1. **Write a Python pipeline** that uses the new or changed proto field. Place it in:
   - `test_data/sdk_compiled_pipelines/valid/critical/` -- for features that are functionally important (e.g., new pipeline features, artifact handling, parameter types, pod metadata, caching behavior)
   - `test_data/sdk_compiled_pipelines/valid/essential/` -- for foundational features that validate core KFP functionality (e.g., basic component composition, pip installs, pipeline nesting, conditionals, loops)

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

### How the test framework uses these files

| Test suite | Input directory | What it does |
|------------|----------------|--------------|
| Compiler tests (`backend/test/compiler`) | `test_data/sdk_compiled_pipelines/valid/**/*.yaml` -> `test_data/compiled-workflows/*.yaml` | Compiles every pipeline spec to an Argo Workflow and compares against the golden YAML |
| E2E essential (`Label(E2eEssential)`) | `test_data/sdk_compiled_pipelines/valid/essential/` | Uploads and runs every pipeline in `essential/` on a live cluster |
| E2E critical (`Label(E2eCritical)`) | `test_data/sdk_compiled_pipelines/valid/critical/` | Uploads and runs every pipeline in `critical/` on a live cluster |
| Smoke tests (`Label(Smoke)`) | Hand-picked files from `essential/` and `critical/` | Runs a small subset for fast validation |

### Naming conventions

- The `.py` and `.yaml` files in `critical/` or `essential/` share the same base name: `my_pipeline.py` <-> `my_pipeline.yaml`
- The Argo Workflow golden in `compiled-workflows/` also uses the same base name: `my_pipeline.yaml`
- Exception: some files use `_compiled` suffix (e.g., `iris_pipeline.py` -> `iris_pipeline_compiled.yaml`)

### When NOT to add test files

- Pure documentation or comment changes in `.proto` files that don't alter the wire format
- Changes to deprecated v1beta1 protos (unless explicitly maintaining v1 compatibility)

## CI/CD (GitHub Actions)

- Workflows: `.github/workflows/` (build, test, lint, release)
- Composite actions: `.github/actions/` (e.g., `kfp-k8s`, `create-cluster`, `deploy`, `test-and-report`)
- Typical checks: Go unit tests (backend), Python SDK tests, frontend tests/lint, image builds.

### Test matrices and variants

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

## Test quality guidelines

For comprehensive test quality, naming conventions, fixture isolation, and proportional coverage rules, see [test-quality.md](../../.github/review-guides/test-quality.md).

## Lint and formatting checks

### Go lint

```bash
golangci-lint run
```

### Python SDK

```bash
pip install -r sdk/python/requirements-dev.txt
pycln --check sdk/python
isort --check --profile google sdk/python
```

### Python SDK formatting (YAPF + string fixer)

```bash
pip install yapf pre_commit_hooks
python3 -m pre_commit_hooks.string_fixer $(find sdk/python/kfp/**/*.py -type f)
yapf --recursive --diff sdk/python/
```

### Python SDK docstring formatting

```bash
pip install docformatter
docformatter --check --recursive sdk/python/ --exclude "compiler_test.py"
```

Notes:

- Legacy `kfp-samples.yml` and `periodic.yml` workflows were removed.
