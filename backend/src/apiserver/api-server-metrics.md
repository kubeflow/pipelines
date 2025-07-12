## Metrics

Standard gRPC metrics are available via https://github.com/grpc-ecosystem/go-grpc-middleware/tree/main/providers/prometheus

### Sample

gRPC RPMs by grpc_method and grpc_code for the ml-pipeline server:
```yaml
sum by (grpc_service, grpc_method, grpc_code) (
rate(grpc_server_handled_total{app=~"ml-pipeline", kubernetes_namespace="kubeflow"}[1m])
) * 60
```

95th percentile gRPC latency by grpc_service and grpc_method for the ml-pipeline server:
```yaml
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

Gap in seconds between creating an execution spec (Argo or other backend) for a recurring run and reporting it via the persistence agent
```yaml
histogram_quantile(0.95, sum(rate(resource_manager_recurring_run_report_gap_bucket[1h])) by (le))
```


