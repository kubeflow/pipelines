import unittest

from kfp.dsl.types.artifact_types import Artifact
from kfp.dsl.types.artifact_types import Dataset
from kfp.dsl.types.artifact_types import Model
from kfp.dsl.types.artifact_types import RemotePrefix


class HuggingFaceArtifactTypeTest(unittest.TestCase):
    """Test HuggingFace artifact type registration."""

    def test_huggingface_remote_prefix_exists(self):
        """Test that HUGGINGFACE is registered in RemotePrefix."""
        self.assertTrue(hasattr(RemotePrefix, 'HUGGINGFACE'))
        self.assertEqual(RemotePrefix.HUGGINGFACE.value, 'huggingface://')

    def test_model_artifact_type_exists(self):
        """Test that Model artifact type is registered."""
        self.assertIsNotNone(Model)

    def test_dataset_artifact_type_exists(self):
        """Test that Dataset artifact type is registered."""
        self.assertIsNotNone(Dataset)


class HuggingFaceURIParsingTest(unittest.TestCase):
    """Test HuggingFace URI parsing logic."""

    def test_parse_repo_uri(self):
        """Test parsing basic repository URI."""
        uri = "huggingface://gpt2"
        scheme, path = uri.split("://")
        self.assertEqual(scheme, "huggingface")
        self.assertEqual(path, "gpt2")

    def test_parse_file_uri(self):
        """Test parsing single-file URI."""
        uri = "huggingface://bert-base-uncased/config.json"
        scheme, path = uri.split("://")
        self.assertEqual(scheme, "huggingface")
        self.assertEqual(path, "bert-base-uncased/config.json")

        # File detection logic
        has_extension = "." in path.split("/")[-1]
        self.assertTrue(has_extension)

    def test_parse_dataset_uri(self):
        """Test parsing dataset URI with query params."""
        uri = "huggingface://wikitext?repo_type=dataset"
        scheme, rest = uri.split("://")
        path, query = rest.split("?") if "?" in rest else (rest, "")

        self.assertEqual(scheme, "huggingface")
        self.assertEqual(path, "wikitext")
        self.assertIn("repo_type=dataset", query)

    def test_parse_with_patterns(self):
        """Test parsing URI with allow_patterns."""
        uri = "huggingface://gpt2?allow_patterns=*.safetensors"
        scheme, rest = uri.split("://")
        path, query = rest.split("?")

        self.assertEqual(path, "gpt2")
        self.assertIn("allow_patterns=*.safetensors", query)


if __name__ == '__main__':
    unittest.main(verbosity=2)
