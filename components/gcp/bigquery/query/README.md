
# Name
Gather training data by querying BigQuery 

# Labels
GCP, BigQuery, Kubeflow, Pipeline

# Summary
A Kubeflow Pipeline component to submit a query to BigQuery and store the result in a Cloud Storage bucket.

# Details
## Intended Use
Use this Kubeflow component to:
* Select training data by submitting a query to BigQuery.
* Output the training data into a Cloud Storage bucket as CSV files.


## Runtime arguments
Name | Description | Data type | Optional | Default
:--- | :---------- | :-------- | :------- | :------
query | The query used by BigQuery to fetch the results. | String | No |
project_id | The project ID of the Google Cloud Platform (GCP)  project to use to execute the query. | GCPProjectID | No |
dataset_id | The ID of the persistent BigQuery dataset to store the results of the query. If the dataset does not exist, the operation will create a new one. | String | Yes | ` `
table_id | The ID of the BigQuery table to store the results of the query. If the table ID is absent, the operation will generate a random ID for the table. | String | Yes | ` `
output_gcs_path | The path to the Cloud Storage bucket to store the query output. | GCSPath | Yes | ` `
dataset_location | The location to create the dataset. Defaults to `US`. | String | Yes | `US`
job_config | The full config spec for the query job. See [QueryJobConfig](https://googleapis.github.io/google-cloud-python/latest/bigquery/generated/google.cloud.bigquery.job.QueryJobConfig.html#google.cloud.bigquery.job.QueryJobConfig) for details. | Dict | Yes | ` `


## Outputs
Name | Description | Type
:--- | :---------- | :---
output_gcs_path | The path to the Cloud Storage bucket containing the query output in CSV format. | GCSPath

## Cautions and requirements
To use the component, the following requirements must be met:
* BigQuery API is enabled
* The component is running under a secret of [Kubeflow user service account](https://www.kubeflow.org/docs/started/getting-started-gke/#gcp-service-accounts) in a Kubeflow cluster. For example:

```python
bigquery_query_op(...).apply(gcp.use_gcp_secret('user-gcp-sa'))

```

* The Kubeflow user service account is a member of `roles/bigquery.admin` role of the project.
* The Kubeflow user service account is also a member of `roles/storage.objectCreator` role of the Cloud Storage output bucket.

## Detailed Description
This Kubeflow Pipeline component is used to:
* Submit a query to BigQuery.
  * The query results are persisted in a dataset table in BigQuery.
  * An extract job is created in BigQuery to extract the data from the dataset table and output it to a Cloud Storage bucket as CSV files.

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

bigquery_query_op = comp.load_component_from_url(
    'https://raw.githubusercontent.com/kubeflow/pipelines/d2f5cc92a46012b9927209e2aaccab70961582dc/components/gcp/bigquery/query/component.yaml')
help(bigquery_query_op)
```

### Sample

Note: The following sample code works in IPython notebook or directly in Python code.

In this sample, we send a query to get the top questions from stackdriver public data and output the data to a Cloud Storage bucket. Here is the query:


```python
QUERY = 'SELECT * FROM `bigquery-public-data.stackoverflow.posts_questions` LIMIT 10'
```

#### Set sample parameters


```python
# Required Parameters
PROJECT_ID = '<Please put your project ID here>'
GCS_WORKING_DIR = 'gs://<Please put your GCS path here>' # No ending slash
```


```python
# Optional Parameters
EXPERIMENT_NAME = 'Bigquery -Query'
OUTPUT_PATH = '{}/bigquery/query/questions.csv'.format(GCS_WORKING_DIR)
```

#### Run the component as a single pipeline


```python
import kfp.dsl as dsl
import kfp.gcp as gcp
import json
@dsl.pipeline(
    name='Bigquery query pipeline',
    description='Bigquery query pipeline'
)
def pipeline(
    query=QUERY, 
    project_id = PROJECT_ID, 
    dataset_id='', 
    table_id='', 
    output_gcs_path=OUTPUT_PATH, 
    dataset_location='US', 
    job_config=''
):
    bigquery_query_op(
        query=query, 
        project_id=project_id, 
        dataset_id=dataset_id, 
        table_id=table_id, 
        output_gcs_path=output_gcs_path, 
        dataset_location=dataset_location, 
        job_config=job_config).apply(gcp.use_gcp_secret('user-gcp-sa'))
```

#### Compile the pipeline


```python
pipeline_func = pipeline
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

#### Inspect the output


```python
!gsutil cat OUTPUT_PATH
```

## References
* [Component python code](https://github.com/kubeflow/pipelines/blob/master/component_sdk/python/kfp_component/google/bigquery/_query.py)
* [Component docker file](https://github.com/kubeflow/pipelines/blob/master/components/gcp/container/Dockerfile)
* [Sample notebook](https://github.com/kubeflow/pipelines/blob/master/components/gcp/bigquery/query/sample.ipynb)
* [BigQuery query REST API](https://cloud.google.com/bigquery/docs/reference/rest/v2/jobs/query)
