FROM jupyter/pyspark-notebook:c1b0cf6bf4d6

RUN pip install --upgrade pip
RUN pip install --upgrade watson-machine-learning-client ibm-ai-openscale --no-cache | tail -n 1
RUN pip install psycopg2-binary | tail -n 1

ENV APP_HOME /app
COPY src $APP_HOME
WORKDIR $APP_HOME

USER root

ENTRYPOINT ["python"]
CMD ["spark_data_preparation.py"]
