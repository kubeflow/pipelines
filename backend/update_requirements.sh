#!/bin/bash

# Usage: ./update_requirements.sh <requirements.in >requirements.txt

set -euo pipefail
IMAGE=${1:-"python:3.7"}
docker run --interactive --rm "$IMAGE" sh -c '
  python3 -m pip install pip setuptools --upgrade --quiet
  python3 -m pip install pip-tools==5.4.0 --quiet
  pip-compile --verbose --output-file - -
'
