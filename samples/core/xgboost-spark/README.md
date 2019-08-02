## Overview

The `xgboost-training-cm.py` pipeline creates XGBoost models on structured data in CSV format. Both classification and regression are supported.

The pipeline starts by creating an Google DataProc cluster, and then running analysis, transformation, distributed training and 
prediction in the created cluster. Then a single node confusion-matrix aggregator is used (for classification case) to
provide the confusion matrix data to the front end. Finally, a delete cluster operation runs to destroy the cluster it creates
in the beginning. The delete cluster operation is used as an exit handler, meaning it will run regardless of whether the pipeline fails
or not.

## Requirements

Preprocessing uses Google Cloud DataProc. Therefore, you must enable the [DataProc API](https://cloud.google.com/endpoints/docs/openapi/enable-api) for the given GCP project.

## Compile

Follow the guide to [building a pipeline](https://www.kubeflow.org/docs/guides/pipelines/build-pipeline/) to install the Kubeflow Pipelines SDK and compile the sample Python into a workflow specification. The specification takes the form of a YAML file compressed into a `.tar.gz` file. 

## Deploy

Open the Kubeflow pipelines UI. Create a new pipeline, and then upload the compiled specification (`.tar.gz` file) as a new pipeline template.

## Run

Most arguments come with default values. Only `output` and `project` need to be filled always. 

* `output` is a Google Storage path which holds
pipeline run results. Note that each pipeline run will create a unique directory under `output` so it will not override previous results. 
* `project` is a GCP project.

## Components source

Create Cluster:
  [source code](https://github.com/kubeflow/pipelines/tree/master/components/dataproc/create_cluster/src) 
  [container](https://github.com/kubeflow/pipelines/tree/master/components/dataproc/create_cluster)

Analyze (step one for preprocessing):
  [source code](https://github.com/kubeflow/pipelines/tree/master/components/dataproc/analyze/src) 
  [container](https://github.com/kubeflow/pipelines/tree/master/components/dataproc/analyze)

Transform (step two for preprocessing):
  [source code](https://github.com/kubeflow/pipelines/tree/master/components/dataproc/transform/src) 
  [container](https://github.com/kubeflow/pipelines/tree/master/components/dataproc/transform)

Distributed Training:
  [source code](https://github.com/kubeflow/pipelines/tree/master/components/dataproc/train/src) 
  [container](https://github.com/kubeflow/pipelines/tree/master/components/dataproc/train)

Distributed Predictions:
  [source code](https://github.com/kubeflow/pipelines/tree/master/components/dataproc/predict/src) 
  [container](https://github.com/kubeflow/pipelines/tree/master/components/dataproc/predict)

Confusion Matrix:
  [source code](https://github.com/kubeflow/pipelines/tree/master/components/local/confusion_matrix/src) 
  [container](https://github.com/kubeflow/pipelines/tree/master/components/local/confusion_matrix)
 

ROC:
  [source code](https://github.com/kubeflow/pipelines/tree/master/components/local/roc/src) 
  [container](https://github.com/kubeflow/pipelines/tree/master/components/local/roc)


Delete Cluster:
  [source code](https://github.com/kubeflow/pipelines/tree/master/components/dataproc/delete_cluster/src) 
  [container](https://github.com/kubeflow/pipelines/tree/master/components/dataproc/delete_cluster)


