
# Submitting a job to Cloud Dataflow service using a template
A Kubeflow Pipeline component to submit a job from a dataflow template to Cloud Dataflow service.

## Intended Use

A Kubeflow Pipeline component to submit a job from a dataflow template to Google Cloud Dataflow service.

## Runtime arguments
Name | Description | Type | Optional | Default
:--- | :---------- | :--- | :------- | :------
project_id | The ID of the Cloud Platform project to which the job belongs. | GCPProjectID | No |
gcs_path | A Cloud Storage path to the job creation template. It must be a valid Cloud Storage URL beginning with `gs://`. | GCSPath | No |
launch_parameters | The parameters that are required for  the template being launched. The Schema is defined in  [LaunchTemplateParameters Parameters](https://cloud.google.com/dataflow/docs/reference/rest/v1b3/LaunchTemplateParameters). | Dict | Yes | `{}`
location | The regional endpoint to which the job request is directed. | GCPRegion | Yes | ``
validate_only | If true, the request is validated but not actually executed. | Bool | Yes | `False`
staging_dir | The Cloud Storage path for keeping staging files. A random subdirectory will be created under the directory to keep job info for resuming the job in case of failure. | GCSPath | Yes | ``
wait_interval | The seconds to wait between calls to get the job status. | Integer | Yes |`30`

## Output:
Name | Description | Type
:--- | :---------- | :---
job_id | The id of the created dataflow job. | String

## Cautions and requirements
To use the components, the following requirements must be met:
* Dataflow API is enabled.
* The component is running under a secret [Kubeflow user service account](https://www.kubeflow.org/docs/started/getting-started-gke/#gcp-service-accounts) in a KFP cluster. For example:
```
component_op(...).apply(gcp.use_gcp_secret('user-gcp-sa'))
```
* The Kubeflow user service account is a member of `roles/dataflow.developer` role of the project.
* The Kubeflow user service account is a member of `roles/storage.objectViewer` role of the Cloud Storage Object `gcs_path`.
* The Kubeflow user service account is a member of `roles/storage.objectCreator` role of the Cloud Storage Object `staging_dir`.

## Detailed description
The input `gcs_path` must contain a valid Dataflow template. The template can be created by following the guide [Creating Templates](https://cloud.google.com/dataflow/docs/guides/templates/creating-templates). Or, you can use [Google-provided templates](https://cloud.google.com/dataflow/docs/guides/templates/provided-templates).

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

dataflow_template_op = comp.load_component_from_url(
    'https://raw.githubusercontent.com/kubeflow/pipelines/d2f5cc92a46012b9927209e2aaccab70961582dc/components/gcp/dataflow/launch_template/component.yaml')
help(dataflow_template_op)
```

For more information about the component, please checkout:
* [Component python code](https://github.com/kubeflow/pipelines/blob/master/component_sdk/python/kfp_component/google/dataflow/_launch_template.py)
* [Component docker file](https://github.com/kubeflow/pipelines/blob/master/components/gcp/container/Dockerfile)
* [Sample notebook](https://github.com/kubeflow/pipelines/blob/master/components/gcp/dataflow/launch_template/sample.ipynb)
* [Cloud Dataflow Templates overview](https://cloud.google.com/dataflow/docs/guides/templates/overview)

### Sample

Note: the sample code below works in both IPython notebook or python code directly.

In this sample, we run a Google provided word count template from `gs://dataflow-templates/latest/Word_Count`. The template takes a text file as input and output word counts to a Cloud Storage bucket. Here is the sample input:


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
import kfp.gcp as gcp
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
        wait_interval = wait_interval).apply(gcp.use_gcp_secret('user-gcp-sa'))
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
