from rlestimator.src.sagemaker_rlestimator_spec import SageMakerRLEstimatorSpec
import unittest


class RLEstimatorSpecTestCase(unittest.TestCase):
    CUSTOM_IMAGE_ARGS = [
        "--region",
        "us-east-1",
        "--entry_point",
        "train-unity.py",
        "--source_dir",
        "s3://input_bucket_name/input_key",
        "--role",
        "arn:aws:iam::123456789012:user/Development/product_1234/*",
        "--image",
        "test-image",
        "--instance_type",
        "ml.m4.xlarge",
        "--instance_count",
        "1",
        "--volume_size",
        "50",
        "--max_run",
        "900",
        "--model_artifact_path",
        "test-path",
    ]

    TOOLKIT_IMAGE_ARGS = [
        "--region",
        "us-east-1",
        "--entry_point",
        "train-unity.py",
        "--source_dir",
        "s3://input_bucket_name/input_key",
        "--role",
        "arn:aws:iam::123456789012:user/Development/product_1234/*",
        "--toolkit",
        "ray",
        "--toolkit_version",
        "0.8.5",
        "--framework",
        "tensorflow",
        "--instance_type",
        "ml.m4.xlarge",
        "--instance_count",
        "1",
        "--volume_size",
        "50",
        "--max_run",
        "900",
        "--model_artifact_path",
        "test-path",
    ]

    def test_custom_image_args(self):
        # Will raise if the inputs are incorrect
        spec = SageMakerRLEstimatorSpec(self.CUSTOM_IMAGE_ARGS)

    def test_toolkit_image_args(self):
        # Will raise if the inputs are incorrect
        spec = SageMakerRLEstimatorSpec(self.TOOLKIT_IMAGE_ARGS)
