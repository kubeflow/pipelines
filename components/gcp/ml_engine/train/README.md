
# Name
Component: Submitting an AI Platform training job as a pipeline step

# Label
 AI Platform, Kubeflow

# Summary
A Kubeflow pipeline component to submit an AI Platform training job as a step in a pipeline.

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
Training

# Details
## Intended use
Use this component to submit a training job to AI Platform from a Kubeflow pipeline. 

## Runtime arguments
| Argument | Description | Optional | Data type | Accepted values | Default |
|:------------------|:------------------|:----------|:--------------|:-----------------|:-------------|
| project_id | The Google Cloud Platform (GCP) project ID of the job. | No | GCPProjectID | - | - |
| python_module | The name of the Python module to run after installing the training program. | Yes | String | - | None |
| package_uris | The Cloud Storage location of the packages that contain the training program and any additional dependencies. The maximum number of package URIs is 100. | Yes | List |  -| None |
| region | The Compute Engine region in which the training job is run. | Yes | GCPRegion |  -| us-central1 |
| args | The command line arguments to pass to the training program. | Yes | List | - | None |
| job_dir | A Cloud Storage path in which to store the training outputs and other data needed for training. This path is passed to your TensorFlow program as the command-line argument, `job-dir`. The benefit of specifying this field is that Cloud ML validates the path for use in training. | Yes | GCSPath | - | None |
| python_version | The version of Python used in training. If it is not set, the default version is 2.7. Python 3.5 is available when the runtime version is set to 1.4 and above. | Yes | String | - | None |
| runtime_version | The runtime version of AI Platform to use for training. If it is not set, AI Platform uses the default. | Yes | String | - | 1 |
| master_image_uri | The Docker image to run on the master replica. This image must be in Container Registry. | Yes | GCRPath | - | None |
| worker_image_uri | The Docker image to run on the worker replica. This image must be in Container Registry. | Yes | GCRPath |-  | None |
| training_input | The input parameters to create a training job. | Yes | Dict | [TrainingInput](https://cloud.google.com/ml-engine/reference/rest/v1/projects.jobs#TrainingInput) | None |
| job_id_prefix | The prefix of the job ID that is generated. | Yes | String | - | None |
| job_id | The ID of the job to create, takes precedence over generated job id if set. | Yes | String | - | None |
| wait_interval | The number of seconds to wait between API calls to get the status of the job. | Yes | Integer | - | 30 |



## Input data schema

The component accepts two types of inputs:
*   A list of Python packages from Cloud Storage.
    * You can manually build a Python package and upload it to Cloud Storage by following this [guide](https://cloud.google.com/ml-engine/docs/tensorflow/packaging-trainer#manual-build).
*   A Docker container from Container Registry. 
    * Follow this [guide](https://cloud.google.com/ml-engine/docs/using-containers) to publish and use a Docker container with this component.

## Output
| Name    | Description                 | Type      |
|:------- |:----                        | :---      |
| job_id  | The ID of the created job.  |  String   |
| job_dir | The Cloud Storage path that contains the output files with the trained model. |  GCSPath  |


## Cautions & requirements

To use the component, you must:

*   Set up a cloud environment by following this [guide](https://cloud.google.com/ml-engine/docs/tensorflow/getting-started-training-prediction#setup).
*   The component can authenticate to GCP. Refer to [Authenticating Pipelines to GCP](https://www.kubeflow.org/docs/gke/authentication-pipelines/) for details.
*   Grant the following access to the Kubeflow user service account: 
    *   Read access to the Cloud Storage buckets which contain the input data, packages, or Docker images.
    *   Write access to the Cloud Storage bucket of the output directory.

## Detailed description

The component builds the [TrainingInput](https://cloud.google.com/ml-engine/reference/rest/v1/projects.jobs#TrainingInput) payload and submits a job via the [AI Platform REST API](https://cloud.google.com/ml-engine/reference/rest/v1/projects.jobs).

The steps to use the component in a pipeline are:


1.  Install the Kubeflow pipeline's SDK:

    ```python
    %%capture --no-stderr

    !pip3 install kfp --upgrade
    ```

2. Load the component using the Kubeflow pipeline's SDK:

    ```python
    import kfp.components as comp

    mlengine_train_op = comp.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/1.4.0-rc.1/components/gcp/ml_engine/train/component.yaml')
    help(mlengine_train_op)
    ```
### Sample
The following sample code works in an IPython notebook or directly in Python code.

In this sample, you use the code from the [census estimator sample](https://github.com/GoogleCloudPlatform/cloudml-samples/tree/master/census/estimator) to train a model on AI Platform. To upload the code to AI Platform, package the Python code and upload it to a Cloud Storage bucket. 

Note: You must have read and write permissions on the bucket that you use as the working directory.

#### Set sample parameters

```python
# Required parameters
PROJECT_ID = '<Put your project ID here>'
GCS_WORKING_DIR = 'gs://<Put your GCS path here>' # No ending slash
```

```python
# Optional parameters
EXPERIMENT_NAME = 'CLOUDML - Train'
TRAINER_GCS_PATH = GCS_WORKING_DIR + '/train/trainer.tar.gz'
OUTPUT_GCS_PATH = GCS_WORKING_DIR + '/train/output/'
```

#### Clean up the working directory

```python
%%capture --no-stderr
!gsutil rm -r $GCS_WORKING_DIR
```

#### Download the sample trainer code to a local directory

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
    job_id = '',
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
        job_id=job_id,
        wait_interval=wait_interval)
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

#### Inspect the results

Use the following command to inspect the contents in the output directory:

```python
!gsutil ls $OUTPUT_GCS_PATH
```

## References
* [Component Python code](https://github.com/kubeflow/pipelines/blob/master/components/gcp/container/component_sdk/python/kfp_component/google/ml_engine/_train.py)
* [Component Docker file](https://github.com/kubeflow/pipelines/blob/master/components/gcp/container/Dockerfile)
* [Sample notebook](https://github.com/kubeflow/pipelines/blob/master/components/gcp/ml_engine/train/sample.ipynb)
* [AI Platform REST API - Resource: Job](https://cloud.google.com/ml-engine/reference/rest/v1/projects.jobs)

## License
By deploying or using this software you agree to comply with the [AI Hub Terms of Service](https://aihub.cloud.google.com/u/0/aihub-tos) and the [Google APIs Terms of Service](https://developers.google.com/terms/). To the extent of a direct conflict of terms, the AI Hub Terms of Service will control.
