
# Creating a Cluster with Cloud Dataproc
A Kubeflow Pipeline component to create a cluster in Cloud Dataproc service.

## Intended Use
This component can be used at the start of a KFP pipeline to create a temporary Dataproc cluster to run Dataproc jobs as subsequent steps in the pipeline. The cluster can be later recycled by the [Dataproc delete cluster component](https://github.com/kubeflow/pipelines/tree/master/components/gcp/dataproc/delete_cluster).


## Runtime arguments
Name | Description | Type | Optional | Default
:--- | :---------- | :--- | :------- | :------
project_id | The Google Cloud Platform (GCP) project ID that the cluster belongs to. | GCPProjectID | No |
region | The Cloud Dataproc region runs the newly created cluster. | GCPRegion | No |
name | The name of the newly created cluster. Cluster names within a project must be unique. Names of deleted clusters can be reused. | String | Yes | ` `
name_prefix | The prefix of the cluster name. | String | Yes | ` `
initialization_actions | List of Cloud Storage URIs of executables to execute on each node after the configuration is completed. By default, executables are run on the master and all the worker nodes. | List | Yes | `[]`
config_bucket | A Cloud Storage bucket used to stage the job dependencies, the configuration files, and the job driver console’s output. | GCSPath | Yes | ` `
image_version | The version of the software inside the cluster. | String | Yes | ` `
cluster | The full [cluster config] (https://cloud.google.com/dataproc/docs/reference/rest/v1/projects.regions.clusters#Cluster). | Dict | Yes | `{}`
wait_interval | The number of seconds to pause between polling the operation done status. | Integer | Yes | `30`

## Output
Name | Description | Type
:--- | :---------- | :---
cluster_name | The cluster name of the created cluster. | String

## Cautions & requirements
To use the component, you must:
* Setup project by following the [guide](https://cloud.google.com/dataproc/docs/guides/setup-project).
* The component is running under a secret of [Kubeflow user service account](https://www.kubeflow.org/docs/started/getting-started-gke/#gcp-service-accounts) in a Kubeflow cluster. For example:
```
component_op(...).apply(gcp.use_gcp_secret('user-gcp-sa'))
```
* Grant Kubeflow user service account the read access to the Cloud Storage buckets which contains initialization action files.
* Grant Kubeflow user service account the `roles/dataproc.editor` role on the project.

## Detailed Description
This component creates a new Dataproc cluster by using [Dataproc create cluster REST API](https://cloud.google.com/dataproc/docs/reference/rest/v1/projects.regions.clusters/create).

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

dataproc_create_cluster_op = comp.load_component_from_url(
    'https://raw.githubusercontent.com/kubeflow/pipelines/d2f5cc92a46012b9927209e2aaccab70961582dc/components/gcp/dataproc/create_cluster/component.yaml')
help(dataproc_create_cluster_op)
```

For more information about the component, please checkout:
* [Component python code](https://github.com/kubeflow/pipelines/blob/master/component_sdk/python/kfp_component/google/dataproc/_create_cluster.py)
* [Component docker file](https://github.com/kubeflow/pipelines/blob/master/components/gcp/container/Dockerfile)
* [Sample notebook](https://github.com/kubeflow/pipelines/blob/master/components/gcp/dataproc/create_cluster/sample.ipynb)
* [Dataproc create cluster REST API](https://cloud.google.com/dataproc/docs/reference/rest/v1/projects.regions.clusters/create)


### Sample

Note: the sample code below works in both IPython notebook or python code directly.

#### Set sample parameters


```python
# Required Parameters
PROJECT_ID = '<Please put your project ID here>'

# Optional Parameters
EXPERIMENT_NAME = 'Dataproc - Create Cluster'
```

#### Example pipeline that uses the component


```python
import kfp.dsl as dsl
import kfp.gcp as gcp
import json
@dsl.pipeline(
    name='Dataproc create cluster pipeline',
    description='Dataproc create cluster pipeline'
)
def dataproc_create_cluster_pipeline(
    project_id = PROJECT_ID, 
    region = 'us-central1', 
    name='', 
    name_prefix='',
    initialization_actions='', 
    config_bucket='', 
    image_version='', 
    cluster='', 
    wait_interval='30'
):
    dataproc_create_cluster_op(
        project_id=project_id, 
        region=region, 
        name=name, 
        name_prefix=name_prefix, 
        initialization_actions=initialization_actions, 
        config_bucket=config_bucket, 
        image_version=image_version, 
        cluster=cluster, 
        wait_interval=wait_interval).apply(gcp.use_gcp_secret('user-gcp-sa'))
```

#### Compile the pipeline


```python
pipeline_func = dataproc_create_cluster_pipeline
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
