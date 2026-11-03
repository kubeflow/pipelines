# v2beta1 compatibility shims

The implementation and editable API contracts live in [`../v2`](../v2).
This directory contains no duplicate protobuf message implementation, HTTP
client implementation, Swagger, Python documentation, or templates.

## Existing Go imports

`go_client` and `go_http_client` contain generated aliases and forwarding
functions. Old names such as `V2beta1Run` refer to the actual v2 types; methods,
serialization, validation, and transport logic come from v2. Recompiled callers
keep their import paths but use canonical v2 HTTP/gRPC endpoints and protobuf
message identities. Explicit `File_backend_api_v2beta1_*_proto` exports alias
the canonical v2 file descriptors.
The small HTTP constructor wrappers preserve assignments
to the legacy packages' `DefaultSchemes` variable.

Regenerate these shims after generating the canonical Go clients:

```bash
make -C backend/api generate-compat
```

Ordinary `make -C backend/api generate` does this automatically. The generator
is `backend/api/hack/generate_compat`; do not edit the aliases by hand.

## Already-built clients and reflection

The server still internally routes `/apis/v2beta1/...` to `/apis/v2/...` and
registers the old gRPC service names against the same canonical handlers.
Already-built clients do not need to be regenerated.

Reflection and protobuf `Any` type URLs use canonical v2 names. The runtime
does not register historical descriptors or dynamic legacy message types.
Clients that discover schemas through reflection must use v2 names; already-built
clients with their own schemas retain the legacy HTTP/gRPC wire routes.

`backend/src/apiserver/testdata/legacy_descriptor.pb` is the frozen 34,519-byte
`FileDescriptorSet` captured from the ten legacy v2beta1 files. The test-only
loader in `legacy_descriptors_test.go` verifies its SHA-256 and loads it into a
private registry for contract and legacy-wire tests. Do not regenerate this
independent historical baseline from v2. Additive v2 fields, enum values, and
RPCs are allowed; changes to the existing contract need explicit adapters.

Python legacy model imports are separately generated aliases in
`kfp.server_api`. See the [API guide](../README.md) for upgrade order and
compatibility scope. Kubernetes CRD versions are unrelated and unchanged.
