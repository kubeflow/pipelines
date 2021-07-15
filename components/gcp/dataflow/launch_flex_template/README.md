# Name

Data preparation by using a Flex template to submit a job to Cloud Dataflow

# Labels

GCP, Cloud Dataflow, Kubeflow, Pipeline

# Summary

A Kubeflow Pipeline component to prepare data by using a Flex template to submit a job to Cloud Dataflow.

# Details

## Intended use

Use this component when you have a pre-built Cloud Dataflow Flex template and want to launch it as a step in a Kubeflow
Pipeline.

## Runtime arguments

Argument        | Description                 | Optional   | Data type  | Accepted values | Default    |
:---            | :----------                 | :----------| :----------| :----------     | :----------|
project_id | The ID of the Google Cloud Platform (GCP) project to which the job belongs. | No | GCPProjectID |  |  |
location | The regional endpoint to which the job request is directed.| No  |  GCPRegion |    |   |
launch_parameters | The parameters that are required to launch the flex template. The schema is defined in [LaunchFlexTemplateParameters](https://cloud.google.com/dataflow/docs/reference/rest/v1b3/projects.locations.flexTemplates/launch#LaunchFlexTemplateParameter) | None |
staging_dir |  The path to the Cloud Storage directory where the staging files are stored. A random subdirectory will be created under the staging directory to keep the job information. This is done so that you can resume the job in case of failure.|  Yes |  GCSPath |   |  None |
wait_interval | The number of seconds to wait between calls to get the status of the job. |  Yes  | Integer  |   |  30 |

## Input data schema

The input `gcs_path` must contain a valid Cloud Dataflow template. The template can be created by following the
instructions in [Creating Templates](https://cloud.google.com/dataflow/docs/guides/templates/creating-templates). You
can also use [Google-provided templates](https://cloud.google.com/dataflow/docs/guides/templates/provided-templates).

## Output

Name | Description
:--- | :----------
job_id | The id of the Cloud Dataflow job that is created.

## Detailed description

This job use the google
provided [BigQuery to Cloud Storage Parquet template](https://cloud.google.com/dataflow/docs/guides/templates/provided-batch#running-the-bigquery-to-cloud-storage-parquet-template)
to run a Dataflow pipeline that reads
the [Shakespeare word index](https://cloud.google.com/bigquery/public-data#sample_tables)
sample public dataset and writes the data in Parquet format to the user provided GCS bucket.

## Caution & requirements

To use the component, the following requirements must be met:

- Cloud Dataflow API is enabled.
- The component can authenticate to GCP. Refer
  to [Authenticating Pipelines to GCP](https://www.kubeflow.org/docs/gke/authentication-pipelines/) for details.
- The output cloud storage bucket must exist before running the pipeline.
- The Kubeflow user service account is a member of:
  - `roles/dataflow.developer` role of the project.
  - `roles/storage.objectCreator` role of the Cloud Storage Object for `staging_dir.` and output data folder.
- The dataflow controller service account is a member of:
  - `roles/bigquery.readSessionUser` role to create read sessions in the project.
  - `roles/bigquery.jobUser` role to run jobs including queries.
  - `roles/bigquery.dataViewer` role to read data and metadata from the table of view

---

### Follow these steps to use the component in a pipeline:

#### 1. Install the Kubeflow Pipeline SDK:

```python
%% capture --no-stderr

!pip3 install kfp - -upgrade
```

#### 2. Load the component using KFP SDK

```python
import kfp.components as comp

dataflow_template_op = comp.load_component_from_url(
  'https://raw.githubusercontent.com/kubeflow/pipelines/1.7.0-rc.1/components/gcp/dataflow/launch_flex_template/component.yaml')
help(dataflow_template_op)
```

#### 3. Configure job parameters

```python
PROJECT_ID = '[Your PROJECT_ID]'
BIGQUERY_TABLE_SPEC = '[Your PROJECT_ID:DATASET_ID.TABLE_ID]'
GCS_OUTPUT_FOLDER = 'gs://[Your output GCS folder]'
GCS_STAGING_FOLDER = 'gs://[Your staging GCS folder]'
LOCATION = 'us'
# Optional Parameters
EXPERIMENT_NAME = 'Dataflow - Launch Flex Template'

flex_temp_launch_parameters = {
  "parameters": {
    "tableRef": BIGQUERY_TABLE_SPEC,
    "bucket": GCS_OUTPUT_FOLDER
  },
  "containerSpecGcsPath": "gs://dataflow-templates/2021-03-29-00_RC00/flex/BigQuery_to_Parquet",
}
```

#### 4. Example pipeline that uses the component

```python
import kfp.dsl as dsl
import json


@dsl.pipeline(
  name='Dataflow launch flex template pipeline',
  description='Dataflow launch flex template pipeline'
)
def pipeline(
        project_id=PROJECT_ID,
        location=LOCATION,
        launch_parameters=json.dumps(flex_temp_launch_parameters),
        staging_dir=GCS_STAGING_FOLDER,
        wait_interval=30):
  dataflow_template_op(
    project_id=project_id,
    location=location,
    launch_parameters=launch_parameters,
    staging_dir=staging_dir,
    wait_interval=wait_interval)

```

#### 5. Create pipeline run

```python
import kfp

pipeline_func = pipeline
run_name = pipeline_func.__name__ + ' run'

kfp.Client().create_run_from_pipeline_func(
  pipeline_func, 
  arguments = {},
  run_name = run_name,
  experiment_name=EXPERIMENT_NAME,
  namespace='default'
)
```

#### 6. Inspect the output

```python
!gsutil cat $GCS_OUTPUT_FOLDER *
```

## References

* [Component python code](https://github.com/kubeflow/pipelines/blob/master/components/gcp/container/component_sdk/python/kfp_component/google/dataflow/_launch_flex_template.py)
* [Component docker file](https://github.com/kubeflow/pipelines/blob/master/components/gcp/container/Dockerfile)
* [Sample notebook](https://github.com/kubeflow/pipelines/blob/master/components/gcp/dataflow/launch_flex_template/sample.ipynb)
* [Cloud Dataflow Templates overview](https://cloud.google.com/dataflow/docs/guides/templates/overview)

## License

By deploying or using this software you agree to comply with
the [AI Hub Terms of Service](https://aihub.cloud.google.com/u/0/aihub-tos) and
the [Google APIs Terms of Service](https://developers.google.com/terms/). To the extent of a direct conflict of terms,
the AI Hub Terms of Service will control.
