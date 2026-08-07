# Development and Deployment

## Python setup

Use a `.venv`:

```bash
python3 -m venv .venv
source .venv/bin/activate
python -m pip install -U pip setuptools wheel
make -C api python-dev
make -C kubernetes_platform python-dev
pip install -e api/v2alpha1/python --config-settings editable_mode=strict
pip install -e sdk/python --config-settings editable_mode=strict
pip install -e kubernetes_platform/python --config-settings editable_mode=strict
```

Install Ginkgo into the repository when running its suites:

```bash
make ginkgo
export PATH="$PWD/bin:$PATH"
```

## Local clusters

| Need | Command |
| --- | --- |
| Standalone development cluster | `make -C backend kind-cluster-agnostic` |
| API-server development cluster | `make -C backend dev-kind-cluster` |

Standalone mode is single-user and unauthenticated. Multi-user deployments require an identity provider, namespace isolation, and Istio; see [`manifests/kustomize/README.md`](../../manifests/kustomize/README.md).

## Environment variables

| Variable | Purpose |
| --- | --- |
| `_KFP_RUNTIME=true` | Runtime mode that disables most SDK imports |
| `VITE_NAMESPACE` | Frontend namespace for multi-user development |
| `LOCAL_API_SERVER=true` | Local API-server integration-test mode |
| `FRONTEND_SERVER_NAMESPACE` | Namespace used by a local frontend server |
| `MINIO_ENDPOINT_REWRITE` | Rewrites object-store endpoints in local proxy mode |
| `MAX_METRICS_FILE_BYTES` | Maximum uncompressed metrics JSON size; defaults to 1 MiB |

`TENSORBOARD_PROXY_SIGNING_SECRET` is optional; it defaults to `MINIO_SECRET_KEY`.
