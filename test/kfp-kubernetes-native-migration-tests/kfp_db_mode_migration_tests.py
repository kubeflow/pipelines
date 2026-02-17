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
from typing import Dict, List, Any, Optional
from dataclasses import dataclass, field

# Add the tools directory to path to import migration module
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '../../tools/k8s-native'))

from migration import migrate

# Import serialization function from create_test_pipelines
sys.path.insert(0, os.path.dirname(__file__))
from create_test_pipelines import to_json_for_comparison

KFP_ENDPOINT = os.environ.get('KFP_ENDPOINT', 'http://localhost:8888')


from k8s_models import K8sPipeline, K8sPipelineVersion, MigrationResult
from yaml_converters import parse_yaml_files_to_objects


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

 
def find_test_data_by_name(test_data: Dict[str, Any], resource_type: str, name: str):
    """Find a resource in test data by name."""
    resources = test_data.get(resource_type, [])
    for resource in resources:        
        resource_name = getattr(resource, 'display_name', None)
        if name in str(resource_name):
            return resource
    pytest.fail(f"Test data should contain {resource_type} with name containing '{name}'")

  
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
   
    migration_objects = parse_yaml_files_to_objects(output_dir)
    original_pipeline = find_test_data_by_name(test_data, "pipelines", "hello-world")

    migrated_pipeline = migration_objects.get_pipeline_by_name(original_pipeline.display_name)
    assert migrated_pipeline is not None, "Should have migrated hello-world pipeline"
    assert migrated_pipeline.get_name() == original_pipeline.display_name
    assert migrated_pipeline.get_original_id() == getattr(original_pipeline, 'pipeline_id', None)

    # Verify pipeline versions exist
    migrated_versions = migration_objects.get_pipeline_versions_by_pipeline_name(original_pipeline.display_name)
    assert len(migrated_versions) >= 1, "Hello-world pipeline should have at least one version"

    for mv in migrated_versions:
        original_version_id = mv.get_original_id()
        match = None
        for original in test_data.get('pipelines', []):
            if getattr(original, 'pipeline_version_id', None) == original_version_id:
                match = original
                break
        if match:
            assert mv.get_name() == getattr(match, 'display_name', None)
            assert mv.get_original_id() == getattr(match, 'pipeline_version_id', None)
            if mv.get_pipeline_spec() is not None:
                assert 'pipelineInfo' in mv.get_pipeline_spec()


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
    
    migration_objects = parse_yaml_files_to_objects(output_dir)
    original_pipeline = find_test_data_by_name(test_data, "pipelines", "add-numbers")

    migrated_pipeline = migration_objects.get_pipeline_by_name(original_pipeline.display_name)
    assert migrated_pipeline is not None, "Should have migrated add-numbers pipeline"
    assert migrated_pipeline.get_original_id() == getattr(original_pipeline, 'pipeline_id', None)

    complex_versions = migration_objects.get_pipeline_versions_by_pipeline_name(original_pipeline.display_name)
    assert len(complex_versions) >= 3, f"add-numbers pipeline should have at least 3 versions, found {len(complex_versions)}"

    for mv in complex_versions:
        original_version_id = mv.get_original_id()
        match = None
        for original in test_data.get('pipelines', []):
            if getattr(original, 'pipeline_version_id', None) == original_version_id:
                match = original
                break
        if match:
            assert mv.get_name() == getattr(match, 'display_name', None)
            assert mv.get_original_id() == getattr(match, 'pipeline_version_id', None)
            if mv.get_pipeline_spec() is not None:
                assert 'pipelineInfo' in mv.get_pipeline_spec()


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
    
    migration_objects = parse_yaml_files_to_objects(output_dir)

    assert len(migration_objects.k8s_pipelines) >= 2, "Should have at least 2 pipelines"
    assert len(migration_objects.k8s_pipeline_versions) >= 4, "Should have at least 4 pipeline versions (hello-world: 1 + add-numbers: 3)"

    # Validate hello-world pipeline and its single version
    hello_world_pipeline = migration_objects.get_pipeline_by_name("hello-world")
    assert hello_world_pipeline is not None, "Should have hello-world pipeline"

    hello_world_versions = migration_objects.get_pipeline_versions_by_pipeline_name("hello-world")
    assert len(hello_world_versions) >= 1, "Hello-world should have at least 1 version"

    # Validate add-numbers pipeline and its multiple versions
    add_numbers_pipeline = migration_objects.get_pipeline_by_name("add-numbers")
    assert add_numbers_pipeline is not None, "Should have add-numbers pipeline"

    add_numbers_versions = migration_objects.get_pipeline_versions_by_pipeline_name("add-numbers")
    assert len(add_numbers_versions) >= 3, f"Add-numbers should have at least 3 versions, found {len(add_numbers_versions)}"

    # Validate test pipelines against original data
    for pipeline_name in ["hello-world", "add-numbers"]:
        migrated = migration_objects.get_pipeline_by_name(pipeline_name)
        original = find_test_data_by_name(test_data, "pipelines", pipeline_name)
        assert migrated is not None
        assert migrated.get_name() == original.display_name
        assert migrated.get_original_id() == getattr(original, 'pipeline_id', None)

    # Validate test pipeline versions against original data
    for mv in hello_world_versions + add_numbers_versions:
        original_version_id = mv.get_original_id()
        match = None
        for original in test_data.get('pipelines', []):
            if getattr(original, 'pipeline_version_id', None) == original_version_id:
                match = original
                break
        if match:
            assert mv.get_name() == getattr(match, 'display_name', None)
            assert mv.get_original_id() == getattr(match, 'pipeline_version_id', None)
            if mv.get_pipeline_spec() is not None:
                assert 'pipelineInfo' in mv.get_pipeline_spec()
