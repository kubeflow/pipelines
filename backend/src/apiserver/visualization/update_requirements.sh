#!/bin/bash

# This image should be in sync with Dockerfile.visualization.
IMAGE=tensorflow/tensorflow:2.4.0
../../../update_requirements.sh $IMAGE <requirements.in >requirements.txt
