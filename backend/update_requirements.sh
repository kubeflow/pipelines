#!/bin/bash
set -euo pipefail

exec docker run -v "$PWD:/src" -w /src --rm python:3.5 bash \
  -c 'pip3 install pip-tools==4.5.0 > /dev/null 2>&1 && pip-compile -v requirements.in'
