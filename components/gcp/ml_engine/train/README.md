
# CloudML - Train

## Intended Use
A Kubeflow Pipeline component to submit a Cloud Machine Learning Engine training job as a step in a pipeline

## Runtime Parameters:
Name | Description
:--- | :----------
project_id | Required. The ID of the parent project of the job.
python_module | The Python module name to run after installing the packages.
package_uris | The Google Cloud Storage location of the packages with the training program and any additional dependencies. The maximum number of package URIs is 100.
region | The Google Compute Engine region to run the training job in.
args | Command line arguments to pass to the program.
job_dir |  The list of args to pass to the python file.
python_version | A Google Cloud Storage path in which to store training outputs and other data needed for training. This path is passed to your TensorFlow program as the `--job-dir` command-line argument. The benefit of specifying this field is that Cloud ML validates the path for use in training.
runtime_version | The Cloud ML Engine runtime version to use for training. If not set, Cloud ML Engine uses the default stable version, 1.0.
master_image_uri | The Docker image to run on the master replica. This image must be in Container Registry.
worker_image_uri | The Docker image to run on the worker replica. This image must be in Container Registry.
training_input | Input parameters to create a training job.
job_id_prefix | The prefix of the generated job id.
wait_interval |  Optional wait interval between calls to get job status. Defaults to 30.

## Output:
Name | Description
:--- | :----------
job_id | The ID of the created job.

## Sample

Note: the sample code below works in both IPython notebook or python code directly.

### Set sample parameters


```python
# Required Parameters
PROJECT_ID = '<Please put your project ID here>'
GCS_WORKING_DIR = 'gs://<Please put your GCS path here>' # No ending slash

# Optional Parameters
EXPERIMENT_NAME = 'CLOUDML - Train'
COMPONENT_SPEC_URI = 'https://raw.githubusercontent.com/kubeflow/pipelines/master/components/gcp/ml_engine/train/component.yaml'
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

mlengine_train_op = comp.load_component_from_url(COMPONENT_SPEC_URI)
display(mlengine_train_op)
```

### Here is an illustrative pipeline that uses the component


```python
import kfp.dsl as dsl
import kfp.gcp as gcp
import json
@dsl.pipeline(
    name='CloudML training pipeline',
    description='CloudML training pipeline'
)
def pipeline(
    project_id,
    python_module,
    package_uris,
    region,
    args = '',
    job_dir = '',
    python_version = '',
    runtime_version = '',
    master_image_uri = '',
    worker_image_uri = '',
    training_input = '',
    job_id_prefix = '',
    wait_interval = '30'):
    task = mlengine_train_op(project_id, python_module, package_uris, region, args, job_dir, python_version,
        runtime_version, master_image_uri, worker_image_uri, training_input, job_id_prefix, 
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
    'python_module': 'trainer.task',
    'package_uris': json.dumps([
        'gs://ml-pipeline-playground/samples/ml_engine/cencus/trainer.tar.gz'
    ]),
    'region': 'us-central1',
    'args': json.dumps([
        '--train-files', 'gs://cloud-samples-data/ml-engine/census/data/adult.data.csv',
        '--eval-files', 'gs://cloud-samples-data/ml-engine/census/data/adult.test.csv',
        '--train-steps', '1000',
        '--eval-steps', '100',
        '--verbosity', 'DEBUG'
    ]),
    'job_dir': GCS_WORKING_DIR + '/train/output/',
    'runtime_version': '1.10'
}

#Get or create an experiment and submit a pipeline run
import kfp
client = kfp.Client()
experiment = client.create_experiment(EXPERIMENT_NAME)

#Submit a pipeline run
run_name = pipeline_func.__name__ + ' run'
run_result = client.run_pipeline(experiment.id, run_name, pipeline_filename, arguments)
```
