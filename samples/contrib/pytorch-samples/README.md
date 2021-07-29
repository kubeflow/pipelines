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

**[Prerequisites](prerequisites.md)**


## Note: The Samples can be run in 2 ways

1. From Kubeflow Jupyter Notebook mentioned in [Option 1](##-Option-1.-Running-from-Kubeflow-Jupyter-Notebook)
2. compiling and uploading to KFP mentioned in [Option 2](##-Option-2.-Compiling-and-Running-by-uploading-to-Kubeflow-Pipelines)

## Option 1. Running from Kubeflow Jupyter Notebook
This involves steps for building and running the pipeline from Kubeflow Jupyter notebook.
Here the pipeline is defined in a Jupyter notebook and run directly from the Jupyter notebook.

Use the following notebook files for running the Cifar 10 and Bert examples

Cifar 10 - [Pipeline-Cifar10.ipynb](Pipeline-Cifar10.ipynb)

Bert - [Pipeline-Bert.ipynb](Pipeline-Bert.ipynb)

**[Steps to Run the example pipelines from Kubeflow Jupyter Notebook](cluster_build.md)**

## Option 2. Compiling and Running by uploading to Kubeflow Pipelines 
This involves steps to build the pipeline in local machine and run it by uploading the 
pipeline file to the Kubeflow Dashboard. Here we have a python file that defines the pipeline. The python file containing the pipeline is compiled and the generated yaml is uploaded to the KFP for creating a run out of it.

Use the following python files building the pipeline locally for Cifar 10 and Bert examples

Cifar 10 - [cifar10/pipeline.py](cifar10/pipeline.py)

Bert - [bert/pipeline.py](bert/pipeline.py)

**[Steps to run the examples pipelines by compiling and uploading to KFP](local_build.md)**

## Other Examples

## PyTorch CIFAR10 with Captum Insights

In this example, we train a PyTorch Lightning model to using image classification cifar10 dataset with Captum Insights. This uses PyTorch KFP components to preprocess, train, visualize and deploy the model in the pipeline
and interpretation of the model using the Captum Insights.

### Run the notebook

Open the example notebook and run to deploy the example in KFP.

Cifar 10 Captum Insights - [Pipeline-Cifar10-Captum-Insights.ipynb](Pipeline-Cifar10-Captum-Insights.ipynb)

## Hyper Parameter Optimization with AX

In this example, we train a PyTorch Lightning model to using image classification cifar10 dataset. A parent run will be created during the training process,which would dump the baseline model and relevant parameters,metrics and model along with its summary,subsequently followed by a set of nested child runs, which will dump the trial results. The best parameters would be dumped into the parent run once the experiments are completed.

### Run the notebook

Open the example notebook and run to deploy the example in KFP.

Cifar 10 HPO - [Pipeline-Cifar10-hpo.ipynb](Pipeline-Cifar10-hpo.ipynb)

## PyTorch Distributed Training with PyTorch Job Operator

In this example, we deploy a pipeline to launch the distributed training of this BERT model file using the pytorch operator and deploy with torchserve using KFServing. 

### Run the notebook

Open the example notebook and run to deploy the example in KFP.

Bert Distributed Training - [Pipeline-Bert-Dist.ipynb](Pipeline-Bert-Dist.ipynb)

**Refer: [Running Pipelines in Kubeflow Jupyter Notebook](cluster_build.md)**

## Contributing to PyTorch KFP Samples

Before you start contributing to PyTorch KFP Samples, read the guidelines in [How to Contribute](contributing.md). To learn how to build and deploy PyTorch components with pytorch-kfp-components SDK. 
