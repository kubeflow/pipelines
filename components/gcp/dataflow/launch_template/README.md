
# Name
Data preparation by using a template to submit a job to Cloud Dataflow

# Labels
GCP, Cloud Dataflow, Kubeflow, Pipeline

# Summary
A Kubeflow Pipeline component to prepare data by using a template to submit a job to Cloud Dataflow.

# Details

## Intended use
Use this component when you have a pre-built Cloud Dataflow template and want to launch it as a step in a Kubeflow Pipeline.

## Runtime arguments
Argument        | Description                 | Optional   | Data type  | Accepted values | Default    |
:---            | :----------                 | :----------| :----------| :----------     | :----------|
project_id | The ID of the Google Cloud Platform (GCP) project to which the job belongs. | No | GCPProjectID |  |  |
gcs_path | The path to a Cloud Storage bucket containing the job creation template. It must be a valid Cloud Storage URL beginning with 'gs://'. | No  | GCSPath  |  |  |
launch_parameters | The parameters that are required to launch the template. The schema is defined in [LaunchTemplateParameters](https://cloud.google.com/dataflow/docs/reference/rest/v1b3/LaunchTemplateParameters). The parameter `jobName` is replaced by a generated name. | Yes  |  Dict | A JSON object which has the same structure as [LaunchTemplateParameters](https://cloud.google.com/dataflow/docs/reference/rest/v1b3/LaunchTemplateParameters) | None |
location | The regional endpoint to which the job request is directed.| Yes  |  GCPRegion |    |  None |
staging_dir |  The path to the Cloud Storage directory where the staging files are stored. A random subdirectory will be created under the staging directory to keep the job information. This is done so that you can resume the job in case of failure.|  Yes |  GCSPath |   |  None |
validate_only | If True, the request is validated but not executed.   |  Yes  |  Boolean |  |  False |
wait_interval | The number of seconds to wait between calls to get the status of the job. |  Yes  | Integer  |   |  30 |

## Input data schema

The input `gcs_path` must contain a valid Cloud Dataflow template. The template can be created by following the instructions in [Creating Templates](https://cloud.google.com/dataflow/docs/guides/templates/creating-templates). You can also use [Google-provided templates](https://cloud.google.com/dataflow/docs/guides/templates/provided-templates).

## Output
Name | Description
:--- | :----------
job_id | The id of the Cloud Dataflow job that is created.

## Caution & requirements

To use the component, the following requirements must be met:
- Cloud Dataflow API is enabled.
- The component can authenticate to GCP. Refer to [Authenticating Pipelines to GCP](https://www.kubeflow.org/docs/gke/authentication-pipelines/) for details.
- The Kubeflow user service account is a member of:
    - `roles/dataflow.developer` role of the project.
    - `roles/storage.objectViewer` role of the Cloud Storage Object `gcs_path.`
    - `roles/storage.objectCreator` role of the Cloud Storage Object `staging_dir.` 

## Detailed description
You can execute the template locally by following the instructions in [Executing Templates](https://cloud.google.com/dataflow/docs/guides/templates/executing-templates). See the sample code below to learn how to execute the template.
Follow these steps to use the component in a pipeline:
1. Install the Kubeflow Pipeline SDK:



```python
%%capture --no-stderr

!pip3 install kfp --upgrade
```

2. Load the component using KFP SDK


```python
import kfp.components as comp

dataflow_template_op = comp.load_component_from_url(
    'https://raw.githubusercontent.com/kubeflow/pipelines/1.4.0-rc.1/components/gcp/dataflow/launch_template/component.yaml')
help(dataflow_template_op)
```

### Sample

Note: The following sample code works in an IPython notebook or directly in Python code.
In this sample, we run a Google-provided word count template from `gs://dataflow-templates/latest/Word_Count`. The template takes a text file as input and outputs word counts to a Cloud Storage bucket. Here is the sample input:


```python
!gsutil cat gs://dataflow-samples/shakespeare/kinglear.txt
```

#### Set sample parameters


```python
# Required Parameters
PROJECT_ID = '<Please put your project ID here>'
GCS_WORKING_DIR = 'gs://<Please put your GCS path here>' # No ending slash
```


```python
# Optional Parameters
EXPERIMENT_NAME = 'Dataflow - Launch Template'
OUTPUT_PATH = '{}/out/wc'.format(GCS_WORKING_DIR)
```

#### Example pipeline that uses the component


```python
import kfp.dsl as dsl
import json
@dsl.pipeline(
    name='Dataflow launch template pipeline',
    description='Dataflow launch template pipeline'
)
def pipeline(
    project_id = PROJECT_ID, 
    gcs_path = 'gs://dataflow-templates/latest/Word_Count', 
    launch_parameters = json.dumps({
       'parameters': {
           'inputFile': 'gs://dataflow-samples/shakespeare/kinglear.txt',
           'output': OUTPUT_PATH
       }
    }), 
    location = '',
    validate_only = 'False', 
    staging_dir = GCS_WORKING_DIR,
    wait_interval = 30):
    dataflow_template_op(
        project_id = project_id, 
        gcs_path = gcs_path, 
        launch_parameters = launch_parameters, 
        location = location, 
        validate_only = validate_only,
        staging_dir = staging_dir,
        wait_interval = wait_interval))
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
!gsutil cat $OUTPUT_PATH*
```

## References

* [Component python code](https://github.com/kubeflow/pipelines/blob/master/components/gcp/container/component_sdk/python/kfp_component/google/dataflow/_launch_template.py)
* [Component docker file](https://github.com/kubeflow/pipelines/blob/master/components/gcp/container/Dockerfile)
* [Sample notebook](https://github.com/kubeflow/pipelines/blob/master/components/gcp/dataflow/launch_template/sample.ipynb)
* [Cloud Dataflow Templates overview](https://cloud.google.com/dataflow/docs/guides/templates/overview)

## License
By deploying or using this software you agree to comply with the [AI Hub Terms of Service](https://aihub.cloud.google.com/u/0/aihub-tos) and the [Google APIs Terms of Service](https://developers.google.com/terms/). To the extent of a direct conflict of terms, the AI Hub Terms of Service will control.

