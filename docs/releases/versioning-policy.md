# Kubeflow Pipelines Versioning

The Kubeflow Pipelines versioning follows the [semantic versioning policy](https://semver.org/). The Kubeflow Pipelines versions are in the format of X.Y.Z, where X is the major version, Y is the minor version, and Z is the patch version. The major version is incremented for API/feature changes that are new and not backward-compatible; the minor version is incremented for API/feature changes that are new and backward-compatible; and the patch version is for bug fixes between two minor versions. The Kubeflow Pipelines X.Y.Z refers to the version (git tag) of the released Kubeflow Pipelines and it versions all servers and the SDK of the Kubeflow Pipelines. Moreover, if the version number includes an appendix -rcN, where N is a number, the appendix indicates a release candidate, which is a pre-release version of an upcoming release.

# Kubeflow Pipelines Feature Status

The features in Kubeflow Pipelines fall into three different phases: general availability,
alpha and beta.

- The features in the general availability phase are stable and have active support for bug fixes.
- The features in beta phase are mostly stable. On a case-by-case basis, they will either
stay in the maintenance mode or be promoted to general availability in the future.
- The features in alpha phase are complete but haven't been tested extensively. They are subject to
future deprecation.

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

### SDK

* Only [ComponentSpec](https://github.com/kubeflow/pipelines/blob/master/sdk/python/kfp/components/structures/components.json_schema.json) in SDK is in general availability phase.

## Features in Beta

* Job API

|                  | Upload | Create | Get | List | Delete | Archive | Unarchive | Enable | Disable | Terminate | Retry |
|:----------------:|:------:|:------:|:---:|:----:|:------:|:-------:|:---------:|:------:|:-------:|:---------:|:-----:|
| Job              |        |  - [x] |- [x]|- [x] | - [x]  |         |           | - [x]  | - [x]   |


* SDK DSL

* [Upgrade/Reinstall the Kubeflow Pipelines instance](https://www.kubeflow.org/docs/pipelines/upgrade/).

* [Built-in visualization of selected artifact types](https://www.kubeflow.org/docs/pipelines/sdk/output-viewer/).


## Features in Alpha

* SDK client (that is, the helper functions). The helper functions mainly rely
on community maintenance. The recommended alternatives are the auto-generated
client APIs, for example, client.pipelines.list_pipelines(),
client.runs.list_runs(), client.pipeline_uploads.upload_pipeline(). Refer to
[more samples](https://www.kubeflow.org/docs/pipelines/tutorials/sdk-examples/)
on how to use them.

* [Python based custom visualization](https://www.kubeflow.org/docs/pipelines/sdk/python-based-visualizations/).

* [Step caching](https://www.kubeflow.org/docs/pipelines/caching/).

// TODO: lineage explorer and multi-tenant support




