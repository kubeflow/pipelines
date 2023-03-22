FROM public.ecr.aws/ubuntu/ubuntu:18.04

RUN apt-get update --allow-releaseinfo-change && apt-get install -y --no-install-recommends \
    curl \
    wget \
    git \
    jq \
    tar \
    sudo \
    python3.8 \
    python3-pip \
    python3.8-dev \
    unzip

# Install eksctl
RUN curl --silent --location "https://github.com/weaveworks/eksctl/releases/download/v0.134.0eksctl_$(uname -s)_amd64.tar.gz" | tar xz -C /tmp \
&& chmod +x /tmp/eksctl && mv /tmp/eksctl /usr/local/bin/

##clean up any temporary files and cahes that were created during the installation process
RUN apt-get clean && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

# Install awscli
RUN curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip" \
 && unzip -qq awscliv2.zip \
 && ./aws/install

# Install aws-iam-authenticator
RUN curl -S -o /usr/local/bin/aws-iam-authenticator https://amazon-eks.s3.us-west-2.amazonaws.com/1.16.8/2020-04-16/bin/linux/amd64/aws-iam-authenticator \
 && chmod +x /usr/local/bin/aws-iam-authenticator

# Install kubectl
RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/v1.21.8/bin/linux/amd64/kubectl \
 && chmod +x ./kubectl \
 && cp ./kubectl /bin

# Install Argo CLI
RUN curl -sSL -o /usr/local/bin/argo https://github.com/argoproj/argo-workflows/releases/download/v2.8.0/argo-linux-amd64 \
 && chmod +x /usr/local/bin/argo

#Install helm(for ack)
RUN wget https://get.helm.sh/helm-v3.8.2-linux-amd64.tar.gz \ 
 && tar xvf helm-v3.8.2-linux-amd64.tar.gz && sudo mv linux-amd64/helm /usr/local/bin

#Install yq for ack
RUN wget https://github.com/mikefarah/yq/releases/download/v4.27.3/yq_linux_amd64 -O /usr/bin/yq &&\
    chmod +x /usr/bin/yq

# install dependencies through pip
RUN ln -s /usr/bin/python3.8 /usr/bin/python \
 && python -m pip install --upgrade pip

RUN python -m pip install -r requirements.txt


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
