## Metrics

Standard gRPC metrics are provided by the [go-grpc-middleware Prometheus provider](https://github.com/grpc-ecosystem/go-grpc-middleware/tree/main/providers/prometheus). All HTTP REST endpoints exposed by the gateway forward through the instrumented gRPC server, allowing unified monitoring of API request rates, success/error outcomes, and latencies across all gRPC services (`kubeflow.pipelines.backend.api.v2.ExperimentService`, `kubeflow.pipelines.backend.api.v2.PipelineService`, `kubeflow.pipelines.backend.api.v2.RecurringRunService`, `kubeflow.pipelines.backend.api.v2.RunService`, etc.).

> **Note:** Direct HTTP handlers that do not pass through the gRPC gateway (such as multipart pipeline uploads in `PipelineUploadServer`) bypass gRPC interceptors and are not captured by `grpc_server_handled_total`.

### Samples & PromQL Queries

#### 1. Request Rate (RPM)

gRPC RPM by service, method, and response code:
```promql
sum by (grpc_service, grpc_method, grpc_code) (
  rate(grpc_server_handled_total{app=~"ml-pipeline", kubernetes_namespace="kubeflow"}[1m])
) * 60
```

#### 2. Success and Error Rates

##### Overall API Server Error Rate (%)
Calculate the percentage of failing requests across all API services:
```promql
sum(rate(grpc_server_handled_total{app=~"ml-pipeline", grpc_code!="OK"}[5m]))
  /
sum(rate(grpc_server_handled_total{app=~"ml-pipeline"}[5m])) * 100
```

##### Successful vs. Failed Requests by Service
Count successful (`OK`) vs. failing (`!=OK`) calls per service:
```promql
# Successful requests
sum by (grpc_service, grpc_method) (
  rate(grpc_server_handled_total{app=~"ml-pipeline", grpc_code="OK"}[5m])
)

# Failed requests
sum by (grpc_service, grpc_method, grpc_code) (
  rate(grpc_server_handled_total{app=~"ml-pipeline", grpc_code!="OK"}[5m])
)
```

##### Service-Specific Error Breakdown (e.g., ExperimentService, PipelineService, RecurringRunService)
Filter error rates and failure codes for a specific service using its fully qualified gRPC service name:

```promql
# ExperimentService error breakdown by method and gRPC code
sum by (grpc_method, grpc_code) (
  rate(grpc_server_handled_total{app=~"ml-pipeline", grpc_service="kubeflow.pipelines.backend.api.v2.ExperimentService", grpc_code!="OK"}[5m])
)

# PipelineService error breakdown by method and gRPC code
sum by (grpc_method, grpc_code) (
  rate(grpc_server_handled_total{app=~"ml-pipeline", grpc_service="kubeflow.pipelines.backend.api.v2.PipelineService", grpc_code!="OK"}[5m])
)

# RecurringRunService error breakdown by method and gRPC code
sum by (grpc_method, grpc_code) (
  rate(grpc_server_handled_total{app=~"ml-pipeline", grpc_service="kubeflow.pipelines.backend.api.v2.RecurringRunService", grpc_code!="OK"}[5m])
)
```

#### 3. Latency

95th percentile gRPC latency by `grpc_service` and `grpc_method`:
```promql
histogram_quantile(
  0.95,
  sum by (grpc_service, grpc_method, le) (
    rate(grpc_server_handling_seconds_bucket{
      app="ml-pipeline",
      kubernetes_namespace="kubeflow"
    }[1m])
  )
)
```

#### 4. Workflow Reporting Lag

Gap in seconds between creating an execution spec for a recurring run and reporting it via the persistence agent:
```promql
histogram_quantile(0.95, sum(rate(resource_manager_recurring_run_report_gap_bucket[1h])) by (le))
```

### Example Prometheus Alerting Rule

Alert when the API server error rate exceeds 5% over a 5-minute window:

```yaml
- alert: KFPAPIServerHighErrorRate
  expr: |
    sum(rate(grpc_server_handled_total{app=~"ml-pipeline", grpc_code!="OK"}[5m]))
      /
    sum(rate(grpc_server_handled_total{app=~"ml-pipeline"}[5m])) > 0.05
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "KFP API Server error rate is above 5%"
    description: "The KFP API server is experiencing elevated errors ({{ $value | humanizePercentage }} failure rate over the last 5 minutes)."
```
