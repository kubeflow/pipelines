
# CloudML - Deploy

## Intended Use
A Kubeflow Pipeline component to deploy a trained model from a Google Cloud Storage path to Google Cloud Machine Learning Engine service.

## Runtime Parameters:
Name | Description
:--- | :----------
model_uri | Required, the GCS URI which contains a model file. Common used TF model search path (export/exporter) will be used if exist. 
project_id | Required. The ID of the parent project.
model_id | Optional, the user provided name of the model.
version_id | Optional, the user provided name of the version. If it is not provided, the operation uses a random name.
runtime_version | Optinal, the Cloud ML Engine runtime version to use for this deployment. If not set, Cloud ML Engine uses the default stable version, 1.0. 
python_version | optinal, the version of Python used in prediction. If not set, the default version is `2.7`. Python `3.5` is available when runtimeVersion is set to `1.4` and above. Python `2.7` works with all supported runtime versions.
version | Optional, the payload of the new version.
replace_existing_version | Boolean flag indicates whether to replace existing version in case of conflict. Defaults to false.
set_default | boolean flag indicates whether to set the new version as default version in the model. Defaults to false.
wait_interval | Optional interval to wait for a long running operation. Defaults to 30.

## Output:
Name | Description
:--- | :----------
model_uri | The GCS URI for the found model.
version_name | The deployed version resource name.

## Sample Code

Note: the sample code below works in both IPython notebook or python code directly.

### Set sample parameters


```python
# Required Parameters
PROJECT_ID = '<Please put your project ID here>'

# Optional Parameters
EXPERIMENT_NAME = 'CLOUDML - Deploy'
COMPONENT_SPEC_URI = 'https://raw.githubusercontent.com/kubeflow/pipelines/d2f5cc92a46012b9927209e2aaccab70961582dc/components/gcp/ml_engine/deploy/component.yaml'
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

mlengine_deploy_op = comp.load_component_from_url(COMPONENT_SPEC_URI)
display(mlengine_deploy_op)
```

### Here is an illustrative pipeline that uses the component


```python
import kfp.dsl as dsl
import kfp.gcp as gcp
import json
@dsl.pipeline(
    name='CloudML deploy pipeline',
    description='CloudML deploy pipeline'
)
def pipeline(
    model_uri,
    project_id,
    model_id = '',
    version_id = '',
    runtime_version = '',
    python_version = '',
    version = '',
    replace_existing_version = 'False',
    set_default = 'False',
    wait_interval = '30'):
    task = mlengine_deploy_op(model_uri, project_id, model_id, version_id, runtime_version, 
        python_version, version, replace_existing_version, set_default, 
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
    'model_uri': 'gs://ml-pipeline-playground/samples/ml_engine/census/trained_model/',
    'project_id': PROJECT_ID,
    'model_id': 'kfp_sample_model',
    'runtime_version': '1.10',
    'set_default': 'True'
}

#Get or create an experiment and submit a pipeline run
import kfp
client = kfp.Client()
experiment = client.create_experiment(EXPERIMENT_NAME)

#Submit a pipeline run
run_name = pipeline_func.__name__ + ' run'
run_result = client.run_pipeline(experiment.id, run_name, pipeline_filename, arguments)
```
