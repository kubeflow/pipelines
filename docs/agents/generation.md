# Generated Code and APIs

Never edit generated files. Update their source and regenerate them.

| Output | Source | Regenerate |
| --- | --- | --- |
| Pipeline-spec Python | `api/v2alpha1/pipeline_spec.proto` | `make -C api python` |
| Pipeline-spec Go | `api/` protos | `make -C api golang` |
| Kubernetes executor config | `kubernetes_platform/proto/kubernetes_executor_config.proto` | `make -C kubernetes_platform python` |
| Backend API clients and Swagger | `backend/api/{v1beta1,v2beta1}/*.proto` | `make -C backend/api API_VERSION=<version> generate` |
| Frontend OpenAPI clients, including the browser and server ArtifactService clients | `backend/api/**/swagger/*.json` | `cd frontend && npm run apis:all` |

- `api/v2alpha1/python/kfp/pipeline_spec/pipeline_spec_pb2.py` is generated but not committed.
- For backend generator changes, use `USE_PREBUILT_IMAGE=false make -C backend/api API_VERSION=<version> generate`.
- Go-based API generator versions are selected by the root `go.mod` when they must match runtime libraries, or by `backend/api/tools/go.mod` for standalone tooling.
- `backend/api/v2beta1/python_http_client` is generated from `kfp_api_single_file.swagger.json` with `cd backend/api && make generate-kfp-server-api-package`.
- `pipeline.upload.swagger.json` is manually maintained.
- Schema changes require both `make -C api python` and `make -C api golang`.
- On SELinux hosts, protoc generation can require temporarily setting SELinux to permissive mode.
