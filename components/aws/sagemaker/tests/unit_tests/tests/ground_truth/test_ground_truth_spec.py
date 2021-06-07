from ground_truth.src.sagemaker_ground_truth_spec import SageMakerGroundTruthSpec
import unittest


class GroundTruthSpecTestCase(unittest.TestCase):

    REQUIRED_ARGS = [
        "--region",
        "us-west-2",
        "--role",
        "arn:aws:iam::123456789012:user/Development/product_1234/*",
        "--manifest_location",
        "s3://fake-bucket/manifest",
        "--output_location",
        "s3://fake-bucket/output",
        "--task_type",
        "fake-task",
        "--worker_type",
        "fake_worker",
        "--ui_template",
        "s3://fake-bucket/ui_template",
        "--title",
        "fake-image-labelling-work",
        "--description",
        "fake job",
        "--num_workers_per_object",
        "1",
        "--time_limit",
        "180",
    ]

    def test_minimum_required_args(self):
        # Will raise if the inputs are incorrect
        spec = SageMakerGroundTruthSpec(self.REQUIRED_ARGS)
