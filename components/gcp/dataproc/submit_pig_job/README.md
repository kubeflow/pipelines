
# Dataproc - Submit Pig Job

## Intended Use
A Kubeflow Pipeline component to submit a Pig job on YARN in Google Cloud Dataproc service. 

## Run-Time Parameters:
Name | Description
:--- | :----------
project_id | Required. The ID of the Google Cloud Platform project that the cluster belongs to.
region | Required. The Cloud Dataproc region in which to handle the request.
cluster_name | Required. The cluster to run the job.
queries | Required. The queries to execute. You do not need to terminate a query with a semicolon. Multiple queries can be specified in one string by separating each with a semicolon. 
query_file_uri | The HCFS URI of the script that contains Pig queries.
script_variables | Optional. Mapping of query variable names to values (equivalent to the Pig command: SET name="value";).
pig_job | Optional. The full payload of a [PigJob](https://cloud.google.com/dataproc/docs/reference/rest/v1/PigJob)
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

### Prepare Pig query
Directly put your Pig queries in the `queries` list or upload your Pig queries into a file to a Google Cloud Storage (GCS) bucket and place the path in `query_file_uri`. In this sample, we will use a hard coded query in the `queries` list to select data from a local `passwd` file.

For more details, please checkout [Pig documentation](http://pig.apache.org/docs/latest/)

### Set sample parameters


```python
PROJECT_ID = '<Please put your project ID here>'
CLUSTER_NAME = '<Please put your existing cluster name here>'

REGION = 'us-central1'
QUERY = '''
natality_csv = load 'gs://public-datasets/natality/csv' using PigStorage(':');
top_natality_csv = LIMIT natality_csv 10; 
dump natality_csv;'''
EXPERIMENT_NAME = 'Dataproc - Submit Pig Job'
COMPONENT_SPEC_URI = 'https://raw.githubusercontent.com/kubeflow/pipelines/master/components/gcp/dataproc/submit_pig_job/component.yaml'
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

dataproc_submit_pig_job_op = comp.load_component_from_url(COMPONENT_SPEC_URI)
display(dataproc_submit_pig_job_op)
```

### Here is an illustrative pipeline that uses the component


```python
import kfp.dsl as dsl
import kfp.gcp as gcp
import json
@dsl.pipeline(
    name='Dataproc submit Pig job pipeline',
    description='Dataproc submit Pig job pipeline'
)
def dataproc_submit_pig_job_pipeline(
    project_id = PROJECT_ID, 
    region = REGION,
    cluster_name = CLUSTER_NAME,
    queries = json.dumps([QUERY]),
    query_file_uri = '',
    script_variables = '', 
    pig_job='', 
    job='', 
    wait_interval='30'
):
    dataproc_submit_pig_job_op(project_id, region, cluster_name, queries, query_file_uri,
        script_variables, pig_job, job, wait_interval).apply(gcp.use_gcp_secret('user-gcp-sa'))
    
```

### Compile the pipeline


```python
pipeline_func = dataproc_submit_pig_job_pipeline
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
