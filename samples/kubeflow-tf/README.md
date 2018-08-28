## Overview
The pipeline creates a TensorFlow model on structured data and image URLs (Google Storage). It works for both classification and regression.
It can run everything inside the pipeline cluster (KubeFlow). The only possible dependency is DataFlow if you enable "cloud" mode for 
preprocessing or prediction.

## The requirements
By default, preprocessing and prediction use local mode and runs inside the cluster. If "cloud" is supplied as the value of "preprocess_mode",
then preprocessing uses Google Cloud DataFlow, and [DataFlow API](https://cloud.google.com/endpoints/docs/openapi/enable-api) needs to be enabled for the given project. 

The trainer relies on KubeFlow tf-job v1alpha2.

## Compiling the pipeline template

Follow [README.md](https://github.com/googleprivate/ml/blob/master/samples/README.md) to install the compiler and then run the following to compile the pipeline:

```bash
dsl-compile --py kubeflow-training-classification.py --output kubeflow-training-classification.yaml
```

## Deploying a pipeline

Open the ML pipeline UI. Create a new pipeline, and then upload the compiled YAML file as a new pipeline template.

The pipeline will require two arguments:

1. The name of a GCP project.
1. An output directory in a GCS bucket, of the form `gs://<BUCKET>/<PATH>`.

## Components Source

Preprocessing:
  [source code](https://github.com/googleprivate/ml/tree/master/components/dataflow/tft) 
  [container](https://github.com/googleprivate/ml/tree/master/components/dataflow/containers/tft)

Training:
  [source code](https://github.com/googleprivate/ml/tree/master/components/kubeflow/launcher) 
  [container](https://github.com/googleprivate/ml/tree/master/components/kubeflow/container/launcher)

Prediction:
  [source code](https://github.com/googleprivate/ml/tree/master/components/dataflow/predict) 
  [container](https://github.com/googleprivate/ml/tree/master/components/dataflow/containers/predict)

Confusion Matrix:
  [source code](https://github.com/googleprivate/ml/tree/master/components/local/evaluation) 
  [container](https://github.com/googleprivate/ml/tree/master/components/local/containers/confusion_matrix)
