# Kubeflow Pipelines Feature Stages

The features in Kubeflow Pipelines fall into three different stages: alpha, beta, and stable.

## Stage criteria and support levels

### Alpha
* The software is very likely to contain bugs.
* The support for a feature may be dropped at any time without notice in advance.
* The API may change in incompatible ways in a later software release without notice.
* The software is recommended for use only in short-lived testing environments, due to increased risk of bugs and lack of long-term support.
* Documentation in https://github.com/kubeflow/pipelines/tree/master/docs (technical writing review not required).

### Beta
* The software is well tested.
* The support for a feature will not be dropped, though the details may change.
* The schema and/or semantics of objects may change in incompatible ways in a subsequent beta or stable release. When this happens, migration instructions are provided.
* The software is recommended for only non-business-critical uses because of potential for incompatible changes in subsequent releases.
* Full documentation on user facing channels (kubeflow.org and reference websites).

### Stable
All of the guarantees for Beta and:
* No breaking changes without a major version release.
* The software is recommended for all uses.

## Stable Features

* [SDK DSL](https://github.com/kubeflow/pipelines/tree/master/sdk/python/kfp/dsl) for constructing a pipeline.
* [ComponentSpec](https://github.com/kubeflow/pipelines/blob/release-1.0/sdk/python/kfp/components/structures/components.json_schema.json).
* [Core APIs](#core-apis)
* The following SDK client helper methods (others are in Alpha stage)
  * [run_pipeline](https://kubeflow-pipelines.readthedocs.io/en/stable/source/kfp.client.html#kfp.Client.run_pipeline)
  * [create_experiment](https://kubeflow-pipelines.readthedocs.io/en/stable/source/kfp.client.html#kfp.Client.create_experiment)
  * [get_experiment](https://kubeflow-pipelines.readthedocs.io/en/stable/source/kfp.client.html#kfp.Client.get_experiment)
  * [wait_for_run_completion](https://kubeflow-pipelines.readthedocs.io/en/stable/source/kfp.client.html#kfp.Client.wait_for_run_completion)
  * [create_run_from_pipelie_func](https://kubeflow-pipelines.readthedocs.io/en/stable/source/kfp.client.html#kfp.Client.create_run_from_pipeline_func)

Note, these packages are stable in general, but specific classes, methods, and arguments might be in a different stage. For more information, refer to [Kubeflow Pipelines SDK documentation](https://kubeflow-pipelines.readthedocs.io/en/stable/source/kfp.dsl.html).

### Core APIs

|                  | Upload | Create | Get | List | Delete | Archive | Unarchive | Enable | Disable | Terminate | Retry |
|:----------------:|:------:|:------:|:---:|:----:|:------:|:-------:|:---------:|:------:|:-------:|:---------:|:-----:|
| Pipeline         | ☑️  |  ☑️ |☑️|☑️ | ☑️  |         |           |
| Pipeline Version | ☑️  |  ☑️ |☑️|☑️ | ☑️  |         |           |
| Run              |        |  ☑️ |☑️|☑️ | ☑️  | ☑️   | ☑️     |        |         | ☑️     | ☑️ |
| Experiment       |        |  ☑️ |☑️|☑️ | ☑️  | ☑️   | ☑️     |

* Refer to the Kubeflow Pipelines documentation to learn more about:
  - The [purpose and goals of Kubeflow Pipelines](https://www.kubeflow.org/docs/pipelines/overview/pipelines-overview/) and the [main concepts required to use Kubeflow Pipelines](https://www.kubeflow.org/docs/pipelines/overview/concepts/). For example, what is a pipeline and what is a pipeline run in Kubeflow Pipelines.
  - [How to manage Kubeflow Pipelines resources using the the Kubeflow Pipelines UI](https://www.kubeflow.org/docs/pipelines/overview/interfaces/).
  -  [How to manage the Kubeflow Pipelines resources using the Kubeflow Pipelines client](https://www.kubeflow.org/docs/pipelines/tutorials/sdk-examples/).
  - The [Kubeflow Pipelines API](https://www.kubeflow.org/docs/pipelines/reference/api/kubeflow-pipeline-api-spec/). The API reference includes a complete list of the methods and parameters that are available in the Kubeflow Pipelines API.

## Features in Beta

* [Job API](https://www.kubeflow.org/docs/pipelines/reference/api/kubeflow-pipeline-api-spec/#tag-JobService)

  |                  | Upload | Create | Get | List | Delete | Archive | Unarchive | Enable | Disable | Terminate | Retry |
  |:----------------:|:------:|:------:|:---:|:----:|:------:|:-------:|:---------:|:------:|:-------:|:---------:|:-----:|
  | Job              |        |  ☑️ |☑️|☑️ | ☑️  |         |           | ☑️  | ☑️   |


* [Upgrade support for the Kubeflow Pipelines standalone deployment](https://www.kubeflow.org/docs/pipelines/installation/standalone-deployment/#upgrading-kubeflow-pipelines).

* [Built-in visualizations](https://www.kubeflow.org/docs/pipelines/sdk/output-viewer/).

* [Pipeline metrics](https://www.kubeflow.org/docs/pipelines/sdk/pipelines-metrics/).

* [Multi-user isolation](https://www.kubeflow.org/docs/pipelines/multi-user/).

## Features in Alpha

* Most of [SDK client helper methods](https://kubeflow-pipelines.readthedocs.io/en/stable/source/kfp.client.html) except those mentioned in stable stage. The helper methods are mainly maintained by the community. These methods can be convenient to use, but they may not provide all the features from the API and they may lack testing. These issues prevent us from moving these methods to Beta stage at this time. For more information, refer to the [SDK client Beta blockers project](https://github.com/kubeflow/pipelines/projects/7).

  We recommend and support the auto-generated client APIs instead. For example,
  `client.pipelines.list_pipelines()`, `client.runs.list_runs()` and
  `client.pipeline_uploads.upload_pipeline()`.

  For more information, refer to the [Using the Kubeflow Pipelines SDK](https://www.kubeflow.org/docs/pipelines/tutorials/sdk-examples/) guide for examples of using the SDK, and the [SDK generated API client reference](https://kubeflow-pipelines.readthedocs.io/en/stable/source/kfp.server_api.html).

* [Step caching](https://www.kubeflow.org/docs/pipelines/caching/).

* [ML Metadata (MLMD)](https://github.com/google/ml-metadata) UI integration.
  For example, artifact and execution list and detail pages.

* [Python based custom visualizations](https://www.kubeflow.org/docs/pipelines/sdk/python-based-visualizations/).
