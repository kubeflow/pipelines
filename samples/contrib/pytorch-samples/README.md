# PyTorch Pipeline Samples

This folder contains different PyTorch Kubeflow pipeline examples using the PyTorch KFP Components SDK.

1. Cifar10 example for Computer Vision
2. BERT example for NLP

Please navigate to the following link for running the examples with Google Vertex AI pipeline

https://github.com/amygdala/code-snippets/tree/master/ml/vertex_pipelines/pytorch/cifar

Use the following link for installing KFP python sdk

https://github.com/kubeflow/pipelines/tree/master/sdk/python

## Prerequisites

Check the following prerequisites before running the examples

[Prequisites](prerequisites.md)

## Steps to Run the examples in Cluster Environment

Use the following notebook files for running the existing Cifar 10 and Bert examples

Cifar 10 - [Pipeline-Cifar10.ipynb](Pipeline-Cifar10.ipynb)

Bert - [Pipeline-Bert.ipynb](Pipeline-Bert.ipynb)

Following steps has to be performed for adding new examples:

### Create new example

Copy the example folder inside `pipelines/samples/contrib/pytorch-samples/`

Ex: `pipelines/samples/contrib/pytorch-samples/iris`

### Build and push the docker image
```
docker build -t image_name:tag .
```

to run the example in gpu, run the following commands for building docker image

```
docker build --build-arg BASE_IMAGE=pytorch/pytorch:1.8.1-cuda10.2-cudnn7-runtime -t image_name:tag .
```

push the docker image

```
docker tag image_name:tag username/image_name:tag
docker push username/image_name
```

Note for gpu testing:

Following changes needs to be done in the examples notebook

1. Make sure to set `node selectors`, `gpus`, `accelerator` variables under the train task

push the docker image.

2. Use `isvc_gpu_yaml` for GPU inference.

### Tensorboard Image Update

A custom tensorboard image is used for viewing pytorch profiler statistics

Update tensorboard image name in the notebook (variable_name: `TENSORBOARD_IMAGE`) for using any other custom tensorboard image.

### Update component.yaml files

To pick the latest changes, component.yaml files needs to be updated.

Update the image tag inside component yaml files.

For example, following component.yaml files needs to be updated for cifar10

[pre_process](cifar10/yaml/pre_process/component.yaml), [train](cifar10/yaml/train/component.yaml), [minio](common/minio/component.yaml)

### Run the notebook

Open the example notebook and run the cells to deploy the example in KFP.

Once the deployment is done, run the prediction and explanations.


### Captum Insights Visualization

Run the following command to port forward kubeflow dashboard

```
kubectl port-forward svc/istio-ingressgateway -n istio-system 8080:80
```

To view the captum insights UI in the local environment, run the following port forwarding command

```
kubectl port-forward <notebook-server-pod-name> -n kubeflow-user-example-com <port>:6080
```

For example:

```
kubectl port-forward pod/root-0 -n kubeflow-user-example-com 8999:6080
```

The captum insights UI can be accessed via

```
http://localhost:8999
```



## Steps to run the examples in local environment

Use the following notebook files for running the existing Cifar 10 and Bert examples

`python cifar10/pipeline.py`

or

`python bert/pipeline.py`

The output of the above script will generate a yaml file which can be uploaded to KFP for invoking a run.


For testing any code changes or adding new examples, use the following build script 

The following actions are done in the build script

1. Bundling the code changes into a docker image
2. Pushing the docker image to dockerhub
3. Changing image tag in component.yaml
4. Run `pipeline.py` file to generate yaml file which can be used to invoke the pipeline.

Run the following command

`./build.sh <example-directory> <dockerhub-user-name>`

For example:

`./build.sh cifar10 johnsmith` 
