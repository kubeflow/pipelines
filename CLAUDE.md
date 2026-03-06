# CLAUDE.md - Kubeflow Pipelines (KFP) Monorepo

Development and review agent for the KFP monorepo. This file is loaded every session -- keep it lean. Load detailed guides on demand based on the task.

## Task routing

Identify the task type, load only the relevant guide, and follow its workflow. Never load all guides at once.

| Task type                                                     | Load | Then |
|---------------------------------------------------------------|------|------|
| Bug fix or feature in `backend/`                              | [backend-guide.md](docs/agents/backend-guide.md) | Run code-tree `--rdeps` on changed files before editing |
| Bug fix or feature in `sdk/python/` or `kubernetes_platform/` | [sdk-guide.md](docs/agents/sdk-guide.md) | Check compilation flow if touching compiler/DSL |
| Bug fix or feature in `frontend/`                             | [frontend-guide.md](docs/agents/frontend-guide.md) | Match Node version to `.nvmrc` first |
| Proto schema change                                           | [protobuf-build.md](docs/agents/protobuf-build.md) | Regenerate ALL targets (Go + Python + frontend) |
| Understanding E2E flow, caching, MLMD, object storage         | [architecture.md](docs/agents/architecture.md) | Start with architecture diagram |
| Writing or running tests, CI changes                          | [testing-ci.md](docs/agents/testing-ci.md) | Match test type to correct suite |
| Codebase exploration, impact analysis, dependency tracing     | [code-tree.md](docs/agents/code-tree.md) | Use `query_graph.py` before reading source |
| PR review                                                     | [.github/copilot-instructions.md](.github/copilot-instructions.md) | Follow batch review strategy |
| Security scan                                                 | [security-code-reviewer.md](https://github.com/anthropics/claude-code-action/blob/main/.claude/agents/security-code-reviewer.md) | Run independently -- OWASP Top 10, input validation, auth/authz. No knowledge graph needed |
| Cross-subsystem change                                        | [architecture.md](docs/agents/architecture.md) + relevant subsystem guide | Verify dependency direction rules |

## Development strategy

### Before writing code

1. **Locate**: Run `python3 tools/code-tree/query_graph.py --symbol <name>` or `--search <keyword>` to find relevant files
2. **Scope**: Run `--rdeps <file>` to understand blast radius before changing anything
3. **Read**: Load the relevant guide from the table above, then read the target files

### While writing code

4. **Guard generated files**: Never hand-edit files listed in the generated files section below. Edit sources and regenerate.
5. **Guard hub files**: Extra care when touching high-fanout files (listed below). Run `--impact <file> --depth 5` first.
6. **Match patterns**: Follow existing code style in the file you're editing. Go: error wrapping with `%w`, no nested goroutines. Python: type hints, docstrings, YAPF formatting.
7. **Cross-boundary changes**: If your change spans SDK + backend, or backend + proto, verify dependency direction rules in [architecture-context.md](.github/review-guides/architecture-context.md#critical-dependency-directions).

### After writing code

8. **Run relevant tests**: Use the commands table below -- run only what applies to your change
9. **Run impact check**: `python3 tools/code-tree/query_graph.py --test-impact <changed-file>` to find tests you may have missed
10. **Lint**: Run the appropriate formatter/linter for the language you changed

## Essential commands

| Task | Command |
|------|---------|
| Compile pipeline | `kfp dsl compile --py pipeline.py --output pipeline.yaml` |
| Generate protos (api) | `make -C api python && make -C api golang` |
| Generate K8s platform protos | `make -C kubernetes_platform python && make -C kubernetes_platform golang` |
| Generate backend API | `make -C backend/api generate` |
| Deploy local cluster | `make -C backend kind-cluster-agnostic` |
| SDK tests | `pytest -v sdk/python/kfp` |
| K8s platform tests | `pytest -v kubernetes_platform/python/test` |
| Backend unit tests | `go test -v $(go list ./backend/... \| grep -v backend/test/)` |
| Compiler tests | `ginkgo -v ./backend/test/compiler` |
| Update compiler goldens | `ginkgo -v ./backend/test/compiler -- -updateCompiledFiles=true` |
| API integration tests | `ginkgo -v --label-filter="Smoke" ./backend/test/v2/api` |
| E2E tests | `ginkgo -v ./backend/test/end2end -- -namespace=kubeflow` |
| Python lint | `yapf --recursive --diff sdk/python/ && pycln --check sdk/python && isort --check --profile google sdk/python` |
| Go lint | `golangci-lint run` |
| Frontend dev | `cd frontend && npm ci && npm start` |
| Frontend tests | `cd frontend && npm run test:ui` |
| Frontend CI check | `cd frontend && npm run test:ci` |
| Ginkgo install | `make ginkgo && export PATH="$PWD/bin:$PATH"` |

## Code-tree queries

Pre-built knowledge graph (24,923 nodes, 87,692 edges) for navigation without reading source files. Use **before** opening files to minimize context.

```bash
python3 tools/code-tree/query_graph.py --symbol Pipeline          # find definition
python3 tools/code-tree/query_graph.py --search "compiler"         # keyword search
python3 tools/code-tree/query_graph.py --rdeps <file>              # what imports this (blast radius)
python3 tools/code-tree/query_graph.py --deps <file>               # what this imports
python3 tools/code-tree/query_graph.py --impact <file> --depth 5   # transitive impact
python3 tools/code-tree/query_graph.py --test-impact <file>        # affected tests
python3 tools/code-tree/query_graph.py --callers <function>        # all call sites
python3 tools/code-tree/query_graph.py --callees <function>        # what it calls
python3 tools/code-tree/query_graph.py --hierarchy <class>         # inheritance tree
python3 tools/code-tree/query_graph.py --module <dir>              # module overview
python3 tools/code-tree/query_graph.py --call-chain <from> <to>    # path between symbols
python3 tools/code-tree/query_graph.py --stats                     # graph statistics
```

Add `--json` for machine-readable output. Regenerate after major changes: `python3 tools/code-tree/code_tree.py --repo-root .`

## Generated files -- never edit directly

| Generated file | Source | Regenerate |
|----------------|--------|------------|
| `api/v2alpha1/python/kfp/pipeline_spec/pipeline_spec_pb2.py` | `api/v2alpha1/pipeline_spec.proto` | `make -C api python` |
| `api/v2alpha1/go/pipelinespec/pipeline_spec.pb.go` | `api/v2alpha1/pipeline_spec.proto` | `make -C api golang` |
| `kubernetes_platform/python/kfp/kubernetes/kubernetes_executor_config_pb2.py` | `kubernetes_platform/proto/*.proto` | `make -C kubernetes_platform python` |
| `kubernetes_platform/go/kubernetesplatform/*.pb.go` | `kubernetes_platform/proto/*.proto` | `make -C kubernetes_platform golang` |
| `backend/api/v2beta1/go_client/**/*.pb.go` | `backend/api/v2beta1/*.proto` | `make -C backend/api generate` |
| `backend/api/v2beta1/swagger/*.swagger.json` | `backend/api/v2beta1/*.proto` | `make -C backend/api generate` |
| `frontend/src/apis/`, `frontend/src/apisv2beta1/` | Backend swagger specs | `cd frontend && npm run apis && npm run apis:v2beta1` |

## Hub files -- highest blast radius

Changes here cascade widely. Always run `--impact` and `--test-impact` before modifying:

| File | Dependents |
|------|------------|
| `backend/src/common/util/json.go` | 659 |
| `backend/src/common/util/time.go` | 470 |
| `backend/src/common/util/string.go` | 303 |
| `backend/src/common/util/error.go` | 284 |
| `backend/src/common/util/uuid.go` | 270 |
| `backend/src/common/util/consts.go` | 259 |
| `backend/src/apiserver/common/utils.go` | 178 |
| `backend/api/v*beta*/python_http_client/kfp_server_api/rest.py` | 95 each |

## Key environment variables

| Variable | Purpose |
|----------|---------|
| `_KFP_RUNTIME=true` | Disables SDK imports during task execution. Don't import SDK-only modules from task code. |
| `VITE_NAMESPACE=...` | Frontend namespace for multi-user mode |
| `LOCAL_API_SERVER=true` | Local API server testing mode on Kind cluster |

## Architecture (minimal)

```
SDK (Python) -> compile -> IR YAML (PipelineSpec proto)
  -> API Server (Go) -> argocompiler.Compile() -> Argo Workflow YAML
    -> Driver (Go init container) -> resolve inputs, check cache
      -> Launcher (Go) -> download artifacts -> invoke user container
        -> Executor (Python) -> component function -> output artifacts
```

Entry points: API Server (`backend/src/apiserver/main.go`), Driver (`backend/src/v2/cmd/driver/main.go`), Compiler (`backend/src/v2/cmd/compiler/main.go`), Launcher (`backend/src/v2/cmd/launcher-v2/main.go`), Cache (`backend/src/cache/main.go`), Executor (`sdk/python/kfp/dsl/executor_main.py`)

Architecture diagram: `images/kfp-cluster-wide-architecture.png`

## Dependency direction rules

Violations of these are bugs -- flag or fix immediately:

```
sdk/python/kfp/compiler  -> sdk/python/kfp/dsl       (never reverse)
sdk/python/kfp/client    -> sdk/python/kfp/dsl       (never reverse)
backend/src/v2/*         -> never imports backend/src/apiserver/
backend/src/crd/controller/* -> backend/src/crd/pkg only, never apiserver/
```

## Validation checklist

Before completing any task, verify what applies:

- [ ] Changed files pass relevant tests (SDK: pytest, Backend: go test, Frontend: npm test)
- [ ] Changed files pass relevant linters (Python: yapf/pycln/isort, Go: golangci-lint, Frontend: prettier/eslint)
- [ ] No generated files hand-edited
- [ ] Proto changes: all language targets regenerated, test pipeline added in `test_data/sdk_compiled_pipelines/valid/`
- [ ] Hub file changes: impact analysis run, affected tests identified
- [ ] Cross-boundary changes: dependency direction rules respected
- [ ] New Go `err` variables checked or returned
- [ ] New Python functions have type hints and docstrings
- [ ] `kfp` runtime path tested: no SDK-only imports in task-executed code

## Troubleshooting

- `kfp` installed with `--no-deps` in containers; ensure runtime deps in `base_image`
- SELinux can break proto gen; `sudo setenforce 0` temporarily
- Frontend needs `swagger-codegen-cli.jar`, `protoc`, `protoc-gen-grpc-web` in PATH
- Node version must match `frontend/.nvmrc`; use nvm/fnm
- Port conflicts: 3000 (Vite), 3001 (server), 3002 (proxy), 6006 (Storybook)
- `ginkgo: command not found`: `make ginkgo` or `GOBIN=$PWD/bin go install github.com/onsi/ginkgo/v2/ginkgo@latest`
- `pipeline_spec_pb2.py` must be generated, not assumed to exist
- Object store failures: verify MinIO running in Kind cluster, check `OBJECTSTORECONFIG_*` env vars

## Self-Review Process

1. Complete implementation
2. Switch perspective: "As a reviewer, what would I flag?"
3. Check each item in checklist above
4. Fix any issues found
5. If significant fix made, note: "Self-review: Fixed [issue]"
6. Maximum 2 self-review iterations to prevent infinite loops

## Example Self-Review Note

Self-review: Fixed edge case for null input in validate_slug()
- Original code would throw NullPointerException
- Added early return for null/empty values
