FROM python:3.13.16-slim-bookworm@sha256:a1165e272e578941b84abc79e4ab38a0305cd12803a5c4247979ac7655f4d641

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1

COPY requirements.txt /tmp/requirements.txt
RUN python -m pip install --no-cache-dir --require-hashes -r /tmp/requirements.txt
COPY sync.py /hooks/sync.py

USER 1000:0
# Exercise the deployed interpreter, dependencies and entrypoint before publishing.
RUN --mount=type=bind,source=runtime_smoke.py,target=/tmp/runtime_smoke.py \
    python /tmp/runtime_smoke.py /hooks/sync.py

EXPOSE 8080
CMD ["python", "/hooks/sync.py"]
