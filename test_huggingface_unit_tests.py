#!/usr/bin/env python3
"""Unit tests for HuggingFace Hub importer functionality.

This file includes unit tests for importer behavior and a helper E2E pipeline that can be
compiled for manual or integration testing. Compiled artifacts (goldens) should be placed
under `test_data/compiled-workflows/valid/huggingface/` when updating expected outputs.

Prerequisites:
- Local development: Python 3.11+, kfp SDK installed in dev env
- For integration/E2E runs: a running Kind cluster and KFP API accessible
"""

import unittest
from unittest import mock
from kfp import dsl


class TestHuggingFaceImporter(unittest.TestCase):

    def test_basic_huggingface_importer(self):
        """Test basic HuggingFace model import."""
        # Test basic repo import
        model = dsl.importer(
            artifact_uri="huggingface://gpt2",
            artifact_class=dsl.Model,
            reimport=False
        ).output
        
        self.assertEqual(model.uri, "huggingface://gpt2")
        self.assertEqual(model._get_path(), "/huggingface/gpt2")

    def test_huggingface_importer_with_revision(self):
        """Test HuggingFace model import with specific revision."""
        model = dsl.importer(
            artifact_uri="huggingface://gpt2/main",
            artifact_class=dsl.Model,
            reimport=False
        ).output
        
        self.assertEqual(model.uri, "huggingface://gpt2/main")
        self.assertEqual(model._get_path(), "/huggingface/gpt2/main")

    def test_huggingface_importer_with_query_params(self):
        """Test HuggingFace import with query parameters."""
        dataset = dsl.importer(
            artifact_uri="huggingface://squad?repo_type=dataset",
            artifact_class=dsl.Dataset,
            reimport=False
        ).output
        
        self.assertEqual(dataset.uri, "huggingface://squad?repo_type=dataset")
        # Query params should be stripped from local path
        self.assertEqual(dataset._get_path(), "/huggingface/squad")

    def test_huggingface_file_import(self):
        """Test importing specific file from HuggingFace repo."""
        artifact = dsl.importer(
            artifact_uri="huggingface://gpt2/tokenizer.json",
            artifact_class=dsl.Artifact,
            reimport=False
        ).output
        
        self.assertEqual(artifact.uri, "huggingface://gpt2/tokenizer.json")
        self.assertEqual(artifact._get_path(), "/huggingface/gpt2/tokenizer.json")

    def test_huggingface_org_repo_import(self):
        """Test importing from organization/repo format."""
        model = dsl.importer(
            artifact_uri="huggingface://meta-llama/Llama-2-7b",
            artifact_class=dsl.Model,
            reimport=False
        ).output
        
        self.assertEqual(model.uri, "huggingface://meta-llama/Llama-2-7b")
        self.assertEqual(model._get_path(), "/huggingface/meta-llama/Llama-2-7b")

    def test_huggingface_complex_query_params(self):
        """Test HuggingFace import with complex query parameters."""
        model = dsl.importer(
            artifact_uri="huggingface://microsoft/DialoGPT-medium?repo_type=model&allow_patterns=*.bin&ignore_patterns=*.safetensors",
            artifact_class=dsl.Model,
            reimport=False
        ).output
        
        # URI should preserve query params
        self.assertIn("repo_type=model", model.uri)
        self.assertIn("allow_patterns=*.bin", model.uri)
        self.assertIn("ignore_patterns=*.safetensors", model.uri)
        
        # Local path should strip query params
        self.assertEqual(model._get_path(), "/huggingface/microsoft/DialoGPT-medium")

    def test_huggingface_path_conversion_round_trip(self):
        """Test local path to remote path conversion for HuggingFace."""
        from kfp.dsl.types.artifact_types import convert_local_path_to_remote_path
        
        # Test round-trip conversion
        test_cases = [
            '/huggingface/gpt2',
            '/huggingface/meta-llama/Llama-2-7b',
            '/huggingface/microsoft/DialoGPT-medium/v1.0'
        ]
        
        for local_path in test_cases:
            remote_path = convert_local_path_to_remote_path(local_path)
            expected_remote = remote_path
            self.assertTrue(remote_path.startswith('huggingface://'))
            
            # Create artifact and test conversion back
            artifact = dsl.Artifact()
            artifact.uri = remote_path
            converted_local = artifact._get_path()
            self.assertEqual(converted_local, local_path)


@dsl.pipeline(name='test-huggingface-e2e')
def test_huggingface_e2e_pipeline():
    """E2E test pipeline for HuggingFace functionality."""
    
    # Import various HuggingFace artifacts
    gpt2_model = dsl.importer(
        artifact_uri="huggingface://gpt2",
        artifact_class=dsl.Model,
        reimport=False
    ).output
    
    distilbert_model = dsl.importer(
        artifact_uri="huggingface://distilbert-base-uncased",
        artifact_class=dsl.Model,
        reimport=False  
    ).output
    
    squad_dataset = dsl.importer(
        artifact_uri="huggingface://squad?repo_type=dataset",
        artifact_class=dsl.Dataset,
        reimport=False
    ).output
    
    # Import specific files
    config_file = dsl.importer(
        artifact_uri="huggingface://gpt2/config.json",
        artifact_class=dsl.Artifact,
        reimport=False
    ).output
    
    # Test with complex query parameters
    filtered_model = dsl.importer(
        artifact_uri="huggingface://gpt2?allow_patterns=*.bin&ignore_patterns=*.safetensors",
        artifact_class=dsl.Model,
        reimport=False
    ).output


if __name__ == "__main__":
    # Run unit tests
    unittest.main(verbosity=2, exit=False)
    
    # Compile E2E test pipeline to a YAML artifact for manual/integration use
    from kfp import compiler
    print("\nCompiling E2E test pipeline to 'test_huggingface_e2e.yaml'...")
    compiler.Compiler().compile(
        pipeline_func=test_huggingface_e2e_pipeline,
        package_path="test_huggingface_e2e.yaml"
    )
    print("E2E test pipeline compiled to test_huggingface_e2e.yaml")