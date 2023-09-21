#!/bin/bash

# This image should be in sync with Dockerfile.
IMAGE="python:3.7"
../hack/update-requirements.sh $IMAGE <requirements.in >requirements.txt
