FROM python:3.6-slim

RUN pip install boto3 ruamel.yaml requests

ENV APP_HOME /app
COPY src $APP_HOME
WORKDIR $APP_HOME

ENTRYPOINT ["python", "train.py"]
