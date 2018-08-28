## Overview
The pipeline does image classification using resnet architecture and TPU technology. There are three steps: preprocessing (Google Cloud DataFlow), training (Google Cloud Machine Learning Engine Training Service), deployment (Google Cloud Machine Learning Model Service).

## Requirements
Preprocessing uses Google Cloud DataFlow. So [DataFlow API](https://cloud.google.com/endpoints/docs/openapi/enable-api) needs to be enabled for the given project.

Training and serving uses Google Cloud Machine Learning Engine. So [Cloud Machine Learning Engine API](https://cloud.google.com/endpoints/docs/openapi/enable-api) needs to be enabled for the given project. Also [TPU](https://cloud.google.com/ml-engine/docs/tensorflow/using-tpus) needs to be enabled for the
given project. 

## Compile
Follow [README.md](https://github.com/googleprivate/ml/blob/master/samples/README.md) to install the compiler and 
compile your python sample into workflow yaml.

## Deploy
Open ML Pipelines UI and upload the generated workflow yaml. Run it from there.
The sample comes with default values (including dataset) of the input parameters. The default dataset are images of ten types of
bolts and nuts. With steps > 10000 it can achieve 99% accuracy.
The only unfilled parameters are:

```
project-id: A GCP project to use and we recommend the same GCP project of the pipeline cluster. 
bucket: A Google storage bucket to store results. 
```

## Components Source

Preprocessing:
  [source code](https://github.com/googleprivate/ml/tree/master/components/resnet-cmle/resnet) 
  [container](https://github.com/googleprivate/ml/tree/master/components/resnet-cmle/containers/preprocess)

Training:
  [source code](https://github.com/googleprivate/ml/tree/master/components/resnet-cmle/resnet) 
  [container](https://github.com/googleprivate/ml/tree/master/components/resnet-cmle/containers/train)

Deployment:
  [source code](https://github.com/googleprivate/ml/tree/master/components/resnet-cmle/resnet) 
  [container](https://github.com/googleprivate/ml/tree/master/components/resnet-cmle/containers/deploy)
