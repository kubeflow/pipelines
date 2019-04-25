FROM python:3.6.8-stretch

RUN pip install --upgrade pip
RUN pip install --upgrade watson-machine-learning-client ibm-ai-openscale --no-cache | tail -n 1
RUN pip install psycopg2-binary | tail -n 1

ENV APP_HOME /app
COPY src $APP_HOME
WORKDIR $APP_HOME

ENTRYPOINT ["python"]
CMD ["monitor_quality.py"]
