## Overview
The pipeline creates XGBoost models on structured data with CSV format. Both classification and regression are supported.

The pipeline starts by creating an Google DataProc cluster, and then run analysis, transormation, distributed training and 
prediction in the created cluster. Then a single node confusion-matrix aggregator is used (for classification case) to
provide frontend the confusion matrix data. At the end, a delete cluster operation runs to destroy the cluster it creates
in the beginning. The delete cluster operation is used as an exit handler, meaning it will run regardless the pipeline fails
or not.

## Requirements
Preprocessing uses Google Cloud DataProc. So the [DataProc API](https://cloud.google.com/endpoints/docs/openapi/enable-api) needs to be enabled for the given project.

## Compile
Follow [README.md](https://github.com/kubeflow/pipelines/blob/master/samples/README.md) to install the compiler and 
compile your sample python into workflow yaml.

## Deploy
Open the ML pipeline UI. Create a new pipeline, and then upload the compiled YAML file as a new pipeline template.

## Run
Most arguments come with default values. Only "output" and "project" need to be filled always. "output" is a Google Storage path which holds
pipeline run results. Note that each pipeline run will create a unique directory under output so it will not override previous results. "project"
is a GCP project.

## Components Source

Create Cluster:
  [source code](https://github.com/kubeflow/pipelines/tree/master/components/dataproc/xgboost/create_cluster) 
  [container](https://github.com/kubeflow/pipelines/tree/master/components/dataproc/containers/create_cluster)

Analyze (step one for preprocessing):
  [source code](https://github.com/kubeflow/pipelines/tree/master/components/dataproc/xgboost/analyze) 
  [container](https://github.com/kubeflow/pipelines/tree/master/components/dataproc/containers/analyze)

Transform (step two for preprocessing):
  [source code](https://github.com/kubeflow/pipelines/tree/master/components/dataproc/xgboost/transform) 
  [container](https://github.com/kubeflow/pipelines/tree/master/components/dataproc/containers/transform)

Distributed Training:
  [source code](https://github.com/kubeflow/pipelines/tree/master/components/dataproc/xgboost/train) 
  [container](https://github.com/kubeflow/pipelines/tree/master/components/dataproc/containers/train)

Distributed Predictions:
  [source code](https://github.com/kubeflow/pipelines/tree/master/components/dataproc/xgboost/predict) 
  [container](https://github.com/kubeflow/pipelines/tree/master/components/dataproc/containers/predict)

Confusion Matrix:
  [source code](https://github.com/kubeflow/pipelines/tree/master/components/local/evaluation) 
  [container](https://github.com/kubeflow/pipelines/tree/master/components/local/containers/confusion_matrix)

Delete Cluster:
  [source code](https://github.com/kubeflow/pipelines/tree/master/components/dataproc/xgboost/delete_cluster) 
  [container](https://github.com/kubeflow/pipelines/tree/master/components/dataproc/containers/delete_cluster)


