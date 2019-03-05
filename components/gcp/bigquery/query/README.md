
# Bigquery - Query

## Intended Use
A Kubeflow Pipeline component to submit a query to Google Cloud Bigquery service and dump outputs to a Google Cloud Storage blob. 

## Run-Time Parameters:
Name | Description
:--- | :----------
query | The query used by Bigquery service to fetch the results.
project_id | The project to execute the query job.
dataset_id | The ID of the persistent dataset to keep the results of the query. If the dataset does not exist, the operation will create a new one.
table_id | The ID of the table to keep the results of the query. If absent, the operation will generate a random id for the table.
output_gcs_path | The GCS blob path to dump the query results to.
dataset_location | The location to create the dataset. Defaults to `US`.
job_config | The full config spec for the query job. See [QueryJobConfig](https://googleapis.github.io/google-cloud-python/latest/bigquery/generated/google.cloud.bigquery.job.QueryJobConfig.html#google.cloud.bigquery.job.QueryJobConfig) for details.

## Output:
Name | Description
:--- | :----------
output_gcs_path | The GCS blob path to dump the query results to.

## Sample

Note: the sample code below works in both IPython notebook or python code directly.

### Set sample parameters


```python
# Required Parameters
PROJECT_ID = '<Please put your project ID here>'
GCS_WORKING_DIR = 'gs://<Please put your GCS path here>' # No ending slash

# Optional Parameters
EXPERIMENT_NAME = 'Bigquery -Query'
COMPONENT_SPEC_URI = 'https://raw.githubusercontent.com/kubeflow/pipelines/master/components/gcp/bigquery/query/component.yaml'
```

### Install KFP SDK


```python
# Install the SDK (Uncomment the code if the SDK is not installed before)
# KFP_PACKAGE = 'https://storage.googleapis.com/ml-pipeline/release/0.1.11/kfp.tar.gz'
# !pip3 install $KFP_PACKAGE --upgrade
```

### Load component definitions


```python
import kfp.components as comp

bigquery_query_op = comp.load_component_from_url(COMPONENT_SPEC_URI)
display(bigquery_query_op)
```

### Here is an illustrative pipeline that uses the component


```python
import kfp.dsl as dsl
import kfp.gcp as gcp
import json
@dsl.pipeline(
    name='Bigquery query pipeline',
    description='Bigquery query pipeline'
)
def pipeline(
    query, 
    project_id, 
    dataset_id='', 
    table_id='', 
    output_gcs_path='', 
    dataset_location='US', 
    job_config=''
):
    bigquery_query_op(query, project_id, dataset_id, table_id, output_gcs_path, dataset_location, 
        job_config).apply(gcp.use_gcp_secret('user-gcp-sa'))
```

### Compile the pipeline


```python
pipeline_func = pipeline
pipeline_filename = pipeline_func.__name__ + '.pipeline.tar.gz'
import kfp.compiler as compiler
compiler.Compiler().compile(pipeline_func, pipeline_filename)
```

### Submit the pipeline for execution


```python
#Specify pipeline argument values
arguments = {
    'query': 'SELECT * FROM `bigquery-public-data.stackoverflow.posts_questions` LIMIT 10',
    'project_id': PROJECT_ID,
    'output_gcs_path': '{}/bigquery/query/questions.csv'.format(GCS_WORKING_DIR)
}

#Get or create an experiment and submit a pipeline run
import kfp
client = kfp.Client()
experiment = client.create_experiment(EXPERIMENT_NAME)

#Submit a pipeline run
run_name = pipeline_func.__name__ + ' run'
run_result = client.run_pipeline(experiment.id, run_name, pipeline_filename, arguments)
```
