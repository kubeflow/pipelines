#!/bin/bash

# This image should be in sync with Dockerfile.
IMAGE="public.ecr.aws/docker/library/python:3.12"
../hack/update-requirements.sh $IMAGE <requirements.in >requirements.txt
