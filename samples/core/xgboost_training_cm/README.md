## Overview

The `xgboost_training_cm.py` pipeline creates XGBoost models on structured data in CSV format. Both classification and regression are supported.

The pipeline starts by creating an Google DataProc cluster, and then running analysis, transformation, distributed training and 
prediction in the created cluster. Finally, a delete cluster operation runs to destroy the cluster it creates
in the beginning. The delete cluster operation is used as an exit handler, meaning it will run regardless of whether the pipeline fails
or not.

## Requirements

Preprocessing uses Google Cloud DataProc. Therefore, you must enable the [DataProc API](https://cloud.google.com/endpoints/docs/openapi/enable-api) for the given GCP project.


## Compile

Follow the guide to [building a pipeline](https://www.kubeflow.org/docs/guides/pipelines/build-pipeline/) to install the Kubeflow Pipelines SDK and compile the sample Python into a workflow specification. The specification takes the form of a YAML file compressed into a `.zip` file. 

## Deploy

Open the Kubeflow pipelines UI. Create a new pipeline, and then upload the compiled specification (`.zip` file) as a new pipeline template.

## Run

Most arguments come with default values. Only `output` and `project` need to be filled always. 

* `output` is a Google Storage path which holds
pipeline run results. Note that each pipeline run will create a unique directory under `output` so it will not override previous results. 
* `project` is a GCP project.

## Components source

Create Cluster:
  [source code](https://github.com/kubeflow/pipelines/blob/master/components/gcp/container/component_sdk/python/kfp_component/google/dataproc/_create_cluster.py) 

Analyze (step one for preprocessing), Transform (step two for preprocessing) are using pyspark job
submission component, with
  [source code](https://github.com/kubeflow/pipelines/blob/master/components/gcp/container/component_sdk/python/kfp_component/google/dataproc/_submit_pyspark_job.py) 

Distributed Training and predictions are using spark job submission component, with
  [source code](https://github.com/kubeflow/pipelines/blob/master/components/gcp/container/component_sdk/python/kfp_component/google/dataproc/_submit_spark_job.py) 

Delete Cluster:
  [source code](https://github.com/kubeflow/pipelines/blob/master/components/gcp/container/component_sdk/python/kfp_component/google/dataproc/_delete_cluster.py) 

The container file is located [here](https://github.com/kubeflow/pipelines/tree/master/components/gcp/container) 

