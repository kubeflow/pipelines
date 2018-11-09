The `taxi-cab-classification-pipeline.py` sample runs a pipeline with TensorFlow's transform and model-analysis components.

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

[Read more](https://cloud.google.com/bigquery/public-data/chicago-taxi) about the
dataset in [Google BigQuery](https://cloud.google.com/bigquery/). Explore the
full dataset in the
[BigQuery UI](https://bigquery.cloud.google.com/dataset/bigquery-public-data:chicago_taxi_trips).

## Requirements

Preprocessing and model analysis use [Apache Beam](https://beam.apache.org/).

When run with the `cloud` mode (instead of the `local` mode), those steps use [Google Cloud DataFlow](https://beam.apache.org/) for running the Beam pipelines.

Therefore, you must enable the DataFlow API for the given GCP project if you want to use `cloud` as the mode for either preprocessing or analysis. See the [guide to enabling the DataFlow API](https://cloud.google.com/endpoints/docs/openapi/enable-api).

## Compiling the pipeline template

Follow the guide to [building a pipeline](https://github.com/kubeflow/pipelines/wiki/Build-a-Pipeline) to install the Kubeflow Pipelines SDK, then run the following command to compile the sample Python into a workflow specification. The specification takes the form of a YAML file compressed into a `.tar.gz` file.

```bash
dsl-compile --py taxi-cab-classification-pipeline.py --output taxi-cab-classification-pipeline.tar.gz
```

## Deploying the pipeline

Open the Kubeflow pipelines UI. Create a new pipeline, and then upload the compiled specification (`.tar.gz` file) as a new pipeline template.

The pipeline requires two arguments:

1. The name of a GCP project.
2. An output directory in a Google Cloud Storage bucket, of the form `gs://<BUCKET>/<PATH>`.

## Components source

Preprocessing:
  [source code](https://github.com/kubeflow/pipelines/tree/master/components/dataflow/tft) 
  [container](https://github.com/kubeflow/pipelines/tree/master/components/dataflow/containers/tft)

Training:
  [source code](https://github.com/kubeflow/pipelines/tree/master/components/kubeflow/launcher) 
  [container](https://github.com/kubeflow/pipelines/tree/master/components/kubeflow/container/launcher)

Analysis:
  [source code](https://github.com/kubeflow/pipelines/tree/master/components/dataflow/tfma) 
  [container](https://github.com/kubeflow/pipelines/tree/master/components/dataflow/containers/tfma)

Prediction:
  [source code](https://github.com/kubeflow/pipelines/tree/master/components/dataflow/predict) 
  [container](https://github.com/kubeflow/pipelines/tree/master/components/dataflow/containers/predict)
