# Generated Code and APIs

Never edit generated files. Update their source and regenerate them.

`backend/api/v2` is canonical. `backend/api/v2beta1` is the frozen HTTP/gRPC
compatibility contract, not another implementation or generation target.
Python generation also creates legacy `V2beta1…` model import aliases.
Backend generation refreshes `docs/_static/kfp_api_single_file.swagger.json`;
frontend generation removes the retired internal `apisv2beta1` client trees.
Do not rename the independent PipelineSpec IR or Kubernetes CRD versions.

| Output | Source | Regenerate |
| --- | --- | --- |
| Pipeline-spec Python | `api/v2alpha1/pipeline_spec.proto` | `make -C api python` |
| Pipeline-spec Go | `api/` protos | `make -C api golang` |
| Kubernetes executor config | `kubernetes_platform/proto/kubernetes_executor_config.proto` | `make -C kubernetes_platform python` |
| Backend API clients and Swagger | `backend/api/v2/*.proto` | `make -C backend/api API_VERSION=v2 generate` |
| Frontend OpenAPI clients, including the browser and server ArtifactService clients | `backend/api/**/swagger/*.json` | `cd frontend && npm run apis:all` |

- Python outputs live in `sdk/python/kfp/{pipeline_spec,kubernetes,server_api}`.
  All generated modules are committed for clean Git/component builds.
  CI still explicitly regenerates them before building/testing and rejects
  differences; never hand-edit them.
- `make -C sdk generate-python` runs all three Python generators.
  `make -C sdk python` generates first, then builds the unified wheel and sdist.
  The package build rejects missing bindings; sdists include them and need no
  protoc or Java when consumers build wheels.
- For backend generator changes, use `USE_PREBUILT_IMAGE=false make -C backend/api API_VERSION=v2 generate`.
- Go-based API generator versions are selected by the root `go.mod` when they must match runtime libraries, or by `backend/api/tools/go.mod` for standalone tooling.
- Register standalone Go tools with a `tool` directive in `backend/api/tools/go.mod` and commit its tidied `go.mod` and `go.sum`. A bare `require` is removed by Dependabot's `go mod tidy`, even when the Dockerfile downloads that tool's binary.
- `sdk/python/kfp/server_api` is generated
  from `backend/api/v2/swagger/kfp_api_single_file.swagger.json` with
  `make -C backend/api API_VERSION=v2 generate-kfp-server-api-package`.
  Generated REST documentation remains under `backend/api/v2/python_http_client`.
- Bundled Python modules use `sdk/python/kfp/version.py`; backend `VERSION`
  remains independent. Regenerate the server client after SDK version changes.
- Python HTTP client generation omits OpenAPI's unimplemented API/model test
  stubs and unused tox configuration. Real coverage remains in SDK client tests
  and `backend/api/v2/python_http_client_smoke`.
- Frontend CI runs `bash scripts/check-spec-generation.sh` from `frontend` with `protoc` installed. It generates pipeline and Kubernetes platform types in a temporary directory and compiles both with the installed TypeScript/compiler dependencies. The individual generation scripts accept `PROTO_OUT_DIR` for isolated verification; ordinary generation still writes the committed source directories.
- `pipeline.upload.swagger.json` and `backend/api/v2/transfer.openapi.yaml` are manually maintained HTTP contracts; transfer archives use raw authenticated HTTP rather than generated gRPC clients.
- Schema changes require both `make -C api python` and `make -C api golang`.
- On SELinux hosts, protoc generation can require temporarily setting SELinux to permissive mode.
