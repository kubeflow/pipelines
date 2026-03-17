FROM python:3.11-slim
WORKDIR /kfp
RUN pip install uv
COPY pyproject.toml uv.lock ./
RUN uv sync --frozen --no-install-workspace --no-install-project --extra legacy-compatible-runtime \
	&& uv pip install \
		"kubernetes>=8.0.0,<=31.0.0" \
		"PyYAML>=5.3,<7" \
		"ml-metadata==1.17.0"
ENV PATH="/kfp/.venv/bin:$PATH"

COPY backend/metadata_writer/src/ /kfp/metadata_writer/
CMD python3 -u /kfp/metadata_writer/metadata_writer.py
