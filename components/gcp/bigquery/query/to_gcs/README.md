
# Name

Gather data by querying BigQuery and save it to GCS. 


# Labels

GCP, BigQuery, Kubeflow, Pipeline


# Summary

A Kubeflow Pipeline component to submit a query to BigQuery and store the result in table on BigQuery.


# Details


## Intended use

Use this Kubeflow component to:
*   Select data by submitting a query to BigQuery.
*   Output the data into a table on BigQuery.


## Runtime arguments:


| Argument | Description | Optional | Data type | Accepted values | Default |
|----------|-------------|----------|-----------|-----------------|---------|
| query | The query used by BigQuery to fetch the results. | No | String |  |  |
| project_id | The project ID of the Google Cloud Platform (GCP) project to use to execute the query. | No | GCPProjectID |  |  |
| dataset_id | The ID of the persistent BigQuery dataset to store the results of the query. If the dataset does not exist, the operation will create a new one. | Yes | String |  | None |
| table_id | The ID of the BigQuery table to store the results of the query. If the table ID is absent, the operation will generate a random ID for the table. | Yes | String |  | None |
| dataset_location | The location where the dataset is created. Defaults to US. | Yes | String |  | US |
| job_config | The full configuration specification for the query job. See [QueryJobConfig](https://googleapis.github.io/google-cloud-python/latest/bigquery/generated/google.cloud.bigquery.job.QueryJobConfig.html#google.cloud.bigquery.job.QueryJobConfig) for details. | Yes | Dict | A JSONobject which has the same structure as [QueryJobConfig](https://googleapis.github.io/google-cloud-python/latest/bigquery/generated/google.cloud.bigquery.job.QueryJobConfig.html#google.cloud.bigquery.job.QueryJobConfig) | None |

## Input data schema

The input data is a BigQuery job containing a query that pulls data from various sources. 


## Output:

Name | Description | Type
:--- | :---------- | :---
output_gcs_path | The path to the Cloud Storage bucket containing the query output in CSV format. | GCSPath


## Cautions & requirements

To use the component, the following requirements must be met:

*   The BigQuery API is enabled.
*   The component can authenticate to GCP. Refer to [Authenticating Pipelines to GCP](https://www.kubeflow.org/docs/gke/authentication-pipelines/) for details.
*   The Kubeflow user service account is a member of the `roles/bigquery.admin` role of the project.
*   The Kubeflow user service account is a member of the `roles/storage.objectCreator `role of the Cloud Storage output bucket.

## Detailed description
This Kubeflow Pipeline component is used to:
*   Submit a query to BigQuery.
    *   The query results are persisted in a dataset table in BigQuery.
    *   The data is extracted localy and stored as a csv file. 

    Use the code below as an example of how to run your BigQuery job.

### Sample

Note: The following sample code works in an IPython notebook or directly in Python code.

#### Set sample parameters


```python
%%capture --no-stderr

!pip3 install kfp --upgrade
```

2. Load the component using KFP SDK


```python
import kfp.components as comp

bigquery_query_op = comp.load_component_from_url(
    'https://raw.githubusercontent.com/kubeflow/pipelines/01a23ae8672d3b18e88adf3036071496aca3552d/components/gcp/bigquery/query/to?gcs/component.yaml')
help(bigquery_query_op)
```

### Query

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
EXPERIMENT_NAME = 'Bigquery-Query'
OUTPUT_PATH = '{}/bigquery/query/questions.csv'.format(GCS_WORKING_DIR)
```

#### Run the component as a single pipeline


```python
import kfp.dsl as dsl
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
        job_config=job_config)
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
!gsutil cat $OUTPUT_PATH
```

## References
* [Component python code](https://github.com/kubeflow/pipelines/blob/master/components/gcp/container/component_sdk/python/kfp_component/google/bigquery/_query.py)
* [Component docker file](https://github.com/kubeflow/pipelines/blob/master/components/gcp/container/Dockerfile)
* [Sample notebook](https://github.com/kubeflow/pipelines/blob/master/components/gcp/bigquery/query/sample.ipynb)
* [BigQuery query REST API](https://cloud.google.com/bigquery/docs/reference/rest/v2/jobs/query)

## License
By deploying or using this software you agree to comply with the [AI Hub Terms of Service](https://aihub.cloud.google.com/u/0/aihub-tos) and the [Google APIs Terms of Service](https://developers.google.com/terms/). To the extent of a direct conflict of terms, the AI Hub Terms of Service will control.
