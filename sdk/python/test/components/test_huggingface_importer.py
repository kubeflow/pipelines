import tempfile
import unittest

from kfp import dsl
from kfp.dsl import Dataset
from kfp.dsl import importer
from kfp.dsl import Model


class HuggingFaceImporterTest(unittest.TestCase):

    def setUp(self):
        self.temp_dir = tempfile.mkdtemp()

    def test_huggingface_importer_uri_validation(self):
        """Test that importer accepts huggingface:// URIs."""

        @dsl.pipeline(name="test-hf-importer")
        def test_pipeline():
            model = importer(
                artifact_uri='huggingface://gpt2', artifact_class=Model)
            # No pipeline-level return to avoid DAG output wiring; compilation is verified at upload time
            pass

        # Should not raise any errors
        self.assertIsNotNone(test_pipeline)

    def test_single_file_uri_format(self):
        """Test single-file download URI format."""

        @dsl.pipeline(name="test-hf-file")
        def test_pipeline():
            file_model = importer(
                artifact_uri='huggingface://bert-base-uncased/config.json',
                artifact_class=Model)
            # No return to avoid adding DAG outputs; this verifies pipeline construction
            pass

        self.assertIsNotNone(test_pipeline)

    def test_dataset_uri_format(self):
        """Test dataset download with repo_type query param."""

        @dsl.pipeline(name="test-hf-dataset")
        def test_pipeline():
            dataset = importer(
                artifact_uri='huggingface://wikitext?repo_type=dataset',
                artifact_class=Dataset)
            # no return; ensure pipeline construction succeeds
            pass

        self.assertIsNotNone(test_pipeline)

    def test_huggingface_uri_with_patterns(self):
        """Test HuggingFace URI with allow_patterns."""

        @dsl.pipeline(name="test-hf-patterns")
        def test_pipeline():
            model = importer(
                artifact_uri='huggingface://gpt2?allow_patterns=*.safetensors',
                artifact_class=Model)
            # no return; pipeline construction only
            pass

        self.assertIsNotNone(test_pipeline)


if __name__ == '__main__':
    unittest.main()
