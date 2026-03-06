# Agent Guide: Kubeflow Pipelines (KFP) Monorepo

Entry point for AI agents and developers. Load only the reference file relevant to your current task.

- Last updated: 2026-03-05
- Scope: KFP master branch (v2 engine), backend (Go), SDK (Python), frontend (React 17, MUI v5)

### Commit policy (agents and contributors)

- Always sign off on commits with `git commit -s` (adds a `Signed-off-by:` trailer).
- Never include AI agents (e.g. Claude Code, Copilot, or similar tools) as co-authors on commits. The human author is responsible for the work.

### Maintenance

- Update the relevant file under `docs/agents/` when changing commands, paths, Make targets, env vars, or workflows.
- Update [protobuf-build.md](docs/agents/protobuf-build.md) when changing generated files.
- Update [testing-ci.md](docs/agents/testing-ci.md) when changing CI matrices or workflows.
- Extend troubleshooting below for new common errors. Bump the date above.

## Reference files

| File | Load when                                                                                                 |
|------|-----------------------------------------------------------------------------------------------------------|
| [architecture.md](docs/agents/architecture.md) | Understanding E2E flow, data flow, execution, object storage, caching, MLMD                               |
| [sdk-guide.md](docs/agents/sdk-guide.md) | Working on `sdk/python/`, `kubernetes_platform/python/`, Python components                                |
| [backend-guide.md](docs/agents/backend-guide.md) | Working on `backend/`, Go code, API server, local dev setup                                               |
| [protobuf-build.md](docs/agents/protobuf-build.md) | Modifying `.proto` files, regenerating code                                                               |
| [frontend-guide.md](docs/agents/frontend-guide.md) | Working on `frontend/`, UI components                                                                     |
| [testing-ci.md](docs/agents/testing-ci.md) | Running tests, adding coverage, CI changes                                                                |
| [code-tree.md](docs/agents/code-tree.md) | Exploring codebase, impact analysis, PR reviews                                                           |
| [security-code-reviewer.md](https://github.com/anthropics/claude-code-action/blob/main/.claude/agents/security-code-reviewer.md) | Security scan -- OWASP Top 10, input validation, auth/authz. Run independently, no knowledge graph needed |

PR review checklists: see [.github/copilot-instructions.md](.github/copilot-instructions.md) and files under `.github/review-guides/`.

## Baseline architecture

Diagram: `images/kfp-cluster-wide-architecture.png` ([source](images/kfp-cluster-wide-architecture.drawio.xml))

## Essential commands

- Compile pipeline: `kfp dsl compile --py pipeline.py --output pipeline.yaml`
- Generate protos: `make -C api python && make -C api golang`
- Generate backend API: `make -C backend/api generate`
- Deploy local cluster: `make -C backend kind-cluster-agnostic` (standalone) / `make -C backend dev-kind-cluster` (dev)
- Run SDK tests: `pytest -v sdk/python/kfp`
- Run backend unit tests: `go test -v $(go list ./backend/... | grep -v backend/test/)`
- Run compiler tests: `ginkgo -v ./backend/test/compiler`
- Run API tests: `ginkgo -v --label-filter="Smoke" ./backend/test/v2/api`
- Run E2E tests: `ginkgo -v ./backend/test/end2end -- -namespace=kubeflow`
- Check formatting: `yapf --recursive --diff sdk/python/ && pycln --check sdk/python && isort --check --profile google sdk/python`
- Frontend: `cd frontend && npm start` / `npm run test:ui` / `npm run format` / `npm run apis`

## Key environment variables

- `_KFP_RUNTIME=true`: Disables SDK imports during task execution
- `VITE_NAMESPACE=...`: Frontend namespace for multi-user mode
- `LOCAL_API_SERVER=true`: Local API server testing mode on Kind cluster

## Troubleshooting

- `_KFP_RUNTIME=true` disables SDK; don't import SDK-only modules from task code
- `kfp` installed with `--no-deps` in containers; ensure runtime deps in `base_image`
- SELinux can break proto gen; `sudo setenforce 0` temporarily
- `pipeline_spec_pb2.py` must be generated, not assumed to exist
- Frontend needs `swagger-codegen-cli.jar`, `protoc`, `protoc-gen-grpc-web` in PATH
- Node version must match `.nvmrc`; use nvm/fnm
- Port conflicts: 3000 (Vite), 3001 (server), 3002 (proxy), 6006 (Storybook)
- "protoc: command not found": use Make targets (run in container)
- "ginkgo: command not found": `make ginkgo` or `GOBIN=$PWD/bin go install github.com/onsi/ginkgo/v2/ginkgo@latest`
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

