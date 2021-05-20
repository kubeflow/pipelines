from hyperparameter_tuning.src.sagemaker_tuning_spec import SageMakerTuningSpec
import unittest


class TuningSpecTestCase(unittest.TestCase):

    REQUIRED_ARGS = [
        "--region",
        "us-west-2",
        "--role",
        "arn:aws:iam::123456789012:user/Development/product_1234/*",
        "--image",
        "test-image",
        "--metric_name",
        "test-metric",
        "--metric_type",
        "Maximize",
        "--channels",
        '[{"ChannelName": "train", "DataSource": {"S3DataSource":{"S3Uri": "s3://fake-bucket/data","S3DataType":"S3Prefix","S3DataDistributionType": "FullyReplicated"}},"ContentType":"","CompressionType": "None","RecordWrapperType":"None","InputMode": "File"}]',
        "--output_location",
        "test-output-location",
        "--max_num_jobs",
        "5",
        "--max_parallel_jobs",
        "2",
    ]

    def test_minimum_required_args(self):
        # Will raise if the inputs are incorrect
        spec = SageMakerTuningSpec(self.REQUIRED_ARGS)
