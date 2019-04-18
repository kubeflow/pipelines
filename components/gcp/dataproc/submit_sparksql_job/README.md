
# Name
Data preparation using SparkSQL  on YARN with Cloud Dataproc

# Label
Cloud Dataproc, GCP, Cloud Storage, YARN, SparkSQL, Kubeflow, pipelines, components 

# Summary
A Kubeflow Pipeline component to prepare data by submitting a SparkSql job on YARN to Cloud Dataproc.

# Details

## Intended use
Use the component to run an Apache SparkSql job as one preprocessing step in a Kubeflow Pipeline.

## Runtime arguments
Argument| Description | Optional | Data type| Accepted values| Default |
:--- | :---------- | :--- | :------- | :------ | :------
project_id | The ID of the Google Cloud Platform (GCP) project that the cluster belongs to. | No| GCPProjectID |   | |
region | The Cloud Dataproc region to handle the request. | No | GCPRegion|
cluster_name | The name of the cluster to run the job. | No | String| | |
queries | The queries to execute the SparkSQL job. Specify multiple queries in one string by separating them with semicolons. You do not need to terminate queries with semicolons. | Yes | List |  | None | 
query_file_uri | The HCFS URI of the script that contains the SparkSQL queries.| Yes | GCSPath |  | None |
script_variables | Mapping of the query’s variable names to their values (equivalent to the SparkSQL command: SET name="value";).| Yes| Dict | | None |
sparksql_job | The payload of a [SparkSqlJob](https://cloud.google.com/dataproc/docs/reference/rest/v1/SparkSqlJob). | Yes | Dict |  | None |
job | The payload of a [Dataproc job](https://cloud.google.com/dataproc/docs/reference/rest/v1/projects.regions.jobs). | Yes | Dict |  | None |
wait_interval | The number of seconds to pause between polling the operation. | Yes |Integer |  | 30 |

## Output
Name | Description | Type
:--- | :---------- | :---
job_id | The ID of the created job. | String

## Cautions & requirements
To use the component, you must:
* Set up a GCP project by following this [guide](https://cloud.google.com/dataproc/docs/guides/setup-project).
* [Create a new cluster](https://cloud.google.com/dataproc/docs/guides/create-cluster).
* Run the component under a secret [Kubeflow user service account](https://www.kubeflow.org/docs/started/getting-started-gke/#gcp-service-accounts) in a Kubeflow cluster. For example:
```
component_op(...).apply(gcp.use_gcp_secret('user-gcp-sa'))
```
* Grant the Kubeflow user service account the role `roles/dataproc.editor` on the project.

## Detailed Description
This component creates a Pig job from [Dataproc submit job REST API](https://cloud.google.com/dataproc/docs/reference/rest/v1/projects.regions.jobs/submit).

Follow these steps to use the component in a pipeline:
1. Install the Kubeflow Pipeline SDK:


```python
%%capture --no-stderr

KFP_PACKAGE = 'https://storage.googleapis.com/ml-pipeline/release/0.1.14/kfp.tar.gz'
!pip3 install $KFP_PACKAGE --upgrade
```

2. Load the component using KFP SDK


```python
import kfp.components as comp

dataproc_submit_sparksql_job_op = comp.load_component_from_url(
    'https://raw.githubusercontent.com/kubeflow/pipelines/d2f5cc92a46012b9927209e2aaccab70961582dc/components/gcp/dataproc/submit_sparksql_job/component.yaml')
help(dataproc_submit_sparksql_job_op)
```

### Sample

Note: The following sample code works in an IPython notebook or directly in Python code. See the sample code below to learn how to execute the template.

#### Setup a Dataproc cluster
[Create a new Dataproc cluster](https://cloud.google.com/dataproc/docs/guides/create-cluster) (or reuse an existing one) before running the sample code.

#### Prepare a SparkSQL job
Either put your SparkSQL queries in the `queires` list, or upload your SparkSQL queries into a file to a Cloud Storage bucket and then enter the Cloud Storage bucket’s path in `query_file_uri`. In this sample, we will use a hard coded query in the `queries` list to select data from a public CSV file from Cloud Storage.

For more details about Spark SQL, see [Spark SQL, DataFrames and Datasets Guide](https://spark.apache.org/docs/latest/sql-programming-guide.html)

#### Set sample parameters


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
```

#### Example pipeline that uses the component


```python
import kfp.dsl as dsl
import kfp.gcp as gcp
import json
@dsl.pipeline(
    name='Dataproc submit SparkSQL job pipeline',
    description='Dataproc submit SparkSQL job pipeline'
)
def dataproc_submit_sparksql_job_pipeline(
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
    dataproc_submit_sparksql_job_op(
        project_id=project_id, 
        region=region, 
        cluster_name=cluster_name, 
        queries=queries, 
        query_file_uri=query_file_uri,
        script_variables=script_variables, 
        sparksql_job=sparksql_job, 
        job=job, 
        wait_interval=wait_interval).apply(gcp.use_gcp_secret('user-gcp-sa'))
    
```

#### Compile the pipeline


```python
pipeline_func = dataproc_submit_sparksql_job_pipeline
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

## References
* [Spark SQL, DataFrames and Datasets Guide](https://spark.apache.org/docs/latest/sql-programming-guide.html)
* [SparkSqlJob](https://cloud.google.com/dataproc/docs/reference/rest/v1/SparkSqlJob)
* [Cloud Dataproc job](https://cloud.google.com/dataproc/docs/reference/rest/v1/projects.regions.jobs)


## License
By deploying or using this software you agree to comply with the [AI Hub Terms of Service](https://aihub.cloud.google.com/u/0/aihub-tos) and the [Google APIs Terms of Service](https://developers.google.com/terms/). To the extent of a direct conflict of terms, the AI Hub Terms of Service will control.
