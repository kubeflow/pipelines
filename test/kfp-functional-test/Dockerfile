# This image has the script to install the base image needed for kfp functional e2e test,

FROM google/cloud-sdk:315.0.0
RUN apt-get update -y
RUN apt-get install --no-install-recommends -y -q libssl-dev libffi-dev wget ssh
RUN apt-get install --no-install-recommends -y -q python3-setuptools python3.7-dev libpython3.7-dev
RUN wget https://bootstrap.pypa.io/get-pip.py && python3 get-pip.py