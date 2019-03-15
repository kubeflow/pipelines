
# Dataproc - Submit Hive Job

## Intended Use
A Kubeflow Pipeline component to submit a Hive job on YARN in Google Cloud Dataproc service. 

## Run-Time Parameters:
Name | Description
:--- | :----------
project_id | Required. The ID of the Google Cloud Platform project that the cluster belongs to.
region | Required. The Cloud Dataproc region in which to handle the request.
cluster_name | Required. The cluster to run the job.
queries | Required. The queries to execute. You do not need to terminate a query with a semicolon. Multiple queries can be specified in one string by separating each with a semicolon. 
query_file_uri | The HCFS URI of the script that contains Hive queries.
script_variables | Optional. Mapping of query variable names to values (equivalent to the Hive command: SET name="value";).
hive_job | Optional. The full payload of a [HiveJob](https://cloud.google.com/dataproc/docs/reference/rest/v1/HiveJob)
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

### Prepare Hive query
Directly put your Hive queries in the `queries` list or upload your Hive queries into a file to a Google Cloud Storage (GCS) bucket and place the path in `query_file_uri`. In this sample, we will use a hard coded query in the `queries` list to select data from a public CSV file from GCS.

For more details, please checkout [Hive language manual](https://cwiki.apache.org/confluence/display/Hive/LanguageManual)

### Set sample parameters


```python
PROJECT_ID = '<Please put your project ID here>'
CLUSTER_NAME = '<Please put your existing cluster name here>'
REGION = 'us-central1'
QUERY = '''
DROP TABLE IF EXISTS natality_csv;
CREATE EXTERNAL TABLE natality_csv (
  source_year BIGINT, year BIGINT, month BIGINT, day BIGINT, wday BIGINT,
  state STRING, is_male BOOLEAN, child_race BIGINT, weight_pounds FLOAT,
  plurality BIGINT, apgar_1min BIGINT, apgar_5min BIGINT,
  mother_residence_state STRING, mother_race BIGINT, mother_age BIGINT,
  gestation_weeks BIGINT, lmp STRING, mother_married BOOLEAN,
  mother_birth_state STRING, cigarette_use BOOLEAN, cigarettes_per_day BIGINT,
  alcohol_use BOOLEAN, drinks_per_week BIGINT, weight_gain_pounds BIGINT,
  born_alive_alive BIGINT, born_alive_dead BIGINT, born_dead BIGINT,
  ever_born BIGINT, father_race BIGINT, father_age BIGINT,
  record_weight BIGINT
)
ROW FORMAT DELIMITED FIELDS TERMINATED BY ','
LOCATION 'gs://public-datasets/natality/csv';

SELECT * FROM natality_csv LIMIT 10;'''
EXPERIMENT_NAME = 'Dataproc - Submit SparkSQL Job'
COMPONENT_SPEC_URI = 'https://raw.githubusercontent.com/kubeflow/pipelines/master/components/gcp/dataproc/submit_hive_job/component.yaml'
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

dataproc_submit_hive_job_op = comp.load_component_from_url(COMPONENT_SPEC_URI)
display(dataproc_submit_hive_job_op)
```

### Here is an illustrative pipeline that uses the component


```python
import kfp.dsl as dsl
import kfp.gcp as gcp
import json
@dsl.pipeline(
    name='Dataproc submit Hive job pipeline',
    description='Dataproc submit Hive job pipeline'
)
def dataproc_submit_hive_job_pipeline(
    project_id = PROJECT_ID, 
    region = REGION,
    cluster_name = CLUSTER_NAME,
    queries = json.dumps([QUERY]),
    query_file_uri = '',
    script_variables = '', 
    sparksql_job='', 
    job='', 
    wait_interval='30'
):
    dataproc_submit_hive_job_op(project_id, region, cluster_name, queries, query_file_uri,
        script_variables, sparksql_job, job, wait_interval).apply(gcp.use_gcp_secret('user-gcp-sa'))
    
```

### Compile the pipeline


```python
pipeline_func = dataproc_submit_hive_job_pipeline
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
