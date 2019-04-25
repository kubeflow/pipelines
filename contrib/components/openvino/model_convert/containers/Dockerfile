FROM ubuntu:16.04
RUN apt-get update && apt-get install -y --no-install-recommends \
        curl ca-certificates \
        python3-pip \
        python-dev \
        gcc \
        python-setuptools \
        python3-setuptools \
        libgfortran3 \
        unzip \
        vim && \
        apt-get clean
RUN curl -L -o 2018_R5.tar.gz https://github.com/opencv/dldt/archive/2018_R5.tar.gz && \
    tar -zxf 2018_R5.tar.gz && \
    rm 2018_R5.tar.gz && \
    rm -Rf dldt-2018_R5/inference-engine
WORKDIR dldt-2018_R5/model-optimizer
RUN pip3 install --upgrade pip setuptools
RUN pip3 install -r requirements.txt
RUN curl -L -o google-cloud-sdk.zip https://dl.google.com/dl/cloudsdk/release/google-cloud-sdk.zip && \
    unzip -qq google-cloud-sdk.zip -d tools && \
    rm google-cloud-sdk.zip && \
    tools/google-cloud-sdk/install.sh --usage-reporting=false \
        --path-update=true --bash-completion=false \
        --disable-installation-options && \
    tools/google-cloud-sdk/bin/gcloud -q components update \
        gcloud core gsutil && \
    tools/google-cloud-sdk/bin/gcloud config set component_manager/disable_update_check true && \
    touch tools/google-cloud-sdk/lib/third_party/google.py && \
    pip install -U crcmod
ENV PATH ${PATH}:/dldt-2018_R5/model-optimizer:/dldt-2018_R5/model-optimizer/tools/google-cloud-sdk/bin
COPY convert_model.py .
RUN chmod 755 *.py
WORKDIR input


