# Kubeflow Pipelines 1.0 Release Notes

## Versioning policy

Kubeflow Pipelines 1.0 is the first major release of Kubeflow Pipelines. The
Kubeflow Pipelines versioning follows the conventional versioning policy. That
is, the major version of Kubeflow Pipelines is incremented for API/feature changes that
are new and not backward-compatible; and on the other hand, the minor version
of Kubeflow Pipelines is incremented for API/feature changes that are new and
backward-compatible.

## Deployment options

* [Standalone Kubeflow Pipelines](https://www.kubeflow.org/docs/pipelines/installation/standalone-deployment/),
which allows users the maximum flexibility in configuring the Kubernetes
clusters and a Kubeflow Pipelines instance.

* [Google Cloud Marketplace Hosted Kubeflow Pipelines](https://www.kubeflow.org/docs/pipelines/installation/overview/#gcp-hosted-ml-pipelines),
which minimizes the user effort in deploying a Kubeflow pipelines instance.

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

Experimental features are the enhanced functionalities available in Kubeflow Pipelines. They might be replaced or deprecated in the future due to the refactoring or redesign of Kubeflow Pipelines.

* Visualization of output artifacts in two ways: [built-in visualization of selected artifact types](https://www.kubeflow.org/docs/pipelines/sdk/python-based-visualizations/) and [python based custom visualization](https://www.kubeflow.org/docs/pipelines/sdk/python-based-visualizations/).

* [Building pipelines with the SDK](https://www.kubeflow.org/docs/pipelines/sdk/).

* [Upgrade/Reinstall the Kubeflow Pipelines instance](https://www.kubeflow.org/docs/pipelines/upgrade/).

* [Step caching](https://www.kubeflow.org/docs/pipelines/caching/).

* Explore lineage or metadata of runs.

* Separate the Kubeflow Pipelines resources with multiple user namespace support. [???]


