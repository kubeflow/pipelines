# This image has the script to kick off the ML pipeline sample e2e test,
FROM google/cloud-sdk:279.0.0

RUN apt-get update -y
RUN apt-get install --no-install-recommends -y -q libssl-dev libffi-dev wget ssh
RUN apt-get install --no-install-recommends -y -q default-jre default-jdk python3-setuptools python3.7-dev gcc libpython3.7-dev zlib1g-dev

RUN wget https://bootstrap.pypa.io/get-pip.py && python3 get-pip.py

COPY ./test/multi-user-test/requirements.txt /python/src/github.com/kubeflow/pipelines/test/multi-user-test/requirements.txt
RUN pip3 install -r /python/src/github.com/kubeflow/pipelines/test/multi-user-test/requirements.txt

# Install python client, including DSL compiler.
COPY ./sdk/python /sdk/python
RUN cd /sdk/python && python3 setup.py install

COPY ./test/multi-user-test /python/src/github.com/kubeflow/pipelines/test/multi-user-test

ENTRYPOINT ["python3", "/python/src/github.com/kubeflow/pipelines/test/multi-user-test/run_multiuser_test.py"]
