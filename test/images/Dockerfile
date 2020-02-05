# This image is the base image for Kubeflow Pipelines e2e test

FROM gcr.io/k8s-testimages/kubekins-e2e:v20200204-8eefa86-master

# install sudo to enable docker command on mkp deployment:
# https://github.com/kubeflow/pipelines/blob/master/test/deploy-pipeline-mkp-cli.sh#L64
RUN apt-get update && \
    apt-get -y install sudo