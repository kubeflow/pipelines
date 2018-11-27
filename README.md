[![Build Status](https://travis-ci.org/kubeflow/pipelines.svg?branch=master)](https://travis-ci.org/kubeflow/pipelines)
[![Coverage Status](https://coveralls.io/repos/github/kubeflow/pipelines/badge.svg?branch=master)](https://coveralls.io/github/kubeflow/pipelines?branch=master)

## Overview of the Kubeflow pipelines service

[Kubeflow](https://www.kubeflow.org/) is a machine learning (ML) toolkit that is dedicated to making deployments of ML workflows on Kubernetes simple, portable, and scalable. 

**Kubeflow pipelines** are reusable end-to-end ML workflows built using the Kubeflow Pipelines SDK.

The Kubeflow pipelines service has the following goals:

* End to end orchestration: enabling and simplifying the orchestration of end to end machine learning pipelines
* Easy experimentation: making it easy for you to try numerous ideas and techniques, and manage your various trials/experiments.
* Easy re-use: enabling you to re-use components and pipelines to quickly cobble together end to end solutions, without having to re-build each time.

## Documentation

Get started with your first pipeline and read further information in the [documentation](https://github.com/kubeflow/pipelines/wiki).

## Blog posts

* [Getting started with Kubeflow Pipelines](https://cloud.google.com/blog/products/ai-machine-learning/getting-started-kubeflow-pipelines) (By Amy Unruh)
* [How to create and deploy a Kubeflow Machine Learning Pipeline](https://towardsdatascience.com/how-to-create-and-deploy-a-kubeflow-machine-learning-pipeline-part-1-efea7a4b650f) (By Lak Lakshmanan)

## Acknowledgments

Kubeflow pipelines uses [Argo](https://github.com/argoproj/argo) under the hood to orchestrate Kubernetes resources. The Argo community has been very supportive and we are very grateful.
