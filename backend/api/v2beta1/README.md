# Frozen v2beta1 compatibility contract

New API development and client generation belong in [`../v2`](../v2).
The API server exposes the old HTTP prefix and gRPC service names through
thin adapters to the canonical v2 implementation; this directory does not
contain a separate server implementation.

Keep these protobuf definitions, descriptors, and legacy Go clients intact for
existing consumers and gRPC reflection. `TestLegacyAPIContractParity` checks
that the shared handlers remain wire- and JSON-compatible. An incompatible
future change needs an explicit adapter, not a silent change here.

Python `V2beta1…` model names remain generated aliases of the canonical `V2…`
models in `kfp.server_api`. See the [API guide](../README.md) for generation
commands and upgrade order. Kubernetes CRDs using `v2beta1` are unrelated.
