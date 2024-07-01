FROM python:3.9.5
RUN python3 -m pip install prophet==1.0.1 google-cloud-storage==1.40.0 google-cloud-bigquery==2.20.0 google-cloud-bigquery-storage==2.5.0 flask==2.0.1 gunicorn==20.1.0
WORKDIR /pipelines/component/src/
COPY fit_model.py fit_model_args.py util.py /pipelines/component/src/
COPY prediction.py prediction_args.py /pipelines/component/src/
COPY online_prediction.py online_prediction_start.sh /pipelines/component/src/
EXPOSE 8080
