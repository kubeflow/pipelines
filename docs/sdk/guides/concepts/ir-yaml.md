# IR YAML

The IR YAML is an intermediate representation of a compiled pipeline or component.
It is an instance of the [`PipelineSpec`][pipeline-spec] protocol buffer message type, which is a platform-agnostic pipeline representation protocol; this makes it possible for pipelines to be submitted on different backends. It is considered an intermediate representation because the KFP backend compiles `PipelineSpec` to [Argo Workflow][argo-workflow] YAML as the final pipeline definition for execution.

Unlike the v1 component YAML, the IR YAML is not intended to be written directly.
While IR YAML is not intended to be easily human-readable, you can still inspect it if you know a bit about its contents:

| Section | Description | Example |
|-------|-------------|---------|
| [`components`][components-schema] | This section is a map of the names of all components used in the pipeline to [`ComponentSpec`][component-spec]. `ComponentSpec` defines the interface, including inputs and outputs, of a component.<br/>For primitive components, `ComponentSpec` contains a reference to the executor containing the component implementation.<br/><br/>For pipelines used as components, `ComponentSpec` contains a [DagSpec][dag-spec] instance, which includes references to the underlying primitive components. | [View on Github][components-example]
| [`deployment_spec`][deployment-spec-schema] | This section contains a map of executor name to [`ExecutorSpec`][executor-spec]. `ExecutorSpec` contains the implementation for a primitive component. | [View on Github][deployment-spec-example]
| [`root`][root-schema] | This section defines the steps of the outermost pipeline definition, also called the pipeline root definition. The root definition is the workflow executed when you submit the IR YAML. It is an instance of [`ComponentSpec`][component-spec]. | [View on Github][root-example]
| [`pipeline_info`][pipeline-info-schema] <a id="kfp_iryaml_pipelineinfo"></a> | This section contains pipeline metadata, including the `pipelineInfo.name` field. This field contains the name of your pipeline template. When you upload your pipeline, a pipeline context name is created based on this template name. The pipeline context lets the backend and the dashboard associate artifacts and executions from pipeline runs using the pipeline template. You can use a pipeline context to determine the best model by comparing metrics and artifacts from multiple pipeline runs based on the same training pipeline. | [View on Github][pipeline-info-example]
| [`sdk_version`][sdk-version-schema] | This section records the version of the KFP SDK used to compile the pipeline. | [View on Github][sdk-version-example]
| [`schema_version`][schema-version-schema] | This section records the version of the `PipelineSpec` schema used for the IR YAML. | [View on Github][schema-version-example]
| [`default_pipeline_root`][default-pipeline-root-schema] | This section records the remote storage root path, such as a SeaweedFS URI or Google Cloud Storage URI, where the pipeline output is written. | [View on Github][default-pipeline-root-example]


## Next steps
* Read an [overview of Kubeflow Pipelines][overview of Kubeflow Pipelines].
* Follow the [pipelines quickstart guide][pipelines quickstart guide]
  to deploy Kubeflow and run a sample pipeline directly from the Kubeflow
  Pipelines UI.


[pipeline-spec]: https://github.com/kubeflow/pipelines/blob/master/api/v2alpha1/pipeline_spec.proto#L50
[argo-workflow]: https://argoproj.github.io/argo-workflows/
[components-schema]: https://github.com/kubeflow/pipelines/blob/41b69fd90da812005965f2209b64fd1278f1cdc9/api/v2alpha1/pipeline_spec.proto#L74-L75
[component-spec]: https://github.com/kubeflow/pipelines/blob/41b69fd90da812005965f2209b64fd1278f1cdc9/api/v2alpha1/pipeline_spec.proto#L85-L96
[components-example]: https://github.com/kubeflow/pipelines/blob/984d8a039d2ff105ca6b21ab26be057b9552b51d/sdk/python/test_data/pipelines/two_step_pipeline.yaml#L1-L21
[deployment-spec-example]: https://github.com/kubeflow/pipelines/blob/984d8a039d2ff105ca6b21ab26be057b9552b51d/sdk/python/test_data/pipelines/two_step_pipeline.yaml#L23-L49
[root-example]: https://github.com/kubeflow/pipelines/blob/984d8a039d2ff105ca6b21ab26be057b9552b51d/sdk/python/test_data/pipelines/two_step_pipeline.yaml#L52-L85
[pipeline-info-example]: https://github.com/kubeflow/pipelines/blob/984d8a039d2ff105ca6b21ab26be057b9552b51d/sdk/python/test_data/pipelines/two_step_pipeline.yaml#L50-L51
[sdk-version-example]: https://github.com/kubeflow/pipelines/blob/984d8a039d2ff105ca6b21ab26be057b9552b51d/sdk/python/test_data/pipelines/two_step_pipeline.yaml#L87
[schema-version-example]: https://github.com/kubeflow/pipelines/blob/984d8a039d2ff105ca6b21ab26be057b9552b51d/sdk/python/test_data/pipelines/two_step_pipeline.yaml#L86
[dag-spec]: https://github.com/kubeflow/pipelines/blob/41b69fd90da812005965f2209b64fd1278f1cdc9/api/v2alpha1/pipeline_spec.proto#L98-L105
[deployment-spec-schema]: https://github.com/kubeflow/pipelines/blob/41b69fd90da812005965f2209b64fd1278f1cdc9/api/v2alpha1/pipeline_spec.proto#L56
[root-schema]: https://github.com/kubeflow/pipelines/blob/41b69fd90da812005965f2209b64fd1278f1cdc9/api/v2alpha1/pipeline_spec.proto#L77-L79
[pipeline-info-schema]: https://github.com/kubeflow/pipelines/blob/41b69fd90da812005965f2209b64fd1278f1cdc9/api/v2alpha1/pipeline_spec.proto#L51-L52
[sdk-version-schema]: https://github.com/kubeflow/pipelines/blob/41b69fd90da812005965f2209b64fd1278f1cdc9/api/v2alpha1/pipeline_spec.proto#L58-L59
[schema-version-schema]: https://github.com/kubeflow/pipelines/blob/41b69fd90da812005965f2209b64fd1278f1cdc9/api/v2alpha1/pipeline_spec.proto#L61-L62
[default-pipeline-root-schema]: https://github.com/kubeflow/pipelines/blob/41b69fd90da812005965f2209b64fd1278f1cdc9/api/v2alpha1/pipeline_spec.proto#L81-L82
[executor-spec]: https://github.com/kubeflow/pipelines/blob/41b69fd90da812005965f2209b64fd1278f1cdc9/api/v2alpha1/pipeline_spec.proto#L788-L803
[default-pipeline-root-example]: https://github.com/kubeflow/pipelines/blob/984d8a039d2ff105ca6b21ab26be057b9552b51d/sdk/python/test_data/pipelines/two_step_pipeline.yaml#L22
[overview of Kubeflow Pipelines]: ../overview.md
[pipelines quickstart guide]: ../getting-started.md
