FROM public.ecr.aws/amazonlinux/amazonlinux:2

ENV PYTHONPATH /app

RUN yum update -y \
    && yum install -y -q \
    python3

RUN pip3 install uv
RUN uv pip install --system \
    "boto3==1.14.12" \
    "sagemaker==2.237.3" \
    "pathlib2==2.3.5" \
    "pyyaml==5.4" \
    "mypy-extensions==0.4.3" \
    "protobuf==3.20.*" \
    "kfp==1.7.0" \
    "docformatter==1.3.1" \
    "black==19.10b0" \
    "coverage==5.1" \
    "pytest==5.4.1"

COPY . /app/

WORKDIR /app/components/aws/sagemaker/tests/unit_tests/

ENTRYPOINT [ "bash", "./run_automated_test.sh" ]