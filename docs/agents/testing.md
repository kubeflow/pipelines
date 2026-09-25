# Testing and Formatting

## Targeted test commands

```bash
# SDK
uv sync --frozen --extra dev
uv run pytest -v sdk/python/kfp

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
