
# Deleting a Cluster with Cloud Dataproc
A Kubeflow Pipeline component to delete a cluster in Cloud Dataproc service.

## Intended Use
Use the component to recycle a Dataproc cluster as one of the step in a KFP pipeline. This component is usually used with an [exit handler](https://github.com/kubeflow/pipelines/blob/master/samples/basic/exit_handler.py) to run at the end of a pipeline.

## Runtime arguments
Name | Description | Type | Optional | Default
:--- | :---------- | :--- | :------- | :------
project_id | The Google Cloud Platform (GCP) project ID that the cluster belongs to. | GCPProjectID | No |
region | The Cloud Dataproc region runs the cluster to delete. | GCPRegion | No |
name | The cluster name to delete. | String | No |
wait_interval | The number of seconds to pause between polling the delete operation done status. | Integer | Yes | `30`

## Cautions & requirements
To use the component, you must:
* Setup project by following the [guide](https://cloud.google.com/dataproc/docs/guides/setup-project).
* The component is running under a secret of [Kubeflow user service account](https://www.kubeflow.org/docs/started/getting-started-gke/#gcp-service-accounts) in a Kubeflow cluster. For example:
```
component_op(...).apply(gcp.use_gcp_secret('user-gcp-sa'))
```
* Grant Kubeflow user service account the `roles/dataproc.editor` role on the project.

## Detailed Description
This component deletes a Dataproc cluster by using [Dataproc delete cluster REST API](https://cloud.google.com/dataproc/docs/reference/rest/v1/projects.regions.clusters/delete).

Here are the steps to use the component in a pipeline:
1. Install KFP SDK



```python
%%capture --no-stderr

KFP_PACKAGE = 'https://storage.googleapis.com/ml-pipeline/release/0.1.14/kfp.tar.gz'
!pip3 install $KFP_PACKAGE --upgrade
```

2. Load the component using KFP SDK


```python
import kfp.components as comp

dataproc_delete_cluster_op = comp.load_component_from_url(
    'https://raw.githubusercontent.com/kubeflow/pipelines/d2f5cc92a46012b9927209e2aaccab70961582dc/components/gcp/dataproc/delete_cluster/component.yaml')
help(dataproc_delete_cluster_op)
```

For more information about the component, please checkout:
* [Component python code](https://github.com/kubeflow/pipelines/blob/master/component_sdk/python/kfp_component/google/dataproc/_delete_cluster.py)
* [Component docker file](https://github.com/kubeflow/pipelines/blob/master/components/gcp/container/Dockerfile)
* [Sample notebook](https://github.com/kubeflow/pipelines/blob/master/components/gcp/dataproc/delete_cluster/sample.ipynb)
* [Dataproc delete cluster REST API](https://cloud.google.com/dataproc/docs/reference/rest/v1/projects.regions.clusters/delete)


### Sample

Note: the sample code below works in both IPython notebook or python code directly.

#### Prerequisites

Before running the sample code, you need to [create a Dataproc cluster](https://cloud.google.com/dataproc/docs/guides/create-cluster).

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
