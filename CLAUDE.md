# Kubeflow Pipelines

ML pipeline orchestration platform. Users define pipelines in Python (SDK), compile to YAML, and submit to a Kubernetes-based backend that schedules, caches, and tracks runs via MLMD.

## Tech stack

- **Backend**: Go (API server, persistence, scheduling) — `backend/`
- **SDK**: Python (DSL, compiler, client) — `sdk/python/`
- **Frontend**: React 18, MUI v5 — `frontend/`
- **Protos**: gRPC/protobuf API definitions — `api/`, `kubernetes_platform/`

## Key directories

| Directory | Purpose |
|-----------|---------|
| `backend/src/apiserver/` | Go API server (REST + gRPC) |
| `sdk/python/kfp/` | Python SDK: compiler, components, client |
| `frontend/src/` | React UI |
| `api/v2alpha1/` | Proto definitions for pipeline IR |
| `kubernetes_platform/` | K8s-specific platform proto + Python package |
| `test_data/` | Golden test files and sample pipelines |
| `tools/code-tree/` | Codebase analysis tool for impact/dependency tracing |

## Essential commands

| Task | Command |
|------|---------|
| Compile pipeline | `kfp dsl compile --py pipeline.py --output pipeline.yaml` |
| SDK tests | `pytest -v sdk/python/kfp` |
| K8s platform tests | `pytest -v kubernetes_platform/python/test` |
| Backend unit tests | `go test -v $(go list ./backend/... \| grep -v backend/test)` |
| Compiler tests | `ginkgo -v ./backend/test/compiler` |
| Frontend dev | `cd frontend && npm ci && npm start` |
| Frontend tests | `cd frontend && npm run test:ci` |
| Generate protos | `make -C api python && make -C api golang` |
| Generate backend API | `make -C backend/api generate` |
| Python lint | `yapf --recursive --diff sdk/python/ && pycln --check sdk/python && isort --check --profile google sdk/python` |
| Go lint | `golangci-lint run` |
| Ginkgo install | `make ginkgo && export PATH="$PWD/bin:$PATH"` |

## Commit policy

- Sign off all commits: `git commit -s`
- Do not include AI agents as co-authors

## Agent workflows

See @AGENTS.md for task routing, PR review workflow, reference docs, and validation checklist.