# PyTorch Pipeline Samples

This folder contains different PyTorch Kubeflow pipeline examples using the PyTorch KFP Components SDK.

1. Cifar10 example for Computer Vision
2. BERT example for NLP


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
docker build -t image_name:tag -f Dockerfile.cpu .
docker tag image_name:tag username/image_name:tag
docker push username/image_name
```

### Update component.yaml files

To pick the latest changes, component.yaml files needs to be updated.

Update the image tag inside component yaml files.

For example, following component.yaml files needs to be updated for cifar10

[pre_process](cifar10/yaml/pre_process/component.yaml), [train](cifar10/yaml/train/component.yaml), [minio](common/minio/component.yaml)

### Run the notebook

Open the example notebook and run the cells to deploy the example in KFP.

Once the deployment is done, run the prediction and explanations.


