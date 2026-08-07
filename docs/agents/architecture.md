# Architecture and Runtime

## Architecture

- Diagram: [`images/kfp-cluster-wide-architecture.png`](../../images/kfp-cluster-wide-architecture.png).
- The SDK compiles Python DSL into the pipeline-spec IR. The API server compiles that IR to Argo Workflows and runs it on Kubernetes.
- The driver resolves inputs and derives pod resource patches. Other Kubernetes configuration belongs in `kubernetes_platform`.
- The launcher transfers artifacts and invokes the Python executor. Local Subprocess and Docker runners skip the launcher.
- The executor entrypoint is `sdk/python/kfp/dsl/executor_main.py`; it does not participate in compilation.
- Runtime containers install `kfp` with `--no-deps`; `_KFP_RUNTIME=true` disables most SDK imports. Task code must not depend on SDK-only modules unless the base image supplies their dependencies.

## Packages and key paths

| Area | Path |
| --- | --- |
| SDK compiler | `sdk/python/kfp/compiler/pipeline_spec_builder.py` |
| DSL | `sdk/python/kfp/dsl/` |
| Platform integration | `kubernetes_platform/python/kfp/` |
| Pipeline-spec APIs | `api/` |
| Backend | `backend/` |
| Frontend | `frontend/` |
| Deployment manifests | `manifests/` |
| Pipeline fixtures and workflow goldens | `test_data/` |

- Python packages share the `kfp` namespace: `kfp`, `kfp-pipeline-spec`, and `kfp-kubernetes`.
- `kfp-kubernetes` rewrites generated pipeline-spec imports through `kubernetes_platform/python/generate_proto.py`.

## Local execution

```python
from kfp import local

local.init(runner=local.SubprocessRunner())  # Lightweight Python components only.
task = my_component(param='value')
print(task.output)
```

Use `local.DockerRunner()` for container components or containerized task images. Pipeline calls return a run whose single output is `run.output`, or whose named outputs are in `run.outputs`. Local outputs default to `./local_outputs`.
