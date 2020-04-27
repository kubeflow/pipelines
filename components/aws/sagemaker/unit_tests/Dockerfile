# a483e75eb9c2:aws kalamadi$ pwd
# /Users/kalamadi/Desktop/work/kfp_my_fork/components/aws
#
# a483e75eb9c2:aws kalamadi$ docker build ./ -f ./sagemaker/unit_tests/Dockerfile -t akartsky/test-aws-sagemaker-kfp-components:0.1
#
# a483e75eb9c2:aws kalamadi$ docker run -it akartsky/test-aws-sagemaker-kfp-components:0.1

FROM amazonlinux:2

ENV PYTHONPATH /app

RUN yum update -y \
 && yum install -y -q \
    ca-certificates \
    python3 \
    wget \
    unzip

RUN pip3 install \
    boto3==1.12.33 \
    sagemaker==1.54.0 \
    pathlib2==2.3.5 \
    pyyaml==3.12

copy ./sagemaker/ /app/

ENTRYPOINT [ "python3", "/app/unit_tests/run_tests.py" ]