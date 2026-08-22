# Output Artifact

An *output artifact* is an output emitted by a pipeline component, which the
Kubeflow Pipelines UI understands and can render as rich visualizations. It’s
useful for pipeline components to include artifacts so that you can provide for
performance evaluation, quick decision making for the run, or comparison across
different runs. Artifacts also make it possible to understand how the pipeline’s
various components work. An artifact can range from a plain textual view of the
data to rich interactive visualizations.

## Next steps

* Read an [overview of Kubeflow Pipelines](../overview.md).
* Follow the [pipelines quickstart guide](../getting-started.md)
  to deploy Kubeflow and run a sample pipeline directly from the Kubeflow
  Pipelines UI.
* Read more about the available
  [output viewers](https://www.kubeflow.org/docs/components/pipelines/legacy-v1/sdk/output-viewer)
  and how to provide the metadata to make use of the visualizations
  that the output viewers provide.
