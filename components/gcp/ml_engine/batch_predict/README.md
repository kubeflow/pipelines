
# Batch predicting using Cloud Machine Learning Engine
A Kubeflow Pipeline component to submit a batch prediction job against a trained model to Cloud ML Engine service.

## Intended use
Use the component to run a batch prediction job against a deployed model in Cloud Machine Learning Engine. The prediction output will be stored in a Cloud Storage bucket.

## Runtime arguments
Name | Description | Type | Optional | Default
:--- | :---------- | :--- | :------- | :------
project_id | The ID of the parent project of the job. | GCPProjectID | No |
model_path | Required. The path to the model. It can be one of the following paths:<ul><li>`projects/[PROJECT_ID]/models/[MODEL_ID]`</li><li>`projects/[PROJECT_ID]/models/[MODEL_ID]/versions/[VERSION_ID]`</li><li>Cloud Storage path of a model file.</li></ul> | String | No |
input_paths | The Cloud Storage location of the input data files. May contain wildcards. For example: `gs://foo/*.csv` | List | No |
input_data_format | The format of the input data files. See [DataFormat](https://cloud.google.com/ml-engine/reference/rest/v1/projects.jobs#DataFormat). | String | No |
output_path | The Cloud Storage location for the output data. | GCSPath | No |
region | The region in Compute Engine where the  prediction job is run. | GCPRegion | No |
output_data_format | The format of the output data files. See [DataFormat](https://cloud.google.com/ml-engine/reference/rest/v1/projects.jobs#DataFormat). | String | Yes | `JSON`
prediction_input | The JSON input parameters to create a prediction job. See [PredictionInput](https://cloud.google.com/ml-engine/reference/rest/v1/projects.jobs#PredictionInput) to know more. | Dict | Yes | ` `
job_id_prefix | The prefix of the generated job id. | String | Yes | ` `
wait_interval | A time-interval to wait for in case the operation has a long run time. | Integer | Yes | `30`

## Output
Name | Description | Type
:--- | :---------- | :---
job_id | The ID of the created batch job. | String
output_path | The output path of the batch prediction job | GCSPath


## Cautions & requirements

To use the component, you must:
* Setup cloud environment by following the [guide](https://cloud.google.com/ml-engine/docs/tensorflow/getting-started-training-prediction#setup).
* The component is running under a secret of [Kubeflow user service account](https://www.kubeflow.org/docs/started/getting-started-gke/#gcp-service-accounts) in a Kubeflow cluster. For example:

```python
mlengine_predict_op(...).apply(gcp.use_gcp_secret('user-gcp-sa'))

```
* Grant Kubeflow user service account the read access to the Cloud Storage buckets which contains the input data.
* Grant Kubeflow user service account the write access to the Cloud Storage bucket of the output directory.


## Detailed Description

The component accepts following input data:
* A trained model: it can be a model file in Cloud Storage, or a deployed model or version in Cloud Machine Learning Engine. The path to the model is specified by the `model_path` parameter.
* Input data: the data will be used to make predictions against the input trained model. The data can be in [multiple formats](https://cloud.google.com/ml-engine/reference/rest/v1/projects.jobs#DataFormat). The path of the data is specified by `input_paths` parameter and the format is specified by `input_data_format` parameter.

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

mlengine_batch_predict_op = comp.load_component_from_url(
    'https://raw.githubusercontent.com/kubeflow/pipelines/d2f5cc92a46012b9927209e2aaccab70961582dc/components/gcp/ml_engine/batch_predict/component.yaml')
help(mlengine_batch_predict_op)
```

For more information about the component, please checkout:
* [Component python code](https://github.com/kubeflow/pipelines/blob/master/component_sdk/python/kfp_component/google/ml_engine/_batch_predict.py)
* [Component docker file](https://github.com/kubeflow/pipelines/blob/master/components/gcp/container/Dockerfile)
* [Sample notebook](https://github.com/kubeflow/pipelines/blob/master/components/gcp/ml_engine/batch_predict/sample.ipynb)
* [Cloud Machine Learning Engine job REST API](https://cloud.google.com/ml-engine/reference/rest/v1/projects.jobs)

### Sample Code

Note: the sample code below works in both IPython notebook or python code directly.

In this sample, we batch predict against a pre-built trained model from `gs://ml-pipeline-playground/samples/ml_engine/census/trained_model/` and use the test data from `gs://ml-pipeline-playground/samples/ml_engine/census/test.json`. 

#### Inspect the test data


```python
!gsutil cat gs://ml-pipeline-playground/samples/ml_engine/census/test.json
```

#### Set sample parameters


```python
# Required Parameters
PROJECT_ID = '<Please put your project ID here>'
GCS_WORKING_DIR = 'gs://<Please put your GCS path here>' # No ending slash
```


```python
# Optional Parameters
EXPERIMENT_NAME = 'CLOUDML - Batch Predict'
OUTPUT_GCS_PATH = GCS_WORKING_DIR + '/batch_predict/output/'
```

#### Example pipeline that uses the component


```python
import kfp.dsl as dsl
import kfp.gcp as gcp
import json
@dsl.pipeline(
    name='CloudML batch predict pipeline',
    description='CloudML batch predict pipeline'
)
def pipeline(
    project_id = PROJECT_ID, 
    model_path = 'gs://ml-pipeline-playground/samples/ml_engine/census/trained_model/', 
    input_paths = '["gs://ml-pipeline-playground/samples/ml_engine/census/test.json"]', 
    input_data_format = 'JSON', 
    output_path = OUTPUT_GCS_PATH, 
    region = 'us-central1', 
    output_data_format='', 
    prediction_input = json.dumps({
        'runtimeVersion': '1.10'
    }), 
    job_id_prefix='',
    wait_interval='30'):
        mlengine_batch_predict_op(
            project_id=project_id, 
            model_path=model_path, 
            input_paths=input_paths, 
            input_data_format=input_data_format, 
            output_path=output_path, 
            region=region, 
            output_data_format=output_data_format, 
            prediction_input=prediction_input, 
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

#### Inspect prediction results


```python
OUTPUT_FILES_PATTERN = OUTPUT_GCS_PATH + '*'
!gsutil cat OUTPUT_FILES_PATTERN
```
