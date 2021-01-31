FROM python:3.6-slim

COPY requirements.txt .
RUN pip3 install -r requirements.txt

ENV APP_HOME /app
COPY src $APP_HOME
WORKDIR $APP_HOME

ENTRYPOINT ["python", "train.py"]
