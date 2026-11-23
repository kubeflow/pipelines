# Istio prerequisite patch

The multi-user deployment script uses this overlay to run Istio 1.31.4 with the
pinned community-distribution 1.31 manifests. It preserves the existing CRDs,
RBAC, mesh configuration, and injection template. The profile prerequisite uses
the same distribution commit and Dashboard 2.0.1.

The image overrides, CNI `CURRENT_AGENT_VERSION`, and injector
`values.global.tag` must advance together. `injector-values.json` is copied from
the rendered upstream `istio-sidecar-injector` ConfigMap; only `global.tag` is
changed. Preserve all other upstream injection settings when refreshing it.
Chart-version labels continue to identify the underlying manifest version.

Release image inventory captures this overlay and injector configuration and
renders the same installation target. Do not replace the images only: newly
injected sidecars would otherwise continue using the old version.
