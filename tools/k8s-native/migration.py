# Copyright 2025 The Kubeflow Authors
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

import argparse
import requests
import yaml
import os
from pathlib import Path
from kfp.dsl import utils

# Constants and defaults
K8S_PIPELINE_API_VERSION = 'pipelines.kubeflow.org/v2beta1'
MAX_NAME_LENGTH = 253
DEFAULT_NAMESPACE = 'kubeflow'
DEFAULT_OUTPUT_DIR = './kfp-exported-pipelines'
REQUEST_TIMEOUT = 10  # seconds

def parse_args():
    parser = argparse.ArgumentParser(description="Migrate KFP pipelines to Kubernetes manifests")
    parser.add_argument("--kfp-server-host", default=os.getenv("KFP_SERVER_HOST"), required=True,
                        help="KFP pipeline server host (e.g., https://<host>). Defaults to the value of the KFP_SERVER_HOST environment variable.")
    parser.add_argument("--token", default=os.getenv("KFP_BEARER_TOKEN"), help="Bearer token for authentication. Defaults to the value of the KFP_BEARER_TOKEN environment variable.")    
    parser.add_argument("--ca-bundle", default=os.getenv("CA_BUNDLE"), help="Path to custom CA bundle file. Defaults to the value of the CA_BUNDLE environment variable")
    parser.add_argument('--output', '-o', default=DEFAULT_OUTPUT_DIR, help="Output directory path where pipeline YAMLs will be written(e.g., '/path/to/exported-pipelines')")
    parser.add_argument('--namespace', default=DEFAULT_NAMESPACE, help="Namespace to filter pipelines from")
    parser.add_argument('--batch-size', type=int, default=20,
                    help="Number of pipelines to fetch per API call (KFP page_size). Defaults to 20.")
    parser.add_argument('--no-pipeline-name-prefix', action='store_true',
                        help="Disable prefixing pipeline name to version names")
    
    return parser.parse_args()

# Fetch all pipelines from the KFP API
def fetch_pipelines(kfp_server_host, headers, verify, namespace, batch_size):
    pipelines = []
    page_token = ""

    while True:
        url = (
            f"{kfp_server_host}/apis/v2beta1/pipelines?"
            f"namespace={namespace}&page_size={batch_size}"
        )
        if page_token:
            url += f"&page_token={page_token}"

        response = requests.get(url, headers=headers, verify=verify, timeout=REQUEST_TIMEOUT)
        if response.status_code != 200:
            raise Exception(f"Error fetching pipelines: {response.status_code} - {response.text}")
        
        response_data = response.json()
        pipelines.extend(response_data.get("pipelines", []))     
        page_token = response_data.get("next_page_token")
        if not page_token:
            break

    return pipelines

# Fetch all versions for a given pipeline
def fetch_pipeline_versions(kfp_server_host, pipeline_id, headers, verify, namespace, batch_size):
    versions = []
    page_token = ""

    while True:
        url = (
            f"{kfp_server_host}/apis/v2beta1/pipelines/{pipeline_id}/versions?"
            f"sort_by=created_at&order_by=asc&namespace={namespace}&page_size={batch_size}"
        )
        if page_token:
            url += f"&page_token={page_token}"

        response = requests.get(url, headers=headers, verify=verify, timeout=REQUEST_TIMEOUT)
        if response.status_code != 200:
            raise Exception(f"Error fetching versions for pipeline {pipeline_id}: {response.status_code} - {response.text}")
        
        response_data = response.json()
        versions.extend(response_data.get("pipeline_versions", [])) 
        page_token = response_data.get("next_page_token")
        if not page_token:
            break

    return versions

# Creates a Kubernetes-safe name for a pipeline version by combining the pipeline name and version ID.
# If the name is too long, it shortens it and adds index at the end to keep it unique.
def get_version_name(pipeline_name, version_display_name, index, add_prefix):
    if add_prefix:
        original_version_name = f"{pipeline_name}-{version_display_name}"
    else:
        original_version_name = f"{version_display_name}"    
    
    # Clean and convert the pipeline name to be k8s-compatible
    formatted_name = utils.maybe_rename_for_k8s(original_version_name)

    if len(formatted_name) > MAX_NAME_LENGTH:
        suffix = f"-{index}"
        max_name_len = MAX_NAME_LENGTH - len(suffix)
        trimmed_name = original_version_name[:max_name_len]
        formatted_name = utils.maybe_rename_for_k8s(trimmed_name + suffix)

    return formatted_name

# Convert a pipeline and its versions into k8s format
def convert_to_k8s_format(pipeline, pipeline_versions, add_prefix, namespace):
    k8s_objects = []

    # Use namespace from pipeline object if available; otherwise fallback to default
    namespace = pipeline.get('namespace') or namespace
    original_id = pipeline.get('pipeline_id')
    display_name = pipeline.get('display_name')

    # Clean and convert the pipeline name to be k8s-compatible
    pipeline_name = utils.maybe_rename_for_k8s(pipeline.get('name', 'unnamed'))
    versions = pipeline_versions

    # Create the Pipeline object
    pipeline_obj = {
        "apiVersion": K8S_PIPELINE_API_VERSION,
        "kind": "Pipeline",
        "metadata": {
            "name": pipeline_name,
            "namespace": namespace,
            "annotations": {
                "pipelines.kubeflow.org/original-id": original_id,
            },
            "spec": {
                "displayName": display_name
            }
        }
    }
    k8s_objects.append(pipeline_obj)

     # Create a PipelineVersion object for each version
    for i, version in enumerate(versions):
        version_name = version.get("name", f"v{i}")
        version_display_name = version.get("display_name", version_name)
        pipeline_version_name = get_version_name(pipeline_name, version_name, i, add_prefix)
        platform_spec = None
        pipeline_spec = version.get("pipeline_spec", {})

        # When a pipeline has a platform spec, pipeline_spec is a dictionary with "pipeline_spec" and "platform_spec"
        # keys.
        if 'pipeline_spec' in pipeline_spec:
            platform_spec = pipeline_spec.get("platform_spec", {})
            pipeline_spec = pipeline_spec.get("pipeline_spec", {})

        pipeline_version_id = version.get("pipeline_version_id")
        
        pipeline_version_obj = {
            "apiVersion": K8S_PIPELINE_API_VERSION,
            "kind": "PipelineVersion",
            "metadata": {
                "name": pipeline_version_name,
                "namespace": namespace,
                "annotations": {
                    "pipelines.kubeflow.org/original-id": pipeline_version_id,
                }
            },
            "spec": {
                "pipelineName": pipeline_name,
                "pipelineSpec": pipeline_spec,
                "displayName": version_display_name,
            }
        }

        if platform_spec:
            pipeline_version_obj["spec"]["platformSpec"] = platform_spec

        k8s_objects.append(pipeline_version_obj)
   
    return pipeline_name, k8s_objects

# Write all collected Kubernetes objects to a seperate YAML file for each pipeline and its versions
def write_pipeline_yaml(pipeline_name, k8s_objects, output_dir):
    output_dir = Path(output_dir)
    output_dir.mkdir(parents=True, exist_ok=True)
    output_path = output_dir / f"{pipeline_name}.yaml"
    with output_path.open('w') as f:
        yaml.dump_all(k8s_objects, f, sort_keys=False)
    print(f"Wrote pipeline '{pipeline_name}' to {output_path}")

def migrate():
    args = parse_args()
    headers = {"Content-Type": "application/json"}
    if args.token:
        headers["Authorization"] = f"Bearer {args.token}"
    verify = args.ca_bundle if args.ca_bundle else True

    try:
        all_objects = []
        pipelines = fetch_pipelines(args.kfp_server_host, headers, verify, args.namespace, args.batch_size)
        for pipeline in pipelines:
            print(f"Processing pipeline: {pipeline['display_name']}")
            versions = fetch_pipeline_versions(args.kfp_server_host, pipeline["pipeline_id"], headers, verify, args.namespace, args.batch_size)
            pipeline_name, k8s_objs = convert_to_k8s_format(pipeline, versions, args.no_pipeline_name_prefix, args.namespace)
            all_objects.extend(k8s_objs)
            write_pipeline_yaml(pipeline_name, k8s_objs, args.output)
    except Exception as e:
        print(f"Migration failed: {e}")

if __name__ == '__main__':    
    migrate()
