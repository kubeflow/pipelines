
# Dataproc - Delete Cluster

## Intended Use
A Kubeflow Pipeline component to delete a cluster in Google Cloud Dataproc service. 

## Run-Time Parameters:
Name | Description
:--- | :----------
project_id | Required. The ID of the Google Cloud Platform project that the cluster belongs to.
region | Required. The Cloud Dataproc region in which to handle the request.
name | Required. The cluster name to delete.
wait_interval | The wait seconds between polling the operation. Defaults to 30s.

## Sample

Note: the sample code below works in both IPython notebook or python code directly.

### Prerequisites

Before running the sample code, you need to [create a Dataproc cluster](https://cloud.google.com/dataproc/docs/guides/create-cluster).

### Set sample parameters


```python
PROJECT_ID = '<Please put your project ID here>'
CLUSTER_NAME = '<Please put your existing cluster name here>'

REGION = 'us-central1'
EXPERIMENT_NAME = 'Dataproc - Delete Cluster'
COMPONENT_SPEC_URI = 'https://raw.githubusercontent.com/kubeflow/pipelines/e5b0081cdcbef6a056c0da114d2eb81ab8d8152d/components/gcp/dataproc/delete_cluster/component.yaml'
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

dataproc_delete_cluster_op = comp.load_component_from_url(COMPONENT_SPEC_URI)
display(dataproc_delete_cluster_op)
```

### Here is an illustrative pipeline that uses the component


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
    dataproc_delete_cluster_op(project_id, region, name).apply(gcp.use_gcp_secret('user-gcp-sa'))
```

### Compile the pipeline


```python
pipeline_func = dataproc_delete_cluster_pipeline
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


```python

```
