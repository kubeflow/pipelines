The `taxi-cab-classification-pipeline.py` sample runs a pipeline with TensorFlow's transform and model-analysis components.

## The dataset

This sample is based on the model-analysis example 
[here](https://github.com/tensorflow/tfx/tree/master/tfx/examples/chicago_taxi).

The sample trains and analyzes a model based on the 
[Taxi Trips dataset](https://data.cityofchicago.org/Transportation/Taxi-Trips/wrvz-psew)
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

For On-Premise cluster, you need to create a [Persistent Volume (PV)](https://kubernetes.io/docs/concepts/storage/persistent-volumes/) if the [Dynamic Volume Provisioning](https://kubernetes.io/docs/concepts/storage/dynamic-provisioning/) is not enabled. The capacity of the PV needs at least 1Gi.

## Compiling the pipeline template

Follow the guide to [building a pipeline](https://www.kubeflow.org/docs/guides/pipelines/build-pipeline/) to install the Kubeflow Pipelines SDK.

For On-Premise cluster, update the `platform` to `onprem` in `taxi-cab-classification-pipeline.py`.

```bash
sed -i.sedbak s"/platform = 'GCP'/platform = 'onprem'/"  taxi-cab-classification-pipeline.py
```
Then run the following command to compile the sample Python into a workflow specification. The specification takes the form of a YAML file compressed into a `.tar.gz` file.

```bash
dsl-compile --py taxi-cab-classification-pipeline.py --output taxi-cab-classification-pipeline.tar.gz
```

## Deploying the pipeline

Open the Kubeflow pipelines UI. Create a new pipeline, and then upload the compiled specification (`.tar.gz` file) as a new pipeline template.

- GCP
  The pipeline requires two arguments:
  
  1. The name of a GCP project.
  2. An output directory in a Google Cloud Storage bucket, of     the form `gs://<BUCKET>/<PATH>`.
- On-Premise
  For On-Premise cluster, the pipeline will create a Persistent Volume Claim (PVC), and download 
  automatically the 
  [source data](https://github.com/kubeflow/pipelines/tree/master/samples/core/tfx_cab_classification/taxi-cab-classification) 
  to the PVC.
  1. The `output` is PVC mount point for the containers, can be set to `/mnt`.
  2. The `project` can be set to `taxi-cab-classification-pipeline-onprem`.
  3. If the PVC mounted to `/mnt`, the value of below parameters need to be set as following:
  - `column-names`: `
/mnt/pipelines/samples/tfx/taxi-cab-classification/column-names.json`
  - `train`: `/mnt/pipelines/samples/tfx/taxi-cab-classification/train.csv`
  -  `evaluation`: `/mnt/pipelines/samples/tfx/taxi-cab-classification/eval.csv`
  -  `preprocess-module`: `/mnt/pipelines/samples/tfx/taxi-cab-classification/preprocessing.py` 



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
