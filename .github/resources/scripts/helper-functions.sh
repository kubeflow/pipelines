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

for_each_runtime_base_image() {
  local images_file=$1
  local image_callback=$2
  local image_count=0
  local image

  while IFS= read -r image; do
    if [[ -z "$image" || "$image" == \#* ]]; then
      continue
    fi

    "$image_callback" "$image" || return 1
    image_count=$((image_count+1))
  done < "$images_file"

  if [[ "$image_count" -eq 0 ]]; then
    echo "No runtime base images configured in $images_file." >&2
    return 1
  fi
}

pull_image_with_backoff() {
  local image=$1
  local max_attempts=5
  local attempt=1
  local pull_output

  while [[ "$attempt" -le "$max_attempts" ]]; do
    if pull_output=$(docker pull "$image" 2>&1); then
      printf '%s\n' "$pull_output"
      return 0
    fi
    printf '%s\n' "$pull_output" >&2

    # ECR's data quota is not a short-lived request throttle. Repeating the
    # same download cannot repair it; preserve the registry error above.
    if [[ "$pull_output" == *"Data limit exceeded"* ]]; then
      echo "Image pull quota exhausted for $image; use the shared cache or an authenticated registry source." >&2
      return 1
    fi

    if [[ "$attempt" -eq "$max_attempts" ]]; then
      echo "Failed to pull $image after $max_attempts attempts; check the registry error above." >&2
      return 1
    fi

    # Docker CLI does not expose Retry-After headers. Bound the exponential
    # delay and jitter so concurrent producers do not all retry together.
    local sleep_seconds=$((20 * (1 << (attempt - 1)) + RANDOM % 11))
    if [[ "$sleep_seconds" -gt 120 ]]; then
      sleep_seconds=120
    fi
    echo "Retrying $image in ${sleep_seconds}s (attempt $((attempt + 1))/$max_attempts)..." >&2
    sleep "$sleep_seconds"
    attempt=$((attempt+1))
  done
}

pull_runtime_image_for_archive() {
  local image=$1
  pull_image_with_backoff "$image" || return 1
  # The inventory may pin acquisition with tag@digest. Docker archives must
  # retain the tag used by the fixture: RepoDigests need not survive save/load.
  if [[ "$image" == *@* ]]; then
    docker tag "$image" "${image%@*}" || return 1
  fi
}

pull_and_save_runtime_base_images() {
  local images_file=$1
  local archive_path=$2
  local runtime_base_images=()

  pull_runtime_base_image() {
    local image=$1

    pull_runtime_image_for_archive "$image" || return 1
    runtime_base_images+=("${image%@*}")
  }

  for_each_runtime_base_image "$images_file" pull_runtime_base_image || return 1

  docker save "${runtime_base_images[@]}" -o "$archive_path"
}

load_runtime_base_images_into_kind() {
  local images_file=$1
  local cluster_name=$2

  load_runtime_base_image() {
    local image=$1

    pull_runtime_image_for_archive "$image" || return 1
    kind --name "$cluster_name" load docker-image "${image%@*}" || return 1
    docker image rm "${image%@*}" || true
  }

  for_each_runtime_base_image "$images_file" load_runtime_base_image
}

verify_runtime_base_images_in_kind() {
  local images_file=$1
  local cluster_name=$2
  local nodes
  nodes=$(kind get nodes --name "$cluster_name") || return 1
  if [[ -z "$nodes" ]]; then
    echo "No Kind nodes found for $cluster_name." >&2
    return 1
  fi

  verify_runtime_base_image() {
    local image=${1%@*}
    local node
    while IFS= read -r node; do
      if ! docker exec "$node" crictl inspecti "$image" > /dev/null; then
        echo "Runtime image $image is missing from $node; rebuild the shared runtime image archive." >&2
        return 1
      fi
    done <<< "$nodes"
  }
  for_each_runtime_base_image "$images_file" verify_runtime_base_image
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
    python -m pip install "kubernetes==30.1.0" "urllib3==2.6.3"
    python "${C_DIR}"/kfp-readiness/wait_for_pods.py
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
