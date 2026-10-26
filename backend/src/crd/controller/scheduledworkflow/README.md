## Controller Flags

The controller supports the following command-line flags:

| Flag                                  | Description | Default |
|---------------------------------------|-------------|---------|
| `--logtostderr`                       | Log output to stderr instead of files. | `true` |
| `--logLevel`                          | Defines the log level for the application. | `""` |
| `--clientQPS`                         | Maximum queries per second to the Kubernetes API server. | `5` |
| `--clientBurst`                       | Maximum burst for throttle from this client. | `10` |
| `--recurringRunResyncIntervalSeconds` | Full resync interval in seconds for recurring run reconciliations. | `30` |
| `--metricsPort`                       | The port for the metrics endpoint. | `9090` |

When the `LEADER_ELECT` environment variable is `true`, as in the default manifests, only the replica that holds the `kfp-scheduledworkflow` Lease in the `POD_NAMESPACE` namespace runs the controller, so that more than one replica can run.

## Prometheus Metrics

The controller exposes the following Prometheus metrics:

| Metric | Type | Description | Labels |
|--------|------|-------------|--------|
| `process_scheduled_workflow_duration_seconds` | Histogram | Duration of scheduled workflow handling in seconds. | `status` |
| `scheduled_workflow_duration_operation_seconds` | Histogram | Duration of scheduled workflow handling by operation. | `operation`, `status` |
| `scheduled_workflow_queue_size` | Gauge | Current number of workflows in the queue. | - |
| `scheduled_workflow_throughput_total` | Counter | Total number of workflows processed (throughput). | - |

### Example Usage

These metrics are automatically registered with Prometheus using `promauto`. You can scrape them at the port specified by `--metricsPort` (default `9090`):
