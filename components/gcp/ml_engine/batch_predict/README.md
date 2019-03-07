
# CloudML - Batch Predict

## Intended Use
A Kubeflow Pipeline component to submit a batch prediction job against a trained model to Google Cloud Machine Learning Engine service.

## Runtime Parameters:
Name | Description
:--- | :----------
project_id | Required. The ID of the parent project of the job.
model_path | Required. The path to the model. It can be either: `projects/[PROJECT_ID]/models/[MODEL_ID]` or `projects/[PROJECT_ID]/models/[MODEL_ID]/versions/[VERSION_ID]` or a GCS path of a model file.
input_paths | Required. The Google Cloud Storage location of the input data files. May contain wildcards.
input_data_format | Required. The format of the input data files. See https://cloud.google.com/ml-engine/reference/rest/v1/projects.jobs#DataFormat.
output_path | Required. The output Google Cloud Storage location.
region | Required. The Google Compute Engine region to run the prediction job in.
output_data_format | Optional. Format of the output data files, defaults to JSON.
prediction_input | Input parameters to create a prediction job. See [PredictionInput](https://cloud.google.com/ml-engine/reference/rest/v1/projects.jobs#PredictionInput).
job_id_prefix | The prefix of the generated job id.
wait_interval | Optional interval to wait for a long running operation. Defaults to 30.

## Output:
Name | Description
:--- | :----------
job_id | The ID of the created batch job.

## Sample Code

Note: the sample code below works in both IPython notebook or python code directly.

### Set sample parameters


```python
# Required Parameters
PROJECT_ID = '<Please put your project ID here>'
GCS_WORKING_DIR = 'gs://<Please put your GCS path here>' # No ending slash

# Optional Parameters
EXPERIMENT_NAME = 'CLOUDML - Batch Predict'
COMPONENT_SPEC_URI = 'https://raw.githubusercontent.com/kubeflow/pipelines/master/components/gcp/ml_engine/batch_predict/component.yaml'
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

mlengine_batch_predict_op = comp.load_component_from_url(COMPONENT_SPEC_URI)
display(mlengine_batch_predict_op)
```

### Here is an illustrative pipeline that uses the component


```python
import kfp.dsl as dsl
import kfp.gcp as gcp
import json
@dsl.pipeline(
    name='CloudML batch predict pipeline',
    description='CloudML batch predict pipeline'
)
def pipeline(
    project_id, 
    model_path, 
    input_paths, 
    input_data_format, 
    output_path, 
    region, 
    output_data_format='', 
    prediction_input='', 
    job_id_prefix='',
    wait_interval='30'):
    task = mlengine_batch_predict_op(project_id, model_path, input_paths, input_data_format, 
    output_path, region, output_data_format, prediction_input, job_id_prefix,
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
    'model_path': 'gs://ml-pipeline-playground/samples/ml_engine/census/trained_model/',
    'input_paths': '["gs://ml-pipeline-playground/samples/ml_engine/census/test.json"]',
    'input_data_format': 'JSON',
    'output_path': GCS_WORKING_DIR + '/batch_predict/output/',
    'region': 'us-central1',
    'prediction_input': json.dumps({
        'runtimeVersion': '1.10'
    })
}

#Get or create an experiment and submit a pipeline run
import kfp
client = kfp.Client()
experiment = client.create_experiment(EXPERIMENT_NAME)

#Submit a pipeline run
run_name = pipeline_func.__name__ + ' run'
run_result = client.run_pipeline(experiment.id, run_name, pipeline_filename, arguments)
```
