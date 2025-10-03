# Driver Pod Configuration Guide

## Overview

Starting from Kubeflow Pipelines v2.15, administrators can configure labels and annotations for driver pods to support infrastructure requirements such as Istio service mesh integration.

## Configuration Methods

### Method 1: Environment Variables

Set environment variables in the `ml-pipeline` deployment:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ml-pipeline
  namespace: kubeflow
spec:
  template:
    spec:
      containers:
        - name: ml-pipeline-api-server
          env:
            - name: DRIVER_POD_LABELS
              value: |
                {
                  "sidecar.istio.io/inject": "true",
                  "app.kubernetes.io/component": "kfp-driver"
                }
            - name: DRIVER_POD_ANNOTATIONS
              value: |
                {
                  "prometheus.io/scrape": "true",
                  "prometheus.io/port": "9090"
                }
```

### Method 2: ConfigMap

Use a ConfigMap for centralized configuration:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: kfp-driver-config
  namespace: kubeflow
data:
  driver-config: |
    {
      "labels": {
        "sidecar.istio.io/inject": "true"
      },
      "annotations": {
        "proxy.istio.io/config": "{\"holdApplicationUntilProxyStarts\": true}"
      }
    }
```

Then reference it in the deployment:

```yaml
env:
  - name: V2_DRIVER_CONFIG
    valueFrom:
      configMapKeyRef:
        name: kfp-driver-config
        key: driver-config
        optional: true
```

### Method 3: Kustomize Overlay

For production deployments, use the provided Istio overlay:

```bash
kubectl apply -k manifests/kustomize/overlays/istio-strict-mtls
```

## Common Use Cases

### Istio Service Mesh with STRICT mTLS

When running KFP in an Istio mesh with STRICT mTLS enabled:

**Problem**: Driver pods cannot communicate with MinIO and MLMD services.

**Solution**: Configure driver pods with Istio sidecar injection:

```json
{
  "labels": {
    "sidecar.istio.io/inject": "true",
    "app.kubernetes.io/component": "kfp-driver"
  },
  "annotations": {
    "proxy.istio.io/config": "{\"holdApplicationUntilProxyStarts\": true}",
    "traffic.sidecar.istio.io/includeInboundPorts": "*",
    "traffic.sidecar.istio.io/excludeOutboundPorts": "15090,15021"
  }
}
```

### Prometheus Monitoring

Enable metrics collection from driver pods:

```json
{
  "annotations": {
    "prometheus.io/scrape": "true",
    "prometheus.io/port": "9090",
    "prometheus.io/path": "/metrics"
  }
}
```

### Node Selection

Route driver pods to specific nodes:

```json
{
  "labels": {
    "workload-type": "kfp-driver",
    "node-role": "pipeline-execution"
  }
}
```

## Configuration Reference

### Supported Environment Variables

| Variable | Description | Format | Default |
|----------|-------------|--------|---------|
| `DRIVER_POD_LABELS` | Labels to apply to driver pods | JSON map | `{}` |
| `DRIVER_POD_ANNOTATIONS` | Annotations to apply to driver pods | JSON map | `{}` |
| `V2_DRIVER_CONFIG` | Combined configuration | JSON object | `{}` |
| `DRIVER_RESOURCE_LIMITS_CPU` | CPU limit for driver pods | Kubernetes quantity | `500m` |
| `DRIVER_RESOURCE_LIMITS_MEMORY` | Memory limit for driver pods | Kubernetes quantity | `512Mi` |
| `DRIVER_RESOURCE_REQUESTS_CPU` | CPU request for driver pods | Kubernetes quantity | `100m` |
| `DRIVER_RESOURCE_REQUESTS_MEMORY` | Memory request for driver pods | Kubernetes quantity | `128Mi` |

### Reserved Labels

The following label prefixes are reserved and will be filtered if provided:
- `pipelines.kubeflow.org/`
- `workflows.argoproj.io/`

## Verification

### 1. Check Configuration Loading

View API server logs:

```bash
kubectl logs deployment/ml-pipeline -n kubeflow | grep -i "driver.*config"
```

### 2. Verify Driver Pod Labels

```bash
# Get a driver pod
DRIVER_POD=$(kubectl get pods -n kubeflow -l workflows.argoproj.io/workflow -o name | grep driver | head -1)

# Check labels
kubectl get $DRIVER_POD -n kubeflow -o jsonpath='{.metadata.labels}' | jq .

# Check annotations
kubectl get $DRIVER_POD -n kubeflow -o jsonpath='{.metadata.annotations}' | jq .
```

### 3. Verify Sidecar Injection (Istio)

```bash
# Check for istio-proxy container
kubectl get $DRIVER_POD -n kubeflow -o jsonpath='{.spec.containers[*].name}'
# Should include: istio-proxy
```

## Troubleshooting

### Configuration Not Applied

1. **Check environment variables are set:**
   ```bash
   kubectl describe deployment ml-pipeline -n kubeflow | grep -A5 "DRIVER_POD"
   ```

2. **Verify ConfigMap exists (if using):**
   ```bash
   kubectl get configmap kfp-driver-config -n kubeflow -o yaml
   ```

3. **Check for parse errors in logs:**
   ```bash
   kubectl logs deployment/ml-pipeline -n kubeflow | grep -i error
   ```

### Istio Connection Issues

1. **Verify sidecar injection:**
   ```bash
   kubectl get pod <driver-pod> -n kubeflow -o yaml | grep sidecar.istio.io/inject
   ```

2. **Check mTLS status:**
   ```bash
   istioctl authn tls-check <driver-pod>.kubeflow minio-service.kubeflow
   ```

3. **Review sidecar logs:**
   ```bash
   kubectl logs <driver-pod> -n kubeflow -c istio-proxy
   ```

## Migration from Earlier Versions

### From KFP < 2.15

No migration required. The feature is disabled by default and only activates when configuration is provided.

### From PERMISSIVE mTLS Workaround

If you were using PERMISSIVE mTLS for MinIO/MLMD as a workaround:

1. Configure driver pods with Istio labels (as shown above)
2. Test with a sample pipeline
3. Switch services back to STRICT mTLS:

```yaml
apiVersion: security.istio.io/v1beta1
kind: PeerAuthentication
metadata:
  name: default
  namespace: kubeflow
spec:
  mtls:
    mode: STRICT
```

## Security Considerations

- Configuration is admin-level only (deployment time)
- Users cannot modify driver pod configuration at runtime
- Reserved system labels are protected from override
- All configurations are logged for audit purposes

## Performance Impact

- Minimal overhead during compilation (<1ms)
- No impact if configuration is not provided
- With Istio sidecar: ~100-200ms additional startup time
- Memory overhead with sidecar: ~50-100Mi per driver pod

## References

- [Issue #12015](https://github.com/kubeflow/pipelines/issues/12015) - Original feature request
- [Istio Sidecar Injection](https://istio.io/latest/docs/setup/additional-setup/sidecar-injection/)
- [Kubernetes Labels and Selectors](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/)
- [KFP Server Configuration](https://www.kubeflow.org/docs/components/pipelines/operator-guides/server-config/)