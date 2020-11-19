from batch_transform.src.sagemaker_transform_spec import SageMakerTransformSpec
import unittest


class TransformSpecTestCase(unittest.TestCase):

    REQUIRED_ARGS = [
        "--region",
        "us-west-2",
        "--model_name",
        "model-test",
        "--input_location",
        "s3://fake-bucket/data",
        "--output_location",
        "s3://fake-bucket/output",
        "--instance_type",
        "ml.c5.18xlarge",
    ]

    def test_minimum_required_args(self):
        # Will raise if the inputs are incorrect
        spec = SageMakerTransformSpec(self.REQUIRED_ARGS)
