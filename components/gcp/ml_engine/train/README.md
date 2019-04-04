
# Submitting a Cloud ML training job as a pipeline step
A Kubeflow Pipeline component to submit a Cloud Machine Learning (Cloud ML) Engine training job as a step in a pipeline.

## Intended Use
This component is intended to submit a training job to Cloud Machine Learning (ML) Engine from a Kubeflow Pipelines workflow.

## Runtime arguments
Name | Description | Type | Optional | Default
:--- | :---------- | :--- | :------- | :------
project_id | The ID of the parent project of the job. | GCPProjectID | No |
python_module | The Python module name to run after installing the packages. | String | Yes | ``
package_uris | The Cloud Storage location of the packages (that contain the training program and any additional dependencies). The maximum number of package URIs is 100. | List<GCSPath> | Yes | ``
region | The Compute Engine region in which the training job is run. | GCPRegion | Yes | ``
args | The command line arguments to pass to the program. | List<String> | Yes | ``
job_dir |  The list of arguments to pass to the Python file. | GCSPath | Yes | ``
python_version | A Cloud Storage path in which to store the training outputs and other data needed for training. This path is passed to your TensorFlow program as the `job-dir` command-line argument. The benefit of specifying this field is that Cloud ML validates the path for use in training. | String | Yes | ``
runtime_version | The Cloud ML Engine runtime version to use for training. If not set, Cloud ML Engine uses the default stable version, 1.0. | String | Yes | ``
master_image_uri | The Docker image to run on the master replica. This image must be in Container Registry. | GCRPath | Yes | ``
worker_image_uri | The Docker image to run on the worker replica. This image must be in Container Registry. | GCRPath | Yes | ``
training_input | The input parameters to create a training job. It is the JSON payload of a [TrainingInput](https://cloud.google.com/ml-engine/reference/rest/v1/projects.jobs#TrainingInput) | Dict | Yes | ``
job_id_prefix | The prefix of the generated job id. | String | Yes | ``
wait_interval |  A time-interval to wait for between calls to get the job status. | Integer | Yes | `30`

## Outputs
Name | Description | Type
:--- | :---------- | :---
job_id | The ID of the created job. | String
job_dir | The output path in Cloud Storage of the trainning job, which contains the trained model files. | GCSPath

## Cautions & requirements

To use the component, you must:
* Setup cloud environment by following the [guide](https://cloud.google.com/ml-engine/docs/tensorflow/getting-started-training-prediction#setup).
* The component is running under a secret of [Kubeflow user service account](https://www.kubeflow.org/docs/started/getting-started-gke/#gcp-service-accounts) in a Kubeflow cluster. For example:

```python
mlengine_train_op(...).apply(gcp.use_gcp_secret('user-gcp-sa'))

```
* Grant Kubeflow user service account the read access to the Cloud Storage buckets which contains the input data, packages or docker images.
* Grant Kubeflow user service account the write access to the Cloud Storage bucket of the output directory.


## Detailed Description

The component accepts one of the two types of executable inputs:
* A list of Python packages from Cloud Storage. You may manually build a Python package by following the [guide](https://cloud.google.com/ml-engine/docs/tensorflow/packaging-trainer#manual-build) and [upload it to Cloud Storage](https://cloud.google.com/ml-engine/docs/tensorflow/packaging-trainer#uploading_packages_manually).
* Docker container from Google Container Registry (GCR). Follow the [guide](https://cloud.google.com/ml-engine/docs/using-containers) to publish and use a Docker container with this component. 

The component builds the payload of a [TrainingInput](https://cloud.google.com/ml-engine/reference/rest/v1/projects.jobs#TrainingInput) and submit a job by Cloud Machine Learning Engine REST API.

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

mlengine_train_op = comp.load_component_from_url(
    'https://raw.githubusercontent.com/kubeflow/pipelines/d2f5cc92a46012b9927209e2aaccab70961582dc/components/gcp/ml_engine/train/component.yaml')
help(mlengine_train_op)
```

For more information about the component, please checkout:
* [Component python code](https://github.com/kubeflow/pipelines/blob/master/component_sdk/python/kfp_component/google/ml_engine/_train.py)
* [Component docker file](https://github.com/kubeflow/pipelines/blob/master/components/gcp/container/Dockerfile)
* [Sample notebook](https://github.com/kubeflow/pipelines/blob/master/components/gcp/ml_engine/train/sample.ipynb)
* [Cloud Machine Learning Engine job REST API](https://cloud.google.com/ml-engine/reference/rest/v1/projects.jobs)


### Sample
Note: The following sample code works in IPython notebook or directly in Python code.

In this sample, we use the code from [census estimator sample](https://github.com/GoogleCloudPlatform/cloudml-samples/tree/master/census/estimator) to train a model in Cloud Machine Learning Engine service. In order to pass the code to the service, we need to package the python code and upload it in a Cloud Storage bucket. Make sure that you have read and write permissions on the bucket that you use as the working directory. 

#### Set sample parameters


```python
# Required Parameters
PROJECT_ID = '<Please put your project ID here>'
GCS_WORKING_DIR = 'gs://<Please put your GCS path here>' # No ending slash
```


```python
# Optional Parameters
EXPERIMENT_NAME = 'CLOUDML - Train'
TRAINER_GCS_PATH = GCS_WORKING_DIR + '/train/trainer.tar.gz'
OUTPUT_GCS_PATH = GCS_WORKING_DIR + '/train/output/'
```

#### Clean up the working directory


```python
%%capture --no-stderr
!gsutil rm -r $GCS_WORKING_DIR
```

#### Download the sample trainer code to local


```python
%%capture --no-stderr
!wget https://github.com/GoogleCloudPlatform/cloudml-samples/archive/master.zip
!unzip master.zip
```

#### Package code and upload the package to Cloud Storage


```python
%%capture --no-stderr
%%bash -s "$TRAINER_GCS_PATH"
pushd ./cloudml-samples-master/census/estimator/
python setup.py sdist
gsutil cp dist/preprocessing-1.0.tar.gz $1
popd
rm -fr ./cloudml-samples-master/ ./master.zip ./dist
```

#### Example pipeline that uses the component


```python
import kfp.dsl as dsl
import kfp.gcp as gcp
import json
@dsl.pipeline(
    name='CloudML training pipeline',
    description='CloudML training pipeline'
)
def pipeline(
    project_id = PROJECT_ID,
    python_module = 'trainer.task',
    package_uris = json.dumps([TRAINER_GCS_PATH]),
    region = 'us-central1',
    args = json.dumps([
        '--train-files', 'gs://cloud-samples-data/ml-engine/census/data/adult.data.csv',
        '--eval-files', 'gs://cloud-samples-data/ml-engine/census/data/adult.test.csv',
        '--train-steps', '1000',
        '--eval-steps', '100',
        '--verbosity', 'DEBUG'
    ]),
    job_dir = OUTPUT_GCS_PATH,
    python_version = '',
    runtime_version = '1.10',
    master_image_uri = '',
    worker_image_uri = '',
    training_input = '',
    job_id_prefix = '',
    wait_interval = '30'):
    task = mlengine_train_op(
        project_id=project_id, 
        python_module=python_module, 
        package_uris=package_uris, 
        region=region, 
        args=args, 
        job_dir=job_dir, 
        python_version=python_version,
        runtime_version=runtime_version, 
        master_image_uri=master_image_uri, 
        worker_image_uri=worker_image_uri, 
        training_input=training_input, 
        job_id_prefix=job_id_prefix, 
        wait_interval=wait_interval).apply(gcp.use_gcp_secret('user-gcp-sa'))
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

#### Inspect the results

Follow the `Run` link to open the KFP UI. In the step logs, you should be able to click on the links to:
* Job dashboard
* And realtime logs on Stackdriver

Use the following command to inspect the contents in the output directory:


```python
!gsutil ls $OUTPUT_GCS_PATH
```
