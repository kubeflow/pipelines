#!/bin/bash

set -o allexport
source .env
set +o allexport

# Function to patch CSV
patch_csv() {
    local CSV_NAME="$1"
    local NAMESPACE_NAME="$2"
    local DSPO_OWNER="$3"
    local DSPO_REPO="$4"
    local DSPO_BRANCH="$5"

    echo "Starting aipipelines component patching for CSV: $CSV_NAME in namespace: $NAMESPACE_NAME"

    # Get the directory where this script is located
    local script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    local overlays_dir="$script_dir/overlays"

    # Step 1: Create PersistentVolumeClaim
    echo "Creating PVC for aipipelines component manifests..."
    if oc apply -f "$overlays_dir/dspo-pvc.yaml" -n "$NAMESPACE_NAME"; then
        echo "PVC created successfully"
    else
        echo "Failed to create PVC, continuing anyway..."
    fi

    # Step 2: Download and build DSPO manifest
    local base_config_path
    base_config_path=$(download_and_prepare_dspo_manifests "$DSPO_OWNER" "$DSPO_REPO" "$DSPO_BRANCH")
    if [[ $? -ne 0 || -z "$base_config_path" ]]; then
        echo "ERROR: Failed to download and prepare DSPO manifests"
        exit 1
    fi

    # Step 3: Apply CSV patch and wait for operator pod readiness
    local operator_label
    operator_label=$(apply_csv_patch_and_wait_for_operator "$CSV_NAME" "$NAMESPACE_NAME" "$overlays_dir")
    if [[ $? -ne 0 || -z "$operator_label" ]]; then
        echo "ERROR: Failed to apply CSV patch or wait for operator pod readiness"
        exit 1
    fi

    # Step 4: Copy DSPO manifests to the operator pods
    copy_dspo_manifests_to_pods "$base_config_path" "$operator_label" "$NAMESPACE_NAME"
    local copy_result=$?
    if [[ $copy_result -eq 1 ]]; then
        echo "ERROR: Failed to copy custom manifests to any pods"
        exit 1
    fi

    # Step 5: Restart operator deployment
    echo "Restarting operator deployment to pick up changes..."
    if oc rollout restart deploy -n "$NAMESPACE_NAME" -l "$operator_label"; then
        echo "Operator deployment restart initiated"

        # Wait for rollout to complete
        echo "Waiting for deployment rollout to complete..."
        if oc rollout status deploy -n "$NAMESPACE_NAME" -l "$operator_label" --timeout=300s; then
            echo "Operator deployment rollout completed successfully"
        else
            echo "Warning: Deployment rollout did not complete within timeout"
            exit 1
        fi
    else
        echo "Failed to restart operator deployment"
        return 1
    fi

    # Step 6: Wait for DSPO controller manager pod to be running
    wait_for_dspo_controller_manager "$NAMESPACE_NAME"

    echo "Finished patching aipipelines component for CSV: $CSV_NAME"
}

# Function to apply CSV patch and wait for operator pod readiness
# Returns the operator label via stdout on success
apply_csv_patch_and_wait_for_operator() {
    local csv_name="$1"
    local namespace_name="$2"
    local overlays_dir="$3"

    echo "Checking if aipipelines volume mount already exists..." >&2

    # Check if the volume mount already exists
    if oc get csv "$csv_name" -n "$namespace_name" -o jsonpath='{.spec.install.spec.deployments[0].spec.template.spec.containers[0].volumeMounts[*].name}' | grep -q "aipipelines-manifests"; then
        echo "aipipelines volume mount already exists, skipping patch..." >&2
    else
        echo "Applying CSV patch to mount aipipelines manifests volume..." >&2
        if oc patch csv "$csv_name" -n "$namespace_name" --type json --patch-file "$overlays_dir/aipipelines-csv-patch.json"; then
            echo "CSV patch applied successfully" >&2
        else
            echo "Failed to apply CSV patch, exiting..." >&2
            return 1
        fi
    fi

    echo "Waiting for operator pod to be ready..." >&2
    local operator_label=""
    if [[ "$namespace_name" == "redhat-ods-operator" ]]; then
        operator_label="name=rhods-operator"
    elif [[ "$namespace_name" == "openshift-operators" ]]; then
        operator_label="name=opendatahub-operator"
    else
        echo "Unknown namespace: $namespace_name, using generic operator label" >&2
        operator_label="app=operator"
    fi

    # Wait up to 5 minutes for pod to be ready
    if oc wait --for='jsonpath={.status.conditions[?(@.type=="Ready")].status}=True' po -l "$operator_label" -n "$namespace_name" --timeout=300s >&2; then
        echo "Operator pod is ready" >&2
    else
        echo "Warning: Operator pod did not become ready within timeout, continuing anyway..." >&2
    fi

    # Return the operator label via stdout
    echo "$operator_label"
    return 0
}

# Function to download and prepare DSPO manifests
# Returns the base config path on stdout, or empty string on failure
download_and_prepare_dspo_manifests() {
    local dspo_owner="$1"
    local dspo_repo="$2"
    local dspo_branch="$3"

    local dspo_path="/tmp/dspo"
    mkdir -p "$dspo_path"
    echo "Downloading DSPO repository from GitHub..." >&2

    # Retry logic: attempt download up to 3 times (1 initial + 2 retries)
    local download_success=false
    local max_attempts=3
    local attempt=1
    local download_url="https://github.com/${dspo_owner}/${dspo_repo}/archive/refs/heads/${dspo_branch}.tar.gz"

    echo "Downloading DSPO from: $download_url" >&2

    while [[ $attempt -le $max_attempts && $download_success == false ]]; do
        echo "Download attempt $attempt of $max_attempts..." >&2
        if wget -q "$download_url" -O "$dspo_path/${dspo_branch}.tar.gz"; then
            download_success=true
            echo "Download successful on attempt $attempt" >&2
        else
            echo "Download failed on attempt $attempt" >&2
            if [[ $attempt -lt $max_attempts ]]; then
                echo "Retrying in 2 seconds..." >&2
                sleep 2
            fi
        fi
        ((attempt++))
    done

    if [[ $download_success == true ]]; then
        echo "Extracting DSPO repository..." >&2
        tar -xzf "$dspo_path/${dspo_branch}.tar.gz" -C "$dspo_path"
        rm -f "$dspo_path/${dspo_branch}.tar.gz"

        echo "Preparing DSPO manifest source directory..." >&2
        local base_config_path="$dspo_path/${dspo_repo}-${dspo_branch}/config"

        echo "DSPO manifest source directory found, processing..." >&2

        # Create the missing overlays/odh/default directory structure if it doesn't exist
        local odh_default_path="$base_config_path/overlays/odh/default"
        if [[ ! -d "$odh_default_path" ]]; then
            echo "Creating missing overlays/odh/default directory..." >&2
            mkdir -p "$odh_default_path"
            # Copy dspo content to default if dspo directory exists
            if [[ -d "$base_config_path/overlays/odh/dspo" ]]; then
                cp -r "$base_config_path/overlays/odh/dspo"/* "$odh_default_path/" 2>/dev/null || true
                echo "Copied dspo overlay content to default directory" >&2
            else
                # Create a basic kustomization.yaml
                cat > "$odh_default_path/kustomization.yaml" << 'EOF'
resources:
- ../../../base
EOF
                echo "Created basic kustomization.yaml in default directory" >&2
            fi
        fi

        # Return the path to the config directory via stdout
        echo "$base_config_path"
        return 0
    else
        echo "Failed to download DSPO repository after $max_attempts attempts" >&2
        return 1
    fi
}

# Function to copy DSPO manifests to operator pods
copy_dspo_manifests_to_pods() {
    local base_config_path="$1"
    local operator_label="$2"
    local namespace_name="$3"

    if [[ -d "$base_config_path" ]]; then
        echo "Copying custom DSPO manifests to operator pods..."

        # Get all running operator pod names
        local pod_names
        pod_names=$(oc get po -l "$operator_label" -n "$namespace_name" --field-selector=status.phase=Running -o jsonpath="{.items[*].metadata.name}" 2>/dev/null)

        if [[ -n "$pod_names" ]]; then
            # Convert space-separated names to array
            read -ra pod_array <<< "$pod_names"
            local pod_count=${#pod_array[@]}
            echo "Found $pod_count running operator pod(s)"

            local copy_success_count=0
            local copy_failure_count=0

            for pod_name in "${pod_array[@]}"; do
                echo "Copying manifests to pod: $pod_name"
                local full_pod_name="$namespace_name/$pod_name"

                if oc cp "$base_config_path/." "$full_pod_name":"/opt/manifests/datasciencepipelines"; then
                    echo "✓ Successfully copied manifests to $pod_name"
                    ((copy_success_count++))
                else
                    echo "✗ Failed to copy manifests to $pod_name"
                    ((copy_failure_count++))
                fi
            done

            echo "Manifest copy summary: $copy_success_count successful, $copy_failure_count failed out of $pod_count pods"

            if [[ $copy_success_count -eq 0 ]]; then
                echo "ERROR: Failed to copy custom manifests to any pods, operator will use default manifests"
                return 1
            elif [[ $copy_failure_count -gt 0 ]]; then
                echo "WARNING: Some pods failed to receive custom manifests, they will use default manifests"
                return 2
            else
                echo "All operator pods successfully received custom manifests"
                return 0
            fi
        else
            echo "Warning: Could not find any running operator pods, skipping manifest copy"
            return 3
        fi
    else
        echo "No custom DSPO manifests found at $base_config_path, operator will use default manifests"
        return 4
    fi
}

# Function to wait for DSPO controller manager pod to be ready
wait_for_dspo_controller_manager() {
    local NAMESPACE_NAME="$1"

    echo "Waiting for data-science-pipelines-operator-controller-manager pod to be ready..."
    local dspo_pod_ready=false
    local max_wait_time=300  # 5 minutes
    local wait_interval=10   # Check every 10 seconds
    local elapsed_time=0

    while [[ $elapsed_time -lt $max_wait_time ]]; do
        # Check if the DSPO controller manager pod exists and is running
        local dspo_pod_status=$(oc get pods -n "$NAMESPACE_NAME" --field-selector=status.phase=Running --no-headers 2>/dev/null | grep "data-science-pipelines-operator-controller-manager" | wc -l)

        if [[ $dspo_pod_status -gt 0 ]]; then
            echo "✓ data-science-pipelines-operator-controller-manager pod is running"
            dspo_pod_ready=true
            break
        else
            echo "Waiting for data-science-pipelines-operator-controller-manager pod... (${elapsed_time}s/${max_wait_time}s)"
            sleep $wait_interval
            elapsed_time=$((elapsed_time + wait_interval))
        fi
    done

    if [[ $dspo_pod_ready == false ]]; then
        echo "WARNING: data-science-pipelines-operator-controller-manager pod did not become ready within ${max_wait_time} seconds"
        echo "This may indicate an issue with the DSPO deployment, but continuing..."
        return 1
    fi

    return 0
}

find_csv_and_update() {
    local DSPO_OWNER="$1"
    local DSPO_REPO="$2"
    local DSPO_BRANCH="$3"

    # Check and update operator images
    echo "Checking for operator namespaces and updating CSV images..."

    # Get list of all namespaces
    NAMESPACE_LIST=$(oc get namespaces -o jsonpath='{.items[*].metadata.name}')

    # Check for redhat-ods-operator namespace
    if echo "$NAMESPACE_LIST" | grep -q "redhat-ods-operator"; then
        echo "Found redhat-ods-operator namespace, updating rhods-operator CSV..."

        # Get CSV matching rhods-operator*
        RHODS_CSV=$(oc get csv -n redhat-ods-operator --no-headers | grep "rhods-operator" | awk '{print $1}' | head -1)

        if [[ -n "$RHODS_CSV" ]]; then
            echo "Found RHODS CSV: $RHODS_CSV"
            patch_csv "$RHODS_CSV" "redhat-ods-operator" "$DSPO_OWNER" "$DSPO_REPO" "$DSPO_BRANCH"
         else
            echo "No rhods-operator CSV found in redhat-ods-operator namespace"
            exit 1
        fi
    # Check for openshift-operators namespace
    elif echo "$NAMESPACE_LIST" | grep -q "openshift-operators"; then
        echo "Found openshift-operators namespace, updating opendatahub-operator CSV..."

        # Get CSV matching opendatahub-operator*
        ODH_CSV=$(oc get csv -n openshift-operators --no-headers | grep "opendatahub-operator" | awk '{print $1}' | head -1)

        if [[ -n "$ODH_CSV" ]]; then
            echo "Found ODH CSV: $ODH_CSV"
            patch_csv "$ODH_CSV" "openshift-operators" "$DSPO_OWNER" "$DSPO_REPO" "$DSPO_BRANCH"
        else
            echo "No opendatahub-operator CSV found in openshift-operators namespace"
            exit 1
        fi
    else
      echo "No RHOAI or ODH operator found, exiting..."
      exit 1
    fi

    echo "Finished checking and updating operator CSV images"
}
