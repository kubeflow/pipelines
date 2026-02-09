#!/bin/bash
#
# Copyright 2023 kubeflow.org
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

retry() {
  local max=$1; shift
  local interval=$1; shift

  until "$@"; do
    echo "trying.."
    max=$((max-1))
    if [[ "$max" -eq 0 ]]; then
      return 1
    fi
    sleep "$interval"
  done
}

wait_for_namespace () {
    if [[ $# -ne 3 ]]
    then
        echo "Usage: wait_for_namespace namespace max_retries sleep_time"
        return 1
    fi

    local namespace=$1
    local max_retries=$2
    local sleep_time=$3

    local i=0

    while [[ $i -lt $max_retries ]]
    do
        if kubectl get ns | grep -qow "$namespace"
        then
            return 0
        fi
        echo "$namespace not found. Checking again in ${sleep_time}s."
        sleep "$sleep_time"
        i=$((i+1))
    done

    return 1
}

wait_for_pods () {
    C_DIR="${BASH_SOURCE%/*}"
    pip install -r "${C_DIR}"/kfp-readiness/requirements.txt
    python "${C_DIR}"/kfp-readiness/wait_for_pods.py
}

wait_for_seaweedfs_init () {
    # Wait for SeaweedFS init job to complete to ensure S3 auth is configured
    local namespace="$1"
    local timeout="$2"
    if kubectl -n "$namespace" get job init-seaweedfs > /dev/null 2>&1; then
        if ! kubectl -n "$namespace" wait --for=condition=complete --timeout="$timeout" job/init-seaweedfs; then
            return 1
        fi
    fi
}

deploy_with_retries () {
    if [[ $# -ne 4 ]]
    then
        echo "Usage: deploy_with_retries (-f FILENAME | -k DIRECTORY) manifest max_retries sleep_time"
        return 1
    fi 

    local flag="$1"
    local manifest="$2"
    local max_retries="$3"
    local sleep_time="$4"
    
    local i=0

    while [[ $i -lt $max_retries ]]
    do
        local exit_code=0

        kubectl apply "$flag" "$manifest" || exit_code=$?

        if [[ $exit_code -eq 0 ]]
        then
            return 0
        fi
        
        echo "Deploy unsuccessful with error code $exit_code. Trying again in ${sleep_time}s."
        sleep "$sleep_time"
        i=$((i+1))
    done

    return 1
}

wait_for_pod () {
    local namespace=$1
    local pod_name=$2
    local max_tries=$3
    local sleep_time=$4

    until pod_is_running "$namespace" "$pod_name"; do
        max_tries=$((max_tries-1))
        if [[ "$max_tries" -eq 0 ]]; then
            return 1
        fi
        echo "Checking again in $sleep_time"
        sleep "$sleep_time"
    done

    return 0
}

pod_is_running () {
    local namespace=$1
    local pod_name=$2

    local pod_status

    # May have unexpected results if pod_name has multiple matches
    pod_status=$(kubectl get pod -n "$namespace" | grep "$pod_name*" | head -1 | awk '{print $3}')

    if [ "$pod_status" = "Running" ]; then
        return 0
    fi

    return 1
}

wait_for_pipeline_run () {
    local run_name=$1
    local max_tries=$2
    local sleep_time=$3

    until pipeline_run_is_success "$run_name"; do
        max_tries=$((max_tries-1))
        if [[ "$max_tries" -eq 0 ]]; then
            return 1
        fi
        echo "Checking pipeline run again in $sleep_time"
        sleep "$sleep_time"
    done

    return 0
}

wait_for_pipeline_run_rev () {
    local run_name=$1
    local max_tries=$2
    local sleep_time=$3

    until [ "$(pipeline_run_is_success_rev "$run_name")" = "0" ]; do
        max_tries=$((max_tries-1))
        if [[ "$max_tries" -eq 0 ]]; then
            echo "1"
            return
        fi
        sleep "$sleep_time"
    done

    echo "0"
    return
}

pipeline_run_is_success () {
    local run_name=$1

    local run_status

    # May have unexpected results if run_status has multiple matches
    run_status=$(kubectl get pipelineruns "$run_name" | tail -1 | awk '{print $2}')

    if [ "$run_status" = "True" ]; then
        return 0
    elif [ "$run_status" = "False" ]; then
        echo "Run Failed"
        exit 1
    fi

    return 1
}

pipeline_run_is_success_rev () {
    local run_name=$1

    local run_status

    # May have unexpected results if run_status has multiple matches
    run_status=$(kubectl get pipelineruns "$run_name" | tail -1 | awk '{print $2}')

    if [ "$run_status" = "True" ]; then
        echo "0"
        return
    elif [ "$run_status" = "False" ]; then
        echo "1"
        return
    fi

    echo "1"
    return
}

collect_artifacts() {
    local kubeflow_ns=$1

    local log_dir=$(mktemp -d)

    pods_kubeflow=$(kubectl get pods -n $kubeflow_ns --no-headers -o custom-columns=NAME:.metadata.name)

    for pod in $pods_kubeflow; do
        kubectl logs -n $kubeflow_ns $pod > $log_dir/$pod.log
    done
}
