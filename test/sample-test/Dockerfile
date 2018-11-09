# This image has the script to kick off the ML pipeline sample e2e test,

FROM google/cloud-sdk

RUN apt-get update -y && \
    apt-get install --no-install-recommends -y -q python3-pip default-jdk python3-setuptools && \
    apt-get install --no-install-recommends -y -q libssl-dev libffi-dev wget ssh

# Install swagger codegen
RUN wget http://central.maven.org/maven2/io/swagger/swagger-codegen-cli/2.3.1/swagger-codegen-cli-2.3.1.jar -O /tmp/swagger-codegen-cli.jar

RUN pip3 install wheel
RUN pip3 install junit-xml
RUN pip3 install kubernetes
RUN pip3 install minio
RUN pip3 install setuptools==40.5.0

COPY ./run_test.sh /run_test.sh
COPY ./run_kubeflow_test.py /run_kubeflow_test.py
COPY ./run_tfx_test.py /run_tfx_test.py
COPY ./run_basic_test.py /run_basic_test.py
COPY ./run_xgboost_test.py /run_xgboost_test.py
COPY ./utils.py /utils.py
RUN chmod +x /run_test.sh

ENTRYPOINT ["/run_test.sh"]
