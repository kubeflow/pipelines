## Overview
The pipeline creates a TensorFlow model on structured data and image URLs (Google Storage). It works for both classification and regression.
Everything runs inside the pipeline cluster (KubeFlow). The only possible dependency is DataFlow if you enable the "*cloud*" mode for 
the preprocessing or prediction step.

## The requirements
By default, the preprocessing and prediction steps use the "*local*" mode and run inside the cluster. If users specify the value of "*preprocess_mode*" as "*cloud*",
[DataFlow API](https://cloud.google.com/endpoints/docs/openapi/enable-api) needs to be enabled for the given project such that the preprocessing step
can use Google Cloud DataFlow. 

Note: The trainer depends on KubeFlow API Version v1alpha2.

## Compiling the pipeline template

Follow [README.md](https://github.com/googleprivate/ml/blob/master/samples/README.md) to install the compiler and then run the following command to compile the pipeline:

```bash
dsl-compile --py kubeflow-training-classification.py --output kubeflow-training-classification.tar.gz
```

## Deploying a pipeline

Open the ML pipeline UI. Create a new pipeline, and then upload the compiled YAML file as a new pipeline template.

The pipeline will require one argument:

1. An output directory in a GCS bucket, of the form `gs://<BUCKET>/<PATH>`.

## Components Source

Preprocessing:
  [source code](https://github.com/googleprivate/ml/tree/master/components/dataflow/tft), 
  [container](https://github.com/googleprivate/ml/tree/master/components/dataflow/containers/tft)

Training:
  [source code](https://github.com/googleprivate/ml/tree/master/components/kubeflow/launcher), 
  [container](https://github.com/googleprivate/ml/tree/master/components/kubeflow/container/launcher)

Prediction:
  [source code](https://github.com/googleprivate/ml/tree/master/components/dataflow/predict), 
  [container](https://github.com/googleprivate/ml/tree/master/components/dataflow/containers/predict)

Confusion Matrix:
  [source code](https://github.com/googleprivate/ml/tree/master/components/local/evaluation), 
  [container](https://github.com/googleprivate/ml/tree/master/components/local/containers/confusion_matrix)
