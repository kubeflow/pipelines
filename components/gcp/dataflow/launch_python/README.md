
# Dataflow - Launch Python

## Intended Use
A Kubeflow Pipeline component to submit a Apache Beam job authored in python, to Google Cloud Dataflow for execution. The python beam code runs with Google Cloud Dataflow runner.

## Run-Time Parameters:
Name | Description | Type | Default
:--- | :---------- | :--- | :------
python_file_path |  The gcs or local path to the python file to run. | String |
project_id |  The ID of the parent project. | GCPProjectID |
staging_dir | Optional. The GCS directory for keeping staging files. A random subdirectory will be created under the directory to keep job info for resuming the job in case of failure and it will be passed as `staging_location` and `temp_location` command line args of the beam code. | GCSPath | ``
requirements_file_path |  Optional, the gcs or local path to the pip requirements file. | GCSPath | ``
args |  The list of args to pass to the python file. | List | `[]`
wait_interval |  Optional wait interval between calls to get job status. Defaults to 30. | Integer | `30`

## Output:
Name | Description | Type
:--- | :---------- | :---
job_id | The id of the created dataflow job. | String

## Sample

Note: the sample code below works in both IPython notebook or python code directly.

### Set sample parameters


```python
# Required Parameters
PROJECT_ID = '<Please put your project ID here>'
GCS_STAGING_DIR = 'gs://<Please put your GCS path here>' # No ending slash

# Optional Parameters
EXPERIMENT_NAME = 'Dataflow - Launch Python'
COMPONENT_SPEC_URI = 'https://raw.githubusercontent.com/kubeflow/pipelines/d2f5cc92a46012b9927209e2aaccab70961582dc/components/gcp/dataflow/launch_python/component.yaml'
```

### Install KFP SDK
Install the SDK (Uncomment the code if the SDK is not installed before)


```python
# KFP_PACKAGE = 'https://storage.googleapis.com/ml-pipeline/release/0.1.12/kfp.tar.gz'
# !pip3 install $KFP_PACKAGE --upgrade
```

### Load component definitions


```python
import kfp.components as comp

dataflow_python_op = comp.load_component_from_url(COMPONENT_SPEC_URI)
display(dataflow_python_op)
```

### Here is an illustrative pipeline that uses the component


```python
import kfp.dsl as dsl
import kfp.gcp as gcp
import json
@dsl.pipeline(
    name='Dataflow launch python pipeline',
    description='Dataflow launch python pipeline'
)
def pipeline(
    python_file_path = 'gs://ml-pipeline-playground/samples/dataflow/wc/wc.py',
    project_id = PROJECT_ID,
    staging_dir = GCS_STAGING_DIR,
    requirements_file_path = 'gs://ml-pipeline-playground/samples/dataflow/wc/requirements.txt',
    args = json.dumps([
        '--output', '{}/wc/wordcount.out'.format(GCS_STAGING_DIR)
    ]),
    wait_interval = 30
):
    dataflow_python_op(
        python_file_path = python_file_path, 
        project_id = project_id, 
        staging_dir = staging_dir, 
        requirements_file_path = requirements_file_path, 
        args = args,
        wait_interval = wait_interval).apply(gcp.use_gcp_secret('user-gcp-sa'))
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
arguments = {}

#Get or create an experiment and submit a pipeline run
import kfp
client = kfp.Client()
experiment = client.create_experiment(EXPERIMENT_NAME)

#Submit a pipeline run
run_name = pipeline_func.__name__ + ' run'
run_result = client.run_pipeline(experiment.id, run_name, pipeline_filename, arguments)
```


```python

```
