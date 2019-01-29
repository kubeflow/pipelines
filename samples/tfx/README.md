The sample runs a pipeline with TensorFlow's transform and model-analysis components. The `taxi-cab-classification-pipeline-gcp.py` is for GCP and `taxi-cab-classification-pipeline-on-prem.py` is for on-prem cluster.

## The dataset

This sample is based on the model-analysis example [here](https://github.com/tensorflow/model-analysis/tree/master/examples/chicago_taxi).

The sample trains and analyzes a model based on the [Taxi Trips dataset](https://data.cityofchicago.org/Transportation/Taxi-Trips/wrvz-psew)
released by the City of Chicago.

Note: This site provides applications using data that has been modified
for use from its original source, www.cityofchicago.org, the official website of
the City of Chicago. The City of Chicago makes no claims as to the content,
accuracy, timeliness, or completeness of any of the data provided at this site.
The data provided at this site is subject to change at any time. It is understood
that the data provided at this site is being used at one’s own risk.

[Read more](https://cloud.google.com/bigquery/public-data/chicago-taxi) about the dataset in [Google BigQuery](https://cloud.google.com/bigquery/). Explore the full dataset in the [BigQuery UI](https://bigquery.cloud.google.com/dataset/bigquery-public-data:chicago_taxi_trips).


## Requirements

### GCP

Preprocessing and model analysis use [Apache Beam](https://beam.apache.org/).

When run with the `cloud` mode (instead of the `local` mode), those steps use [Google Cloud DataFlow](https://beam.apache.org/) for running the Beam pipelines.

Therefore, you must enable the DataFlow API for the given GCP project if you want to use `cloud` as the mode for either preprocessing or analysis. See the [guide to enabling the DataFlow API](https://cloud.google.com/endpoints/docs/openapi/enable-api).

### On-prem cluster

When run the on-prem clusters, follow the [document](https://kubernetes.io/docs/concepts/storage/persistent-volumes/) to create the persistent volume and persistent volume claim to storage the intermediate data and result. Note that the `accessModes` should be `ReadWriteMany` so that the volume can be mounted as read-write by many nodes.

**Limitation**: The pvc_name parameter does not support setting from Pipelines UI. The default value of the PVC name is `pipeline-pvc`, if you want to use a different name, you must be update the `taxi-cab-classification-pipeline-on-prem.py` to replace `pipeline-pvc` with real value. See the [issue](https://github.com/kubeflow/pipelines/issues/1303) issue for details.

## Compiling the pipeline template

Follow the guide to [building a pipeline](https://www.kubeflow.org/docs/guides/pipelines/build-pipeline/) to install the Kubeflow Pipelines SDK, then run the following command to compile the sample Python into a workflow specification. The specification takes the form of a YAML file compressed into a `.tar.gz` file.

### GCP
```bash
dsl-compile --py taxi-cab-classification-pipeline-gcp.py --output taxi-cab-classification-pipeline-gcp.tar.gz
```
### On-prem cluster
```bash
dsl-compile --py taxi-cab-classification-pipeline-on-prem.py --output taxi-cab-classification-pipeline-on-prem.tar.gz
```

## Deploying the pipeline

Open the Kubeflow pipelines UI. Create a new pipeline, and then upload the compiled specification (`.tar.gz` file) as a new pipeline template.

### GCP

The pipeline requires two arguments:

1. The name of a GCP project.
2. An output directory in a Google Cloud Storage bucket, of the form `gs://<BUCKET>/<PATH>`.

### On-prem cluster

- Before deploying the pipeline, download the training and evaluation data [taxi-cab-classification](https://github.com/kubeflow/pipelines/tree/master/samples/tfx/taxi-cab-classification), and copy the **directory** to the persistent volume storage.
  
- Following the [guide](https://www.kubeflow.org/docs/pipelines/pipelines-ui/) to run an experiment and a run inside the experiment.

## Components source

Preprocessing:
  [source code](https://github.com/kubeflow/pipelines/tree/master/components/dataflow/tft/src) 
  [container](https://github.com/kubeflow/pipelines/tree/master/components/dataflow/tft)

Training:
  [source code](https://github.com/kubeflow/pipelines/tree/master/components/kubeflow/launcher/src) 
  [container](https://github.com/kubeflow/pipelines/tree/master/components/kubeflow/launcher)

Analysis:
  [source code](https://github.com/kubeflow/pipelines/tree/master/components/dataflow/tfma/src) 
  [container](https://github.com/kubeflow/pipelines/tree/master/components/dataflow/tfma)

Prediction:
  [source code](https://github.com/kubeflow/pipelines/tree/master/components/dataflow/predict/src) 
  [container](https://github.com/kubeflow/pipelines/tree/master/components/dataflow/predict)