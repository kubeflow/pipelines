FROM ubuntu:18.04

# Directories for model codes and secrets
RUN mkdir /app
RUN mkdir /app/secrets

# Python3 and pip3
RUN apt-get update
RUN apt-get install python3-pip -y

# Watson studio and machine learning python client
RUN pip3 install watson_machine_learning_client boto3

# Python functions with endpoints to Watson Machine Learning
COPY src/wml-train.py /app
