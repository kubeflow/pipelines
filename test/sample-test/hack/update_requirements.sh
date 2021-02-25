#!/bin/bash

set -ex

cat ../../sdk/python/requirements.in ../../backend/requirements.in requirements.in | \
    ../../backend/update_requirements.sh google/cloud-sdk:315.0.0 >requirements.txt
