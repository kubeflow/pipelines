This sample runs a pipeline with tensorflow transform and model-analysis components.

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

As such, the DataFlow API needs to be enabled for the given project if you want to use `cloud` as the mode for either preprocessing or analysis.

Instructions for enabling that can be found [here](https://cloud.google.com/endpoints/docs/openapi/enable-api).

## Compiling the pipeline template

Follow [README.md](https://github.com/googleprivate/ml/blob/master/samples/README.md) to install the compiler and then run the following to compile the pipeline:

```bash
dsl-compile --py taxi-cab-classification-pipeline.py --output taxi-cab-classification-pipeline.tar.gz
```

## Deploying a pipeline

Open the ML pipeline UI. Create a new pipeline, and then upload the compiled YAML file as a new pipeline template.

The pipeline will require two arguments:

1. The name of a GCP project.
2. An output directory in a GCS bucket, of the form `gs://<BUCKET>/<PATH>`.

## Components Source

Preprocessing:
  [source code](https://github.com/googleprivate/ml/tree/master/components/dataflow/tft) 
  [container](https://github.com/googleprivate/ml/tree/master/components/dataflow/containers/tft)

Training:
  [source code](https://github.com/googleprivate/ml/tree/master/components/kubeflow/launcher) 
  [container](https://github.com/googleprivate/ml/tree/master/components/kubeflow/container/launcher)

Analysis:
  [source code](https://github.com/googleprivate/ml/tree/master/components/dataflow/tfma) 
  [container](https://github.com/googleprivate/ml/tree/master/components/dataflow/containers/tfma)

Prediction:
  [source code](https://github.com/googleprivate/ml/tree/master/components/dataflow/predict) 
  [container](https://github.com/googleprivate/ml/tree/master/components/dataflow/containers/predict)
