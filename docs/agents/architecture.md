# Architecture and Runtime

## Architecture

- Diagram: [`images/kfp-cluster-wide-architecture.png`](../../images/kfp-cluster-wide-architecture.png).
- This fork captures DSL tasks/groups/channels before IR construction and compiles them directly to native Argo resources. See `plan.md` for the migration checkpoint and `samples/argo-native/README.md` for the executable prototype.
- Argo owns scheduling, input resolution, artifact transfer, memoization and status. `argo_runtime.py` invokes unchanged Python functions inside component containers; no KFP driver, launcher service or API server is required by this path.
- Legacy backend/IR code remains in the monorepo during migration but is not the native prototype's execution model.
- The executor entrypoint is `sdk/python/kfp/dsl/executor_main.py`; it does not participate in compilation.
- Runtime containers install `kfp` with `--no-deps`; `_KFP_RUNTIME=true` disables most SDK imports. Task code must not depend on SDK-only modules unless the base image supplies their dependencies.

## Packages and key paths

| Area | Path |
| --- | --- |
| Native SDK compiler | `sdk/python/kfp/compiler/argo_compiler.py`, `argo_groups.py` |
| DSL | `sdk/python/kfp/dsl/` |
| Platform integration | `sdk/python/kfp/kubernetes/` |
| Pipeline-spec APIs | `api/` |
| Backend | `backend/` |
| Frontend | `frontend/` |
| Deployment manifests | `manifests/` |
| Pipeline fixtures and workflow goldens | `test_data/` |

- `kfp` is one distribution containing `kfp.pipeline_spec`, `kfp.kubernetes`,
  and the generated REST client `kfp.server_api`. It no longer extends a shared
  namespace with separately installed distributions.
- Kubernetes bindings rewrite generated pipeline-spec imports through
  `kubernetes_platform/python/generate_proto.py`; proto and Go sources stay in
  their domain directories.

## Execution boundary

Components are called inside `@dsl.pipeline` to capture tasks. Submit compiled
resources to Argo on Kubernetes; there is no in-process runner or compatibility
fallback. Offline compilation, validation and runtime unit tests remain supported.
Docker's Python package remains necessary for `kfp component build`, not execution.
