#!/bin/bash

# This image should be in sync with Dockerfile.
IMAGE=""
../hack/update-requirements.sh $IMAGE <requirements.in >requirements.txt
