
# Submitting a PySpark Job to Cloud Dataproc
A Kubeflow Pipeline component to submit a PySpark job to Google Cloud Dataproc service. 

## Intended Use
Use the component to run an Apache PySpark job as one preprocessing step in a KFP pipeline. 

## Runtime arguments
Name | Description | Type | Optional | Default
:--- | :---------- | :--- | :------- | :------
project_id | The Google Cloud Platform (GCP) project ID that the cluster belongs to. | GCPProjectID | No |
region | The Dataproc region that handles the request. | GCPRegion | No |
cluster_name | The name of the cluster that runs the job. | String | No |
main_python_file_uri | The Hadoop Compatible Filesystem (HCFS) URI of the main Python file to use as the driver. Must be a .py file. | GCSPath | No |
args | The arguments to pass to the driver. Do not include arguments, such as --conf, that can be set as job properties, since a collision may occur that causes an incorrect job submission. | List | Yes | `[]`
pyspark_job | The payload of a [PySparkJob](https://cloud.google.com/dataproc/docs/reference/rest/v1/PySparkJob). | Dict | Yes | `{}`
job | The payload of a [Dataproc job](https://cloud.google.com/dataproc/docs/reference/rest/v1/projects.regions.jobs). | Dict | Yes | `{}`
wait_interval | The number of seconds to pause between polling the operation. | Integer | Yes | `30`

## Output
Name | Description | Type
:--- | :---------- | :---
job_id | The ID of the created job. | String

## Cautions & requirements
To use the component, you must:
* Setup project by following the [guide](https://cloud.google.com/dataproc/docs/guides/setup-project).
* [Create a new cluster](https://cloud.google.com/dataproc/docs/guides/create-cluster).
* The component is running under a secret of [Kubeflow user service account](https://www.kubeflow.org/docs/started/getting-started-gke/#gcp-service-accounts) in a Kubeflow cluster. For example:
```
component_op(...).apply(gcp.use_gcp_secret('user-gcp-sa'))
```
* Grant Kubeflow user service account the `roles/dataproc.editor` role on the project.

## Detailed Description
This component creates a PySpark job from [Dataproc submit job REST API](https://cloud.google.com/dataproc/docs/reference/rest/v1/projects.regions.jobs/submit).

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

dataproc_submit_pyspark_job_op = comp.load_component_from_url(
    'https://raw.githubusercontent.com/kubeflow/pipelines/d2f5cc92a46012b9927209e2aaccab70961582dc/components/gcp/dataproc/submit_pyspark_job/component.yaml')
help(dataproc_submit_pyspark_job_op)
```

For more information about the component, please checkout:
* [Component python code](https://github.com/kubeflow/pipelines/blob/master/component_sdk/python/kfp_component/google/dataproc/_submit_pyspark_job.py)
* [Component docker file](https://github.com/kubeflow/pipelines/blob/master/components/gcp/container/Dockerfile)
* [Sample notebook](https://github.com/kubeflow/pipelines/blob/master/components/gcp/dataproc/submit_pyspark_job/sample.ipynb)
* [Dataproc PySparkJob](https://cloud.google.com/dataproc/docs/reference/rest/v1/PySparkJob)

### Sample

Note: the sample code below works in both IPython notebook or python code directly.

#### Setup a Dataproc cluster
[Create a new Dataproc cluster](https://cloud.google.com/dataproc/docs/guides/create-cluster) (or reuse an existing one) before running the sample code.

#### Prepare PySpark job
Upload your PySpark code file to a Cloud Storage bucket. For example, thisis a publicly accessible hello-world.py in Cloud Storage:


```python
!gsutil cat gs://dataproc-examples-2f10d78d114f6aaec76462e3c310f31f/src/pyspark/hello-world/hello-world.py
```

#### Set sample parameters


```python
PROJECT_ID = '<Please put your project ID here>'
CLUSTER_NAME = '<Please put your existing cluster name here>'
REGION = 'us-central1'
PYSPARK_FILE_URI = 'gs://dataproc-examples-2f10d78d114f6aaec76462e3c310f31f/src/pyspark/hello-world/hello-world.py'
ARGS = ''
EXPERIMENT_NAME = 'Dataproc - Submit PySpark Job'
```

#### Example pipeline that uses the component


```python
import kfp.dsl as dsl
import kfp.gcp as gcp
import json
@dsl.pipeline(
    name='Dataproc submit PySpark job pipeline',
    description='Dataproc submit PySpark job pipeline'
)
def dataproc_submit_pyspark_job_pipeline(
    project_id = PROJECT_ID, 
    region = REGION,
    cluster_name = CLUSTER_NAME,
    main_python_file_uri = PYSPARK_FILE_URI, 
    args = ARGS, 
    pyspark_job='{}', 
    job='{}', 
    wait_interval='30'
):
    dataproc_submit_pyspark_job_op(
        project_id=project_id, 
        region=region, 
        cluster_name=cluster_name, 
        main_python_file_uri=main_python_file_uri, 
        args=args, 
        pyspark_job=pyspark_job, 
        job=job, 
        wait_interval=wait_interval).apply(gcp.use_gcp_secret('user-gcp-sa'))
    
```

#### Compile the pipeline


```python
pipeline_func = dataproc_submit_pyspark_job_pipeline
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


```python

```
