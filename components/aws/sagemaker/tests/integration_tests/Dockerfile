FROM public.ecr.aws/lts/ubuntu:18.04_stable

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
    unzip \
    gzip

# Install eksctl
RUN curl --silent --location "https://github.com/weaveworks/eksctl/releases/download/v0.134.0/eksctl_$(uname -s)_amd64.tar.gz" | tar xz -C /tmp \
&& chmod +x /tmp/eksctl && mv /tmp/eksctl /usr/local/bin/

##clean up any temporary files and cahes that were created during the installation process
RUN apt-get clean && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

# Install awscli
RUN curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip" \
 && unzip -qq awscliv2.zip \
 && ./aws/install

# Install aws-iam-authenticator
RUN curl -Lo /usr/local/bin/aws-iam-authenticator https://github.com/kubernetes-sigs/aws-iam-authenticator/releases/download/v0.5.9/aws-iam-authenticator_0.5.9_linux_amd64 \
 && chmod +x /usr/local/bin/aws-iam-authenticator

# Install kubectl
RUN curl -LO https://dl.k8s.io/release/v1.23.0/bin/linux/amd64/kubectl \
 && chmod +x ./kubectl \
 && cp ./kubectl /bin

# Install Argo CLI
RUN curl -sLO https://github.com/argoproj/argo-workflows/releases/download/v3.4.5/argo-linux-amd64.gz \
&& gunzip argo-linux-amd64.gz && chmod +x argo-linux-amd64 && mv ./argo-linux-amd64 /usr/local/bin/argo

#Install helm(for ack)
RUN wget https://get.helm.sh/helm-v3.8.2-linux-amd64.tar.gz \ 
 && tar xvf helm-v3.8.2-linux-amd64.tar.gz && sudo mv linux-amd64/helm /usr/local/bin

#Install yq for ack
RUN wget https://github.com/mikefarah/yq/releases/download/v4.31.2/yq_linux_amd64 -O /usr/bin/yq &&\
    chmod +x /usr/bin/yq

# install dependencies through pip
RUN ln -s /usr/bin/python3.8 /usr/bin/python \
 && python -m pip install --upgrade pip

COPY ./components/aws/sagemaker/tests/integration_tests/requirements.txt requirements.txt

RUN python -m pip install -r requirements.txt


# Environment variables to be used by tests
ENV REGION="us-west-2"
ENV SAGEMAKER_EXECUTION_ROLE_ARN="arn:aws:iam::1234567890:role/sagemaker-role"
ENV ROBOMAKER_EXECUTION_ROLE_ARN="arn:aws:iam::1234567890:role/robomaker-role"
ENV S3_DATA_BUCKET="kfp-test-data"
ENV MINIO_LOCAL_PORT=9000
ENV KFP_NAMESPACE="kubeflow"

WORKDIR /pipelines/components/aws/sagemaker/tests/integration_tests/scripts/

ENTRYPOINT ["./run_integration_tests"]
