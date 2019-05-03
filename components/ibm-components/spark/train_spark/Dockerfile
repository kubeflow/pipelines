FROM python:3.6.8-stretch
RUN apt-get update
RUN apt-get install -y vim wget curl

ENV APP_HOME /app
COPY src $APP_HOME
WORKDIR $APP_HOME
