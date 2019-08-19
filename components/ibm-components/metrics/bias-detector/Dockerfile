FROM pytorch/pytorch:latest

RUN pip install aif360 pandas Minio Pillow torchsummary

ENV APP_HOME /app
COPY src $APP_HOME
WORKDIR $APP_HOME

ENTRYPOINT ["python"]
CMD ["fairness_check.py"]
