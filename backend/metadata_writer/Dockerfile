FROM python:3.11-slim

ARG TARGETARCH

RUN apt-get update \
    && apt-get install -y --no-install-recommends libcap2 \
    && rm -rf /var/lib/apt/lists/*
RUN python3 -m pip install --no-cache-dir 'wheel>=0.46.2'
COPY backend/metadata_writer/requirements.txt /kfp/metadata_writer/

# ml-metadata 1.21.0 does not publish a Linux ARM64 wheel. Build the exact
# pinned release from source on ARM64. MLMD pins rules_java 7.10.0, while the
# Bazel local-JDK repository still requests the legacy bootstrap toolchain label
# @rules_java//tools/jdk. Provide that compatibility alias in a local copy of
# rules_java instead of weakening the MLMD dependency pin.
RUN if [ "${TARGETARCH}" = "arm64" ]; then \
      apt-get update \
      && apt-get install -y --no-install-recommends build-essential ca-certificates cmake curl git python3-dev \
      && curl -fsSL https://github.com/bazelbuild/bazelisk/releases/download/v1.27.0/bazelisk-linux-arm64 -o /usr/local/bin/bazel \
      && chmod +x /usr/local/bin/bazel \
      && git clone --depth 1 --branch v1.21.0 https://github.com/google/ml-metadata.git /tmp/ml-metadata \
      && git clone --depth 1 --branch 7.10.0 https://github.com/bazelbuild/rules_java.git /tmp/rules_java \
      && mkdir -p /tmp/rules_java/tools/jdk \
      && printf '%s\n' 'package(default_visibility = ["//visibility:public"])' 'alias(name = "bootstrap_runtime_toolchain_type", actual = "//toolchains:bootstrap_runtime_toolchain_type")' > /tmp/rules_java/tools/jdk/BUILD \
      && cd /tmp/ml-metadata \
      && BAZELFLAGS="--copt=-march=armv8.3-a --host_copt=-march=armv8.3-a --experimental_google_legacy_api --override_repository=rules_java=/tmp/rules_java" python3 setup.py bdist_wheel --dist-dir /tmp/mlmd-wheel \
      && python3 -m pip install --no-cache-dir /tmp/mlmd-wheel/ml_metadata-1.21.0-*.whl \
      && cd / \
      && rm -rf /tmp/ml-metadata /tmp/rules_java /tmp/mlmd-wheel \
      && apt-get purge -y --auto-remove build-essential cmake curl git python3-dev \
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