FROM python:3.7-slim

COPY requirements.txt .
RUN pip install -r requirements.txt

COPY queue-pipeline/src/queue_pipeline.py /scripts/queue_pipeline.py

ENTRYPOINT [ "bash" ]
