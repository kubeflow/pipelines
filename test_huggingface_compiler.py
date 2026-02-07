#!/usr/bin/env python3
"""Test for adding HuggingFace pipeline to backend compiler test suite.

Notes:
- This test verifies pipeline compilation and should remain deterministic.
- When updating compiled golden files, save them under `test_data/compiled-workflows/valid/huggingface/` and update CI goldens accordingly.
"""

import os
import tempfile
import unittest
from kfp import dsl, compiler


class TestHuggingFaceCompilerIntegration(unittest.TestCase):

    def test_huggingface_importer_compilation(self):
        """Test that HuggingFace importer pipelines compile correctly."""
        
        @dsl.pipeline(name='test-hf-compilation')
        def huggingface_compilation_test():
            """Test pipeline for HuggingFace compiler integration."""
            
            # Test basic repo import
            gpt2_model = dsl.importer(
                artifact_uri="huggingface://gpt2",
                artifact_class=dsl.Model,
                reimport=False
            ).output
            
            # Test repo with revision
            gpt2_main = dsl.importer(
                artifact_uri="huggingface://gpt2/main",
                artifact_class=dsl.Model,
                reimport=False
            ).output
            
            # Test dataset with query params
            squad_dataset = dsl.importer(
                artifact_uri="huggingface://squad?repo_type=dataset",
                artifact_class=dsl.Dataset,
                reimport=False
            ).output
            
            # Test specific file
            config_file = dsl.importer(
                artifact_uri="huggingface://gpt2/config.json",
                artifact_class=dsl.Artifact,
                reimport=False
            ).output
            
            # Test complex query params
            filtered_model = dsl.importer(
                artifact_uri="huggingface://gpt2?allow_patterns=*.bin&ignore_patterns=*.safetensors",
                artifact_class=dsl.Model,
                reimport=False
            ).output
        
        # Test compilation to standard format
        with tempfile.NamedTemporaryFile(mode='w', suffix='.yaml', delete=False) as f:
            try:
                compiler.Compiler().compile(
                    pipeline_func=huggingface_compilation_test,
                    package_path=f.name
                )
                
                # Check that file was created and has content
                self.assertTrue(os.path.exists(f.name))
                with open(f.name, 'r') as compiled_file:
                    content = compiled_file.read()
                    
                    # Verify HuggingFace URIs are preserved in compilation
                    self.assertIn('huggingface://gpt2', content)
                    self.assertIn('huggingface://squad?repo_type=dataset', content)
                    self.assertIn('huggingface://gpt2/config.json', content)
                    self.assertIn('allow_patterns=*.bin', content)
                    self.assertIn('ignore_patterns=*.safetensors', content)
                    
                    # Verify importer components are present
                    self.assertIn('comp-importer', content)
                    self.assertIn('exec-importer', content)
                    
            finally:
                os.unlink(f.name)

    def test_huggingface_kubernetes_manifest_compilation(self):
        """Test HuggingFace pipeline compiles to Kubernetes manifest format."""
        
        @dsl.pipeline(name='test-hf-k8s')
        def huggingface_k8s_test():
            model = dsl.importer(
                artifact_uri="huggingface://distilbert-base-uncased",
                artifact_class=dsl.Model,
                reimport=False
            ).output
        
        # Test Kubernetes manifest compilation
        with tempfile.NamedTemporaryFile(mode='w', suffix='.yaml', delete=False) as f:
            try:
                compiler.Compiler().compile(
                    pipeline_func=huggingface_k8s_test,
                    package_path=f.name,
                    kubernetes_manifest_format=True
                )
                
                # Check that file was created and has Kubernetes content
                self.assertTrue(os.path.exists(f.name))
                with open(f.name, 'r') as compiled_file:
                    content = compiled_file.read()
                    
                    # Verify Kubernetes manifest structure
                    self.assertIn('apiVersion: pipelines.kubeflow.org', content)
                    self.assertIn('kind: PipelineVersion', content)
                    self.assertIn('huggingface://distilbert-base-uncased', content)
                    
            finally:
                os.unlink(f.name)


if __name__ == "__main__":
    print("Running HuggingFace compiler integration tests...")
    unittest.main(verbosity=2)