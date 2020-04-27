# How to build this docker file
#
# For now using my branch as an example
# git clone --single-branch --branch unit_test https://github.com/akartsky/pipelines.git
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
    sagemaker==1.54.0 \
    pyyaml==3.12

copy ./sagemaker/ /app/

ENTRYPOINT [ "python3", "/app/unit_tests/run_tests.py" ]