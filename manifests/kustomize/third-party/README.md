# Optional Monitoring of Kubeflow Pipelines With Prometheus and Grafana

We provide the option of monitoring Kubeflow Pipelines with [Prometheus](https://prometheus.io/) and [Grafana](https://grafana.com/). Kubeflow Pipelines instance has some backend operations that are directly related to resource consumption and request latency. Therefore, we can monitor the health of a running Kubeflow Pipelines instance by collecting metrics of its critical operations and resources. On the other hand, Prometheus and Grafana are widely adopted for collecting runtime metrics, building monitor dashboard, and setting up alerts; and Kubeflow Pipelines users might be familiar with them already. So naturally, we decided to provide optional monitoring with Prometheus and Grafana.

## Why we need monitoring

Kubeflow Pipelines can be used in a variety of application scenarios. For example, some application scenarios might have simple pipelines, but have a large amount of them; some might have quite complex pipelines, but only have a few of them; others might have very frequent requests on creating/listing pipelines/runs. In different application scenarios, Kubeflow Pipelines instance would likely show different characteristics and possibly expose different bottlenecks. In order to investigate any performance issue of Kubeflow Pipelines, we can collect runtime metrics and align performance issues with anomalies in those metrics, which will help a lot in determine the root cause(s) of the performance issues.

## How to monitor with Prometheus and Grafana

A basic way to monitor with Prometheus/Grafana is to deploy a Prometheus/Grafana server in the Kubeflow Pipelines cluster. The Prometheus server collects and stores metrics from running servers while the Grafana server hosts dashboards of plotting those metrics.

We haven't made the Prometheus/Grafana server part of the default deployment process of Kubeflow Pipelines. Instead, we provide a basic configuration file for Prometheus and Grafana respectively in the [third-party](https://github.com/kubeflow/pipelines/tree/master/manifests/kustomize/third-party) folder. If the users want to try monitoring with them, they can

- Use [kustomization.yaml](https://github.com/kubeflow/pipelines/blob/master/manifests/kustomize/sample/kustomization.yaml) located at manifests/kustomize/sample folder. Please pay attention to the customization part of this file as shown below.

![]


## example dashboard/basic metrics

## Customization

### config wise

### code wise

### node exporter