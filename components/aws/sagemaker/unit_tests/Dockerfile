# How to build this docker file
#
# https://github.com/kubeflow/pipelines.git
#
# cd pipelines/components/aws
#
# docker build ./ -f ./sagemaker/unit_tests/Dockerfile -t amazon/unit-test-aws-sagemaker-kfp-components:20200427
#
# Run tests
# docker run -it amazon/unit-test-aws-sagemaker-kfp-components:20200427


FROM amazonlinux:2

ENV PYTHONPATH /app

RUN yum update -y \
 && yum install -y -q \
    python3

RUN pip3 install \
    coverage==5.1 \
    pytest==5.4.1 \
    pyyaml==3.12 \
    sagemaker==1.54.0

copy ./sagemaker/ /app/

WORKDIR /app/unit_tests/

CMD [ "bash", "./run_all_tests.sh" ]