#!/usr/bin/env bash

GIT_ROOT=$(git rev-parse --show-toplevel)

pip-compile --no-emit-find-links --no-header --no-emit-index-url requirements.in \
  --find-links="${GIT_ROOT}/api/v2alpha1/python/dist" \
  --find-links="${GIT_ROOT}/backend/api/v2beta1/python_http_client/dist" > requirements.txt
