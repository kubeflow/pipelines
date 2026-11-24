#!/bin/sh
# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Older TensorFlow readers accepted a bare host and a separate transport flag.
case ${S3_ENDPOINT:-} in
    ''|*://*) ;;
    *)
        case ${S3_USE_HTTPS:-} in
            0*) S3_ENDPOINT=http://$S3_ENDPOINT ;;
            *) S3_ENDPOINT=https://$S3_ENDPOINT ;;
        esac
        export S3_ENDPOINT
        ;;
esac

# The controller's first argument contains the UI-generated Series1, Series2,
# ... specification when users select more than one run.
case ${1:-} in
    --logdir=Series1:*,Series2:*)
        logdirs=${1#--logdir=}
        shift
        set -- "--logdir_spec=$logdirs" "$@"
        ;;
esac

# Keep remote event reads on the Python filesystem clients shipped in this image.
exec tensorboard-cli --load_fast=false "$@"
