
# Name
Component: Data preparation using Apache Pig on YARN with Cloud Dataproc

# Labels
Cloud Dataproc, YARN, Apache Pig, Kubeflow


# Summary
A Kubeflow pipeline component to prepare data by submitting an Apache Pig job on YARN to Cloud Dataproc.

# Facets
<!--Make sure the asset has data for the following facets:
Use case
Technique
Input data type
ML workflow

The data must map to the acceptable values for these facets, as documented on the “taxonomy” sheet of go/aihub-facets
https://gitlab.aihub-content-external.com/aihubbot/kfp-components/commit/fe387ab46181b5d4c7425dcb8032cb43e70411c1
--->
Use case:
Other

Technique: 
Other

Input data type:
Tabular

ML workflow: 
Data preparation

# Details
## Intended use
Use this component to run an Apache Pig job as one preprocessing step in a Kubeflow pipeline.

## Runtime arguments
| Argument | Description | Optional | Data type | Accepted values | Default |
|:----------|:-------------|:----------|:-----------|:-----------------|:---------|
| project_id | The ID of the Google Cloud Platform (GCP) project that the cluster belongs to. | No | GCPProjectID |-  |  -|
| region | The Cloud Dataproc region that handles the request. | No | GCPRegion | - |-  |
| cluster_name | The name of the cluster that runs the job. | No | String | - | - |
| queries | The queries to execute the Pig job. Specify multiple queries in one string by separating them with semicolons. You do not need to terminate queries with semicolons. | Yes | List |  -| None |
| query_file_uri | The Cloud Storage bucket path pointing to a file that contains the Pig queries. | Yes | GCSPath | - | None |
| script_variables | Mapping of the query’s variable names to their values (equivalent to the Pig command: SET name="value";). | Yes | Dict |  -| None |
| pig_job | The payload of a [PigJob](https://cloud.google.com/dataproc/docs/reference/rest/v1/PigJob). | Yes | Dict | - | None |
| job | The payload of a [Dataproc job](https://cloud.google.com/dataproc/docs/reference/rest/v1/projects.regions.jobs). | Yes | Dict |  | None |
| wait_interval | The number of seconds to pause between polling the operation. | Yes | Integer | - | 30 |

## Output
Name | Description | Type
:--- | :---------- | :---
job_id | The ID of the created job. | String

## Cautions & requirements

To use the component, you must:
*   Set up a GCP project by following this [guide](https://cloud.google.com/dataproc/docs/guides/setup-project).
*   [Create a new cluster](https://cloud.google.com/dataproc/docs/guides/create-cluster).
*   The component can authenticate to GCP. Refer to [Authenticating Pipelines to GCP](https://www.kubeflow.org/docs/gke/authentication-pipelines/) for details.
*   Grant the Kubeflow user service account the role, `roles/dataproc.editor`, on the project.

## Detailed description
This component creates a Pig job from the [Dataproc submit job REST API](https://cloud.google.com/dataproc/docs/reference/rest/v1/projects.regions.jobs/submit).

Follow these steps to use the component in a pipeline:
1.  Install the Kubeflow pipeline's SDK



    ```python
    %%capture --no-stderr

    !pip3 install kfp --upgrade
    ```

2. Load the component using the Kubeflow pipeline's SDK


    ```python
    import kfp.components as comp

    dataproc_submit_pig_job_op = comp.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/1.4.1/components/gcp/dataproc/submit_pig_job/component.yaml')
    help(dataproc_submit_pig_job_op)
    ```

### Sample

The following sample code works in an IPython notebook or directly in Python code. See the sample code below to learn how to execute the template.

#### Setup a Dataproc cluster

[Create a new Dataproc cluster](https://cloud.google.com/dataproc/docs/guides/create-cluster) (or reuse an existing one) before running the sample code.

#### Prepare a Pig query

You can put your Pig queries in the `queries` list, or you can use `query_file_uri`. In this sample, we will use a hard-coded query in the `queries` list to select data from a local password file.

For more details on Apache Pig, see the [Pig documentation.](http://pig.apache.org/docs/latest/)

#### Set sample parameters


```python
PROJECT_ID = '<Put your project ID here>'
CLUSTER_NAME = '<Put your existing cluster name here>'

REGION = 'us-central1'
QUERY = '''
natality_csv = load 'gs://public-datasets/natality/csv' using PigStorage(':');
top_natality_csv = LIMIT natality_csv 10; 
dump natality_csv;'''
EXPERIMENT_NAME = 'Dataproc - Submit Pig Job'
```

#### Example pipeline that uses the component


```python
import kfp.dsl as dsl
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
    dataproc_submit_pig_job_op(
        project_id=project_id, 
        region=region, 
        cluster_name=cluster_name, 
        queries=queries, 
        query_file_uri=query_file_uri,
        script_variables=script_variables, 
        pig_job=pig_job, 
        job=job, 
        wait_interval=wait_interval)
    
```

#### Compile the pipeline


```python
pipeline_func = dataproc_submit_pig_job_pipeline
pipeline_filename = pipeline_func.__name__ + '.zip'
import kfp.compiler as compiler
compiler.Compiler().compile(pipeline_func, pipeline_filename)
```

#### Submit the pipeline for execution


```python
#Specify values for the pipeline's arguments
arguments = {}

#Get or create an experiment
import kfp
client = kfp.Client()
experiment = client.create_experiment(EXPERIMENT_NAME)

#Submit a pipeline run
run_name = pipeline_func.__name__ + ' run'
run_result = client.run_pipeline(experiment.id, run_name, pipeline_filename, arguments)
```

## References
*   [Create a new Dataproc cluster](https://cloud.google.com/dataproc/docs/guides/create-cluster) 
*   [Pig documentation](http://pig.apache.org/docs/latest/)
*   [Dataproc job](https://cloud.google.com/dataproc/docs/reference/rest/v1/projects.regions.jobs)
*   [PigJob](https://cloud.google.com/dataproc/docs/reference/rest/v1/PigJob)

## License
By deploying or using this software you agree to comply with the [AI Hub Terms of Service](https://aihub.cloud.google.com/u/0/aihub-tos) and the [Google APIs Terms of Service](https://developers.google.com/terms/). To the extent of a direct conflict of terms, the AI Hub Terms of Service will control.
