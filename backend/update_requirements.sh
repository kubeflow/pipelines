#!/bin/bash
set -euo pipefail
IMAGE=${1:-"python:3.5"}
exec docker run -v "$PWD:/src" -w /src --rm "$IMAGE" bash \
  -c 'pip3 install pip setuptools --upgrade && pip3 install pip-tools==4.5.0 > /dev/null 2>&1 && pip-compile -v requirements.in'
