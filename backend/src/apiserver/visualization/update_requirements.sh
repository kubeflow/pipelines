#!/bin/bash

# This image should be in sync with Dockerfile.visualization.
IMAGE="tensorflow/tfx:0.30.1"
# tensorflow/tfx default entrypoint is Apache BEAM, because Apache BEAM doesn't
# support custom entrypoint for now. We need to override with --entrypoint ""
# for other `docker run` usecase.
# https://github.com/tensorflow/tfx/blob/master/tfx/tools/docker/Dockerfile#L71
../../../update_requirements.sh $IMAGE <requirements.in >requirements.txt
