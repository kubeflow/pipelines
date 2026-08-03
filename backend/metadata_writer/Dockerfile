FROM python:3.11-slim
RUN apt-get update \
    && apt-get install -y --no-install-recommends libcap2 \
    && rm -rf /var/lib/apt/lists/*
RUN python3 -m pip install --no-cache-dir 'wheel>=0.46.2'
WORKDIR /kfp
RUN pip install uv
COPY pyproject.toml uv.lock ./
COPY sdk/python ./sdk/python
COPY kubernetes_platform/python ./kubernetes_platform/python
COPY api/v2alpha1/python ./api/v2alpha1/python
COPY backend/api/v2beta1/python_http_client ./backend/api/v2beta1/python_http_client
RUN uv sync --frozen --extra metadata-writer --no-install-workspace --no-install-project --no-dev
RUN cd /tmp && uv pip install --python /kfp/.venv/bin/python "ml-metadata==1.21.0"
RUN uv pip check --python /kfp/.venv/bin/python
ENV PATH="/kfp/.venv/bin:$PATH"

COPY backend/metadata_writer/src/ /kfp/metadata_writer/
CMD python3 -u /kfp/metadata_writer/metadata_writer.py
