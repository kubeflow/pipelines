FROM python:3.11-slim
RUN apt-get update \
    && apt-get upgrade -y \
    && apt-get install -y --no-install-recommends libcap2 \
    && rm -rf /var/lib/apt/lists/*
# These services do not install packages or bootstrap virtualenvs at runtime.
RUN python3 -m pip install --no-cache-dir --upgrade pip setuptools wheel
COPY backend/metadata_writer/requirements.txt /kfp/metadata_writer/
RUN python3 -m pip install --no-cache-dir -r /kfp/metadata_writer/requirements.txt \
    && python3 -m pip uninstall --yes pip \
    && rm -rf /usr/local/lib/python3.11/ensurepip \
    && python3 -c "import importlib.util; assert importlib.util.find_spec('pip') is None; assert importlib.util.find_spec('ensurepip') is None"

COPY backend/metadata_writer/src/* /kfp/metadata_writer/
CMD python3 -u /kfp/metadata_writer/metadata_writer.py
