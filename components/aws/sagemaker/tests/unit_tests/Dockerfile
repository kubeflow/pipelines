FROM amazonlinux:2

ENV PYTHONPATH /app

RUN yum update -y \
 && yum install -y -q \
    python3

copy ./sagemaker/ /app/

WORKDIR /app/tests/unit_tests/

RUN pip3 install -r requirements.txt

ENTRYPOINT [ "bash", "./run_all_tests.sh" ]