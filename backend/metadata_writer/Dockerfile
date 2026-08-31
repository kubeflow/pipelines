FROM python:3.11-slim

ARG TARGETARCH

RUN apt-get update \
    && apt-get install -y --no-install-recommends libcap2 \
    && rm -rf /var/lib/apt/lists/*
RUN python3 -m pip install --no-cache-dir 'wheel>=0.46.2'
COPY backend/metadata_writer/requirements.txt /kfp/metadata_writer/

# ml-metadata 1.21.0 does not publish a Linux ARM64 wheel. Build the exact
# pinned release from source on ARM64. Its release WORKSPACE pins rules_java
# 7.10.0, so use the matching Bazel 7.10.0 toolchain instead of the upstream
# .bazelversion (7.7.0), whose local JDK integration is incompatible with the
# rules_java 7.10.0 toolchain package layout.
RUN if [ "${TARGETARCH}" = "arm64" ]; then \
      apt-get update \
      && apt-get install -y --no-install-recommends build-essential ca-certificates curl git \
      && curl -fsSL https://github.com/bazelbuild/bazelisk/releases/download/v1.27.0/bazelisk-linux-arm64 -o /usr/local/bin/bazel \
      && chmod +x /usr/local/bin/bazel \
      && git clone --depth 1 --branch v1.21.0 https://github.com/google/ml-metadata.git /tmp/ml-metadata \
      && cd /tmp/ml-metadata \
      && printf '7.10.0\n' > .bazelversion \
      && BAZELFLAGS="--copt=-march=armv8.3-a --host_copt=-march=armv8.3-a --experimental_google_legacy_api" python3 -m pip wheel . --no-deps --wheel-dir /tmp/mlmd-wheel \
      && python3 -m pip install --no-cache-dir /tmp/mlmd-wheel/ml_metadata-1.21.0-*.whl \
      && cd / \
      && rm -rf /tmp/ml-metadata /tmp/mlmd-wheel \
      && apt-get purge -y --auto-remove build-essential curl git \
      && rm -rf /var/lib/apt/lists/*; \
    else \
      python3 -m pip install --no-cache-dir -r /kfp/metadata_writer/requirements.txt; \
    fi

# Remove ml-metadata from the requirements before the common install on ARM64;
# it has already been built and installed from the pinned upstream source.
RUN if [ "${TARGETARCH}" = "arm64" ]; then \
      sed -i '/^ml-metadata==/d' /kfp/metadata_writer/requirements.txt; \
      python3 -m pip install --no-cache-dir -r /kfp/metadata_writer/requirements.txt; \
    fi

COPY backend/metadata_writer/src/* /kfp/metadata_writer/
CMD ["python3", "-u", "/kfp/metadata_writer/metadata_writer.py"]