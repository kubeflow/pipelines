
# Name

Batch prediction using Cloud Machine Learning Engine


# Label

Cloud Storage, Cloud ML Engine, Kubeflow, Pipeline, Component


# Summary

A Kubeflow Pipeline component to submit a batch prediction job against a deployed model on Cloud ML Engine.


# Details


## Intended use

Use the component to run a batch prediction job against a deployed model on Cloud ML Engine. The prediction output is stored in a Cloud Storage bucket.


## Runtime arguments

| Argument | Description | Optional | Data type | Accepted values | Default |
|--------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|----------|--------------|-----------------|---------|
| project_id | The ID of the Google Cloud Platform (GCP) project of the job. | No | GCPProjectID |  |  |
| model_path | The path to the model. It can be one of the following:<br/> <ul>   <li>projects/[PROJECT_ID]/models/[MODEL_ID]</li>  <li>projects/[PROJECT_ID]/models/[MODEL_ID]/versions/[VERSION_ID]</li> <li>The path to a Cloud Storage location containing a model file.</li>  </ul> | No | GCSPath |  |  |
| input_paths | The path to the Cloud Storage location containing the input data files. It can contain wildcards, for example, `gs://foo/*.csv` | No | List | GCSPath |  |
| input_data_format | The format of the input data files. See [REST Resource: projects.jobs](https://cloud.google.com/ml-engine/reference/rest/v1/projects.jobs#DataFormat)  for more details. | No | String | DataFormat |  |
| output_path | The path to the Cloud Storage location for the output data. | No | GCSPath |  |  |
| region | The Compute Engine region where the prediction job is run. | No | GCPRegion |  |  |
| output_data_format | The format of the output data files. See [REST Resource: projects.jobs](https://cloud.google.com/ml-engine/reference/rest/v1/projects.jobs#DataFormat) for more details. | Yes | String | DataFormat | JSON |
| prediction_input | The JSON input parameters to create a prediction job. See [PredictionInput](https://cloud.google.com/ml-engine/reference/rest/v1/projects.jobs#PredictionInput) for more information. | Yes | Dict |  | None |
| job_id_prefix | The prefix of the generated job id. | Yes | String |  | None |
| wait_interval | The number of seconds to wait in case the operation has a long run time. | Yes |  |  | 30 |


## Input data schema

The component accepts the following as input:

*   A trained model: It can be a model file in Cloud Storage, a deployed model, or a version in Cloud ML Engine. Specify the path to the model in the `model_path `runtime argument.
*   Input data: The data used to make predictions against the trained model. The data can be in [multiple formats](https://cloud.google.com/ml-engine/reference/rest/v1/projects.jobs#DataFormat). The data path is specified by `input_paths` and the format is specified by `input_data_format`.

## Output
Name | Description | Type
:--- | :---------- | :---
job_id | The ID of the created batch job. | String


## Cautions & requirements

To use the component, you must:

*   Set up a cloud environment by following this [guide](https://cloud.google.com/ml-engine/docs/tensorflow/getting-started-training-prediction#setup).
*   The component can authenticate to GCP. Refer to [Authenticating Pipelines to GCP](https://www.kubeflow.org/docs/gke/authentication-pipelines/) for details.
*   Grant the following types of access to the Kubeflow user service account:
    *   Read access to the Cloud Storage buckets which contains the input data.
    *   Write access to the Cloud Storage bucket of the output directory.


## Detailed description

Follow these steps to use the component in a pipeline:



1.  Install the Kubeflow Pipeline SDK:




```python
%%capture --no-stderr

!pip3 install kfp --upgrade
```

2. Load the component using KFP SDK


```python
import kfp.components as comp

mlengine_batch_predict_op = comp.load_component_from_url(
    'https://raw.githubusercontent.com/kubeflow/pipelines/1.4.0-rc.1/components/gcp/ml_engine/batch_predict/component.yaml')
help(mlengine_batch_predict_op)
```


### Sample Code
Note: The following sample code works in an IPython notebook or directly in Python code. 

In this sample, you batch predict against a pre-built trained model from `gs://ml-pipeline-playground/samples/ml_engine/census/trained_model/` and use the test data from `gs://ml-pipeline-playground/samples/ml_engine/census/test.json`.

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

#### Inspect prediction results


```python
OUTPUT_FILES_PATTERN = OUTPUT_GCS_PATH + '*'
!gsutil cat OUTPUT_FILES_PATTERN
```

## References
* [Component python code](https://github.com/kubeflow/pipelines/blob/master/components/gcp/container/component_sdk/python/kfp_component/google/ml_engine/_batch_predict.py)
* [Component docker file](https://github.com/kubeflow/pipelines/blob/master/components/gcp/container/Dockerfile)
* [Sample notebook](https://github.com/kubeflow/pipelines/blob/master/components/gcp/ml_engine/batch_predict/sample.ipynb)
* [Cloud Machine Learning Engine job REST API](https://cloud.google.com/ml-engine/reference/rest/v1/projects.jobs)

## License
By deploying or using this software you agree to comply with the [AI Hub Terms of Service](https://aihub.cloud.google.com/u/0/aihub-tos) and the [Google APIs Terms of Service](https://developers.google.com/terms/). To the extent of a direct conflict of terms, the AI Hub Terms of Service will control.
