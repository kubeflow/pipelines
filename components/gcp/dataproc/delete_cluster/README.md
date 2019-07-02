
# Name

Data preparation by deleting a cluster in Cloud Dataproc

# Label
Cloud Dataproc, cluster, GCP, Cloud Storage, Kubeflow, Pipeline


# Summary
A Kubeflow Pipeline component to delete a cluster in Cloud Dataproc.

## Intended use
Use this component at the start of a Kubeflow Pipeline to delete a temporary Cloud Dataproc cluster to run Cloud Dataproc jobs as steps in the pipeline. This component is usually used with an [exit handler](https://github.com/kubeflow/pipelines/blob/master/samples/basic/exit_handler.py) to run at the end of a pipeline.


## Runtime arguments
| Argument | Description | Optional | Data type | Accepted values | Default |
|----------|-------------|----------|-----------|-----------------|---------|
| project_id | The Google Cloud Platform (GCP) project ID that the cluster belongs to. | No | GCPProjectID |  |  |
| region | The Cloud Dataproc region in which to handle the request. | No | GCPRegion |  |  |
| name | The name of the cluster to delete. | No | String |  |  |
| wait_interval | The number of seconds to pause between polling the operation. | Yes | Integer |  | 30 |


## Cautions & requirements
To use the component, you must:
*   Set up a GCP project by following this [guide](https://cloud.google.com/dataproc/docs/guides/setup-project).
*   Run the component under a secret [Kubeflow user service account](https://www.kubeflow.org/docs/started/getting-started-gke/#gcp-service-accounts) in a Kubeflow cluster. For example:

    ```
    component_op(...).apply(gcp.use_gcp_secret('user-gcp-sa'))
    ```
*   Grant the Kubeflow user service account the role `roles/dataproc.editor` on the project.

## Detailed description
This component deletes a Dataproc cluster by using [Dataproc delete cluster REST API](https://cloud.google.com/dataproc/docs/reference/rest/v1/projects.regions.clusters/delete).

Follow these steps to use the component in a pipeline:
1.  Install the Kubeflow Pipeline SDK:


```python
%%capture --no-stderr

KFP_PACKAGE = 'https://storage.googleapis.com/ml-pipeline/release/0.1.14/kfp.tar.gz'
!pip3 install $KFP_PACKAGE --upgrade
```

2. Load the component using KFP SDK


```python
import kfp.components as comp

dataproc_delete_cluster_op = comp.load_component_from_url(
    'https://raw.githubusercontent.com/kubeflow/pipelines/2e52e54166795d20e92d287bde7b800b181eda02/components/gcp/dataproc/delete_cluster/component.yaml')
help(dataproc_delete_cluster_op)
```

### Sample

Note: The following sample code works in an IPython notebook or directly in Python code. See the sample code below to learn how to execute the template.

#### Prerequisites

[Create a Dataproc cluster](https://cloud.google.com/dataproc/docs/guides/create-cluster) before running the sample code.

#### Set sample parameters


```python
PROJECT_ID = '<Please put your project ID here>'
CLUSTER_NAME = '<Please put your existing cluster name here>'

REGION = 'us-central1'
EXPERIMENT_NAME = 'Dataproc - Delete Cluster'
```

#### Example pipeline that uses the component


```python
import kfp.dsl as dsl
import kfp.gcp as gcp
import json
@dsl.pipeline(
    name='Dataproc delete cluster pipeline',
    description='Dataproc delete cluster pipeline'
)
def dataproc_delete_cluster_pipeline(
    project_id = PROJECT_ID, 
    region = REGION,
    name = CLUSTER_NAME
):
    dataproc_delete_cluster_op(
        project_id=project_id, 
        region=region, 
        name=name).apply(gcp.use_gcp_secret('user-gcp-sa'))
```

#### Compile the pipeline


```python
pipeline_func = dataproc_delete_cluster_pipeline
pipeline_filename = pipeline_func.__name__ + '.zip'
import kfp.compiler as compiler
compiler.Compiler().compile(pipeline_func, pipeline_filename)
```

#### Submit the pipeline for execution


```python
#Specify pipeline argument values
arguments = {}

#Get or create an experiment and submit a pipeline run
import kfp
client = kfp.Client()
experiment = client.create_experiment(EXPERIMENT_NAME)

#Submit a pipeline run
run_name = pipeline_func.__name__ + ' run'
run_result = client.run_pipeline(experiment.id, run_name, pipeline_filename, arguments)
```

## References

*   [Component Python code](https://github.com/kubeflow/pipelines/blob/master/component_sdk/python/kfp_component/google/dataproc/_delete_cluster.py)
*   [Component Docker file](https://github.com/kubeflow/pipelines/blob/master/components/gcp/container/Dockerfile)
*   [Sample notebook](https://github.com/kubeflow/pipelines/blob/master/components/gcp/dataproc/delete_cluster/sample.ipynb)
*   [Dataproc delete cluster REST API](https://cloud.google.com/dataproc/docs/reference/rest/v1/projects.regions.clusters/delete)


## License
By deploying or using this software you agree to comply with the [AI Hub Terms of Service](https://aihub.cloud.google.com/u/0/aihub-tos) and the [Google APIs Terms of Service](https://developers.google.com/terms/). To the extent of a direct conflict of terms, the AI Hub Terms of Service will control.
