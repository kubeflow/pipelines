# Kubeflow Pipelines Feature Stages

The features in Kubeflow Pipelines fall into three different stages: stable, beta and alpha.

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
* No breaking changes without a major version bump.
* The software is recommended for all uses.

## Stable Features

* [SDK DSL](https://github.com/kubeflow/pipelines/tree/master/sdk/python/kfp/dsl) for constructing a pipeline.
* [ComponentSpec](https://github.com/kubeflow/pipelines/blob/release-1.0/sdk/python/kfp/components/structures/components.json_schema.json).

Note, these packages are stable in general, but specific classes, methods, and arguments might be in a different stage. Refer to [their own documentation](https://kubeflow-pipelines.readthedocs.io/en/latest/source/kfp.dsl.html).

### Core APIs

|                  | Upload | Create | Get | List | Delete | Archive | Unarchive | Enable | Disable | Terminate | Retry |
|:----------------:|:------:|:------:|:---:|:----:|:------:|:-------:|:---------:|:------:|:-------:|:---------:|:-----:|
| Pipeline         | ☑️  |  ☑️ |☑️|☑️ | ☑️  |         |           |
| Pipeline Version | ☑️  |  ☑️ |☑️|☑️ | ☑️  |         |           |
| Run              |        |  ☑️ |☑️|☑️ | ☑️  | ☑️   | ☑️     |        |         | ☑️     | ☑️ |
| Experiment       |        |  ☑️ |☑️|☑️ | ☑️  | ☑️   | ☑️     |

* Refer to the Kubeflow Pipelines [introduction]((https://www.kubeflow.org/docs/pipelines/overview/)) for
  - An [overview](https://www.kubeflow.org/docs/pipelines/overview/pipelines-overview/) and [main concepts](https://www.kubeflow.org/docs/pipelines/overview/concepts/), for example, what is a pipeline or run in Kubeflow Pipelines.
  - How to manage the Kubeflow Pipelines resources via the the [Kubeflow Pipelines UI](https://www.kubeflow.org/docs/pipelines/overview/interfaces/).
* Refer to the Kubeflow Pipelines [SDK samples and tutorials](https://www.kubeflow.org/docs/pipelines/tutorials/sdk-examples/)
for how to manage the Kubeflow Pipelines resources via the Kubeflow Pipelines client.
* A complete API method list and a detailed description of parameters to those API methods are available at the Kubeflow Pipelines [API reference](https://www.kubeflow.org/docs/pipelines/reference/api/kubeflow-pipeline-api-spec/).

## Features in Beta

* Job API

  |                  | Upload | Create | Get | List | Delete | Archive | Unarchive | Enable | Disable | Terminate | Retry |
  |:----------------:|:------:|:------:|:---:|:----:|:------:|:-------:|:---------:|:------:|:-------:|:---------:|:-----:|
  | Job              |        |  ☑️ |☑️|☑️ | ☑️  |         |           | ☑️  | ☑️   |


* [Upgrade support for the Kubeflow Pipelines standalone deployment](https://www.kubeflow.org/docs/pipelines/installation/standalone-deployment/#upgrading-kubeflow-pipelines).

* [Built-in Visualizations](https://www.kubeflow.org/docs/pipelines/sdk/output-viewer/).

* [Pipeline Metrics](https://www.kubeflow.org/docs/pipelines/sdk/pipelines-metrics/).

* [Multi-user Isolation](https://www.kubeflow.org/docs/pipelines/multi-user/).

## Features in Alpha

* [SDK client helper methods](https://kubeflow-pipelines.readthedocs.io/en/latest/source/kfp.client.html). The helper methods are mainly maintained by
community. They are convenient to use, but some problems including not providing all features from the API and lack of testing
prevent us moving it to Beta stage right now. Refer to [SDK client Beta blockers project](https://github.com/kubeflow/pipelines/projects/7) for specific issues.

  We recommend and support auto-generated client APIs instead. For example,
  `client.pipelines.list_pipelines()`, `client.runs.list_runs()` and
  `client.pipeline_uploads.upload_pipeline()`.
  
  Refer to [Using the Kubeflow Pipelines SDK](https://www.kubeflow.org/docs/pipelines/tutorials/sdk-examples/) for some example usages and [SDK generated API reference](https://kubeflow-pipelines.readthedocs.io/en/latest/source/kfp.client.html#generated-apis) for exact method signatures.

* [Step caching](https://www.kubeflow.org/docs/pipelines/caching/).

* [ML Metadata (MLMD)](https://github.com/google/ml-metadata) UI integration.
  For example, artifact and execution list and detail pages.

* [Python based custom visualization](https://www.kubeflow.org/docs/pipelines/sdk/python-based-visualizations/).
