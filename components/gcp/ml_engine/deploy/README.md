
# Name

Deploying a trained model to Cloud Machine Learning Engine 


# Label

Cloud Storage, Cloud ML Engine, Kubeflow, Pipeline


# Summary

A Kubeflow Pipeline component to deploy a trained model from a Cloud Storage location to Cloud ML Engine.


# Details


## Intended use

Use the component to deploy a trained model to Cloud ML Engine. The deployed model can serve online or batch predictions in a Kubeflow Pipeline.


## Runtime arguments

| Argument | Description | Optional | Data type | Accepted values | Default |
|--------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|----------|--------------|-----------------|---------|
| model_uri | The URI of a Cloud Storage directory that contains a trained model file.<br/> Or <br/> An [Estimator export base directory](https://www.tensorflow.org/guide/saved_model#perform_the_export) that contains a list of subdirectories named by timestamp. The directory with the latest timestamp is used to load the trained model file. | No | GCSPath |  |  |
| project_id | The ID of the Google Cloud Platform (GCP) project of the serving model. | No | GCPProjectID |  |  |
| model_id | The name of the trained model. | Yes | String |  | None |
| version_id | The name of the version of the model. If it is not provided, the operation uses a random name. | Yes | String |  | None |
| runtime_version | The Cloud ML Engine runtime version to use for this deployment. If it is not provided, the default stable version, 1.0, is used. | Yes | String |  | None |
| python_version | The version of Python used in the prediction. If it is not provided, version 2.7 is used. You can use Python 3.5 if runtime_version is set to 1.4 or above. Python 2.7 works with all supported runtime versions. | Yes | String |  | 2.7 |
| model | The JSON payload of the new [model](https://cloud.google.com/ml-engine/reference/rest/v1/projects.models). | Yes | Dict |  | None |
| version | The new [version](https://cloud.google.com/ml-engine/reference/rest/v1/projects.models.versions) of the trained model. | Yes | Dict |  | None |
| replace_existing_version | Indicates whether to replace the existing version in case of a conflict (if the same version number is found.) | Yes | Boolean |  | FALSE |
| set_default | Indicates whether to set the new version as the default version in the model. | Yes | Boolean |  | FALSE |
| wait_interval | The number of seconds to wait in case the operation has a long run time. | Yes | Integer |  | 30 |



## Input data schema

The component looks for a trained model in the location specified by the  `model_uri` runtime argument. The accepted trained models are:


*   [Tensorflow SavedModel](https://cloud.google.com/ml-engine/docs/tensorflow/exporting-for-prediction) 
*   [Scikit-learn & XGBoost model](https://cloud.google.com/ml-engine/docs/scikit/exporting-for-prediction)

The accepted file formats are:

*   *.pb
*   *.pbtext
*   model.bst
*   model.joblib
*   model.pkl

`model_uri` can also be an [Estimator export base directory, ](https://www.tensorflow.org/guide/saved_model#perform_the_export)which contains a list of subdirectories named by timestamp. The directory with the latest timestamp is used to load the trained model file.

## Output
Name | Description | Type
:--- | :---------- | :---
| model_uri | The Cloud Storage URI of the trained model.  |  GCSPath |
| model_name | The name of the deployed model. |  String  |
| version_name | The name of the deployed version. |  String  |


## Cautions & requirements

To use the component, you must:

*   [Set up the cloud environment](https://cloud.google.com/ml-engine/docs/tensorflow/getting-started-training-prediction#setup).
*   The component can authenticate to GCP. Refer to [Authenticating Pipelines to GCP](https://www.kubeflow.org/docs/gke/authentication-pipelines/) for details.
*   Grant read access to the Cloud Storage bucket that contains the trained model to the Kubeflow user service account.

## Detailed description

Use the component to: 
*   Locate the trained model at the Cloud Storage location you specify.
*   Create a new model if a model provided by you doesn’t exist.
*   Delete the existing model version if `replace_existing_version` is enabled.
*   Create a new version of the model from the trained model.
*   Set the new version as the default version of the model if `set_default` is enabled.

Follow these steps to use the component in a pipeline:

1.  Install the Kubeflow Pipeline SDK:




```python
%%capture --no-stderr

!pip3 install kfp --upgrade
```

2. Load the component using KFP SDK


```python
import kfp.components as comp

mlengine_deploy_op = comp.load_component_from_url(
    'https://raw.githubusercontent.com/kubeflow/pipelines/1.4.0-rc.1/components/gcp/ml_engine/deploy/component.yaml')
help(mlengine_deploy_op)
```

### Sample
Note: The following sample code works in IPython notebook or directly in Python code.

In this sample, you deploy a pre-built trained model from `gs://ml-pipeline-playground/samples/ml_engine/census/trained_model/` to Cloud ML Engine. The deployed model is `kfp_sample_model`. A new version is created every time the sample is run, and the latest version is set as the default version of the deployed model.

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
    version = {},
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

## References
* [Component python code](https://github.com/kubeflow/pipelines/blob/master/components/gcp/container/component_sdk/python/kfp_component/google/ml_engine/_deploy.py)
* [Component docker file](https://github.com/kubeflow/pipelines/blob/master/components/gcp/container/Dockerfile)
* [Sample notebook](https://github.com/kubeflow/pipelines/blob/master/components/gcp/ml_engine/deploy/sample.ipynb)
* [Cloud Machine Learning Engine Model REST API](https://cloud.google.com/ml-engine/reference/rest/v1/projects.models)
* [Cloud Machine Learning Engine Version REST API](https://cloud.google.com/ml-engine/reference/rest/v1/projects.versions)

## License
By deploying or using this software you agree to comply with the [AI Hub Terms of Service](https://aihub.cloud.google.com/u/0/aihub-tos) and the [Google APIs Terms of Service](https://developers.google.com/terms/). To the extent of a direct conflict of terms, the AI Hub Terms of Service will control.
