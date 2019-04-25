# This image has the script to kick off the ML pipeline sample e2e test,

FROM google/cloud-sdk:236.0.0

RUN apt-get update -y && \
    apt-get install --no-install-recommends -y -q python3-pip default-jdk python3-setuptools python3-dev && \
    apt-get install --no-install-recommends -y -q libssl-dev libffi-dev wget ssh

# Install swagger codegen
RUN wget http://central.maven.org/maven2/io/swagger/swagger-codegen-cli/2.4.1/swagger-codegen-cli-2.4.1.jar -O /tmp/swagger-codegen-cli.jar

RUN pip3 install wheel
RUN pip3 install junit-xml
RUN pip3 install kubernetes==9.0.0
RUN pip3 install minio
RUN pip3 install setuptools==40.5.0
RUN pip3 install papermill==0.16.1
RUN pip3 install ipykernel==5.1.0
RUN pip3 install google-api-python-client==1.7.0

#Needs test/sample-test and the files needed to run sdk/python/build.sh
COPY . /python/src/github.com/kubeflow/pipelines 

ENTRYPOINT ["/python/src/github.com/kubeflow/pipelines/test/sample-test/run_test.sh"]
