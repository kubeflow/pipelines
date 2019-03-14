
# Dataproc - Submit PySpark Job

## Intended Use
A Kubeflow Pipeline component to submit a PySpark job to Google Cloud Dataproc service. 

## Run-Time Parameters:
Name | Description
:--- | :----------
project_id | Required. The ID of the Google Cloud Platform project that the cluster belongs to.
region | Required. The Cloud Dataproc region in which to handle the request.
cluster_name | Required. The cluster to run the job.
main_python_file_uri | Required. The HCFS URI of the main Python file to use as the driver. Must be a .py file.
args | Optional. The arguments to pass to the driver. Do not include arguments, such as --conf, that can be set as job properties, since a collision may occur that causes an incorrect job submission.
pyspark_job | Optional. The full payload of a [PySparkJob](https://cloud.google.com/dataproc/docs/reference/rest/v1/PySparkJob).
job | Optional. The full payload of a [Dataproc job](https://cloud.google.com/dataproc/docs/reference/rest/v1/projects.regions.jobs).
wait_interval | Optional. The wait seconds between polling the operation. Defaults to 30s.

## Output:
Name | Description
:--- | :----------
job_id | The ID of the created job.

## Sample

Note: the sample code below works in both IPython notebook or python code directly.

### Setup a Dataproc cluster
Follow the [guide](https://cloud.google.com/dataproc/docs/guides/create-cluster) to create a new Dataproc cluster or reuse an existing one.

### Prepare PySpark job
Upload your PySpark code file to a Google Cloud Storage (GCS) bucket. For example, here is a public accessible hello-world.py in GCS:


```python
!gsutil cat gs://dataproc-examples-2f10d78d114f6aaec76462e3c310f31f/src/pyspark/hello-world/hello-world.py
```

### Set sample parameters


```python
PROJECT_ID = '<Please put your project ID here>'
CLUSTER_NAME = '<Please put your existing cluster name here>'
REGION = 'us-central1'
PYSPARK_FILE_URI = 'gs://dataproc-examples-2f10d78d114f6aaec76462e3c310f31f/src/pyspark/hello-world/hello-world.py'
ARGS = ''
EXPERIMENT_NAME = 'Dataproc - Submit PySpark Job'
SUBMIT_PYSPARK_JOB_SPEC_URI = 'https://raw.githubusercontent.com/kubeflow/pipelines/e5b0081cdcbef6a056c0da114d2eb81ab8d8152d/components/gcp/dataproc/submit_pyspark_job/component.yaml'
```

### Install KFP SDK
Install the SDK (Uncomment the code if the SDK is not installed before)


```python
# KFP_PACKAGE = 'https://storage.googleapis.com/ml-pipeline/release/0.1.12/kfp.tar.gz'
# !pip3 install $KFP_PACKAGE --upgrade
```

### Load component definitions


```python
import kfp.components as comp

dataproc_submit_pyspark_job_op = comp.load_component_from_url(SUBMIT_PYSPARK_JOB_SPEC_URI)
display(dataproc_submit_pyspark_job_op)
```

### Here is an illustrative pipeline that uses the component


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
    dataproc_submit_pyspark_job_op(project_id, region, cluster_name, main_python_file_uri, 
        args, pyspark_job, job, wait_interval).apply(gcp.use_gcp_secret('user-gcp-sa'))
    
```

### Compile the pipeline


```python
pipeline_func = dataproc_submit_pyspark_job_pipeline
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
