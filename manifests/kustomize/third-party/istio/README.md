# Istio prerequisite images

The multi-user installer uses Istio 1.31.4 distroless for the control plane,
CNI, gateway and injected sidecars. For shell-based troubleshooting, use an
ephemeral debug container.

When updating the Community Distribution pin in `deploy-kfp.sh`, update the
base in `install/kustomization.yaml` and copy the rendered sidecar injector's
`values` into `install/injector-values.json`. Set `global.tag` to the Istio
version and `global.variant` to `distroless`. Keep the three image tags in sync
with those settings and use the unsuffixed version for `CURRENT_AGENT_VERSION`.
