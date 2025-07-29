# Seldon Core - Serve PyTorch Model

## Intended Use
Serve PyTorch Models remotely as web service using Seldon Core
  
## Run-Time Parameters:
Name | Description
:--- | :----------
model_id | Required. Model training_id from Fabric for Deep Learning
deployment_name | Required. Deployment name for the seldon service
model_class_name | PyTorch model class name', default: 'ModelClass'
model_class_file | File that contains the PyTorch model class', default: 'model_class.py'
serving_image | Model serving images', default: 'aipipeline/seldon-pytorch:0.1

## Output:
Name | Description
:--- | :----------
output | Model Serving status

## Sample

Note: the sample code below works in both IPython notebook or python code directly.

### Set sample parameters
```python
# Parameters
model_id = 'Model training_id'
deployment_name = 'Deployment name for the seldon service'
model_class_name = 'PyTorch model class name'
model_class_file = 'File that contains the PyTorch model class'
serving_image =  'aipipeline/seldon-pytorch:0.1'
```

```python
# Additional Parameters
EXPERIMENT_NAME = 'Seldon Core - Serve PyTorch Model'
COMPONENT_SPEC_URI = 'https://raw.githubusercontent.com/kubeflow/pipelines/eb830cd73ca148e5a1a6485a9374c2dc068314bc/components/ibm-components/ffdl/serve/component.yaml'

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

ffdl_serve_op = comp.load_component_from_url(COMPONENT_SPEC_URI)
display(ffdl_serve_op)
```

### Here is an illustrative pipeline that uses the component


```python
import kfp.dsl as dsl
import ai_pipeline_params as params
import json
@dsl.pipeline(
    name='FfDL Serve Pipeline',
    description='FfDL Serve pipeline leveraging Sledon'
)
def ffdl_train_pipeline(
   model_id,
   deployment_name,
   model_class_name,
   model_class_file,
   serving_image
):
    ffdl_serve_op(model_id, deployment_name,model_class_name,model_class_file,serving_image).apply(params.use_ai_pipeline_params('kfp-creds'))
```

### Compile the pipeline


```python
pipeline_func = ffdl_serve_pipeline
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
