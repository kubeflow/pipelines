FROM aipipeline/pyspark:spark-2.1

RUN pip install --upgrade pip
RUN pip install --upgrade watson-machine-learning-client ibm-ai-openscale Minio --no-cache | tail -n 1
RUN pip install psycopg2-binary | tail -n 1

ENV APP_HOME /app
COPY src $APP_HOME
WORKDIR $APP_HOME

USER root

ENTRYPOINT ["python"]
CMD ["store_spark_model.py"]
