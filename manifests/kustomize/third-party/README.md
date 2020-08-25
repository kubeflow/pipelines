# Optional Monitoring of Kubeflow Pipelines With Prometheus and Grafana

We provide the option of monitoring Kubeflow Pipelines with [Prometheus](https://prometheus.io/) and [Grafana](https://grafana.com/). Kubeflow Pipelines instance has some backend operations that are directly related to resource consumption and request latency. Therefore, we can monitor the health of a running Kubeflow Pipelines instance by collecting metrics of its critical operations and resources. On the other hand, Prometheus and Grafana are widely adopted for collecting runtime metrics, building monitor dashboard, and setting up alerts; and Kubeflow Pipelines user might be familiar with them already. So naturally, we decided to provide optional monitoring with Prometheus and Grafana.

## Why we need monitoring

Kubeflow Pipelines can be used in a variety of application scenarios. For example, some application scenarios might have simple pipelines, but have a large amount of them; some might have quite complex pipelines, but only have a few of them; others might have very frequent requests on creating or listing pipelines/runs. In different application scenarios, Kubeflow Pipelines instance is likely to show different characteristics and different bottlenecks. The runtime metrics

## How to monitor with Prometheus and Grafana



## example dashboard/basic metrics

## Customization

### config wise

### code wise
