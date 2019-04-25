#!/bin/bash
#
# Copyright 2018 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -ex


usage()
{
    echo "usage: deploy.sh
    [--gcr_image_base_dir   the gcr image base directory including images such as apiImage and persistenceAgentImage]
    [--gcr_image_tag   		the tags for images such as apiImage and persistenceAgentImage]
    [-h help]"
}
GCR_IMAGE_TAG=latest

while [ "$1" != "" ]; do
    case $1 in
             --gcr_image_base_dir )   shift
                                      GCR_IMAGE_BASE_DIR=$1
                                      ;;
             --gcr_image_tag )   	  shift
                                      GCR_IMAGE_TAG=$1
                                      ;;                                      
             -h | --help )            usage
                                      exit
                                      ;;
             * )                      usage
                                      exit 1
    esac
    shift
done

cd ${DIR}/${KFAPP}

## Update pipeline component image
pushd ks_app
ks param set pipeline apiImage ${GCR_IMAGE_BASE_DIR}/api:${GCR_IMAGE_TAG}
ks param set pipeline persistenceAgentImage ${GCR_IMAGE_BASE_DIR}/persistenceagent:${GCR_IMAGE_TAG}
ks param set pipeline scheduledWorkflowImage ${GCR_IMAGE_BASE_DIR}/scheduledworkflow:${GCR_IMAGE_TAG}
ks param set pipeline uiImage ${GCR_IMAGE_BASE_DIR}/frontend:${GCR_IMAGE_TAG}
ks apply default -c pipeline
popd