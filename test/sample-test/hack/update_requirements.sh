#!/bin/bash

set -ex

cat ../../sdk/python/requirements.in ../../backend/api/python_http_client/requirements.txt requirements.in | \
    ../../backend/update_requirements.sh google/cloud-sdk:315.0.0 >requirements.txt
