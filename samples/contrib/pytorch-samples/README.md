# PyTorch Pipeline Samples

This folder contains different Kubeflow pipeline PyTorch examples using the PyTorch KFP Components SDK.

1. Cifar10 example for Computer Vision
2. BERT example for NLP

Please navigate to the following link for running the examples with Google Vertex AI pipeline

https://github.com/amygdala/code-snippets/tree/master/ml/vertex_pipelines/pytorch/cifar

Use the following link for installing KFP python sdk

https://github.com/kubeflow/pipelines/tree/master/sdk/python

## Prerequisites

Check the following prerequisites before running the examples

**[Prequisites](prerequisites.md)**

## Cluster Build
This involves steps for building and running the pipeline from Kubeflow Jupyter notebook.

Use the following notebook files for running the Cifar 10 and Bert examples

Cifar 10 - [Pipeline-Cifar10.ipynb](Pipeline-Cifar10.ipynb)

Bert - [Pipeline-Bert.ipynb](Pipeline-Bert.ipynb)

**[Steps to Run the example pipelines from Kubeflow Jupyter Notebook](cluster_build.md)**

## Local Build
This involves steps to build the pipeline in local machine and run it by uploading the 
pipeline file to the Kubeflow Dashboard.

Use the following python files building the pipeline locally for Cifar 10 and Bert examples

Cifar 10 - [cifar10/pipeline.py](cifar10/pipeline.py)

Bert - [bert/pipeline.py](bert/pipeline.py)

**[Steps to build the examples pipelines in local machine](local_build.md)**

## Adding new example

### Steps for adding new examples:

1. Create new example

    1. Copy the new example to `pipelines/samples/contrib/pytorch-samples/` (Ex: `pipelines/samples/contrib/pytorch-samples/iris`)
    2. Add yaml files for all components in the pipeline
    3. Create a pipeline file (Ex: `pipelines/samples/contrib/pytorch-samples/iris/pipeline.py`)

2. Build image and compile pipeline

    ```./build.sh <example-directory> <dockerhub-user-name>```

    For example:

    ```./build.sh cifar10 johnsmith```

    The following actions are done in the build script

    1. Bundling the code changes into a docker image
    2. Pushing the docker image to dockerhub
    3. Changing image tag in component.yaml
    4. Run `pipeline.py` file to generate yaml file which can be used to invoke the pipeline.

3. Upload pipeline and create a run 

    1. Create an Experiment from `kubeflow dashboard -> Experiments -> Create Experiment`.
    2. Upload the generated pipeline to `kubeflow dashboard -> Pipelines -> Upload Pipeline` 
    3. Create run from `kubeflow dashboard -> Pipelines -> {Pipeline Name} -> Create Run`

        **Pipeline params can be added while creating a run.**

    Refer: [Kubeflow Pipelines Quickstart](https://www.kubeflow.org/docs/components/pipelines/pipelines-quickstart/)

## Hyper Parameter Optimization with AX

In this example, we train a Pytorch Lightning model to using image classification cifar10 dataset. A parent run will be created during the training process,which would dump the baseline model and relevant parameters,metrics and model along with its summary,subsequently followed by a set of nested child runs, which will dump the trial results. The best parameters would be dumped into the parent run once the experiments are completed.

### Run the notebook

Open the example notebook and run to deploy the example in KFP.

Cifar 10 HPO - [Pipeline-Cifar10-hpo.ipynb](Pipeline-Cifar10-hpo.ipynb)

## Pytorch Distributed Training with Pytorch Job Operator

In this example, we deploy a pipeline to launch the distributed training of this BERT model file using the pytorch operator and deploy with torchserve using KFServing. 

### Run the notebook

Open the example notebook and run to deploy the example in KFP.

Bert Distributed Training - [Pipeline-Bert-Dist.ipynb](Pipeline-Bert-Dist.ipynb)

**Refer: [Running Pipelines in Kubeflow Jupyter Notebook](cluster_build.md)**

