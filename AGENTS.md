# Agent Guide: Kubeflow Pipelines (KFP) Monorepo

Operational playbook for AI agents working in this repo. For onboarding (what/why/how), see [CLAUDE.md](CLAUDE.md).

- Last updated: 2026-03-24
- Scope: KFP master branch (v2 engine)

## Reference files

| File | Load when |
|------|-----------|
| [architecture.md](docs/agents/architecture.md) | Understanding E2E flow, caching, MLMD, object storage, or cross-subsystem changes |
| [sdk-guide.md](docs/agents/sdk-guide.md) | Working on `sdk/python/`, `kubernetes_platform/python/`, Python components |
| [backend-guide.md](docs/agents/backend-guide.md) | Working on `backend/`, Go code, API server, local dev setup |
| [protobuf-build.md](docs/agents/protobuf-build.md) | Modifying `.proto` files, regenerating code |
| [frontend-guide.md](docs/agents/frontend-guide.md) | Working on `frontend/`, UI components |
| [testing-ci.md](docs/agents/testing-ci.md) | Running tests, adding coverage, CI changes |
| [code-tree.md](docs/agents/code-tree.md) | Codebase exploration, impact analysis, PR reviews |
| [security-code-reviewer.md](https://github.com/anthropics/claude-code-action/blob/main/.claude/agents/security-code-reviewer.md) | Security scan — OWASP Top 10, input validation, auth/authz |

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
| Codebase exploration, impact analysis, dependency tracing | [code-tree.md](docs/agents/code-tree.md) | Use `query_graph.py` before reading source |
| PR review | [.github/copilot-instructions.md](.github/copilot-instructions.md) + [code-tree.md](docs/agents/code-tree.md) | Run code-tree `--rdeps` on changed files first |
| Security scan | [security-code-reviewer.md](https://github.com/anthropics/claude-code-action/blob/main/.claude/agents/security-code-reviewer.md) | Run independently |
| Cross-subsystem change | [architecture.md](docs/agents/architecture.md) + relevant subsystem guide | Verify dependency direction rules |

## PR review workflow

When reviewing a PR, follow this order. Detailed batch strategy is in `.github/copilot-instructions.md`.

1. **Code-tree first** (before reading any diffs):
   ```bash
   python3 tools/code-tree/code_tree.py --repo-root . --incremental -q
   python3 tools/code-tree/query_graph.py --rdeps <changed-file>
   python3 tools/code-tree/query_graph.py --test-impact <changed-file>
   python3 tools/code-tree/query_graph.py --callers <changed-function>
   ```
   Run for every changed file. Identifies untested blast radius, missing caller updates, affected tests.

2. **Read diffs in batches**: Proto/API → Backend Go → SDK Python → Tests → Manifests/CI.

3. **Write comments**: Combine code-tree findings with diff analysis. Flag untested callers, missing test coverage, blast radius.

4. **Backend endpoint/query checks** (for API/storage PRs): Use [PR #12999](https://github.com/kubeflow/pipelines/pull/12999) as reference for endpoint completeness, test coverage, SQL/query construction, and cache correctness.

## Validation checklist

Before completing any task, verify what applies:

- [ ] Changed files pass relevant tests (SDK: pytest, Backend: go test, Frontend: npm test)
- [ ] Changed files pass relevant linters
- [ ] No generated files hand-edited
- [ ] Proto changes: all language targets regenerated, test pipeline added in `test_data/sdk_compiled_pipelines/valid/`
- [ ] New Go `err` variables checked or returned
- [ ] New Python functions have type hints and docstrings
- [ ] `kfp` runtime path tested: no SDK-only imports in task-executed code
- [ ] Cross-boundary changes: dependency direction rules respected

## Maintenance

- Update the relevant file under `docs/agents/` when changing commands, paths, Make targets, env vars, or workflows.
- Update [protobuf-build.md](docs/agents/protobuf-build.md) when changing generated files.
- Update [testing-ci.md](docs/agents/testing-ci.md) when changing CI matrices or workflows.
- Extend troubleshooting in the relevant guide for new common errors. Bump the date above.