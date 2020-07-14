# Kubeflow Pipelines Versioning Policy

The Kubeflow Pipelines versions follows [the semantic versioning](https://semver.org/).
The Kubeflow Pipelines versions are in the format of `X.Y.Z`, where `X` is the major version,
`Y` is the minor version, and `Z` is the patch version.

We increment the:
* MAJOR version when we make incompatible API changes on [generally available features](#features-in-general-availability),
* MINOR version when we add functionality in a backwards compatible manner, and
* PATCH version when we make backwards compatible bug fixes.

Additionally, we do pre-releases as an extension in the format of `X.Y.Z-rc.N`
where `N` is a number. The appendix indicates the Nth release candidate before
an upcoming public release.

The Kubeflow Pipelines version `X.Y.Z` refers to the version (git tag) of the released
Kubeflow Pipelines. It versions all released artifacts, including:
* `kfp` and `kfp-server-api` python packages
* install manifests
* docker images on gcr.io
* first party components

# Kubeflow Pipelines Feature Status

The features in Kubeflow Pipelines fall into three different phases: general availability, beta and alpha.

- The features in the general availability phase are stable. They have active
support for bug fixes. Only backward compatible features will be added.
- The features in beta phase are mostly stable. We reserve the right to make
breaking changes, but we'll provide deprecation notice and a migration path.
- The features in alpha phase are early. They haven't been tested extensively.
They are subject to drastic and potentially backward incompatible changes.

## Features in general availability

### APIs

|                  | Upload | Create | Get | List | Delete | Archive | Unarchive | Enable | Disable | Terminate | Retry |
|:----------------:|:------:|:------:|:---:|:----:|:------:|:-------:|:---------:|:------:|:-------:|:---------:|:-----:|
| Pipeline         | - [x]  |  - [x] |- [x]|- [x] | - [x]  |         |           |
| Pipeline Version | - [x]  |  - [x] |- [x]|- [x] | - [x]  |         |           |
| Run              |        |  - [x] |- [x]|- [x] | - [x]  | - [x]   | - [x]     |        |         | - [x]     | - [x] |
| Experiment       |        |  - [x] |- [x]|- [x] | - [x]  | - [x]   | - [x]     |

* Refer to the Kubeflow Pipelines [introduction]((https://www.kubeflow.org/docs/pipelines/overview/)) for
  - An [overview](https://www.kubeflow.org/docs/pipelines/overview/pipelines-overview/) and [main concepts](https://www.kubeflow.org/docs/pipelines/overview/concepts/), for example, what is a pipeline or run in Kubeflow Pipelines.
  - How to manage the Kubeflow Pipelines resources via the the [Kubeflow Pipelines UI](https://www.kubeflow.org/docs/pipelines/overview/interfaces/).
* Refer to the Kubeflow Pipelines [SDK samples and tutorials](https://www.kubeflow.org/docs/pipelines/tutorials/sdk-examples/)
for how to manage the Kubeflow Pipelines resources via the Kubeflow Pipelines client.
* A complete API method list and a detailed description of parameters to those API methods are available at the Kubeflow Pipelines [API reference](https://www.kubeflow.org/docs/pipelines/reference/api/kubeflow-pipeline-api-spec/).

## Features in Beta

* [ComponentSpec](https://github.com/kubeflow/pipelines/blob/release-1.0/sdk/python/kfp/components/structures/components.json_schema.json)

* Job API

|                  | Upload | Create | Get | List | Delete | Archive | Unarchive | Enable | Disable | Terminate | Retry |
|:----------------:|:------:|:------:|:---:|:----:|:------:|:-------:|:---------:|:------:|:-------:|:---------:|:-----:|
| Job              |        |  - [x] |- [x]|- [x] | - [x]  |         |           | - [x]  | - [x]   |


* SDK DSL for constructing a pipeline.

* [Upgrade support for the Kubeflow Pipelines standalone deployment](https://www.kubeflow.org/docs/pipelines/installation/standalone-deployment/#upgrading-kubeflow-pipelines).

* [Built-in Visualizations](https://www.kubeflow.org/docs/pipelines/sdk/output-viewer/).


## Features in Alpha

* [SDK client helper methods](https://kubeflow-pipelines.readthedocs.io/en/latest/source/kfp.client.html). The helper methods are mainly maintained by
community. They are convenient to use, but lack of error handling and testing
prevent us moving it to Beta quality right now.

  We recommend and support auto-generated client APIs instead. For example,
  `client.pipelines.list_pipelines()`, `client.runs.list_runs()` and
  `client.pipeline_uploads.upload_pipeline()`.
  
  Refer to [Using the Kubeflow Pipelines SDK](https://www.kubeflow.org/docs/pipelines/tutorials/sdk-examples/) for some example usages and [SDK generated API reference](https://kubeflow-pipelines.readthedocs.io/en/latest/source/kfp.client.html#generated-apis) for exact method signatures.

* [Step caching](https://www.kubeflow.org/docs/pipelines/caching/).

* [ML Metadata (MLMD)](https://github.com/google/ml-metadata) UI integration.
  For example, artifact and execution list and detail pages.

* [Python based custom visualization](https://www.kubeflow.org/docs/pipelines/sdk/python-based-visualizations/).

* [Pipeline Metrics](https://www.kubeflow.org/docs/pipelines/sdk/pipelines-metrics/).

* [Multi User Support](https://github.com/kubeflow/pipelines/issues/1223) will
  not be present in Kubeflow Pipelines 1.0's standalone deployment. It
  will instead get released with Kubeflow 1.1.
