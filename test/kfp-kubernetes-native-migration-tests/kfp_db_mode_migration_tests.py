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
Kubeflow Pipelines Database Mode Migration Tests

These tests verify the migration script that exports KFP resources from database mode
to Kubernetes native format
"""

import json
import os
import pickle
import sys
import subprocess
import requests
import pytest
import yaml
from pathlib import Path
from unittest.mock import patch
from typing import Dict, List, Any

# Add the tools directory to path to import migration module
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '../../tools/k8s-native'))

from migration import migrate

# Import serialization function from create_test_pipelines
sys.path.insert(0, os.path.dirname(__file__))
from create_test_pipelines import to_json_for_comparison

KFP_ENDPOINT = os.environ.get('KFP_ENDPOINT', 'http://localhost:8888')


@pytest.fixture(scope="session")
def test_data():
    """Load test data created by create_test_pipelines.py.
    
    """
    test_data_file = Path("migration_test_data.pkl")
    if test_data_file.exists():
        with open(test_data_file, "rb") as f:
            return pickle.load(f)
    else:
        pytest.skip("Test data file not found. Run create_test_pipelines.py first when KFP server is available.")

@pytest.fixture(scope="function")
def migration_output_dir(request):
    """Create a unique output directory for each test's migration results.
    
    This directory is shared with K8s mode tests to apply migrated resources.
    """
    # Use a shared persistent directory that K8s mode tests can access
    shared_migration_base = Path("/tmp/kfp_shared_migration_outputs")
    shared_migration_base.mkdir(exist_ok=True)
    
    output_dir = shared_migration_base / f"migration_output_{request.node.name}"
    output_dir.mkdir(exist_ok=True)
 
    # Write the migration output directory to a shared location for K8s mode tests
    migration_info_file = Path("/tmp/kfp_migration_output_dir.txt")
    with open(migration_info_file, "w") as f:
        f.write(str(output_dir))
    
    yield output_dir  

@pytest.fixture
def run_migration(migration_output_dir):
    """Execute the migration script and return the output directory
    containing the generated YAML files.
    
    """
    with patch('sys.argv', [
        'migration.py',
        '--kfp-server-host', KFP_ENDPOINT,
        '--output', str(migration_output_dir),
        '--namespace', 'kubeflow'
    ]):
        migrate()
    
    return migration_output_dir

def parse_yaml_files(output_dir: Path) -> Dict[str, List[Dict[str, Any]]]:
    """Parse all YAML files in the output directory and group by kind."""
    resources = {"Pipeline": [], "PipelineVersion": [], "Experiment": [], "Run": [], "RecurringRun": []}
    
    for yaml_file in output_dir.glob("*.yaml"):
        with open(yaml_file) as f:
            docs = list(yaml.safe_load_all(f))
            for doc in docs:
                if doc and 'kind' in doc:
                    kind = doc['kind']
                    if kind in resources:
                        resources[kind].append(doc)
    
    return resources

def find_test_data_by_name(test_data: Dict[str, Any], resource_type: str, name: str):
    """Find a resource in test data by name."""
    resources = test_data.get(resource_type, [])
    for resource in resources:        
        resource_name = getattr(resource, 'display_name', None)
        if name in str(resource_name):
            return resource
    pytest.fail(f"Test data should contain {resource_type} with name containing '{name}'")


def compare_complete_objects(migrated_resource: Dict[str, Any], original_resource, resource_type: str) -> None:
    
    original_json = to_json_for_comparison(original_resource)
    original_object = original_resource.to_dict()  
    
    # Core validations based on resource type
    if resource_type == "Pipeline":
        # Validate pipeline-specific attributes
        original_name = original_object.get('display_name')
        migrated_name = migrated_resource.get('metadata', {}).get('name')
        assert migrated_name == original_name, \
            f"Pipeline name mismatch: migrated={migrated_name}, original={original_name}"
        
        # Validate pipeline ID preservation in annotations
        original_id = original_object.get('pipeline_id')
        annotations = migrated_resource.get('metadata', {}).get('annotations', {})
        assert annotations.get('pipelines.kubeflow.org/original-id') == original_id, \
            f"Pipeline ID should be preserved in annotations: {original_id}"
    
    elif resource_type == "PipelineVersion":
        # Validate pipeline version-specific attributes
        original_name = original_object.get('display_name')
        migrated_name = migrated_resource.get('metadata', {}).get('name')
        
        # Validate version ID preservation in annotations
        original_id = original_object.get('pipeline_version_id')
        annotations = migrated_resource.get('metadata', {}).get('annotations', {})
        assert annotations.get('pipelines.kubeflow.org/original-id') == original_id, \
            f"Pipeline version ID should be preserved in annotations: {original_id}"
        
        # Validate pipeline spec preservation
        if (hasattr(original_object, 'get') and 'pipeline_spec' in original_object) or (hasattr(original_object, 'pipeline_spec') and getattr(original_object, 'pipeline_spec', None) is not None):
            assert 'spec' in migrated_resource, "Migrated pipeline version should have spec"
            assert 'pipelineSpec' in migrated_resource['spec'], "Should have pipelineSpec in spec"
    
    elif resource_type == "Experiment":
        # Validate experiment-specific attributes
        original_name = original_object.get('display_name')
        migrated_name = migrated_resource.get('metadata', {}).get('name')
        assert migrated_name == original_name, \
            f"Experiment name mismatch: migrated={migrated_name}, original={original_name}"
        
        # Validate experiment ID preservation in annotations
        original_id = original_object.get('experiment_id')
        annotations = migrated_resource.get('metadata', {}).get('annotations', {})
        assert annotations.get('pipelines.kubeflow.org/original-id') == original_id, \
            f"Experiment ID should be preserved in annotations: {original_id}"
    
    elif resource_type == "Run":
        # Validate run-specific attributes
        original_name = original_object.get('display_name')
        migrated_name = migrated_resource.get('metadata', {}).get('name')
        
        # Validate run ID preservation in annotations
        original_id = original_object.get('run_id')
        annotations = migrated_resource.get('metadata', {}).get('annotations', {})
        assert annotations.get('pipelines.kubeflow.org/original-id') == original_id, \
            f"Run ID should be preserved in annotations: {original_id}"
    
    elif resource_type == "RecurringRun":
        # Validate recurring run-specific attributes
        original_name = original_object.get('display_name')
        migrated_name = migrated_resource.get('metadata', {}).get('name')
        
        # Validate recurring run ID preservation in annotations
        original_id = original_object.get('recurring_run_id')
        annotations = migrated_resource.get('metadata', {}).get('annotations', {})
        assert annotations.get('pipelines.kubeflow.org/original-id') == original_id, \
            f"Recurring run ID should be preserved in annotations: {original_id}"
  
def test_migration_single_pipeline_single_version(test_data, run_migration):
    """Test migration of a single pipeline with single version.
    
    Runs migration on a simple pipeline created in DB mode.
    Validates YAML files are generated with correct Kubernetes resources.
    Verifies original IDs are preserved in annotations and migrated pipeline spec matches original data.
   
    """
    output_dir = run_migration
    
    yaml_files = list(output_dir.glob("*.yaml"))
    for yaml_file in yaml_files:
        print(f"Generated file: {yaml_file}")
    
    # Verify YAML files were created
    assert len(yaml_files) > 0, "Migration should create YAML files"    
   
    migrated_resources = parse_yaml_files(output_dir)    
    original_pipeline = find_test_data_by_name(test_data, "pipelines", "hello-world")   
    pipelines = migrated_resources["Pipeline"]
    assert len(pipelines) >= 1, "Should have at least one Pipeline resource"    
    simple_pipeline_resources = [p for p in pipelines 
                                if "hello-world" in p.get("metadata", {}).get("name", "")]
    assert len(simple_pipeline_resources) >= 1, "Should have migrated hello-world pipeline"
    
    # Compare migrated pipeline with original
    migrated_pipeline = simple_pipeline_resources[0]    
    compare_complete_objects(migrated_pipeline, original_pipeline, "Pipeline")
    
    # Verify pipeline versions exist
    pipeline_versions = migrated_resources["PipelineVersion"]
    simple_versions = [v for v in pipeline_versions 
                      if "hello-world" in v.get("metadata", {}).get("name", "")]
    assert len(simple_versions) >= 1, "Hello-world pipeline should have at least one version"
    
    # Validate version structure
    for version in simple_versions:       
        annotations = version.get('metadata', {}).get('annotations', {})
        original_version_id = annotations.get('pipelines.kubeflow.org/original-id')
        original_version = None
        for original in test_data.get('pipelines', []):
            # Only match pipeline versions (have pipeline_version_id)
            original_id = getattr(original, 'pipeline_version_id', None)
            if original_id == original_version_id:
                original_version = original
                break
        
        if original_version:
            compare_complete_objects(version, original_version, "PipelineVersion")


def test_migration_single_pipeline_multiple_versions(test_data, run_migration):
    """Test migration of pipeline with multiple versions having mixed specifications.
    
    Runs migration on pipeline with multiple versions (some identical, some different specs).
    Validates multiple pipeline versions are correctly exported with original ID annotations.
    Verifies pipeline-version relationships are preserved and specs are handled properly.
    
    """
    output_dir = run_migration
    
    yaml_files = list(output_dir.glob("*.yaml"))
    for yaml_file in yaml_files:
        print(f"Generated file: {yaml_file}")
    
    assert len(yaml_files) > 0, "Migration should create YAML files"    
    
    migrated_resources = parse_yaml_files(output_dir)
    original_pipeline = find_test_data_by_name(test_data, "pipelines", "add-numbers")
    pipelines = migrated_resources["Pipeline"]
    complex_pipeline_resources = [p for p in pipelines 
                                if "add-numbers" in p.get("metadata", {}).get("name", "")]
    assert len(complex_pipeline_resources) >= 1, "Should have migrated add-numbers pipeline"
    
    # Compare migrated pipeline with original
    migrated_pipeline = complex_pipeline_resources[0]    
    compare_complete_objects(migrated_pipeline, original_pipeline, "Pipeline")
    
    pipeline_versions = migrated_resources["PipelineVersion"]    
    
    # Find versions that belong to add-numbers pipeline
    complex_versions = []
    for version in pipeline_versions:        
        pipeline_name = version.get('spec', {}).get('pipelineSpec', {}).get('pipelineInfo', {}).get('name', '')
        if pipeline_name == 'add-numbers':
            complex_versions.append(version)
    assert len(complex_versions) >= 3, f"add-numbers pipeline should have at least 3 versions, found {len(complex_versions)}"
    
    print(f"Found {len(complex_versions)} versions for add-numbers pipeline")
    
    # Validate version structure and compare with originals
    for version in complex_versions:       
        annotations = version.get('metadata', {}).get('annotations', {})
        original_version_id = annotations.get('pipelines.kubeflow.org/original-id')
        original_version = None
        for original in test_data.get('pipelines', []):            
            # Only match pipeline versions (have pipeline_version_id)
            original_id = getattr(original, 'pipeline_version_id', None)
            if original_id == original_version_id:
                original_version = original
                break
        
        if original_version:
            compare_complete_objects(version, original_version, "PipelineVersion")
  
def test_migration_multiple_pipelines_multiple_versions(test_data, run_migration):
    """Test migration of multiple pipelines with multiple versions.
    
    Runs migration on multiple pipelines where one has single version and another has multiple versions.
    Validates all pipelines and their versions are exported with correct relationships.
    Verifies pipeline-version associations are preserved and cross-pipeline relationships are handled correctly.
    
    """
    output_dir = run_migration
    
    yaml_files = list(output_dir.glob("*.yaml"))
    for yaml_file in yaml_files:
        print(f"Generated file: {yaml_file}")
    
    assert len(yaml_files) > 0, "Migration should create YAML files"
    
    migrated_resources = parse_yaml_files(output_dir)
    pipelines = migrated_resources["Pipeline"]
    pipeline_versions = migrated_resources["PipelineVersion"]
    
    assert len(pipelines) >= 2, "Should have at least 2 pipelines"
    assert len(pipeline_versions) >= 4, "Should have at least 4 pipeline versions (hello-world: 1 + add-numbers: 3)"
   
    # Validate hello-world pipeline and its single version
    hello_world_pipelines = [p for p in pipelines if "hello-world" in p.get("metadata", {}).get("name", "")]
    assert len(hello_world_pipelines) >= 1, "Should have hello-world pipeline"    
   
    hello_world_versions = []
    for version in pipeline_versions:
        pipeline_name = version.get('spec', {}).get('pipelineSpec', {}).get('pipelineInfo', {}).get('name', '')
        if pipeline_name == 'echo':
            hello_world_versions.append(version)
    assert len(hello_world_versions) >= 1, "Hello-world should have at least 1 version"
    
    # Validate add-numbers pipeline and its multiple versions
    add_numbers_pipelines = [p for p in pipelines if "add-numbers" in p.get("metadata", {}).get("name", "")]
    assert len(add_numbers_pipelines) >= 1, "Should have add-numbers pipeline"
    
    # Find add-numbers versions
    add_numbers_versions = []
    for version in pipeline_versions:
        pipeline_name = version.get('spec', {}).get('pipelineSpec', {}).get('pipelineInfo', {}).get('name', '')
        if pipeline_name == 'add-numbers':
            add_numbers_versions.append(version)
    assert len(add_numbers_versions) >= 3, f"Add-numbers should have at least 3 versions, found {len(add_numbers_versions)}"
   
    # Validate test pipelines against original data
    test_pipelines = [p for p in pipelines 
                     if any(test_name in p.get("metadata", {}).get("name", "") 
                           for test_name in ["hello-world", "add-numbers"])]
    
    for pipeline in test_pipelines:
        pipeline_name = pipeline.get("metadata", {}).get("name", "")
        original_pipeline = find_test_data_by_name(test_data, "pipelines", pipeline_name)        
            
        compare_complete_objects(pipeline, original_pipeline, "Pipeline")
    
    # Validate test pipeline versions against original data
    test_versions = hello_world_versions + add_numbers_versions
    
    for version in test_versions:              
        annotations = version.get('metadata', {}).get('annotations', {})
        original_version_id = annotations.get('pipelines.kubeflow.org/original-id')
        original_version = None
        for original in test_data.get('pipelines', []):
            # Only match pipeline versions (have pipeline_version_id)
            if getattr(original, 'pipeline_version_id', None) == original_version_id:
                original_version = original
                break
        
        if original_version:
            compare_complete_objects(version, original_version, "PipelineVersion")
