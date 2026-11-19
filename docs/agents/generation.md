# Schema generation

Only the shared authoring and Kubernetes configuration schemas remain:

| Binding | Source |
|---|---|
| `sdk/python/kfp/pipeline_spec/pipeline_spec_pb2.py` | `api/v2alpha1/pipeline_spec.proto` |
| `sdk/python/kfp/kubernetes/kubernetes_executor_config_pb2.py` | `kubernetes_platform/proto/kubernetes_executor_config.proto` |

Run `make generate-python`. The generator uses the pinned `grpcio-tools` workspace
extra and installed Google common proto sources. It rewrites the Kubernetes
binding's import to the unified SDK path. Never hand-edit the generated files.
There is no Go or OpenAPI/RPC generator in this fork.

The offline viewer is generated from the retained frontend modules:

```sh
cd frontend
npm ci
npm run build:viewer
```

Commit `sdk/python/kfp/cli/viewer_assets/` with renderer or dependency changes.
The bundle includes its licenses and is tested without network access.
