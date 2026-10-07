# v2beta1 compatibility shims

The implementation and editable API contracts live in [`../v2`](../v2).
This directory contains no duplicate protobuf message implementation, HTTP
client implementation, Swagger, Python documentation, or templates.

## Existing Go imports

`go_client` and `go_http_client` contain generated aliases and forwarding
functions. Old names such as `V2beta1Run` refer to the actual v2 types; methods,
serialization, validation, and transport logic come from v2. Recompiled callers
keep their import paths but use canonical v2 HTTP/gRPC endpoints and protobuf
message identities. The small HTTP constructor wrappers preserve assignments
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

`legacy_descriptor.pb` is a frozen 34,519-byte `FileDescriptorSet`, captured
from the ten v2beta1 generated Go file descriptors at commit `2953513cf` before
removing them. It contains schema metadata only, not executable client/server
logic. `descriptors.go` registers this snapshot for legacy gRPC reflection and
uses generic dynamic protobuf types to preserve legacy `Any` type URLs, resolving
external dependencies from the canonical API. Do not regenerate the
snapshot from v2: it is the independent historical baseline used by schema
parity and real legacy-wire tests. Incompatible changes need explicit adapters.

Python legacy model imports are separately generated aliases in
`kfp.server_api`. See the [API guide](../README.md) for upgrade order and
compatibility scope. Kubernetes CRD versions are unrelated and unchanged.
