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
