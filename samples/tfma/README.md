This sample runs a pipeline with tensorflow transform and model-analysis components.

## Requirements:

* A GKE cluster with [argo](https://github.com/argoproj/argo) and
  [kubeflow](https://github.com/kubeflow/kubeflow) installed.
  The GKE cluster needs cloud-platform scope. For example:

  `gcloud container clusters create [your-gke-cluster-name] --zone us-central1-a --scopes cloud-platform`

* Preprocessing uses Google Cloud DataFlow. So DataFlow API needs to be enabled for given project.

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

## Running the sample

```sh
argo submit taxi-cab-classification/pipeline.yaml \
     -p project=MY_GCP_PROJECT \
     -p output="gs://my-bucket/taximodel" \
     --entrypoint kubeflow-training
```
