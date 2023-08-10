#!/bin/bash
set -euo pipefail

# This image should be in sync with Dockerfile.visualization.
IMAGE="tensorflow/tensorflow:2.5.1"

# tensorflow/tfx default entrypoint is Apache BEAM, because Apache BEAM doesn't
# support custom entrypoint for now. We need to override with --entrypoint ""
# for other `docker run` usecase.
# https://github.com/tensorflow/tfx/blob/master/tfx/tools/docker/Dockerfile#L71
docker run -i --rm  --entrypoint "" "$IMAGE" sh -c '
  python3 -m pip install pip setuptools --upgrade --quiet
  python3 -m pip install pip-tools==5.4.0 --quiet
  pip-compile --verbose --output-file - -
' <requirements.in >requirements.txt