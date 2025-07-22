# Kubeflow Pipelines

[![Coverage Status](https://coveralls.io/repos/github/kubeflow/pipelines/badge.svg?branch=master)](https://coveralls.io/github/kubeflow/pipelines?branch=master)
[![SDK Documentation Status](https://readthedocs.org/projects/kubeflow-pipelines/badge/?version=latest)](https://kubeflow-pipelines.readthedocs.io/en/stable/?badge=latest)
[![SDK Package version](https://img.shields.io/pypi/v/kfp?color=%2334D058&label=pypi%20package)](https://pypi.org/project/kfp)
[![SDK Supported Python versions](https://img.shields.io/pypi/pyversions/kfp.svg?color=%2334D058)](https://pypi.org/project/kfp)
[![OpenSSF Best Practices](https://www.bestpractices.dev/projects/9938/badge)](https://www.bestpractices.dev/projects/9938)
[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/kubeflow/pipelines)

## Overview of the Kubeflow pipelines service

[Kubeflow](https://www.kubeflow.org/) is a machine learning (ML) toolkit that is dedicated to making deployments of ML workflows on Kubernetes simple, portable, and scalable.

**Kubeflow pipelines** are reusable end-to-end ML workflows built using the Kubeflow Pipelines SDK.

The Kubeflow pipelines service has the following goals:

* End to end orchestration: enabling and simplifying the orchestration of end to end machine learning pipelines
* Easy experimentation: making it easy for you to try numerous ideas and techniques, and manage your various trials/experiments.
* Easy re-use: enabling you to re-use components and pipelines to quickly cobble together end to end solutions, without having to re-build each time.

## Installation

* Kubeflow Pipelines can be installed as part of the [Kubeflow Platform](https://www.kubeflow.org/docs/started/installing-kubeflow/#kubeflow-platform). Alternatively you can deploy [Kubeflow Pipelines](https://www.kubeflow.org/docs/components/pipelines/operator-guides/installation/) as a standalone service.

* The Docker container runtime has been deprecated on Kubernetes 1.20+. Kubeflow Pipelines has switched to use [Emissary Executor](https://www.kubeflow.org/docs/components/pipelines/legacy-v1/installation/choose-executor/#emissary-executor) by default from Kubeflow Pipelines 1.8. Emissary executor is Container runtime agnostic, meaning you are able to run Kubeflow Pipelines on Kubernetes cluster with any [Container runtimes](https://kubernetes.io/docs/setup/production-environment/container-runtimes/).

## Documentation

Get started with your first pipeline and read further information in the [Kubeflow Pipelines overview](https://www.kubeflow.org/docs/components/pipelines/overview/).

See the various ways you can [use the Kubeflow Pipelines SDK](https://kubeflow-pipelines.readthedocs.io/en/stable/).

See the Kubeflow [Pipelines API doc](https://www.kubeflow.org/docs/components/pipelines/reference/api/kubeflow-pipeline-api-spec/) for API specification.

Consult the [Python SDK reference docs](https://kubeflow-pipelines.readthedocs.io/en/stable/) when writing pipelines using the Python SDK.

## Deep Wiki
Check out our AI Powered repo documentation on [DeepWiki](https://deepwiki.com/kubeflow/pipelines). 

> :warning: Please note, this is AI generated and may not have completely accurate information.

## Contributing to Kubeflow Pipelines

Before you start contributing to Kubeflow Pipelines, read the guidelines in [How to Contribute](./CONTRIBUTING.md). To learn how to build and deploy Kubeflow Pipelines from source code, read the [developer guide](./developer_guide.md).

## Kubeflow Pipelines Community

### Community Meeting

The Kubeflow Pipelines Community Meeting occurs every other Wed 10-11AM (PST).

[Calendar Invite](https://calendar.google.com/event?action=TEMPLATE&tmeid=NTdoNG5uMDBtcnJlYmdlOWt1c2lkY25jdmlfMjAxOTExMTNUMTgwMDAwWiBqZXNzaWV6aHVAZ29vZ2xlLmNvbQ&tmsrc=jessiezhu%40google.com&scp=ALL)

[Direct Meeting Link](https://zoom.us/j/92607298595?pwd%3DVlKLUbiguGkbT9oKbaoDmCxrhbRop7.1&sa=D&source=calendar&ust=1736264977415448&usg=AOvVaw1EIkjFsKy0d4yQPptIJS3x)

[Meeting notes](http://bit.ly/kfp-meeting-notes)

### Slack

We also have a slack channel (#kubeflow-pipelines) on the Cloud Native Computing Foundation Slack workspace. You can find more details at [https://www.kubeflow.org/docs/about/community/#kubeflow-slack-channels](https://www.kubeflow.org/docs/about/community/#kubeflow-slack-channels)

## Architecture

Details about the KFP Architecture can be found at [Architecture.md](docs/Architecture.md)

## Blog posts

* [From Raw Data to Model Serving: A Blueprint for the AI/ML Lifecycle with Kubeflow](https://blog.kubeflow.org/fraud-detection-e2e/) (By [Helber Belmiro](https://github.com/hbelmiro))
* [Getting started with Kubeflow Pipelines](https://cloud.google.com/blog/products/ai-machine-learning/getting-started-kubeflow-pipelines) (By Amy Unruh)
* How to create and deploy a Kubeflow Machine Learning Pipeline (By Lak Lakshmanan)
  * [Part 1: How to create and deploy a Kubeflow Machine Learning Pipeline](https://medium.com/data-science/how-to-create-and-deploy-a-kubeflow-machine-learning-pipeline-part-1-efea7a4b650f)
  * [Part 2: How to deploy Jupyter notebooks as components of a Kubeflow ML pipeline](https://medium.com/data-science/how-to-deploy-jupyter-notebooks-as-components-of-a-kubeflow-ml-pipeline-part-2-b1df77f4e5b3)
  * [Part 3: How to carry out CI/CD in Machine Learning (“MLOps”) using Kubeflow ML pipelines](https://medium.com/google-cloud/how-to-carry-out-ci-cd-in-machine-learning-mlops-using-kubeflow-ml-pipelines-part-3-bdaf68082112)

## Acknowledgments

Kubeflow pipelines uses [Argo Workflows](https://github.com/argoproj/argo-workflows) by default under the hood to orchestrate Kubernetes resources. The Argo community has been very supportive and we are very grateful.
