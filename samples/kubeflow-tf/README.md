## Overview

The `kubeflow-training-classification.py` pipeline creates a TensorFlow model on structured data and image URLs (Google Cloud Storage). It works for both classification and regression.
Everything runs inside the pipeline cluster (Kubeflow). The only possible dependency is Google Cloud DataFlow if you enable the "*cloud*" mode for 
the preprocessing or prediction step.

## The requirements

By default, the preprocessing and prediction steps use the "*local*" mode and run inside the cluster. If you specify the value of "*preprocess_mode*" as "*cloud*", you must enable the
[DataFlow API](https://cloud.google.com/endpoints/docs/openapi/enable-api) for the given GCP project so that the preprocessing step
can use Cloud DataFlow. 

Note: The trainer depends on Kubeflow API version v1alpha2.

## Compiling the pipeline template

Follow the guide to [building a pipeline](https://github.com/kubeflow/pipelines/wiki/Build-a-Pipeline) to install the Kubeflow Pipelines SDK, then run the following command to compile the sample Python into a workflow specification. The specification takes the form of a YAML file compressed into a `.tar.gz` file.

```bash
dsl-compile --py kubeflow-training-classification.py --output kubeflow-training-classification.tar.gz
```

## Deploying the pipeline

Open the Kubeflow pipelines UI. Create a new pipeline, and then upload the compiled specification (`.tar.gz` file) as a new pipeline template.

The pipeline requires one argument:

1. An output directory in a Google Cloud Storage bucket, of the form `gs://<BUCKET>/<PATH>`.

## Components source

Preprocessing:
  [source code](https://github.com/kubeflow/pipelines/tree/master/components/dataflow/tft), 
  [container](https://github.com/kubeflow/pipelines/tree/master/components/dataflow/containers/tft)

Training:
  [source code](https://github.com/kubeflow/pipelines/tree/master/components/kubeflow/launcher), 
  [container](https://github.com/kubeflow/pipelines/tree/master/components/kubeflow/container/launcher)

Prediction:
  [source code](https://github.com/kubeflow/pipelines/tree/master/components/dataflow/predict), 
  [container](https://github.com/kubeflow/pipelines/tree/master/components/dataflow/containers/predict)

Confusion Matrix:
  [source code](https://github.com/kubeflow/pipelines/tree/master/components/local/evaluation), 
  [container](https://github.com/kubeflow/pipelines/tree/master/components/local/containers/confusion_matrix)
