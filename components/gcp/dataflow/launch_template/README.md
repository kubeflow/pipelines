
# Dataflow - Launch Template

## Intended Use

A Kubeflow Pipeline component to submit a job from a dataflow template to Google Cloud Dataflow service.

## Runtime Parameters:
Name | Description
:--- | :----------
project_id | Required. The ID of the Cloud Platform project that the job belongs to.
gcs_path | Required. A Cloud Storage path to the template from which to create the job. Must be valid Cloud Storage URL, beginning with 'gs://'.
launch_parameters | Parameters to provide to the template being launched. Schema defined in https://cloud.google.com/dataflow/docs/reference/rest/v1b3/LaunchTemplateParameters. `jobName` will be replaced by generated name.
location | Optional. The regional endpoint to which to direct the request.
job_name_prefix |  Optional. The prefix of the genrated job name. If not provided, the method will generated a random name.
validate_only | If true, the request is validated but not actually executed. Defaults to false.
wait_interval | Optional wait interval between calls to get job status. Defaults to 30.

## Output:
Name | Description
:--- | :----------
job_id | The id of the created dataflow job.

## Sample

Note: the sample code below works in both IPython notebook or python code directly.

### Set sample parameters


```python
# Required Parameters
PROJECT_ID = '<Please put your project ID here>'
GCS_WORKING_DIR = 'gs://<Please put your GCS path here>' # No ending slash

# Optional Parameters
EXPERIMENT_NAME = 'Dataflow - Launch Template'
COMPONENT_SPEC_URI = 'https://raw.githubusercontent.com/kubeflow/pipelines/master/components/gcp/dataflow/launch_template/component.yaml'
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

dataflow_template_op = comp.load_component_from_url(COMPONENT_SPEC_URI)
display(dataflow_template_op)
```

### Here is an illustrative pipeline that uses the component


```python
import kfp.dsl as dsl
import kfp.gcp as gcp
import json
@dsl.pipeline(
    name='Dataflow launch template pipeline',
    description='Dataflow launch template pipeline'
)
def pipeline(
    project_id, 
    gcs_path, 
    launch_parameters, 
    location='', 
    job_name_prefix='', 
    validate_only='', 
    wait_interval = 30
):
    dataflow_template_op(project_id, gcs_path, launch_parameters, location, job_name_prefix, validate_only, 
        wait_interval).apply(gcp.use_gcp_secret('user-gcp-sa'))
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
    'project_id': PROJECT_ID,
    'gcs_path': 'gs://dataflow-templates/latest/Word_Count',
    'launch_parameters': json.dumps({
       'parameters': {
           'inputFile': 'gs://dataflow-samples/shakespeare/kinglear.txt',
           'output': '{}/dataflow/launch-template/'.format(GCS_WORKING_DIR)
       }
    })
}

#Get or create an experiment and submit a pipeline run
import kfp
client = kfp.Client()
experiment = client.create_experiment(EXPERIMENT_NAME)

#Submit a pipeline run
run_name = pipeline_func.__name__ + ' run'
run_result = client.run_pipeline(experiment.id, run_name, pipeline_filename, arguments)
```
