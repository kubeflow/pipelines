# Testing and Formatting

## Targeted test commands

```bash
# SDK
uv sync --frozen --extra dev
uv run pytest -v sdk/python/kfp

# Complete SDK unit CI selection, including cross-file parallelism
SETUP_ENV=false PYTEST_PARALLEL_WORKERS=4 \
  ./test/presubmit-tests-sdk-unit.sh

# Bundled Kubernetes helpers
uv run pytest -v sdk/python/test/kubernetes

# Backend unit tests
go test -v $(go list ./backend/... | \
  grep -v backend/test/v2/api | \
  grep -v backend/test/v2/integration | \
  grep -v backend/test/v2/initialization | \
  grep -v backend/test/compiler | \
  grep -v backend/test/end2end)

# Compiler, API, and end-to-end suites
ginkgo -v ./backend/test/compiler
ginkgo -v --label-filter="Smoke" ./backend/test/v2/api
ginkgo -v --label-filter="Smoke" ./backend/test/end2end -- -namespace=kubeflow
```

Compiler and API/E2E suites require Ginkgo; API and E2E tests require a cluster. Use a label filter on CPU-only clusters because `gpu-scheduling-check` requires `nvidia.com/gpu`.

Runtime command regressions require a wheel built with `uv build --package kfp --wheel`. Set `KFP_PACKAGE_PATH` to its absolute path, then run `uv run pytest sdk/python/test/runtime -m regression`. Each case installs the wheel with the stored `--no-deps` bootstrap in a clean uv venv using the test runner's Python version; missing artifacts fail instead of selecting a published SDK.

Pipeline inputs live in `test_data/pipeline_files/valid/`; compiler goldens live in `test_data/compiled-workflows/`.

## Formatting and linting

```bash
golangci-lint run
bash test/presubmit-isort-sdk.sh
bash test/presubmit-yapf-sdk.sh
bash test/presubmit-docformatter-sdk.sh
```

These SDK scripts exclude generated protobuf and OpenAPI modules. The YAPF
script also normalizes Python string quotes before checking formatting.

Run the complete SDK unit selection with both Python 3.11 and 3.13 after
changing packaging or local-runner test setup; a runtime-only or serial subset
does not exercise cross-worker installation races. `LocalRunnerEnvironmentTestCase`
isolates both the SDK build source and the subprocess installation destination.
Even `use_venv=False` tests use a disposable per-test interpreter, while retaining
the runner's actual install locking and per-task-venv behavior. Never let test
components reinstall packages into the pytest workers' shared environment.
