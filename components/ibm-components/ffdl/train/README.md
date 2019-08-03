
# Fabric for Deep Learning - Train Model

## Intended Use
Train Machine Learning and Deep Learning Models remotely using Fabric for Deep Learning

## Run-Time Parameters:
Name | Description
:--- | :----------
model_def_file_path | Required. Path for model training code in object storage
manifest_file_path | Required. Path for model manifest definition in object storage

## Output:
Name | Description
:--- | :----------
output | Model training_id

## Sample

Note: the sample code below works in both IPython notebook or python code directly.

### Set sample parameters
```python
# Required Parameters
MODEL_DEF_FILE_PATH = '<Please put your path for model training code in the object storage bucket>'
MANIFEST_FILE_PATH = '<Please put your path for model manifest definition in the object storage bucket>'
```

```python
# Optional Parameters
EXPERIMENT_NAME = 'Fabric for Deep Learning - Train Model'
COMPONENT_SPEC_URI = 'https://raw.githubusercontent.com/kubeflow/pipelines/eb830cd73ca148e5a1a6485a9374c2dc068314bc/components/ibm-components/ffdl/train/component.yaml'
```

### Install KFP SDK
Install the SDK (Uncomment the code if the SDK is not installed before)


```python
#KFP_PACKAGE = 'https://storage.googleapis.com/ml-pipeline/release/0.1.12/kfp.tar.gz'
#!pip3 install $KFP_PACKAGE --upgrade
```
### Load component definitions


```python
import kfp.components as comp

ffdl_train_op = comp.load_component_from_url(COMPONENT_SPEC_URI)
display(ffdl_train_op)
```

### Here is an illustrative pipeline that uses the component


```python
import kfp.dsl as dsl
import ai_pipeline_params as params
import json
@dsl.pipeline(
    name='FfDL train pipeline',
    description='FfDL train pipeline'
)
def ffdl_train_pipeline(
    model_def_file_path=MODEL_DEF_FILE_PATH, 
    manifest_file_path=MANIFEST_FILE_PATH
):
    ffdl_train_op(model_def_file_path, manifest_file_path).apply(params.use_ai_pipeline_params('kfp-creds'))
```

### Compile the pipeline


```python
pipeline_func = ffdl_train_pipeline
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
