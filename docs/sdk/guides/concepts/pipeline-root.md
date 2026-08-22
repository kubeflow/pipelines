# Pipeline Root

A *pipeline root* represents a path within an object store bucket (SeaweedFS, S3, GCS) where Kubeflow Pipelines stores [artifacts][artifact] from pipeline runs. Pipeline roots can be set at the cluster, [pipeline][pipeline], and [run][run] level, with support for authentication and overrides for specific paths provided.

:::{note}
It's important to understand how pipeline roots fit in KFP's data ecosystem. Pipeline roots are KFP's way to store [artifacts][artifact] (user data files) from [runs][run] in particular. Metadata on these [artifacts][artifact] (including their storage paths) is stored in an SQL database. Independently--and not to be confused with pipeline roots--KFP uses another object storage specification (in `Deployment/ml-pipeline`) to support operations of the KFP API server; more information on this backend specification can be found [here][API Server Storage].
:::

## The Why Behind Pipeline Roots
Machine learning workflows are highly iterative, and they tend to produce a lot of artifacts. These artifacts must be stored and tracked in connection with their workflows, so that ML engineers can access outputs and compare results.

KFP provides support for default pipeline root specification at several levels, as well as override capabilities; this way, ML engineers can spend less time on storage specifications. On the other hand, if they need to customize, they have the opportunity to do this at either pipeline specification or pipeline run time. Meanwhile, MLOps administrators also get the tools they need to  centrally manage the foundations of where and how data gets stored. The flexibility in the level of pipeline root options KFP provides, means ML engineers have support to operate pipelines in different cluster environments with different storage requirements, which can be common during ML development.

Note that while nothing is stopping ML engineers from creating components with their own object store persistence capabilities (for example, a component with internal code that writes directly to S3), such implementations will not benefit from the operational support for [artifact][artifact] outputs that KFP provides.


## Pipeline Root Implementation

1. **KFP Cluster Level**

    The default pipeline root can be set at the cluster level via the Kubernetes `ConfigMap/kfp-launcher` resource; this can be done at KFP deployment, or as an update, by setting the `data.defaultPipelineRoot` path.

    The out-of-the-box default setting for the pipeline root (in the manifests) is:
    ```yaml
    data:
      defaultPipelineRoot: "minio://mlpipeline/v2/artifacts"
    ```

    The `ConfigMap/kfp-launcher` can also be given pipeline root authentication details (multiple auth types supported), as well as override details (for more specific paths). For more details on setting this up, see the [Object Store Configuration: KFP Launcher Object Store Configuration][ConfigMap-kfp-launcher-config] page.

    A basic pipeline root setup at the cluster level `ConfigMap/kfp-launcher` can be done by an MLOps administrator for KFP. From this central location, they can manage the general options that are available for downstream ML engineer users to support different use-cases. For example, an MLOps administrator might implement the following `defaultPipelineRoot` settings:
    ```yaml
    apiVersion: v1
    kind: ConfigMap
    metadata:
      name: kfp-launcher
      namespace: user-namespace
    data:
      defaultPipelineRoot: gs://ml-models/
      providers: |-
          gs:
            default:
              credentials:
                fromEnv: false
                secretRef:
                  secretName: gs-secret-1
                  tokenKey: gs-tokenKey
            overrides:
            # Matches pipeline root: gs://ml-models/fraud-models/
              - bucketName: ml-models
                keyPrefix: fraud-models
                credentials:
                  fromEnv: true
    ```
    In this example, the `defaultPipelineRoot` is set to `gs://ml-models`, and authentication uses a secret and token. However, in the specific case of [runs][run] using the `gs://ml-models/fraud-models` directory as the pipeline root, a different set of credentials will be required, which will be expected to be supplied in the environment at runtime. (Presumably, in this example fraud-model artifacts are more sensitive or specific, so authentication is done differently here).

    Note that KFP does not create or configure cloud resources like buckets and IAM policies. The `ConfigMap/kfp-launcher` configuration is meant to use cloud resources that are assumed to already exist.

2. **KFP Pipeline Level**

    The KFP Python SDK also provides functionality for ML Engineers to set the `pipeline_root` parameter when creating a [pipeline][pipeline] via the `@dsl.pipeline` decorator. When set, this parameter overrides cluster-level default settings for the pipeline root when the [pipeline][pipeline] is run.
    ```python
    # example
    from kfp.dsl import pipeline
    ...
    @pipeline(
        name: "ranking-model-trainer",
        definition: "rank model pipeline for recommendation",
        pipeline_root: "gs://ml-models/recommendation/rank-model/"
    )
    def rank_model_pipeline(...):
        ...
    ```
    In the example above, an ML engineer sets the pipeline root to `gs://ml-models/recommendation/rank-model/`. This gives them control over the location of artifacts for this pipeline, while potentially also relying on the cluster-level settings for authentication.

3. **KFP Run Level**

    ML Engineers can also override the cluster and pipeline level settings for the pipeline root at run time. This can be done during [run][run] submission:
    ```python
    # example
    ...
    from kfp.client import Client

    cl = Client()

    cl.create_run_from_pipeline_func(
        pipeline_func=rank_model_pipeline,
        pipeline_root="gs://ml-models/recommendation/sandbox"
        ...
    )
    ```
    In this example, the ML engineer overrides the cluster-level and pipeline-level settings with a pipeline root of `gs://ml-models/recommendation/sandbox`. Presumably, the `sandbox` folder here is useful for testing outside of official runs.

    Finally, setting the pipeline root can also be done similarly through the UI when submitting a pipeline run by setting the `pipeline-root` "Run Parameter".

<!-- TODO: there is interest in profile-level settings for pipeline-root, which is not well documented. Update here when this is added
https://github.com/kubeflow/pipelines/issues/8406 -->

For more details on setting up pipeline root defaults and overrides at the cluster level, see the [Kubeflow Pipelines Deployment Guide][Kubeflow Pipelines Deployment
Guide]. Also, consult the [Pipeline Root Configuration Guide][Pipeline Root Guide] for more details on setting up authentication, configuration, and usage.


## Next steps
* Read an [overview of Kubeflow Pipelines][overview of Kubeflow Pipelines].
* Follow the [pipelines quickstart guide][pipelines quickstart guide]
  to deploy Kubeflow and run a sample pipeline directly from the Kubeflow
  Pipelines UI.


[artifact]: output-artifact.md
[run]: run.md
[pipeline]: pipeline.md
[Pipeline Root Guide]: ../user-guides/data-handling/pipeline-root.md
[Kubeflow Pipelines deployment guide]: ../operator-guides/installation/index.md
[API Server Storage]: ../operator-guides/configure-object-store.md#kfp-api-server
[ConfigMap-kfp-launcher-config]: ../operator-guides/configure-object-store.md#kfp-launcher-object-store-configuration
[overview of Kubeflow Pipelines]: ../overview.md
[pipelines quickstart guide]: ../getting-started.md
