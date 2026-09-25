# Output Artifact

An *output artifact* is an output emitted by a pipeline component, which the
Kubeflow Pipelines UI understands and can render as rich visualizations. It’s
useful for pipeline components to include artifacts so that you can provide for
performance evaluation, quick decision making for the run, or comparison across
different runs. Artifacts also make it possible to understand how the pipeline’s
various components work. An artifact can range from a plain textual view of the
data to rich interactive visualizations.

## Migrating from the Python visualization server

The Python visualization server and its public
`POST /apis/v2beta1/visualizations/{namespace}` endpoint have been removed,
including the `VisualizationService.CreateVisualizationV1` RPC and generated
clients. The `ALLOW_CUSTOM_VISUALIZATIONS` UI setting and
`GET /visualizations/allowed` route are also removed.

Instead of asking the server to execute Python, generate HTML within a pipeline
component and write it to a `dsl.HTML` output:

```python
from kfp import dsl


@dsl.component
def create_report(report: dsl.Output[dsl.HTML]):
    with open(report.path, 'w') as output:
        output.write('<html><body><h1>Evaluation report</h1></body></html>')
```

Install any plotting, TFDV, or TFMA dependencies in the component's image, and
export a self-contained HTML report there. The UI displays the resulting
artifact without a separate visualization service. Scalar metrics, ROC curves,
confusion matrices, HTML/Markdown artifacts, supported legacy UI metadata
viewers, and TensorBoard retain their existing rendering paths.

### Deployment cleanup

New manifests no longer deploy `ml-pipeline-visualizationserver`. Applying
updated YAML with `kubectl apply` alone does not delete resources omitted from
it. After migrating direct API consumers, remove the old Deployment, Service,
and ServiceAccount named `ml-pipeline-visualizationserver` from the KFP namespace.
For Istio installations, also remove its AuthorizationPolicy and DestinationRule.
Check for copies in user namespaces left by older profile controllers or custom
deployments. Remove service-specific image overrides, environment variables,
NodePorts, and RBAC grants from custom overlays. Do not remove the TensorBoard
viewer controller or artifact-proxy deployments.

## Next steps

* Read an [overview of Kubeflow Pipelines](../overview.md).
* Follow the [pipelines quickstart guide](../getting-started.md)
  to deploy Kubeflow and run a sample pipeline directly from the Kubeflow
  Pipelines UI.
* Read more about the available
  [output viewers](../user-guides/data-handling/artifacts.md)
  and how to provide the metadata to make use of the visualizations
  that the output viewers provide.
