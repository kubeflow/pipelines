FROM amazonlinux:2

ENV PYTHONPATH /app

RUN yum update -y \
 && yum install -y -q \
    python3

# requirements.txt is copied separately to preserve cache
COPY ./sagemaker/tests/unit_tests/requirements.txt .
RUN pip3 install -r requirements.txt

COPY ./sagemaker/ /app/

WORKDIR /app/tests/unit_tests/

ENTRYPOINT [ "bash", "./run_all_tests.sh" ]