# Kubeflow Pipelines backend API

## Versions and compatibility

`v2` is the canonical backend API: HTTP endpoints use `/apis/v2`, protobuf
services use `kubeflow.pipelines.backend.api.v2`, and generated clients use
`V2…` model names.

`v2beta1` is a thin compatibility layer, not a second implementation:

- `/apis/v2beta1/...` is internally rewritten to `/apis/v2/...` before routing,
  including uploads, health checks, log/artifact streams, and data transfer.
  There are no HTTP redirects or additional upstream requests.
- The old gRPC service names register the same v2 handlers. Authentication,
  validation, persistence, and error handling remain shared. A small frozen
  descriptor snapshot supports legacy reflection and independent wire tests.
- Old Go client import paths and exported names are generated aliases and
  forwarding functions to v2, not duplicate clients or message implementations.
  Recompiled callers use canonical v2 endpoints and protobuf identities;
  already-built clients still work through the legacy server aliases.
- Generated Python `V2beta1…` imports, including module-qualified imports under
  `kfp.server_api.models.v2beta1_*`, are aliases of the canonical `V2…` classes.
  They serialize the same fields and use the v2 endpoints.
- Request/response fields, resource names, filter syntax, and page tokens are
  unchanged. Removed v1 APIs are not supported.

PipelineSpec IR (`api/v2alpha1`) and Kubernetes CRDs
(`pipelines.kubeflow.org/v2beta1`) have version lifecycles independent of the backend API. Old clients can call the new server;
new v2 clients require a server that exposes v2 (upgrade the server first).

Make schema changes under `backend/api/v2`. The descriptor parity regression
in `backend/src/apiserver/api_compat_test.go` compares v2 against the frozen
`v2beta1/legacy_descriptor.pb` snapshot, protecting the shared-handler assumption.
Incompatible future changes require explicit compatibility adapters rather than
changes to that historical baseline. There is no second editable proto tree.

## Generate Go clients and OpenAPI definitions

Requires Docker and Make:

```bash
make -C backend/api generate
```

`API_VERSION` defaults to `v2`; other generation versions are rejected.
Prebuilt generator images are used by default. After toolchain changes, or
before validating an API change, build the generator from the pinned source:

```bash
USE_PREBUILT_IMAGE=false make -C backend/api generate
```

Outputs live under `backend/api/v2/{go_client,go_http_client,swagger}`.
The same command regenerates the legacy Go import shims from those outputs.
`make -C backend/api generate-compat` regenerates just the shims without Docker.
`swagger/pipeline.upload.swagger.json` and `transfer.openapi.yaml` are manually
maintained HTTP contracts. All other Swagger and client outputs are generated;
do not edit them directly. `generate-from-scratch` is an alias for the source
build. Docker caches the image through the `.image-built` target.

## Generate the Python client

```bash
USE_PREBUILT_IMAGE=false make -C backend/api generate-kfp-server-api-package
```

The generated modules live in `sdk/python/kfp/server_api` and ship only in the
unified `kfp` distribution. Documentation lives in
`backend/api/v2/python_http_client`; it is not a separate Python distribution.
`hack/generate_python_compat.py` generates the legacy model import aliases as
part of this command. Import/endpoint smoke tests live outside the regenerated
directory, in `backend/api/v2/python_http_client_smoke`.

With Java, Python 3, and jq installed, the Python generator can also run directly:

```bash
backend/api/build_kfp_server_api_python_package.sh
```

## Generate frontend clients

After regenerating Swagger:

```bash
cd frontend
npm run apis:all
```

This regenerates browser and server clients from the v2 contracts and removes
the retired internal v2beta1 TypeScript client directories.

## API reference documentation

The merged `backend/api/v2/swagger/kfp_api_single_file.swagger.json` is the
canonical API reference. Keep `docs/_static/kfp_api_single_file.swagger.json`
synchronized with it when generating the contracts.

For kubeflow.org, generate the self-contained reference with
[bootprint-openapi](https://github.com/bootprint/bootprint-monorepo/tree/master/packages/bootprint-openapi)
and [html-inline](https://www.npmjs.com/package/html-inline), then update
[the website reference](https://github.com/kubeflow/website/blob/master/content/en/docs/components/pipelines/v2/reference/api/kubeflow-pipeline-api-spec.html).

The `build-tools-images.yml` workflow publishes the API generator and release
images. See that workflow for native-platform validation and publication.
