FROM python:3.6-slim

RUN pip3 install kubernetes==10.0.1 kfserving==0.3.0 requests==2.22.0 Flask==1.1.1 flask-cors==3.0.8

ENV APP_HOME /app
COPY src $APP_HOME
WORKDIR $APP_HOME

ENTRYPOINT ["python"]
CMD ["kfservingdeployer.py"]
