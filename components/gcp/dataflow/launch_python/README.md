
# Dataflow - Launch Python

Launch a self-executing beam python file. The input python code will be executed inside the KFP's cluster with Dataflow Runner. Client can optionally provide a requirements file to install additional packages to the container. The code will be executed under python 2.7, which is supported beam SDK.

## Input Parameters:
Name | Description
:--- | :----------
python_file_path |  The gcs or local path to the python file to run.
project_id |  The ID of the parent project.
requirements_file_path |  Optional, the gcs or local path to the pip requirements file.
location |  Optional. The regional endpoint to which to direct the request.
job_name_prefix |  Optional. The prefix of the genrated job name. If not provided, the method will generated a random name.
args |  The list of args to pass to the python file.
wait_interval |  Optional wait interval between calls to get job status. Defaults to 30.

## Output Parameters:
Name | Description
:--- | :----------
job_id | The id of the created dataflow job.

## Sample Code

Note: the sample code below works in both IPython notebook or python code directly.

### Set sample parameters


```python
# Required Parameters
PROJECT_ID = '<Please put your project ID here>'
GCS_WORKING_DIR = 'gs://<Please put your GCS path here>' # No ending slash

# Optional Parameters
EXPERIMENT_NAME = 'Dataflow - Launch Python'
COMPONENT_SPEC_URI = 'https://raw.githubusercontent.com/kubeflow/pipelines/master/components/gcp/dataflow/launch_python/component.yaml'
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

dataflow_python_op = comp.load_component_from_url(COMPONENT_SPEC_URI)
display(dataflow_python_op)
```

### Run the component as a single pipeline


```python
import kfp.dsl as dsl
import kfp.gcp as gcp
import json
@dsl.pipeline(
    name='Dataflow launch python pipeline',
    description='Dataflow launch python pipeline'
)
def pipeline():
    dataflow_python_op(
        python_file_path='gs://ml-pipeline-playground/samples/dataflow/wc/wc.py',
        project_id=PROJECT_ID,
        requirements_file_path='gs://ml-pipeline-playground/samples/dataflow/wc/requirements.txt',
        location='',
        job_name_prefix='',
        args=json.dumps([
            '--output', '{}/wc/wordcount.out'.format(GCS_WORKING_DIR),
            '--temp_location', '{}/dataflow/wc/tmp'.format(GCS_WORKING_DIR),
            '--staging_location', '{}/dataflow/wc/staging'.format(GCS_WORKING_DIR)
        ]),
        wait_interval = '30').apply(gcp.use_gcp_secret('user-gcp-sa'))
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
