
# Name
Data preparation using PySpark on Cloud Dataproc


# Label
Cloud Dataproc, GCP, Cloud Storage,PySpark, Kubeflow, pipelines, components


# Summary
A Kubeflow Pipeline component to prepare data by submitting a PySpark job to Cloud Dataproc.


# Details
## Intended use
Use the component to run an Apache PySpark job as one preprocessing step in a Kubeflow Pipeline.


## Runtime arguments
| Argument | Description | Optional | Data type | Accepted values | Default |
|----------------------|------------|----------|--------------|-----------------|---------|
| project_id | The ID of the Google Cloud Platform (GCP) project that the cluster belongs to. | No | GCPProjectID |  |  |
| region | The Cloud Dataproc region to handle the request. | No | GCPRegion |  |  |
| cluster_name | The name of the cluster to run the job. | No | String |  |  |
| main_python_file_uri | The HCFS URI of the Python file to use as the driver. This must be a .py file. | No | GCSPath |  |  |
| args | The arguments to pass to the driver. Do not include arguments, such as --conf, that can be set as job properties, since a collision may occur that causes an incorrect job submission. | Yes | List |  | None |
| pyspark_job | The payload of a [PySparkJob](https://cloud.google.com/dataproc/docs/reference/rest/v1/PySparkJob). | Yes | Dict |  | None |
| job | The payload of a [Dataproc job](https://cloud.google.com/dataproc/docs/reference/rest/v1/projects.regions.jobs). | Yes | Dict |  | None |

## Output
Name | Description | Type
:--- | :---------- | :---
job_id | The ID of the created job. | String

## Cautions & requirements

To use the component, you must:
*   Set up a GCP project by following this [guide](https://cloud.google.com/dataproc/docs/guides/setup-project).
*   [Create a new cluster](https://cloud.google.com/dataproc/docs/guides/create-cluster).
*   Run the component under a secret [Kubeflow user service account](https://www.kubeflow.org/docs/started/getting-started-gke/#gcp-service-accounts) in a Kubeflow cluster. For example:

    ```
    component_op(...).apply(gcp.use_gcp_secret('user-gcp-sa'))
    ```
*   Grant the Kubeflow user service account the role `roles/dataproc.editor` on the project.

## Detailed description

This component creates a PySpark job from the [Dataproc submit job REST API](https://cloud.google.com/dataproc/docs/reference/rest/v1/projects.regions.jobs/submit).

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

dataproc_submit_pyspark_job_op = comp.load_component_from_url(
    'https://raw.githubusercontent.com/kubeflow/pipelines/a97f1d0ad0e7b92203f35c5b0b9af3a314952e05/components/gcp/dataproc/submit_pyspark_job/component.yaml')
help(dataproc_submit_pyspark_job_op)
```

### Sample

Note: The following sample code works in an IPython notebook or directly in Python code. See the sample code below to learn how to execute the template.


#### Setup a Dataproc cluster

[Create a new Dataproc cluster](https://cloud.google.com/dataproc/docs/guides/create-cluster) (or reuse an existing one) before running the sample code.


#### Prepare a PySpark job

Upload your PySpark code file to a Cloud Storage bucket. For example, this is a publicly accessible `hello-world.py` in Cloud Storage:


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

## References

*   [Create a new Dataproc cluster](https://cloud.google.com/dataproc/docs/guides/create-cluster) 
*   [PySparkJob](https://cloud.google.com/dataproc/docs/reference/rest/v1/PySparkJob)
*   [Dataproc job](https://cloud.google.com/dataproc/docs/reference/rest/v1/projects.regions.jobs)

## License
By deploying or using this software you agree to comply with the [AI Hub Terms of Service](https://aihub.cloud.google.com/u/0/aihub-tos) and the [Google APIs Terms of Service](https://developers.google.com/terms/). To the extent of a direct conflict of terms, the AI Hub Terms of Service will control.
