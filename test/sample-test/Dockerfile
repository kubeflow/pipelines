# This image has the script to kick off the ML pipeline sample e2e test,

FROM google/cloud-sdk:236.0.0

RUN apt-get update -y && \
    apt-get install --no-install-recommends -y -q default-jdk python3-setuptools python3-dev && \
    apt-get install --no-install-recommends -y -q libssl-dev libffi-dev wget ssh

RUN wget https://bootstrap.pypa.io/get-pip.py && python3 get-pip.py

RUN pip3 install wheel
RUN pip3 install junit-xml
RUN pip3 install kubernetes==9.0.0
RUN pip3 install minio
RUN pip3 install setuptools==40.5.0
RUN pip3 install papermill==1.0.1
RUN pip3 install ipykernel==5.1.0
RUN pip3 install google-api-python-client==1.7.0
RUN pip3 install google-cloud-storage==1.17.0
RUN pip3 install fire==0.2.1
RUN pip3 install yamale==2.0

# Install python client, including DSL compiler.
COPY ./sdk/python /sdk/python
RUN cd /sdk/python && python3 setup.py install

COPY ./test/sample-test /python/src/github.com/kubeflow/pipelines/test/sample-test
COPY ./samples /python/src/github.com/kubeflow/pipelines/samples
RUN cd /python/src/github.com/kubeflow/pipelines

# Install argo.
RUN echo "install argo"
RUN mkdir -p /usr/local/bin/
RUN ARGO_VERSION=v2.3.0 && curl -sSL -o /usr/local/bin/argo \
    "https://github.com/argoproj/argo/releases/download/$ARGO_VERSION/argo-linux-amd64" && \
    chmod +x /usr/local/bin/argo
ENV PATH $PATH:/usr/local/bin/argo

ENTRYPOINT ["python3", "/python/src/github.com/kubeflow/pipelines/test/sample-test/sample_test_launcher.py"]
