## Disclaimer: 
**TFMA DSL is under testing.**

## The requirements:
Preprocessing uses Google Cloud DataFlow. So [DataFlow API](https://cloud.google.com/endpoints/docs/openapi/enable-api) needs to be enabled for the given project.

## Compile
<!---
Follow [README.md](https://github.com/googleprivate/ml/blob/master/samples/README.md) to install the compiler and 
compile your sample python into workflow yaml.
--->
Currently, the DSL python file is not generated for TFMA yet. You can submit the yaml file directly to the Web UI.

## Deploy
This sample runs a pipeline with tensorflow transform and model-analysis components.
* Prepare output directory  
Create a GCS bucket to store the generated model. Make sure it's in the same project as the ML pipeline deployed above.

```bash
gsutil mb gs://[YOUR_GCS_BUCKET]
```

* Deploy  
Open the ML pipeline UI.  
Kubeflow-training-classification requires two argument:

```
project: MY_GCP_PROJECT
output: gs://[YOUR_GCS_BUCKET]
```


### The dataset

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

<!---
## Running the sample

```sh
argo submit taxi-cab-classification/pipeline.yaml \
     -p project=MY_GCP_PROJECT \
     -p output="gs://my-bucket/taximodel" \
     --entrypoint kubeflow-training
```
--->