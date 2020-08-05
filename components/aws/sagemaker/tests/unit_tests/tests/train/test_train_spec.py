from train.src.sagemaker_training_spec import SageMakerTrainingSpec
import unittest


class TrainingSpecTestCase(unittest.TestCase):
    REQUIRED_ARGS = [
        "--region",
        "us-west-2",
        "--role",
        "arn:aws:iam::123456789012:user/Development/product_1234/*",
        "--image",
        "test-image",
        "--channels",
        '[{"ChannelName": "train", "DataSource": {"S3DataSource":{"S3Uri": "s3://fake-bucket/data","S3DataType":"S3Prefix","S3DataDistributionType": "FullyReplicated"}},"ContentType":"","CompressionType": "None","RecordWrapperType":"None","InputMode": "File"}]',
        "--instance_type",
        "ml.m4.xlarge",
        "--instance_count",
        "1",
        "--volume_size",
        "50",
        "--max_run_time",
        "3600",
        "--model_artifact_path",
        "test-path",
        "--model_artifact_url_output_path",
        "/tmp/model_artifact_url_output_path",
        "--job_name_output_path",
        "/tmp/job_name_output_path",
        "--training_image_output_path",
        "/tmp/training_image_output_path",
    ]

    def test_minimum_required_args(self):
        # Will raise if the inputs are incorrect
        spec = SageMakerTrainingSpec(self.REQUIRED_ARGS)
