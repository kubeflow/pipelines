
# Dataproc - Create Cluster

## Intended Use
A Kubeflow Pipeline component to create a cluster in Google Cloud Dataproc service. 

## Run-Time Parameters:
Name | Description
:--- | :----------
project_id | Required. The ID of the Google Cloud Platform project that the cluster belongs to.
region | Required. The Cloud Dataproc region in which to handle the request.
name | Optional. The cluster name. Cluster names within a project must be unique. Names of deleted clusters can be reused.
name_prefix | Optional. The prefix of the cluster name.
initialization_actions | Optional. List of GCS URIs of executables to execute on each node after config is completed. By default, executables are run on master and all worker nodes. 
config_bucket | Optional. A Google Cloud Storage bucket used to stage job dependencies, config files, and job driver console output.
image_version | Optional. The version of software inside the cluster.
cluster | Optional. The full [cluster config](https://cloud.google.com/dataproc/docs/reference/rest/v1/projects.regions.clusters#Cluster)
wait_interval | The wait seconds between polling the operation. Defaults to 30s.

## Output:
Name | Description
:--- | :----------
cluster_name | The cluster name of the created cluster.

## Sample

Note: the sample code below works in both IPython notebook or python code directly.

### Set sample parameters


```python
# Required Parameters
PROJECT_ID = '<Please put your project ID here>'

# Optional Parameters
EXPERIMENT_NAME = 'Dataproc - Create Cluster'
COMPONENT_SPEC_URI = 'https://raw.githubusercontent.com/kubeflow/pipelines/e5b0081cdcbef6a056c0da114d2eb81ab8d8152d/components/gcp/dataproc/create_cluster/component.yaml'
```

### Install KFP SDK
Install the SDK (Uncomment the code if the SDK is not installed before)


```python
#KFP_PACKAGE = 'https://storage.googleapis.com/ml-pipeline/release/0.1.12/kfp.tar.gz'
#!pip3 install $KFP_PACKAGE --upgrade
```

### Load component definitions


```python
import kfp.components as comp

dataproc_create_cluster_op = comp.load_component_from_url(COMPONENT_SPEC_URI)
display(dataproc_create_cluster_op)
```

### Here is an illustrative pipeline that uses the component


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
    job_name_prefix='', 
    initialization_actions='', 
    config_bucket='', 
    image_version='', 
    cluster='', 
    wait_interval='30'
):
    dataproc_create_cluster_op(project_id, region, name, name_prefix, job_name_prefix, initialization_actions, 
       config_bucket, image_version, cluster, wait_interval).apply(gcp.use_gcp_secret('user-gcp-sa'))
```

### Compile the pipeline


```python
pipeline_func = dataproc_create_cluster_pipeline
pipeline_filename = pipeline_func.__name__ + '.pipeline.tar.gz'
import kfp.compiler as compiler
compiler.Compiler().compile(pipeline_func, pipeline_filename)
```

### Submit the pipeline for execution


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
