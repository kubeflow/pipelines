FROM continuumio/miniconda:4.7.12

RUN apt-get update && apt-get install -y --no-install-recommends \
    curl \
    wget \
    git \
    jq

# Install eksctl
RUN curl --location "https://github.com/weaveworks/eksctl/releases/download/0.19.0/eksctl_$(uname -s)_amd64.tar.gz" | tar xz -C /tmp \
 && mv /tmp/eksctl /usr/local/bin

# Install aws-iam-authenticator
RUN curl -S -o /usr/local/bin/aws-iam-authenticator https://amazon-eks.s3.us-west-2.amazonaws.com/1.16.8/2020-04-16/bin/linux/amd64/aws-iam-authenticator \
 && chmod +x /usr/local/bin/aws-iam-authenticator

# Install Kubectl
RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/v1.18.0/bin/linux/amd64/kubectl \
 && chmod +x ./kubectl \
 && mv ./kubectl /usr/local/bin/kubectl

# Install Argo CLI
RUN curl -sSL -o /usr/local/bin/argo https://github.com/argoproj/argo/releases/download/v2.8.0/argo-linux-amd64 \
 && chmod +x /usr/local/bin/argo

# Copy conda environment early to avoid cache busting
COPY ./components/aws/sagemaker/tests/integration_tests/environment.yml environment.yml

# Create conda environment for running tests and set as start-up environment
RUN conda env create -f environment.yml
RUN echo "source activate kfp_test_env" > ~/.bashrc
ENV PATH "/opt/conda/envs/kfp_test_env/bin":$PATH

# Environment variables to be used by tests
ENV REGION="us-west-2"
ENV SAGEMAKER_EXECUTION_ROLE_ARN="arn:aws:iam::1234567890:role/sagemaker-role"
ENV ROBOMAKER_EXECUTION_ROLE_ARN="arn:aws:iam::1234567890:role/robomaker-role"
ENV S3_DATA_BUCKET="kfp-test-data"
ENV MINIO_LOCAL_PORT=9000
ENV KFP_NAMESPACE="kubeflow"

RUN mkdir pipelines
COPY ./ ./pipelines/

WORKDIR /pipelines/components/aws/sagemaker/tests/integration_tests/scripts/

ENTRYPOINT [ "/bin/bash", "./run_integration_tests" ]