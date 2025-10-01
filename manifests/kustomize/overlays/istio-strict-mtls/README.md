# Istio STRICT mTLS Overlay for Kubeflow Pipelines

This overlay configures Kubeflow Pipelines to work with Istio service mesh in STRICT mTLS mode.

## Features

- Configures driver pods with Istio sidecar injection labels
- Enables STRICT mTLS for the entire Kubeflow namespace
- Provides authorization policies for proper service communication
- Optimizes resource allocations for sidecar overhead
- Ensures proper proxy initialization order

## Prerequisites

1. Istio installed in your cluster
2. Kubeflow namespace labeled for Istio injection:
   ```bash
   kubectl label namespace kubeflow istio-injection=enabled
   ```

## Installation

Deploy KFP with Istio STRICT mTLS support:

```bash
kubectl apply -k manifests/kustomize/overlays/istio-strict-mtls
```

## Configuration

### Driver Pod Labels

The following labels are automatically applied to driver pods:
- `sidecar.istio.io/inject: "true"` - Enables sidecar injection
- `app.kubernetes.io/component: "kfp-driver"` - Component identification

### Driver Pod Annotations

The following annotations are applied:
- `proxy.istio.io/config` - Ensures proxy starts before the application
- `traffic.sidecar.istio.io/includeInboundPorts` - Includes all inbound ports
- `traffic.sidecar.istio.io/excludeOutboundPorts` - Excludes Istio control ports

### Resource Allocations

Increased resource limits to account for sidecar overhead:
- CPU: 1000m (limit), 200m (request)
- Memory: 1Gi (limit), 256Mi (request)

## Verification

### 1. Check Driver Pod Sidecar Injection

```bash
# Get a driver pod name
kubectl get pods -n kubeflow -l workflows.argoproj.io/workflow -o name | grep driver

# Verify sidecar container exists
kubectl get pod <driver-pod-name> -n kubeflow -o jsonpath='{.spec.containers[*].name}'
# Should show: istio-proxy
```

### 2. Verify mTLS Configuration

```bash
# Check PeerAuthentication
kubectl get peerauthentication -n kubeflow

# Check AuthorizationPolicy
kubectl get authorizationpolicy -n kubeflow
```

### 3. Test Pipeline Execution

Run a sample pipeline to verify driver pods can communicate with MinIO and MLMD:

```python
import kfp
import kfp.dsl as dsl

@dsl.component
def test_component():
    print("Testing Istio STRICT mTLS")

@dsl.pipeline(name='istio-test')
def test_pipeline():
    test_component()

client = kfp.Client()
client.create_run_from_pipeline_func(test_pipeline)
```

## Troubleshooting

### Connection Reset Errors

If you see "connection reset by peer" errors:

1. Verify sidecar injection is enabled:
   ```bash
   kubectl get pod <driver-pod> -n kubeflow -o yaml | grep -A5 "sidecar.istio.io/inject"
   ```

2. Check sidecar container logs:
   ```bash
   kubectl logs <driver-pod> -n kubeflow -c istio-proxy
   ```

### Filter Chain Not Found

This typically indicates mTLS mismatch:

1. Verify the driver pod has labels:
   ```bash
   kubectl get pod <driver-pod> -n kubeflow -o jsonpath='{.metadata.labels}'
   ```

2. Check if MinIO/MLMD services have proper Istio configuration:
   ```bash
   istioctl authn tls-check <driver-pod>.kubeflow minio-service.kubeflow
   istioctl authn tls-check <driver-pod>.kubeflow metadata-grpc-service.kubeflow
   ```

## Security Considerations

This overlay enforces STRICT mTLS, which means:
- All communication between services must use mTLS
- Pods without sidecars cannot communicate with services in the mesh
- External traffic must go through proper ingress gateways

## References

- [Issue #12015](https://github.com/kubeflow/pipelines/issues/12015)
- [Istio mTLS Migration](https://istio.io/latest/docs/tasks/security/authentication/mtls-migration/)
- [Istio Sidecar Injection](https://istio.io/latest/docs/setup/additional-setup/sidecar-injection/)