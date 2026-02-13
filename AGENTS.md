# Agent Guide: Kubeflow Pipelines (KFP) Monorepo

Entry point for AI agents and developers. Load only the reference file relevant to your current task.

- Last updated: 2026-03-13
- Scope: KFP master branch (v2 engine), backend (Go), SDK (Python), frontend (React 17, MUI v5)

### Code reuse policy (agents and contributors)

- Always reuse existing functions, helpers, and utilities before writing new code. Search the codebase for existing implementations that accomplish the same goal.
- Do not duplicate logic that already exists elsewhere in the repo. If a function, method, or pattern is already implemented, import and call it rather than reimplementing it.
- When adding new functionality, check related packages and modules for shared code that can be leveraged.
- If existing code needs slight modifications to be reusable, prefer refactoring the existing code to be more general over duplicating it with changes.
- Use descriptive variable and function names. Avoid abbreviations or single-letter names — prefer full, meaningful names that clearly convey purpose (e.g., `executionID` over `execID`, `fingerPrint` over `fp`).

### Testing policy (agents and contributors)

- Every new non-trivial function, method, or exported API must have accompanying unit tests before merging. Trivial helpers and glue code may be excluded when testing adds no meaningful value.
- All existing tests must pass locally before pushing changes. Run the relevant test suites listed in the essential commands section.
- When modifying existing functions, verify that existing tests still pass and add new test cases if the behavior changes.
- Do not submit changes that break existing tests. If a test failure is pre-existing and unrelated to your changes, note it explicitly in the PR description.

### Commit policy (agents and contributors)

- Always sign off on commits with `git commit -s` (adds a `Signed-off-by:` trailer).
- Never include AI agents (e.g. Claude Code, Copilot, or similar tools) as co-authors on commits. The human author is responsible for the work.

### Maintenance

- Update the relevant file under `docs/agents/` when changing commands, paths, Make targets, env vars, or workflows.
- Update [protobuf-build.md](docs/agents/protobuf-build.md) when changing generated files.
- Update [testing-ci.md](docs/agents/testing-ci.md) when changing CI matrices or workflows.
- Extend troubleshooting in the relevant guide for new common errors. Bump the date above.

## Reference files

| File | Load when |
|------|-----------|
| [architecture.md](docs/agents/architecture.md) | Understanding E2E flow, caching, MLMD, object storage, or cross-subsystem changes |
| [sdk-guide.md](docs/agents/sdk-guide.md) | Working on `sdk/python/`, `kubernetes_platform/python/`, Python components |
| [backend-guide.md](docs/agents/backend-guide.md) | Working on `backend/`, Go code, API server, local dev setup |
| [protobuf-build.md](docs/agents/protobuf-build.md) | Modifying `.proto` files, regenerating code |
| [frontend-guide.md](docs/agents/frontend-guide.md) | Working on `frontend/`, UI components |
| [testing-ci.md](docs/agents/testing-ci.md) | Running tests, adding coverage, CI changes |
| [code-tree.md](docs/agents/code-tree.md) | **Development workflow**, codebase exploration, impact analysis, PR reviews |
| [security-code-reviewer.md](https://github.com/anthropics/claude-code-action/blob/main/.claude/agents/security-code-reviewer.md) | Security scan -- OWASP Top 10, input validation, auth/authz |

PR review checklists: see [.github/copilot-instructions.md](.github/copilot-instructions.md) and files under `.github/review-guides/`.

## Task routing

| Task type | Load | Then |
|-----------|------|------|
| Bug fix or feature in `backend/` | [backend-guide.md](docs/agents/backend-guide.md) | Run code-tree `--rdeps` on changed files before editing |
| Bug fix or feature in `sdk/python/` or `kubernetes_platform/` | [sdk-guide.md](docs/agents/sdk-guide.md) | Check compilation flow if touching compiler/DSL |
| Bug fix or feature in `frontend/` | [frontend-guide.md](docs/agents/frontend-guide.md) | Match Node version to `.nvmrc` first |
| Proto schema change | [protobuf-build.md](docs/agents/protobuf-build.md) | Regenerate ALL targets (Go + Python + frontend) |
| Understanding E2E flow, caching, MLMD, object storage | [architecture.md](docs/agents/architecture.md) | Start with architecture diagram |
| Writing or running tests, CI changes | [testing-ci.md](docs/agents/testing-ci.md) | Match test type to correct suite |
| Codebase exploration, impact analysis, dependency tracing | [code-tree.md](docs/agents/code-tree.md) | Follow development workflow, use `query_graph.py` before reading source |
| PR review | [.github/copilot-instructions.md](.github/copilot-instructions.md) | Follow batch review strategy |
| Security scan | [security-code-reviewer.md](https://github.com/anthropics/claude-code-action/blob/main/.claude/agents/security-code-reviewer.md) | Run independently |
| Cross-subsystem change | [architecture.md](docs/agents/architecture.md) + relevant subsystem guide | Verify dependency direction rules |

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

After completing, self-review: switch perspective ("as a reviewer, what would I flag?"), check each item above, fix issues found. Maximum 2 self-review iterations.