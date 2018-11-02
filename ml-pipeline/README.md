# ML Pipeline Deployment

The ML pipeline system uses [Ksonnet](https://ksonnet.io/) as part of the deployment process. Ksonnet provides the flexibility to generate Kubernetes manifests from parameterized templates and makes it easy to customize Kubernetes manifests for different use cases. 
The Ksonnet is wrapped in a customized bootstrap container so user don't need to explicitly install Ksonnet to install ML pipeline.


This directory contains
- ml-pipeline Ksonnet registry
- files for bootstrap docker container 


The container is published to **gcr.io/ml-pipeline/bootstrapper**
