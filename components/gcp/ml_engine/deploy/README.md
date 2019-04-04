
# Deploying a trained model to Cloud Machine Learning Engine
A Kubeflow Pipeline component to deploy a trained model from a Cloud Storage path to a Cloud Machine Learning Engine service.

## Intended use
Use the component to deploy a trained model to Cloud Machine Learning Engine service. The deployed model can serve online or batch predictions in a KFP pipeline.

## Runtime arguments:
Name | Description | Type | Optional | Default
:--- | :---------- | :--- | :------- | :------
model_uri | The Cloud Storage URI which contains a model file. Commonly used TF model search paths (export/exporter) will be used. | GCSPath | No |
project_id | The ID of the parent project of the serving model. | GCPProjectID | No | 
model_id | The user-specified name of the model. If it is not provided, the operation uses a random name. | String | Yes | ` `
version_id | The user-specified name of the version. If it is not provided, the operation uses a random name. | String | Yes | ` `
runtime_version | The [Cloud ML Engine runtime version](https://cloud.google.com/ml-engine/docs/tensorflow/runtime-version-list) to use for this deployment. If it is not set, the Cloud ML Engine uses the default stable version, 1.0. | String | Yes | ` ` 
python_version | The version of Python used in the prediction. If it is not set, the default version is `2.7`. Python `3.5` is available when the runtime_version is set to `1.4` and above. Python `2.7` works with all supported runtime versions. | String | Yes | ` `
version | The JSON payload of the new [Version](https://cloud.google.com/ml-engine/reference/rest/v1/projects.models.versions). | Dict | Yes | ` `
replace_existing_version | A Boolean flag that indicates whether to replace existing version in case of conflict. | Bool | Yes | False
set_default | A Boolean flag that indicates whether to set the new version as default version in the model. | Bool | Yes | False
wait_interval | A time-interval to wait for in case the operation has a long run time. | Integer | Yes | 30

## Output:
Name | Description | Type
:--- | :---------- | :---
model_uri | The Cloud Storage URI of the trained model. | GCSPath
model_name | The name of the serving model. | String
version_name | The name of the deployed version of the model. | String

## Cautions & requirements

To use the component, you must:
* Setup cloud environment by following the [guide](https://cloud.google.com/ml-engine/docs/tensorflow/getting-started-training-prediction#setup).
* The component is running under a secret of [Kubeflow user service account](https://www.kubeflow.org/docs/started/getting-started-gke/#gcp-service-accounts) in a Kubeflow cluster. For example:

```python
mlengine_deploy_op(...).apply(gcp.use_gcp_secret('user-gcp-sa'))

```
* Grant Kubeflow user service account the read access to the Cloud Storage buckets which contains the trained model.


## Detailed Description

The component does:
* Search for the trained model from the user provided Cloud Storage path.
* Create a new model if user provided model doesn’t exist.
* Delete the existing model version if `replace_existing_version` is enabled.
* Create a new model version from the trained model.
* Set the new version as the default version of the model if ‘set_default’ is enabled.

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

mlengine_deploy_op = comp.load_component_from_url(
    'https://raw.githubusercontent.com/kubeflow/pipelines/d2f5cc92a46012b9927209e2aaccab70961582dc/components/gcp/ml_engine/deploy/component.yaml')
help(mlengine_deploy_op)
```

For more information about the component, please checkout:
* [Component python code](https://github.com/kubeflow/pipelines/blob/master/component_sdk/python/kfp_component/google/ml_engine/_deploy.py)
* [Component docker file](https://github.com/kubeflow/pipelines/blob/master/components/gcp/container/Dockerfile)
* [Sample notebook](https://github.com/kubeflow/pipelines/blob/master/components/gcp/ml_engine/deploy/sample.ipynb)
* [Cloud Machine Learning Engine Model REST API](https://cloud.google.com/ml-engine/reference/rest/v1/projects.models)
* [Cloud Machine Learning Engine Version REST API](https://cloud.google.com/ml-engine/reference/rest/v1/projects.versions)


### Sample
Note: The following sample code works in IPython notebook or directly in Python code.

In this sample, we will deploy a pre-built trained model from `gs://ml-pipeline-playground/samples/ml_engine/census/trained_model/` to Cloud Machine Learning Engine service. The deployed model is named `kfp_sample_model`. A new version will be created every time when the sample is run, and the latest version will be set as the default version of the deployed model.

#### Set sample parameters


```python
# Required Parameters
PROJECT_ID = '<Please put your project ID here>'

# Optional Parameters
EXPERIMENT_NAME = 'CLOUDML - Deploy'
TRAINED_MODEL_PATH = 'gs://ml-pipeline-playground/samples/ml_engine/census/trained_model/'
```

#### Example pipeline that uses the component


```python
import kfp.dsl as dsl
import kfp.gcp as gcp
import json
@dsl.pipeline(
    name='CloudML deploy pipeline',
    description='CloudML deploy pipeline'
)
def pipeline(
    model_uri = 'gs://ml-pipeline-playground/samples/ml_engine/census/trained_model/',
    project_id = PROJECT_ID,
    model_id = 'kfp_sample_model',
    version_id = '',
    runtime_version = '1.10',
    python_version = '',
    version = '',
    replace_existing_version = 'False',
    set_default = 'True',
    wait_interval = '30'):
    task = mlengine_deploy_op(
        model_uri=model_uri, 
        project_id=project_id, 
        model_id=model_id, 
        version_id=version_id, 
        runtime_version=runtime_version, 
        python_version=python_version,
        version=version, 
        replace_existing_version=replace_existing_version, 
        set_default=set_default, 
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
