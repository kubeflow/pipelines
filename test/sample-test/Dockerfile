# This image has the script to kick off the ML pipeline sample e2e test,

FROM google/cloud-sdk:315.0.0

RUN apt-get update -y
RUN apt-get install --no-install-recommends -y -q libssl-dev libffi-dev wget ssh
RUN apt-get install --no-install-recommends -y -q default-jre default-jdk python3-setuptools python3.7-dev gcc libpython3.7-dev zlib1g-dev

RUN wget https://bootstrap.pypa.io/get-pip.py && python3 get-pip.py

COPY ./test/sample-test/requirements.txt /python/src/github.com/kubeflow/pipelines/test/sample-test/requirements.txt
RUN pip3 install -r /python/src/github.com/kubeflow/pipelines/test/sample-test/requirements.txt

# Install KFP server API from commit.
COPY ./backend/api/python_http_client /python_http_client
RUN cd /python_http_client && python3 setup.py install

# Install python client, including DSL compiler.
COPY ./sdk/python /sdk/python
RUN cd /sdk/python && python3 setup.py install

COPY ./test/sample-test /python/src/github.com/kubeflow/pipelines/test/sample-test
COPY ./samples /python/src/github.com/kubeflow/pipelines/samples
RUN cd /python/src/github.com/kubeflow/pipelines

# Install argo.
RUN echo "install argo"
RUN mkdir -p /usr/local/bin/
#RUN ARGO_VERSION=v2.7.5 && curl -sSL -o /usr/local/bin/argo \
RUN ARGO_VERSION=v2.4.3 && curl -sSL -o /usr/local/bin/argo \
    "https://github.com/argoproj/argo/releases/download/$ARGO_VERSION/argo-linux-amd64" && \
    chmod +x /usr/local/bin/argo
ENV PATH $PATH:/usr/local/bin/argo

ENTRYPOINT ["python3", "/python/src/github.com/kubeflow/pipelines/test/sample-test/sample_test_launcher.py"]
