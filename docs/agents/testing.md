# Testing and Formatting

## Targeted test commands

```bash
# SDK
pip install -r sdk/python/requirements-dev.txt
pytest -v sdk/python/kfp

# kfp-kubernetes
pytest -v kubernetes_platform/python/test

# Backend unit tests
go test -v $(go list ./backend/... | \
  grep -v backend/test/v2/api | \
  grep -v backend/test/integration | \
  grep -v backend/test/v2/integration | \
  grep -v backend/test/initialization | \
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
pycln --check sdk/python
isort --check --profile google sdk/python
yapf --recursive --diff sdk/python/
docformatter --check --recursive sdk/python/ --exclude "compiler_test.py"
```

Run the Python string fixer before YAPF when needed:

```bash
python3 -m pre_commit_hooks.string_fixer $(find sdk/python/kfp -name '*.py' -type f)
```
