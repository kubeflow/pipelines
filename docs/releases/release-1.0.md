# Kubeflow Pipelines 1.0 Release Notes

## Versioning policy

Kubeflow Pipelines 1.0 is the first major release of Kubeflow Pipelines. It follows the Kubeflow Pipelines [versioning policy](https://github.com/kubeflow/pipelines/blob/master/docs/releases/versioning-policy.md).

## Deployment options

There are two available ways to [deployment Kubeflow Pipelines](https://www.kubeflow.org/docs/pipelines/installation/overview/).

* [Standalone Kubeflow Pipelines](https://www.kubeflow.org/docs/pipelines/installation/standalone-deployment/), which allows users the maximum flexibility in configuring the Kubernetes clusters and a Kubeflow Pipelines instance.

* [AI Platform Pipelines](https://cloud.google.com/ai-platform/pipelines/docs), which minimizes the user effort in deploying a Kubeflow pipelines instance.

* Note: as of 2020-06-30, the Kubeflow Pipelines 1.0 instance is not available in [the Kubeflow deployment](https://www.kubeflow.org/docs/pipelines/installation/overview/#full-kubeflow) yet. Please use the above two ways
to deploy the Kubeflow Pipelines 1.0 instance.

## Core features and APIs

Core features and APIs are the fundamental functionalities of Kubeflow Pipelines. The most fundamental functionality is to manage the Kubeflow Pipelines resources (that is, pipeline, pipeline version, run, job, experiment) via the operations listed below. Active supports are available for core features and APIs.

|                  | Upload | Create | Get | List | Delete | Archive | Unarchive | Enable | Disable | Terminate | Retry |
|:----------------:|:------:|:------:|:---:|:----:|:------:|:-------:|:---------:|:------:|:-------:|:---------:|:-----:|
| Pipeline         | - [x]  |  - [x] |- [x]|- [x] | - [x]  | - [x]   | - [x]     |
| Pipeline Version | - [x]  |  - [x] |- [x]|- [x] | - [x]  | - [x]   | - [x]     |
| Run              | - [x]  |  - [x] |- [x]|- [x] | - [x]  | - [x]   | - [x]     |        |         | - [x]     | - [x] |
| Job              | - [x]  |  - [x] |- [x]|- [x] | - [x]  |         |           | - [x]  | - [x]   |
| Experiment       | - [x]  |  - [x] |- [x]|- [x] | - [x]  | - [x]   | - [x]     |

* Refer to the Kubeflow Pipelines [introduction]((https://www.kubeflow.org/docs/pipelines/overview/)) for
  - An [overview](https://www.kubeflow.org/docs/pipelines/overview/pipelines-overview/) and [main concepts](https://www.kubeflow.org/docs/pipelines/overview/concepts/), for example, what is a pipeline or run in Kubeflow Pipelines.
  - How to manage the Kubeflow Pipelines resources via the the [Kubeflow Pipelines UI](https://www.kubeflow.org/docs/pipelines/overview/interfaces/).
* Refer to the Kubeflow Pipelines [SDK samples and tutorials](https://www.kubeflow.org/docs/pipelines/tutorials/sdk-examples/)
for how to manage the Kubeflow Pipelines resources via the Kubeflow Pipelines client.
* A complete API method list and a detailed description of parameters to those API methods are available at the Kubeflow Pipelines [API reference](https://www.kubeflow.org/docs/pipelines/reference/api/kubeflow-pipeline-api-spec/).

## Experimental features

Experimental features are the feature in the early phases of development and may change drastically in the future.

* Visualization of output artifacts in two ways: [built-in visualization of selected artifact types](https://www.kubeflow.org/docs/pipelines/sdk/python-based-visualizations/) and [python based custom visualization](https://www.kubeflow.org/docs/pipelines/sdk/python-based-visualizations/).

* [Building pipelines with the SDK](https://www.kubeflow.org/docs/pipelines/sdk/).

* [Upgrade/Reinstall the Kubeflow Pipelines instance](https://www.kubeflow.org/docs/pipelines/upgrade/).

* [Step caching](https://www.kubeflow.org/docs/pipelines/caching/).

* Explore lineage or metadata of runs.

* Separate the Kubeflow Pipelines resources with multiple user namespace support. [???]


