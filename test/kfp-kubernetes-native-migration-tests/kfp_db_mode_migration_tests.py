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

import json
import os
import sys
import time
import tempfile
import unittest
import subprocess
import requests
from pathlib import Path
from unittest.mock import patch

# Add the tools directory to path to import migration module
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '../../tools/k8s-native'))

from migration import migrate

KFP_ENDPOINT = os.environ.get('KFP_ENDPOINT', 'http://localhost:8888')

class TestMigrationIntegration(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        """Set up test environment."""
        cls.temp_dir = tempfile.mkdtemp()
        
        # Load test data created by create_test_pipelines.py
        test_data_file = Path("migration_test_data.json")
        if test_data_file.exists():
            with open(test_data_file) as f:
                cls.test_data = json.load(f)
        else:
            cls.test_data = {"pipelines": [], "experiments": [], "runs": [], "recurring_runs": []}

    @classmethod
    def tearDownClass(cls):
        """Clean up test environment."""
        import shutil
        shutil.rmtree(cls.temp_dir)

    def setUp(self):
        """Set up for each test."""
        # Use a shared persistent directory that K8s mode tests can access
        shared_migration_base = Path("/tmp/kfp_shared_migration_outputs")
        shared_migration_base.mkdir(exist_ok=True)
        
        self.output_dir = shared_migration_base / f"migration_output_{self._testMethodName}"
        self.output_dir.mkdir(exist_ok=True)
     
        # Write the migration output directory to a shared location for K8s mode tests
        migration_info_file = Path("/tmp/kfp_migration_output_dir.txt")
        with open(migration_info_file, "w") as f:
            f.write(str(self.output_dir))

    def test_migration_single_pipeline_single_version(self): 
        """Test that the migration script correctly exports a single pipeline with single version from DB mode to Kubernetes native format"""
        
        with patch('sys.argv', [
            'migration.py',
            '--kfp-server-host', KFP_ENDPOINT,
            '--output', str(self.output_dir),
            '--namespace', 'kubeflow'
        ]):
            migrate()        
        
        yaml_files = list(self.output_dir.glob("*.yaml"))        
        for yaml_file in yaml_files:
            print(f"Generated file: {yaml_file}")
        
        self.assertGreater(len(yaml_files), 0, "Migration should create YAML files")        
        
        pipeline_files = [f for f in yaml_files if "simple-pipeline" in f.name]
        self.assertGreaterEqual(len(pipeline_files), 1, "Should have files for simple pipeline")        
        
        for yaml_file in yaml_files:
            content = yaml_file.read_text()
            if "kind: Pipeline" in content or "kind: PipelineVersion" in content:
                self.assertIn("pipelines.kubeflow.org/original-id", content, "Should have original ID annotation")

    def test_migration_single_pipeline_multiple_versions_same_spec(self):
        """Test that the migration script correctly exports a single pipeline with multiple versions that have the same specification"""
        
        with patch('sys.argv', [
            'migration.py',
            '--kfp-server-host', KFP_ENDPOINT,
            '--output', str(self.output_dir),
            '--namespace', 'kubeflow'
        ]):
            migrate()
                
        yaml_files = list(self.output_dir.glob("*.yaml"))        
        for yaml_file in yaml_files:
            print(f"Generated file: {yaml_file}")
        
        self.assertGreater(len(yaml_files), 0, "Migration should create YAML files")        
        
        version_count = len([f for f in yaml_files if "kind: PipelineVersion" in f.read_text()])
        self.assertGreaterEqual(version_count, 2, "Should have at least 2 pipeline versions")        
        
        for yaml_file in yaml_files:
            content = yaml_file.read_text()
            if "kind: PipelineVersion" in content:
                self.assertIn("pipelines.kubeflow.org/original-id", content, "Should have original ID annotation")

    def test_migration_single_pipeline_multiple_versions_different_specs(self):
        """Test that the migration script correctly exports a single pipeline with multiple versions that have different specifications"""
        
        with patch('sys.argv', [
            'migration.py',
            '--kfp-server-host', KFP_ENDPOINT,
            '--output', str(self.output_dir),
            '--namespace', 'kubeflow'
        ]):
            migrate()        
        
        yaml_files = list(self.output_dir.glob("*.yaml"))
        for yaml_file in yaml_files:
            print(f"Generated file: {yaml_file}")
        
        self.assertGreater(len(yaml_files), 0, "Migration should create YAML files")        
       
        complex_pipeline_files = [f for f in yaml_files if "complex-pipeline" in f.name]
        self.assertEqual(len(complex_pipeline_files), 1, "Should have one file for complex pipeline")        
       
        # Check that the complex pipeline file contains multiple versions
        complex_pipeline_file = complex_pipeline_files[0]
        content = complex_pipeline_file.read_text()
        version_count = content.count("kind: PipelineVersion")
        self.assertGreaterEqual(version_count, 2, "Complex pipeline should have multiple versions in the file")        
        
        # Check for original ID annotations
        self.assertIn("pipelines.kubeflow.org/original-id", content, "Should have original ID annotation")

    def test_migration_multiple_pipelines_single_version_each(self):
        """Test that the migration script correctly exports multiple pipelines, each with a single version"""
       
        with patch('sys.argv', [
            'migration.py',
            '--kfp-server-host', KFP_ENDPOINT,
            '--output', str(self.output_dir),
            '--namespace', 'kubeflow'
        ]):
            migrate()        
        
        yaml_files = list(self.output_dir.glob("*.yaml"))
        for yaml_file in yaml_files:
            print(f"Generated file: {yaml_file}")
        
        self.assertGreater(len(yaml_files), 0, "Migration should create YAML files")        
       
        pipeline_count = len([f for f in yaml_files if "kind: Pipeline" in f.read_text()])
        version_count = len([f for f in yaml_files if "kind: PipelineVersion" in f.read_text()])        
        
        self.assertGreaterEqual(pipeline_count, 2, "Should have at least 2 pipelines")
        self.assertGreaterEqual(version_count, 2, "Should have at least 2 pipeline versions")

    def test_migration_multiple_pipelines_multiple_versions_different_specs(self):
        """Test that the migration script correctly exports multiple pipelines with multiple versions having different specifications"""
        
        with patch('sys.argv', [
            'migration.py',
            '--kfp-server-host', KFP_ENDPOINT,
            '--output', str(self.output_dir),
            '--namespace', 'kubeflow'
        ]):
            migrate()        
        
        yaml_files = list(self.output_dir.glob("*.yaml"))
        for yaml_file in yaml_files:
            print(f"Generated file: {yaml_file}")
        
        self.assertGreater(len(yaml_files), 0, "Migration should create YAML files")     
      
        pipeline_files = [f for f in yaml_files if "simple-pipeline" in f.name or "complex-pipeline" in f.name]
        self.assertGreaterEqual(len(pipeline_files), 2, "Should have files for both pipelines")       
       
        # Check that complex pipeline file contains multiple versions
        complex_pipeline_files = [f for f in yaml_files if "complex-pipeline" in f.name]
        self.assertEqual(len(complex_pipeline_files), 1, "Should have one file for complex pipeline")
        
        complex_pipeline_file = complex_pipeline_files[0]
        content = complex_pipeline_file.read_text()
        version_count = content.count("kind: PipelineVersion")
        self.assertGreaterEqual(version_count, 2, "Complex pipeline should have multiple versions in the file")   
   
if __name__ == '__main__':
    unittest.main()
