from model.src.sagemaker_model_spec import SageMakerCreateModelSpec
import unittest


class CreateModelSpecTestCase(unittest.TestCase):

    REQUIRED_ARGS = [
        "--region",
        "us-west-2",
        "--model_name",
        "model_test",
        "--role",
        "arn:aws:iam::123456789012:user/Development/product_1234/*",
        "--image",
        "test-image",
        "--model_artifact_url",
        "s3://fake-bucket/model_artifact",
    ]

    def test_minimum_required_args(self):
        # Will raise if the inputs are incorrect
        spec = SageMakerCreateModelSpec(self.REQUIRED_ARGS)
