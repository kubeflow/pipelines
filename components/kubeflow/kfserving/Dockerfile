FROM python:3.6-slim

RUN pip install kubernetes==9.0.0 kfserving==0.1.1 requests==2.22.0

ENV APP_HOME /app
COPY src $APP_HOME
WORKDIR $APP_HOME

ENTRYPOINT ["python"]
CMD ["kfservingdeployer.py"]
