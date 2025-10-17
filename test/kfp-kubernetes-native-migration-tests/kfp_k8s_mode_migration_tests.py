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

"""
Kubeflow Pipelines Kubernetes Native Mode Migration Tests

These tests verify that migrated resources work correctly in KFP Kubernetes native mode.
They test the end-to-end migration flow from database mode to K8s native mode.

"""

import json
import os
import pickle
import tempfile
import subprocess
import requests
from kfp.client import Client
import pytest
import yaml
from pathlib import Path
from typing import Dict, List, Any, Optional
from requests.adapters import HTTPAdapter
from urllib3.util.retry import Retry
from create_test_pipelines import to_json_for_comparison
from k8s_models import K8sMetadata, K8sPipeline, K8sPipelineVersion

KFP_ENDPOINT = os.environ.get('KFP_ENDPOINT', 'http://localhost:8888')


@pytest.fixture(scope="session")
def kfp_client():
    """Create a KFP client for testing K8s native mode."""
    return Client(host=KFP_ENDPOINT)


@pytest.fixture(scope="session")
def api_base():
    """KFP API base URL for direct HTTP requests."""
    return f"{KFP_ENDPOINT}/apis/v2beta1"


@pytest.fixture(scope="session")
def test_data():
    """Load test data created in database mode before migration."""
    test_data_file = Path("migration_test_data.pkl")
    if test_data_file.exists():
        with open(test_data_file, "rb") as f:
            return pickle.load(f)
    else:
        return {"pipelines": [], "experiments": [], "runs": [], "recurring_runs": []}


def get_k8s_pipelines() -> List[str]:
    """Get list of Pipeline resources from Kubernetes cluster."""
    result = subprocess.run([
        'kubectl', 'get', 'pipeline', '-n', 'kubeflow', '-o', 'jsonpath={.items[*].metadata.name}'
    ], check=True, capture_output=True, text=True)
    
    return result.stdout.strip().split() if result.stdout.strip() else []


def get_migrated_pipelines() -> List[str]:
    """Get list of migrated Pipeline resources."""
    pipeline_names = get_k8s_pipelines()
    migrated_pipelines = []
    
    for pipeline_name in pipeline_names:
        annotation_result = subprocess.run([
            'kubectl', 'get', 'pipeline', pipeline_name, '-n', 'kubeflow', 
            '-o', r'jsonpath={.metadata.annotations.pipelines\.kubeflow\.org/original-id}'
        ], capture_output=True, text=True)
        
        if annotation_result.stdout.strip():
            migrated_pipelines.append(pipeline_name)
    
    return migrated_pipelines


def get_pipeline_details(pipeline_name: str) -> Dict[str, Any]:
    """Get detailed information about a specific pipeline from Kubernetes."""
    result = subprocess.run([
        'kubectl', 'get', 'pipeline', pipeline_name, '-n', 'kubeflow', '-o', 'json'
    ], check=True, capture_output=True, text=True)
    
    return json.loads(result.stdout)


def _to_metadata(meta: Dict[str, Any]) -> K8sMetadata:
    return K8sMetadata(
        name=meta.get('name', ''),
        namespace=meta.get('namespace', ''),
        annotations=meta.get('annotations', {}) or {},
        labels=meta.get('labels', {}) or {},
        creationTimestamp=meta.get('creationTimestamp'),
        uid=meta.get('uid')
    )


def get_pipeline_resource(pipeline_name: str) -> K8sPipeline:    
    raw = get_pipeline_details(pipeline_name)
    return K8sPipeline(
        apiVersion=raw.get('apiVersion', ''),
        kind=raw.get('kind', 'Pipeline'),
        metadata=_to_metadata(raw.get('metadata', {})),
        spec=raw.get('spec', {}) or {}
    )


def list_pipeline_versions_objects() -> List[K8sPipelineVersion]:    
    result = subprocess.run([
        'kubectl', 'get', 'pipelineversion', '-n', 'kubeflow', '-o', 'json'
    ], check=True, capture_output=True, text=True)
    data = json.loads(result.stdout)
    items = data.get('items', [])
    versions: List[K8sPipelineVersion] = []
    for item in items:
        versions.append(
            K8sPipelineVersion(
                apiVersion=item.get('apiVersion', ''),
                kind=item.get('kind', 'PipelineVersion'),
                metadata=_to_metadata(item.get('metadata', {})),
                spec=item.get('spec', {}) or {}
            )
        )
    return versions


def get_pipeline_versions_for_pipeline(pipeline_name: str) -> List[K8sPipelineVersion]:
    """Filter PipelineVersion objects belonging to a given Pipeline name."""
    all_versions = list_pipeline_versions_objects()
    filtered: List[K8sPipelineVersion] = []
    for v in all_versions:
        if v.get_pipeline_name() == pipeline_name:
            filtered.append(v)
    return filtered
   
def test_k8s_mode_pipeline_execution(kfp_client):
    """Test that migrated pipelines are available and executable in K8s native mode.
    
    Validates migrated pipelines exist as Kubernetes Pipeline resources with original-id annotations.
    Tests pipeline discovery via KFP client API and verifies runs can be created and executed.
    Ensures run details match expected structure and contain proper metadata.   
    """
    # Get migrated pipelines from Kubernetes
    migrated_pipelines = get_migrated_pipelines()
    assert len(migrated_pipelines) > 0, "Should have at least one migrated pipeline in K8s mode"
    
    print(f"Found {len(migrated_pipelines)} migrated pipelines: {migrated_pipelines}")
   
    # Use the 'hello-world' pipeline explicitly for this test
    test_pipeline_name = 'hello-world'
   
    migrated_pipeline_obj = get_pipeline_resource(test_pipeline_name)
    assert migrated_pipeline_obj.get_name() == test_pipeline_name
    assert migrated_pipeline_obj.get_original_id() is not None and migrated_pipeline_obj.get_original_id() != "", \
        "Migrated pipeline should have original-id annotation"
    
    k8s_versions = get_pipeline_versions_for_pipeline(test_pipeline_name)
    assert len(k8s_versions) > 0, f"Pipeline {test_pipeline_name} should have at least one PipelineVersion"
    first_version = k8s_versions[0]
    assert first_version.get_original_id() is not None and first_version.get_original_id() != "", \
        "PipelineVersion should have original-id annotation"
    spec = first_version.get_pipeline_spec()
    assert spec is not None and 'pipelineInfo' in spec, "PipelineVersion should contain pipelineSpec with pipelineInfo"
    
    # Test pipeline execution
    experiment = kfp_client.create_experiment(
        name="k8s-execution-test-experiment",
        description="Test experiment for K8s mode pipeline execution"
    )
    experiment_id = experiment.experiment_id
    assert experiment_id is not None, "Experiment should be created successfully"
    
    # Get pipeline
    pipelines = kfp_client.list_pipelines()
    pipeline_list = pipelines.pipelines if hasattr(pipelines, 'pipelines') else []
    
    migrated_pipeline = None
    for pipeline in pipeline_list:
        pipeline_name = getattr(pipeline, 'display_name', None)
        if pipeline_name == test_pipeline_name:
            migrated_pipeline = pipeline
            break
    
    assert migrated_pipeline is not None, f"Pipeline {test_pipeline_name} should be discoverable via KFP client"
    
    pipeline_id = migrated_pipeline.pipeline_id
    assert pipeline_id is not None, "Pipeline should have an ID"
    
    # Get pipeline versions
    versions = kfp_client.list_pipeline_versions(pipeline_id=pipeline_id)
    version_list = versions.pipeline_versions if hasattr(versions, 'pipeline_versions') else []
    assert len(version_list) > 0, "Pipeline should have at least one version"

    # Get the version whose spec indicates the hello-world pipeline
    def _is_hello_world_version(v):
        v_spec = getattr(v, 'pipeline_spec', None)
        if isinstance(v_spec, dict):
            info = v_spec.get('pipelineInfo')
            if isinstance(info, dict):
                return info.get('name') == 'hello-world'
        return False

    migrated_version = next((v for v in version_list if _is_hello_world_version(v)), version_list[0])
    version_id = migrated_version.pipeline_version_id
    assert version_id is not None, "Pipeline version should have an ID"
    
    # Create and execute a run
    test_params = {}
    run_data = kfp_client.run_pipeline(
        experiment_id=experiment_id,
        job_name="k8s-execution-test-run",
        pipeline_id=pipeline_id,
        version_id=version_id,
        params=test_params
    )
    
    run_id = run_data.run_id
    run_name = run_data.display_name or "k8s-execution-test-run"
    
    assert run_id is not None, "Run should be created successfully"
    print(f"Successfully created and started run: {run_name} (ID: {run_id})")
    
    # Validate that run was created and can be retrieved
    run_details = kfp_client.get_run(run_id=run_id)
    assert run_details.run_id == run_id, "Retrieved run should have correct ID"
    assert run_details.experiment_id == experiment_id, "Run should be associated with correct experiment"
    

def test_k8s_mode_duplicate_pipeline_creation():
    """Test duplicate pipeline name handling in K8s native mode.
    
    Validates existing pipelines are properly managed in K8s native mode.
    Tests that attempting to create duplicate pipeline names is handled correctly.
    Verifies Kubernetes resource uniqueness constraints work and pipelines maintain proper metadata.
    """
    pipeline_names = get_k8s_pipelines()
    assert len(pipeline_names) > 0, "Should have at least one pipeline in cluster"
    
    print(f"Found {len(pipeline_names)} pipelines in cluster: {pipeline_names}")
    
    existing_pipeline_name = pipeline_names[0]
    print(f"Testing duplicate creation for pipeline: {existing_pipeline_name}")
    
    # Get original pipeline details
    original_pipeline = get_pipeline_details(existing_pipeline_name)
    original_creation_time = original_pipeline['metadata']['creationTimestamp']
    original_uid = original_pipeline['metadata']['uid']
    
    # Create duplicate pipeline YAML
    duplicate_pipeline_data = {
        "apiVersion": "pipelines.kubeflow.org/v2beta1",
        "kind": "Pipeline",
        "metadata": {
            "name": existing_pipeline_name,
            "namespace": "kubeflow"
        },
        "spec": {
            "description": "Duplicate pipeline test"
        }
    }
    
    # Apply duplicate pipeline
    with tempfile.NamedTemporaryFile(mode='w', suffix='.yaml', delete=False) as f:
        yaml.dump(duplicate_pipeline_data, f)
        temp_file = f.name
    
    result = subprocess.run([
        'kubectl', 'apply', '-f', temp_file
    ], capture_output=True, text=True)
    
    print(f"kubectl apply result: {result.returncode}, stdout: {result.stdout}, stderr: {result.stderr}")
    
    # Verify pipeline still exists and is unique
    updated_pipeline = get_pipeline_details(existing_pipeline_name)
    
    # Check that there's still only one pipeline with this name
    all_pipelines = get_k8s_pipelines()
    name_count = all_pipelines.count(existing_pipeline_name)
    assert name_count == 1, f"Should have exactly 1 pipeline named {existing_pipeline_name}, but found {name_count}"
    
    # Verify the pipeline wasn't replaced
    assert updated_pipeline['metadata']['uid'] == original_uid, \
        "Pipeline UID should remain the same (not replaced)"
    assert updated_pipeline['metadata']['creationTimestamp'] == original_creation_time, \
        "Pipeline creation time should remain the same (not replaced)"
    
    print(f"Duplicate pipeline handling works correctly - {existing_pipeline_name} remains unique")
    
    # Clean up temp file
    os.unlink(temp_file)

def test_k8s_mode_experiment_creation(kfp_client):
    """Test experiment and run creation in K8s native mode after migration.
    
    Validates new experiments can be created in K8s native mode with proper structure and metadata.
    Tests runs can be created against migrated pipelines.
    Verifies complete experiment/pipeline/run relationship works end-to-end.
    """
    # Create experiment
    experiment = kfp_client.create_experiment(
        name="k8s-mode-test-experiment",
        description="Test experiment created in K8s mode"
    )
    
    experiment_id = experiment.experiment_id
    experiment_name = experiment.display_name
    
    assert experiment_id is not None, "Experiment should have an ID"
    assert experiment_name == "k8s-mode-test-experiment", "Experiment name should match input"
    
    print(f"Created experiment: {experiment_name} (ID: {experiment_id})")
    
    # Find simple pipeline for testing
    pipelines = kfp_client.list_pipelines()
    pipeline_list = pipelines.pipelines if hasattr(pipelines, 'pipelines') else []
    print(f"Available pipelines: {[p.display_name for p in pipeline_list]}")
    
    target_pipeline = None
    for pipeline in pipeline_list:
        pipeline_name = getattr(pipeline, 'display_name', None)
        if pipeline_name == "hello-world":
            target_pipeline = pipeline
            break
    
    assert target_pipeline is not None, "Should find hello-world pipeline for testing"
    
    k8s_pipeline_obj = get_pipeline_resource("hello-world")
    assert k8s_pipeline_obj.get_original_id() is not None and k8s_pipeline_obj.get_original_id() != "", \
        "hello-world pipeline should carry original-id annotation"
  
    hv_versions = get_pipeline_versions_for_pipeline("hello-world")
    assert len(hv_versions) > 0, "hello-world should have at least one PipelineVersion in K8s mode"
    hv_first = hv_versions[0]
    assert hv_first.get_original_id() is not None and hv_first.get_original_id() != "", \
        "hello-world PipelineVersion should carry original-id annotation"
    hv_spec = hv_first.get_pipeline_spec()
    assert hv_spec is not None and 'pipelineInfo' in hv_spec, "hello-world PipelineVersion should contain pipelineSpec with pipelineInfo"
    
    pipeline_id = target_pipeline.pipeline_id
    assert pipeline_id is not None, "Pipeline should have an ID"
    
    # Get pipeline versions
    versions = kfp_client.list_pipeline_versions(pipeline_id=pipeline_id)
    version_list = versions.pipeline_versions if hasattr(versions, 'pipeline_versions') else []
    assert len(version_list) > 0, "Pipeline should have at least one version"
    
    version = version_list[0]
    version_id = version.pipeline_version_id
    assert version_id is not None, "Pipeline version should have an ID"
    
    # Create run
    run_data = kfp_client.run_pipeline(
        experiment_id=experiment_id,
        job_name="k8s-mode-test-run",
        pipeline_id=pipeline_id,
        version_id=version_id
    )
    
    run_id = run_data.run_id
    run_name = run_data.display_name
    
    assert run_id is not None, "Run should be created successfully"
    assert run_name == "k8s-mode-test-run", "Run name should match input"
    
    print(f"Created run: {run_name} (ID: {run_id})")
    
    # Validate experiment structure
    experiment_details = kfp_client.get_experiment(experiment_id=experiment_id)
    assert experiment_details.experiment_id == experiment_id, "Experiment should have correct ID"
    assert experiment_details.display_name == experiment_name, "Experiment should have correct name"
    experiment_description = getattr(experiment_details, 'description', None)
    assert experiment_description == "Test experiment created in K8s mode", "Experiment should have correct description"
    
    # Validate run structure
    run_details = kfp_client.get_run(run_id=run_id)
    assert run_details.run_id == run_id, "Run should have correct ID"
    assert run_details.display_name == run_name, "Run should have correct name"
    assert run_details.experiment_id == experiment_id, "Run should be associated with correct experiment"
      
    run_state = getattr(run_details, 'state', None)   
    assert run_state is not None, "Run should have state information"
            
def test_k8s_mode_recurring_run_continuation(api_base, test_data):
    """Test recurring run continuity after migration to K8s native mode.
    
    Validates recurring runs created in database mode still exist in K8s native mode.
    Tests recurring run metadata (name, ID, schedule) is preserved with correct cron schedules.
    Verifies API endpoints continue to work for existing recurring runs.
    """
    if not test_data.get("recurring_runs"):
        print("Note: No recurring runs in test data to validate - skipping test")
        return
    
    original_recurring_run = test_data["recurring_runs"][0]
    recurring_run_name = getattr(original_recurring_run, 'display_name', None)
    recurring_run_id = getattr(original_recurring_run, 'recurring_run_id', None)
    
    print(f"Testing recurring run continuity: {recurring_run_name} (ID: {recurring_run_id})")
    
    # Check if the recurring run still exists via API
    response = requests.get(
        f"{api_base}/recurringruns/{recurring_run_id}",
        headers={"Content-Type": "application/json"}
    )
    
    response.raise_for_status()
    recurring_run = response.json()
    
    # Extract current run details
    current_run_name = recurring_run.get('display_name') or recurring_run.get('name')
    current_run_id = recurring_run.get('recurring_run_id') or recurring_run.get('id')
    
    print(f"Recurring run found in K8s mode: {current_run_name} (ID: {current_run_id})")
    
    # Validate recurring run identity preservation
    assert current_run_name == recurring_run_name, \
        f"Recurring run name should be preserved: expected {recurring_run_name}, got {current_run_name}"
    assert current_run_id == recurring_run_id, \
        f"Recurring run ID should be preserved: expected {recurring_run_id}, got {current_run_id}"
    
    # Validate cron schedule preservation
    original_cron = getattr(getattr(getattr(original_recurring_run, 'trigger', {}), 'cron_schedule', {}), 'cron', None)
    current_cron = recurring_run.get('trigger', {}).get('cron_schedule', {}).get('cron')
    
    # Validate cron schedule preservation
    assert current_cron == original_cron, f"Cron schedule should be preserved: expected={original_cron}, got={current_cron}"
    print(f"Cron schedule preserved: {current_cron}")
    
    # Validate recurring run structure matches original format
    if hasattr(recurring_run, 'keys'):
        current_structure_keys = set(recurring_run.keys())
        # Check that key fields are preserved
        key_fields = {'id', 'name', 'trigger'}
        for field in key_fields:
            field_exists = (field in current_structure_keys or 
                          any(field in k for k in current_structure_keys))
            assert field_exists, f"Key field {field} should be preserved in recurring run structure"
   
    original_json = to_json_for_comparison(original_recurring_run)
    original_object = original_recurring_run.to_dict()    
    
    # Validate recurring run name preservation
    original_name = original_object.get('display_name')
    if original_name and current_run_name:
        assert current_run_name == original_name, \
            f"Recurring run name should be preserved: expected={original_name}, got={current_run_name}"
    
    # Validate trigger/schedule preservation
    if 'trigger' in original_object and original_object['trigger']:
        current_trigger = recurring_run.get('trigger', {})
        original_trigger = original_object.get('trigger')
        
        # Check cron schedule preservation
        if ((hasattr(original_trigger, 'get') and 'cron_schedule' in original_trigger) or 
            (hasattr(original_trigger, 'cron_schedule') and getattr(original_trigger, 'cron_schedule', None) is not None)):
            assert 'cron_schedule' in current_trigger, "Cron schedule should be preserved"
            original_cron = original_trigger['cron_schedule'].get('cron') if hasattr(original_trigger['cron_schedule'], 'get') else None
            current_cron = current_trigger['cron_schedule'].get('cron') if hasattr(current_trigger['cron_schedule'], 'get') else None
            assert current_cron == original_cron, \
                f"Cron schedule should match: expected={original_cron}, got={current_cron}"
    
    # Validate status and enabled state if available
    if 'enabled' in original_object and original_object['enabled'] is not None:
        current_enabled = recurring_run.get('enabled')
        if current_enabled is not None:
            original_enabled = original_object.get('enabled')
            assert current_enabled == original_enabled, \
                "Recurring run enabled state should be preserved"
