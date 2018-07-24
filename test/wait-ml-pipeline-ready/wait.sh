#!/bin/sh

# Copyright 2018 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -xe

usage()
{
    echo "usage: wait.sh
    [--namespace     namespace where ML pipeline is deployed]
    [-h help]"
}

while [ "$1" != "" ]; do
    case $1 in
             --namespace )     shift
                               NAMESPACE=$1
                               ;;
             -h | --help )     usage
                               exit
                               ;;
             * )               usage
                               exit 1
    esac
    shift
done

MAX_ATTEMPT=60
READY_KEYWORD="\"apiServerReady\":true"
CA_CERT=/var/run/secrets/kubernetes.io/serviceaccount/ca.crt
TOKEN=$(cat /var/run/secrets/kubernetes.io/serviceaccount/token)

# probing the UI healthz/ status until it's ready. Timeout after 3 minutes
for i in $(seq 1 ${MAX_ATTEMPT})
do
  UI_STATUS=`curl --cacert $CA_CERT -H "Authorization: Bearer $TOKEN" https://${KUBERNETES_SERVICE_HOST}:${KUBERNETES_SERVICE_PORT}/api/v1/proxy/namespaces/${NAMESPACE}/services/ml-pipeline-ui:80/apis/v1alpha2/healthz`
  echo $UI_STATUS | grep ${READY_KEYWORD} && s=0 && break || s=$? && printf "ML Pipeline Not Ready...\n" && sleep 3
done

if [[ $s != 0 ]]
 then echo "Timeout.." && exit $s
else
 echo "ML Pipeline Ready" && exit $s
fi
