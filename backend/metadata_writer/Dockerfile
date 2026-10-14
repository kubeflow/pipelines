FROM python:3.11-slim
RUN apt-get update \
    && apt-get upgrade -y \
    && apt-get install -y --no-install-recommends libcap2 \
    && rm -rf /var/lib/apt/lists/*
RUN python3 -m pip install --no-cache-dir --upgrade pip setuptools wheel
COPY backend/metadata_writer/requirements.txt /kfp/metadata_writer/
RUN python3 -m pip install --no-cache-dir -r /kfp/metadata_writer/requirements.txt \
    && python3 -m pip uninstall --yes pip

COPY backend/metadata_writer/src/* /kfp/metadata_writer/
CMD python3 -u /kfp/metadata_writer/metadata_writer.py
