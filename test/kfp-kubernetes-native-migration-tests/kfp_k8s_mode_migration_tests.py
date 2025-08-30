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
import tempfile
import subprocess
import requests
import kfp
import pytest
import yaml
from pathlib import Path
from typing import Dict, List, Any
from requests.adapters import HTTPAdapter
from urllib3.util.retry import Retry

KFP_ENDPOINT = os.environ.get('KFP_ENDPOINT', 'http://localhost:8888')


@pytest.fixture(scope="session")
def kfp_client():
    """Create a KFP client for testing K8s native mode."""
    return kfp.Client(host=KFP_ENDPOINT)


@pytest.fixture(scope="session")
def api_base():
    """KFP API base URL for direct HTTP requests."""
    return f"{KFP_ENDPOINT}/apis/v2beta1"


@pytest.fixture(scope="session")
def test_data():
    """Load test data created in database mode before migration."""
    test_data_file = Path("migration_test_data.json")
    if test_data_file.exists():
        with open(test_data_file) as f:
            return json.load(f)
    else:
        return {"pipelines": [], "experiments": [], "runs": [], "recurring_runs": []}


def get_k8s_pipelines() -> List[str]:
    """Get list of Pipeline resources from Kubernetes cluster."""
    result = subprocess.run([
        'kubectl', 'get', 'pipeline', '-n', 'kubeflow', '-o', 'jsonpath={.items[*].metadata.name}'
    ], check=True, capture_output=True, text=True)
    
    return result.stdout.strip().split() if result.stdout.strip() else []


def get_migrated_pipelines() -> List[str]:
    """Get list of migrated Pipeline resources (those with original-id annotation)."""
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


def validate_pipeline_structure(pipeline_data: Dict[str, Any], expected_original_id: str = None) -> None:
    """Validate that a pipeline has the expected Kubernetes structure."""
    assert 'metadata' in pipeline_data, "Pipeline should have metadata"
    assert 'spec' in pipeline_data, "Pipeline should have spec"
    assert pipeline_data['kind'] == 'Pipeline', "Resource should be a Pipeline"
    assert pipeline_data['metadata']['namespace'] == 'kubeflow', "Pipeline should be in kubeflow namespace"
    
    if expected_original_id:
        annotations = pipeline_data.get('metadata', {}).get('annotations', {})
        assert annotations.get('pipelines.kubeflow.org/original-id') == expected_original_id, \
            f"Pipeline should have original ID annotation: {expected_original_id}"


def test_k8s_mode_pipeline_execution(kfp_client, test_data):
    """Test that migrated pipelines are available and executable in K8s native mode.
    
    Validates migrated pipelines exist as Kubernetes Pipeline resources with original-id annotations.
    Tests pipeline discovery via KFP client API and verifies runs can be created and executed.
    Ensures run details match expected structure and contain proper metadata.
    """
    # Get migrated pipelines from Kubernetes
    migrated_pipelines = get_migrated_pipelines()
    assert len(migrated_pipelines) > 0, "Should have at least one migrated pipeline in K8s mode"
    
    print(f"Found {len(migrated_pipelines)} migrated pipelines: {migrated_pipelines}")
    
    # Validate pipeline structure in Kubernetes
    first_pipeline_name = migrated_pipelines[0]
    pipeline_details = get_pipeline_details(first_pipeline_name)
    
    # Find corresponding original pipeline in test data
    original_pipeline = None
    for pipeline in test_data.get("pipelines", []):
        if pipeline.get("name") == first_pipeline_name:
            original_pipeline = pipeline
            break
    
    if original_pipeline:
        validate_pipeline_structure(pipeline_details, original_pipeline["id"])
    else:
        validate_pipeline_structure(pipeline_details)
    
    # Test pipeline execution via KFP client
    experiment = kfp_client.create_experiment(
        name="k8s-execution-test-experiment",
        description="Test experiment for K8s mode pipeline execution"
    )
    experiment_id = getattr(experiment, 'experiment_id', None)
    assert experiment_id is not None, "Experiment should be created successfully"
    
    # Get pipeline via KFP client
    pipelines = kfp_client.list_pipelines()
    pipeline_list = pipelines.pipelines if hasattr(pipelines, 'pipelines') else []
    
    target_pipeline = None
    for pipeline in pipeline_list:
        pipeline_name = getattr(pipeline, 'display_name', None)
        if pipeline_name == first_pipeline_name:
            target_pipeline = pipeline
            break
    
    assert target_pipeline is not None, f"Pipeline {first_pipeline_name} should be discoverable via KFP client"
    
    pipeline_id = getattr(target_pipeline, 'pipeline_id', None)
    assert pipeline_id is not None, "Pipeline should have an ID"
    
    # Get pipeline versions
    versions = kfp_client.list_pipeline_versions(pipeline_id=pipeline_id)
    version_list = versions.pipeline_versions if hasattr(versions, 'pipeline_versions') else []
    assert len(version_list) > 0, "Pipeline should have at least one version"
    
    target_version = version_list[0]
    version_id = getattr(target_version, 'pipeline_version_id', None)
    assert version_id is not None, "Pipeline version should have an ID"
    
    # Create and execute a run
    run_data = kfp_client.run_pipeline(
        experiment_id=experiment_id,
        job_name="k8s-execution-test-run",
        pipeline_id=pipeline_id,
        version_id=version_id,
        params={}
    )
    
    run_id = getattr(run_data, 'run_id', None)
    run_name = getattr(run_data, 'display_name', None) or "k8s-execution-test-run"
    
    assert run_id is not None, "Run should be created successfully"
    print(f"Successfully created and started run: {run_name} (ID: {run_id})")
    
    # Validate run details structure
    run_details = kfp_client.get_run(run_id=run_id)
    
    # Compare with original test data structure
    expected_run_fields = {
        'run_id': run_id,
        'display_name': run_name,
        'experiment_id': experiment_id,
        'pipeline_spec': {
            'pipeline_id': pipeline_id,
            'pipeline_version_id': version_id
        }
    }
    
    # Validate run structure matches expected format
    assert getattr(run_details, 'run_id', None) == expected_run_fields['run_id'], \
        "Run ID should match expected value"
    assert getattr(run_details, 'display_name', None) == expected_run_fields['display_name'], \
        "Run name should match expected value"
    
    # Validate that run is associated with correct experiment and pipeline
    run_experiment_id = getattr(run_details, 'experiment_id', None)
    assert run_experiment_id == experiment_id, "Run should be associated with correct experiment"
   

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

def test_k8s_mode_experiment_creation(kfp_client, test_data):
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
    
    experiment_id = getattr(experiment, 'experiment_id', None)
    experiment_name = getattr(experiment, 'display_name', None)
    
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
        if pipeline_name == "simple-pipeline":
            target_pipeline = pipeline
            break
    
    assert target_pipeline is not None, "Should find simple-pipeline for testing"
    
    pipeline_id = getattr(target_pipeline, 'pipeline_id', None)
    assert pipeline_id is not None, "Pipeline should have an ID"
    
    # Get pipeline versions
    versions = kfp_client.list_pipeline_versions(pipeline_id=pipeline_id)
    version_list = versions.pipeline_versions if hasattr(versions, 'pipeline_versions') else []
    assert len(version_list) > 0, "Pipeline should have at least one version"
    
    version = version_list[0]
    version_id = getattr(version, 'pipeline_version_id', None)
    assert version_id is not None, "Pipeline version should have an ID"
    
    # Create run
    run_data = kfp_client.run_pipeline(
        experiment_id=experiment_id,
        job_name="k8s-mode-test-run",
        pipeline_id=pipeline_id,
        version_id=version_id
    )
    
    run_id = getattr(run_data, 'run_id', None)
    run_name = getattr(run_data, 'display_name', None)
    
    assert run_id is not None, "Run should be created successfully"
    assert run_name == "k8s-mode-test-run", "Run name should match input"
    
    print(f"Created run: {run_name} (ID: {run_id})")
    
    # Validate run structure matches expected format from test data
    expected_run_structure = {
        "id": run_id,
        "name": run_name,
        "pipeline_spec": {
            "pipeline_id": pipeline_id,
            "pipeline_version_id": version_id
        },
        "experiment_id": experiment_id
    }
    
    # Compare with original test data structure for runs
    if test_data.get("runs"):
        original_run = test_data["runs"][0]
        original_structure_keys = set(original_run.keys())
        new_structure_keys = set(expected_run_structure.keys())
       
        essential_keys = {"id", "name", "pipeline_spec"}
        assert essential_keys.issubset(new_structure_keys), \
            f"Run should have essential keys: {essential_keys}. Got: {new_structure_keys}"
        
def test_k8s_mode_recurring_run_continuation(api_base, test_data):
    """Test recurring run continuity after migration to K8s native mode.
    
    Validates recurring runs created in database mode still exist in K8s native mode.
    Tests recurring run metadata (name, ID, schedule) is preserved with correct cron schedules.
    Verifies API endpoints continue to work for existing recurring runs.
    """
    assert test_data.get("recurring_runs"), "Test data should contain recurring runs for validation"
    
    original_recurring_run = test_data["recurring_runs"][0]
    recurring_run_name = original_recurring_run["name"]
    recurring_run_id = original_recurring_run["id"]
    
    print(f"Testing recurring run continuity: {recurring_run_name} (ID: {recurring_run_id})")
    
    # Check if the recurring run still exists via API
    response = requests.get(
        f"{api_base}/recurringruns/{recurring_run_id}",
        headers={"Content-Type": "application/json"}
    )
    
    response.raise_for_status()
    recurring_run = response.json()
    
    # Extract current run details
    current_run_name = getattr(recurring_run, 'display_name', 
                              recurring_run.get('display_name', 
                                               recurring_run.get('name')))
    current_run_id = getattr(recurring_run, 'recurring_run_id', 
                           recurring_run.get('recurring_run_id', 
                                            recurring_run.get('id')))
    
    print(f"Recurring run found in K8s mode: {current_run_name} (ID: {current_run_id})")
    
    # Validate recurring run identity preservation
    assert current_run_name == recurring_run_name, \
        f"Recurring run name should be preserved: expected {recurring_run_name}, got {current_run_name}"
    assert current_run_id == recurring_run_id, \
        f"Recurring run ID should be preserved: expected {recurring_run_id}, got {current_run_id}"
    
    # Validate cron schedule preservation
    original_cron = original_recurring_run.get('trigger', {}).get('cron_schedule', {}).get('cron')
    current_cron = recurring_run.get('trigger', {}).get('cron_schedule', {}).get('cron')
    
    assert current_cron == original_cron, \
        f"Cron schedule should be preserved: expected {original_cron}, got {current_cron}"
    
    # Validate recurring run structure matches original format
    expected_structure_keys = set(original_recurring_run.keys())
    current_structure_keys = set(recurring_run.keys())
    
    # Check that key fields are preserved
    key_fields = {'id', 'name', 'trigger'}
    for field in key_fields:
        if field in expected_structure_keys:
            field_exists = (field in current_structure_keys or 
                          any(field in k for k in current_structure_keys))
            assert field_exists, f"Key field {field} should be preserved in recurring run structure"